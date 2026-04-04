package jwtverifier

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Overland-East-Bay/trip-planner-api/internal/platform/auth/jwks_testutil"
	"github.com/Overland-East-Bay/trip-planner-api/internal/platform/config"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

func mintRS256JWTWithClaims(t *testing.T, kp jwks_testutil.Keypair, header map[string]any, claims map[string]any) string {
	t.Helper()

	hb, err := json.Marshal(header)
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}
	cb, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	enc := base64.RawURLEncoding
	signingInput := enc.EncodeToString(hb) + "." + enc.EncodeToString(cb)
	sum := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, kp.Private, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return signingInput + "." + enc.EncodeToString(sig)
}

func TestVerifier_Verify_AudienceArrayAndNbf(t *testing.T) {
	t.Parallel()

	jwksSrv, setKeys := jwks_testutil.NewRotatingJWKSServer()
	defer jwksSrv.Close()

	kp, _ := jwks_testutil.GenerateRSAKeypair("kid-1")
	setKeys([]jwks_testutil.Keypair{kp})

	clk := fixedClock{now: time.Unix(1700000000, 0)}
	cfg := config.JWTConfig{
		Issuer:                 "test-iss",
		Audience:               "test-aud",
		JWKSURL:                jwksSrv.URL,
		ClockSkew:              0,
		JWKSRefreshInterval:    10 * time.Minute,
		JWKSMinRefreshInterval: 0,
		HTTPTimeout:            2 * time.Second,
	}
	v := NewWithOptions(cfg, nil, clk)

	// aud as array should be accepted when it contains expected.
	jwtOK, err := jwks_testutil.MintRS256JWT(kp, cfg.Issuer, []string{"other", cfg.Audience}, "member-123", clk.Now(), 5*time.Minute, nil)
	if err != nil {
		t.Fatalf("MintRS256JWT: %v", err)
	}
	if sub, err := v.Verify(context.Background(), jwtOK); err != nil || sub != "member-123" {
		t.Fatalf("Verify(sub=%q, err=%v)", sub, err)
	}

	// nbf in the future should be rejected.
	nbfDelta := 1 * time.Minute
	jwtNBF, err := jwks_testutil.MintRS256JWT(kp, cfg.Issuer, cfg.Audience, "member-123", clk.Now(), 5*time.Minute, &nbfDelta)
	if err != nil {
		t.Fatalf("MintRS256JWT(nbf): %v", err)
	}
	if _, err := v.Verify(context.Background(), jwtNBF); err == nil {
		t.Fatalf("expected nbf token to be rejected")
	}
}

func TestVerifier_Verify_MissingExpOrSubRejected(t *testing.T) {
	t.Parallel()

	jwksSrv, setKeys := jwks_testutil.NewRotatingJWKSServer()
	defer jwksSrv.Close()

	kp, _ := jwks_testutil.GenerateRSAKeypair("kid-1")
	setKeys([]jwks_testutil.Keypair{kp})

	clk := fixedClock{now: time.Unix(1700000000, 0)}
	cfg := config.JWTConfig{
		Issuer:                 "test-iss",
		Audience:               "test-aud",
		JWKSURL:                jwksSrv.URL,
		ClockSkew:              0,
		JWKSRefreshInterval:    10 * time.Minute,
		JWKSMinRefreshInterval: 0,
		HTTPTimeout:            2 * time.Second,
	}
	v := NewWithOptions(cfg, nil, clk)

	header := map[string]any{"alg": "RS256", "typ": "JWT", "kid": kp.Kid}

	// Missing exp.
	jwtNoExp := mintRS256JWTWithClaims(t, kp, header, map[string]any{
		"iss": cfg.Issuer,
		"aud": cfg.Audience,
		"sub": "member-123",
	})
	if _, err := v.Verify(context.Background(), jwtNoExp); err == nil {
		t.Fatalf("expected missing-exp token to be rejected")
	}

	// Missing sub.
	jwtNoSub, err := jwks_testutil.MintRS256JWT(kp, cfg.Issuer, cfg.Audience, "", clk.Now(), 5*time.Minute, nil)
	if err != nil {
		t.Fatalf("MintRS256JWT: %v", err)
	}
	if _, err := v.Verify(context.Background(), jwtNoSub); err == nil {
		t.Fatalf("expected missing-sub token to be rejected")
	}
}

func TestVerifier_Verify_InvalidAlgOrKidShortCircuits(t *testing.T) {
	t.Parallel()

	// alg != RS256.
	h := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","kid":"kid-1","typ":"JWT"}`))
	c := base64.RawURLEncoding.EncodeToString([]byte(`{"iss":"x","aud":"y","sub":"z","exp":1700000001}`))
	s := base64.RawURLEncoding.EncodeToString([]byte("sig"))
	if _, err := New(config.JWTConfig{}).Verify(context.Background(), h+"."+c+"."+s); err == nil {
		t.Fatalf("expected error")
	}

	// kid empty.
	h2 := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","kid":"","typ":"JWT"}`))
	if _, err := New(config.JWTConfig{}).Verify(context.Background(), h2+"."+c+"."+s); err == nil {
		t.Fatalf("expected error")
	}
}

func TestVerifier_Verify_JWKSFetchErrorsRejectToken(t *testing.T) {
	t.Parallel()

	kp, _ := jwks_testutil.GenerateRSAKeypair("kid-1")
	clk := fixedClock{now: time.Unix(1700000000, 0)}

	jwt, err := jwks_testutil.MintRS256JWT(kp, "test-iss", "test-aud", "member-123", clk.Now(), 5*time.Minute, nil)
	if err != nil {
		t.Fatalf("MintRS256JWT: %v", err)
	}

	t.Run("non-2xx", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("nope"))
		}))
		defer srv.Close()

		cfg := config.JWTConfig{Issuer: "test-iss", Audience: "test-aud", JWKSURL: srv.URL, HTTPTimeout: 2 * time.Second}
		v := NewWithOptions(cfg, nil, clk)
		if _, err := v.Verify(context.Background(), jwt); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("invalid-json", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("{not-json"))
		}))
		defer srv.Close()

		cfg := config.JWTConfig{Issuer: "test-iss", Audience: "test-aud", JWKSURL: srv.URL, HTTPTimeout: 2 * time.Second}
		v := NewWithOptions(cfg, nil, clk)
		if _, err := v.Verify(context.Background(), jwt); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("no-keys", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"keys":[]}`))
		}))
		defer srv.Close()

		cfg := config.JWTConfig{Issuer: "test-iss", Audience: "test-aud", JWKSURL: srv.URL, HTTPTimeout: 2 * time.Second}
		v := NewWithOptions(cfg, nil, clk)
		if _, err := v.Verify(context.Background(), jwt); err == nil {
			t.Fatalf("expected error")
		}
	})
}
