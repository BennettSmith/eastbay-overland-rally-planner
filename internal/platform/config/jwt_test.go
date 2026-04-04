package config

import (
	"testing"
	"time"
)

func TestLoadJWTConfigFromEnv_MissingRequiredVars(t *testing.T) {
	// Ensure required vars are missing.
	t.Setenv("JWT_ISSUER", "")
	t.Setenv("JWT_AUDIENCE", "")
	t.Setenv("JWT_JWKS_URL", "")

	if _, err := LoadJWTConfigFromEnv(); err == nil {
		t.Fatalf("expected error")
	}
}

func TestLoadJWTConfigFromEnv_Defaults(t *testing.T) {
	t.Setenv("JWT_ISSUER", "iss")
	t.Setenv("JWT_AUDIENCE", "aud")
	t.Setenv("JWT_JWKS_URL", "https://example.com/jwks.json")

	cfg, err := LoadJWTConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadJWTConfigFromEnv: %v", err)
	}
	if cfg.Issuer != "iss" || cfg.Audience != "aud" || cfg.JWKSURL != "https://example.com/jwks.json" {
		t.Fatalf("cfg=%+v", cfg)
	}
	if cfg.ClockSkew != 30*time.Second {
		t.Fatalf("ClockSkew=%s want=30s", cfg.ClockSkew)
	}
	if cfg.JWKSRefreshInterval != 5*time.Minute {
		t.Fatalf("JWKSRefreshInterval=%s want=5m", cfg.JWKSRefreshInterval)
	}
	if cfg.JWKSMinRefreshInterval != 10*time.Second {
		t.Fatalf("JWKSMinRefreshInterval=%s want=10s", cfg.JWKSMinRefreshInterval)
	}
	if cfg.HTTPTimeout != 5*time.Second {
		t.Fatalf("HTTPTimeout=%s want=5s", cfg.HTTPTimeout)
	}
}

func TestLoadJWTConfigFromEnv_OverridesAndValidatesDurations(t *testing.T) {
	t.Setenv("JWT_ISSUER", "iss")
	t.Setenv("JWT_AUDIENCE", "aud")
	t.Setenv("JWT_JWKS_URL", "https://example.com/jwks.json")
	t.Setenv("JWT_CLOCK_SKEW", "1m")
	t.Setenv("JWT_JWKS_REFRESH_INTERVAL", "2m")
	t.Setenv("JWT_JWKS_MIN_REFRESH_INTERVAL", "3s")

	cfg, err := LoadJWTConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadJWTConfigFromEnv: %v", err)
	}
	if cfg.ClockSkew != time.Minute {
		t.Fatalf("ClockSkew=%s want=1m", cfg.ClockSkew)
	}
	if cfg.JWKSRefreshInterval != 2*time.Minute {
		t.Fatalf("JWKSRefreshInterval=%s want=2m", cfg.JWKSRefreshInterval)
	}
	if cfg.JWKSMinRefreshInterval != 3*time.Second {
		t.Fatalf("JWKSMinRefreshInterval=%s want=3s", cfg.JWKSMinRefreshInterval)
	}

	t.Run("invalid clock skew", func(t *testing.T) {
		t.Setenv("JWT_CLOCK_SKEW", "nope")
		if _, err := LoadJWTConfigFromEnv(); err == nil {
			t.Fatalf("expected error")
		}
	})
}
