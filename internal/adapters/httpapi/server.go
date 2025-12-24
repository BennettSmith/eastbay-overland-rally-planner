package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/oapi-codegen/nullable"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"eastbay-overland-rally-planner/internal/adapters/httpapi/oas"
	"eastbay-overland-rally-planner/internal/app/members"
	"eastbay-overland-rally-planner/internal/app/trips"
	"eastbay-overland-rally-planner/internal/domain"
	"eastbay-overland-rally-planner/internal/ports/out/idempotency"
)

// Server is the real HTTP adapter implementation. For endpoints not yet implemented,
// it embeds StrictUnimplemented.
type Server struct {
	StrictUnimplemented

	Members *members.Service
	Trips   *trips.Service
	Idem    idempotency.Store
}

func NewServer(membersSvc *members.Service, tripsSvc *trips.Service, idem idempotency.Store) *Server {
	return &Server{
		Members: membersSvc,
		Trips:   tripsSvc,
		Idem:    idem,
	}
}

func (s *Server) ListMembers(ctx context.Context, req oas.ListMembersRequestObject) (oas.ListMembersResponseObject, error) {
	sub, ok := SubjectFromContext(ctx)
	if !ok {
		return oas.ListMembers401JSONResponse{UnauthorizedJSONResponse: oas.UnauthorizedJSONResponse(oasError(ctx, "UNAUTHORIZED", "missing subject", nil))}, nil
	}
	// In v1, directory access requires the caller to have a provisioned member profile.
	if _, err := s.Members.GetMyMemberProfile(ctx, domain.SubjectID(sub)); err != nil {
		if isMemberNotProvisioned(err) {
			return oas.ListMembers401JSONResponse{UnauthorizedJSONResponse: oas.UnauthorizedJSONResponse(oasError(ctx, "MEMBER_NOT_PROVISIONED", "No member profile exists for the authenticated subject.", nil))}, nil
		}
		return nil, err
	}

	includeInactive := false
	if req.Params.IncludeInactive != nil {
		includeInactive = bool(*req.Params.IncludeInactive)
	}
	ms, err := s.Members.ListMembers(ctx, domain.SubjectID(sub), includeInactive)
	if err != nil {
		return nil, err
	}
	out := make([]oas.MemberDirectoryEntry, 0, len(ms))
	for _, m := range ms {
		out = append(out, oas.MemberDirectoryEntry{
			MemberId:    string(m.ID),
			DisplayName: m.DisplayName,
		})
	}
	return oas.ListMembers200JSONResponse{Members: out}, nil
}

func (s *Server) SearchMembers(ctx context.Context, req oas.SearchMembersRequestObject) (oas.SearchMembersResponseObject, error) {
	sub, ok := SubjectFromContext(ctx)
	if !ok {
		return oas.SearchMembers401JSONResponse{UnauthorizedJSONResponse: oas.UnauthorizedJSONResponse(oasError(ctx, "UNAUTHORIZED", "missing subject", nil))}, nil
	}
	// Require provisioned member (see note in ListMembers).
	if _, err := s.Members.GetMyMemberProfile(ctx, domain.SubjectID(sub)); err != nil {
		if isMemberNotProvisioned(err) {
			return oas.SearchMembers401JSONResponse{UnauthorizedJSONResponse: oas.UnauthorizedJSONResponse(oasError(ctx, "MEMBER_NOT_PROVISIONED", "No member profile exists for the authenticated subject.", nil))}, nil
		}
		return nil, err
	}

	ms, err := s.Members.SearchMembers(ctx, string(req.Params.Q))
	if err != nil {
		if ae := (*members.Error)(nil); errors.As(err, &ae) {
			switch ae.Status {
			case http.StatusUnprocessableEntity:
				return oas.SearchMembers422JSONResponse{UnprocessableEntityJSONResponse: oas.UnprocessableEntityJSONResponse(oasError(ctx, ae.Code, ae.Message, ae.Details))}, nil
			default:
				return nil, err
			}
		}
		return nil, err
	}
	out := make([]oas.MemberDirectoryEntry, 0, len(ms))
	for _, m := range ms {
		out = append(out, oas.MemberDirectoryEntry{
			MemberId:    string(m.ID),
			DisplayName: m.DisplayName,
		})
	}
	return oas.SearchMembers200JSONResponse{Members: out}, nil
}

func (s *Server) CreateMyMember(ctx context.Context, req oas.CreateMyMemberRequestObject) (oas.CreateMyMemberResponseObject, error) {
	sub, ok := SubjectFromContext(ctx)
	if !ok {
		return oas.CreateMyMember401JSONResponse{UnauthorizedJSONResponse: oas.UnauthorizedJSONResponse(oasError(ctx, "UNAUTHORIZED", "missing subject", nil))}, nil
	}
	if req.Body == nil {
		return oas.CreateMyMember422JSONResponse{UnprocessableEntityJSONResponse: oas.UnprocessableEntityJSONResponse(oasError(ctx, "VALIDATION_ERROR", "missing request body", nil))}, nil
	}

	in := members.CreateMyMemberInput{
		DisplayName: req.Body.DisplayName,
		Email:       string(req.Body.Email),
	}
	if req.Body.GroupAliasEmail.IsSpecified() {
		if req.Body.GroupAliasEmail.IsNull() {
			in.GroupAliasEmail = nil
		} else {
			v, err := req.Body.GroupAliasEmail.Get()
			if err == nil {
				s := string(v)
				in.GroupAliasEmail = &s
			}
		}
	}
	if req.Body.VehicleProfile != nil {
		in.VehicleProfile = vehicleProfilePatchFromOAS(*req.Body.VehicleProfile)
	}

	m, err := s.Members.CreateMyMember(ctx, domain.SubjectID(sub), in)
	if err != nil {
		if ae := (*members.Error)(nil); errors.As(err, &ae) {
			switch ae.Status {
			case http.StatusConflict:
				return oas.CreateMyMember409JSONResponse(oasError(ctx, ae.Code, ae.Message, ae.Details)), nil
			case http.StatusUnprocessableEntity:
				return oas.CreateMyMember422JSONResponse{UnprocessableEntityJSONResponse: oas.UnprocessableEntityJSONResponse(oasError(ctx, ae.Code, ae.Message, ae.Details))}, nil
			default:
				return nil, err
			}
		}
		return nil, err
	}

	return oas.CreateMyMember201JSONResponse{
		Member: memberProfileFromDomain(m),
	}, nil
}

func (s *Server) GetMyMemberProfile(ctx context.Context, _ oas.GetMyMemberProfileRequestObject) (oas.GetMyMemberProfileResponseObject, error) {
	sub, ok := SubjectFromContext(ctx)
	if !ok {
		return oas.GetMyMemberProfile401JSONResponse{UnauthorizedJSONResponse: oas.UnauthorizedJSONResponse(oasError(ctx, "UNAUTHORIZED", "missing subject", nil))}, nil
	}
	m, err := s.Members.GetMyMemberProfile(ctx, domain.SubjectID(sub))
	if err != nil {
		if ae := (*members.Error)(nil); errors.As(err, &ae) {
			switch ae.Status {
			case http.StatusNotFound:
				return oas.GetMyMemberProfile404JSONResponse(oasError(ctx, ae.Code, ae.Message, ae.Details)), nil
			default:
				return nil, err
			}
		}
		return nil, err
	}
	return oas.GetMyMemberProfile200JSONResponse{Member: memberProfileFromDomain(m)}, nil
}

func (s *Server) UpdateMyMemberProfile(ctx context.Context, req oas.UpdateMyMemberProfileRequestObject) (oas.UpdateMyMemberProfileResponseObject, error) {
	sub, ok := SubjectFromContext(ctx)
	if !ok {
		return oas.UpdateMyMemberProfile401JSONResponse{UnauthorizedJSONResponse: oas.UnauthorizedJSONResponse(oasError(ctx, "UNAUTHORIZED", "missing subject", nil))}, nil
	}
	if req.Body == nil {
		return oas.UpdateMyMemberProfile422JSONResponse{UnprocessableEntityJSONResponse: oas.UnprocessableEntityJSONResponse(oasError(ctx, "VALIDATION_ERROR", "missing request body", nil))}, nil
	}

	// Idempotency handling (v1):
	// - Replay if same actor+key+route+bodyHash
	// - Reject if same actor+key+route with different bodyHash (409)
	bodyHash, err := hashUpdateMyMemberProfileBody(*req.Body)
	if err != nil {
		return nil, err
	}
	idemKey := idempotency.Key(req.Params.IdempotencyKey)
	metaFP := idempotency.Fingerprint{
		Key:      idemKey,
		Subject:  domain.SubjectID(sub),
		Method:   http.MethodPatch,
		Route:    "/members/me",
		BodyHash: "",
	}
	if s.Idem != nil {
		if meta, ok, err := s.Idem.Get(ctx, metaFP); err != nil {
			return nil, err
		} else if ok {
			if string(meta.Body) != bodyHash {
				return oas.UpdateMyMemberProfile409JSONResponse{ConflictJSONResponse: oas.ConflictJSONResponse(oasError(ctx, "IDEMPOTENCY_KEY_REUSE", "idempotency key reuse with different payload", nil))}, nil
			}
		} else {
			_ = s.Idem.Put(ctx, metaFP, idempotency.Record{
				StatusCode:  0,
				ContentType: "text/plain",
				Body:        []byte(bodyHash),
				CreatedAt:   time.Now().UTC(),
			})
		}

		respFP := metaFP
		respFP.BodyHash = bodyHash
		if rec, ok, err := s.Idem.Get(ctx, respFP); err != nil {
			return nil, err
		} else if ok && rec.StatusCode == http.StatusOK && strings.HasPrefix(rec.ContentType, "application/json") {
			var payload oas.UpdateMyMemberProfileResponse
			if err := json.Unmarshal(rec.Body, &payload); err == nil {
				return oas.UpdateMyMemberProfile200JSONResponse(payload), nil
			}
		}
	}

	in := updateMyMemberProfileInputFromOAS(*req.Body)
	m, err := s.Members.UpdateMyMemberProfile(ctx, domain.SubjectID(sub), in)
	if err != nil {
		if ae := (*members.Error)(nil); errors.As(err, &ae) {
			switch ae.Status {
			case http.StatusNotFound:
				return oas.UpdateMyMemberProfile404JSONResponse(oasError(ctx, ae.Code, ae.Message, ae.Details)), nil
			case http.StatusConflict:
				return oas.UpdateMyMemberProfile409JSONResponse{ConflictJSONResponse: oas.ConflictJSONResponse(oasError(ctx, ae.Code, ae.Message, ae.Details))}, nil
			case http.StatusUnprocessableEntity:
				return oas.UpdateMyMemberProfile422JSONResponse{UnprocessableEntityJSONResponse: oas.UnprocessableEntityJSONResponse(oasError(ctx, ae.Code, ae.Message, ae.Details))}, nil
			default:
				return nil, err
			}
		}
		return nil, err
	}

	resp := oas.UpdateMyMemberProfileResponse{
		Member: memberProfileFromDomain(m),
	}

	// Store successful response for replay.
	if s.Idem != nil {
		respFP := idempotency.Fingerprint{
			Key:      idempotency.Key(req.Params.IdempotencyKey),
			Subject:  domain.SubjectID(sub),
			Method:   http.MethodPatch,
			Route:    "/members/me",
			BodyHash: bodyHash,
		}
		if b, err := json.Marshal(resp); err == nil {
			_ = s.Idem.Put(ctx, respFP, idempotency.Record{
				StatusCode:  http.StatusOK,
				ContentType: "application/json",
				Body:        b,
				CreatedAt:   time.Now().UTC(),
			})
		}
	}

	return oas.UpdateMyMemberProfile200JSONResponse(resp), nil
}

func (s *Server) ListVisibleTripsForMember(ctx context.Context, _ oas.ListVisibleTripsForMemberRequestObject) (oas.ListVisibleTripsForMemberResponseObject, error) {
	sub, ok := SubjectFromContext(ctx)
	if !ok {
		return oas.ListVisibleTripsForMember401JSONResponse{UnauthorizedJSONResponse: oas.UnauthorizedJSONResponse(oasError(ctx, "UNAUTHORIZED", "missing subject", nil))}, nil
	}
	me, err := s.Members.GetMyMemberProfile(ctx, domain.SubjectID(sub))
	if err != nil {
		if isMemberNotProvisioned(err) {
			return oas.ListVisibleTripsForMember401JSONResponse{UnauthorizedJSONResponse: oas.UnauthorizedJSONResponse(oasError(ctx, "MEMBER_NOT_PROVISIONED", "No member profile exists for the authenticated subject.", nil))}, nil
		}
		return nil, err
	}

	ts, err := s.Trips.ListVisibleTripsForMember(ctx, me.ID)
	if err != nil {
		if ae := (*trips.Error)(nil); errors.As(err, &ae) {
			switch ae.Status {
			default:
				return nil, err
			}
		}
		return nil, err
	}

	out := make([]oas.TripSummary, 0, len(ts))
	for _, t := range ts {
		out = append(out, tripSummaryFromDomain(t))
	}
	return oas.ListVisibleTripsForMember200JSONResponse{Trips: out}, nil
}

func (s *Server) ListMyDraftTrips(ctx context.Context, _ oas.ListMyDraftTripsRequestObject) (oas.ListMyDraftTripsResponseObject, error) {
	sub, ok := SubjectFromContext(ctx)
	if !ok {
		return oas.ListMyDraftTrips401JSONResponse{UnauthorizedJSONResponse: oas.UnauthorizedJSONResponse(oasError(ctx, "UNAUTHORIZED", "missing subject", nil))}, nil
	}
	me, err := s.Members.GetMyMemberProfile(ctx, domain.SubjectID(sub))
	if err != nil {
		if isMemberNotProvisioned(err) {
			return oas.ListMyDraftTrips401JSONResponse{UnauthorizedJSONResponse: oas.UnauthorizedJSONResponse(oasError(ctx, "MEMBER_NOT_PROVISIONED", "No member profile exists for the authenticated subject.", nil))}, nil
		}
		return nil, err
	}

	ts, err := s.Trips.ListMyDraftTrips(ctx, me.ID)
	if err != nil {
		if ae := (*trips.Error)(nil); errors.As(err, &ae) {
			switch ae.Status {
			default:
				return nil, err
			}
		}
		return nil, err
	}

	out := make([]oas.TripSummary, 0, len(ts))
	for _, t := range ts {
		out = append(out, tripSummaryFromDomain(t))
	}
	return oas.ListMyDraftTrips200JSONResponse{Trips: out}, nil
}

func (s *Server) GetTripDetails(ctx context.Context, req oas.GetTripDetailsRequestObject) (oas.GetTripDetailsResponseObject, error) {
	sub, ok := SubjectFromContext(ctx)
	if !ok {
		return oas.GetTripDetails401JSONResponse{UnauthorizedJSONResponse: oas.UnauthorizedJSONResponse(oasError(ctx, "UNAUTHORIZED", "missing subject", nil))}, nil
	}
	me, err := s.Members.GetMyMemberProfile(ctx, domain.SubjectID(sub))
	if err != nil {
		if isMemberNotProvisioned(err) {
			return oas.GetTripDetails401JSONResponse{UnauthorizedJSONResponse: oas.UnauthorizedJSONResponse(oasError(ctx, "MEMBER_NOT_PROVISIONED", "No member profile exists for the authenticated subject.", nil))}, nil
		}
		return nil, err
	}

	td, err := s.Trips.GetTripDetails(ctx, me.ID, domain.TripID(req.TripId))
	if err != nil {
		if ae := (*trips.Error)(nil); errors.As(err, &ae) {
			switch ae.Status {
			case http.StatusNotFound:
				return oas.GetTripDetails404JSONResponse{NotFoundJSONResponse: oas.NotFoundJSONResponse(oasError(ctx, ae.Code, ae.Message, ae.Details))}, nil
			default:
				return nil, err
			}
		}
		return nil, err
	}
	return oas.GetTripDetails200JSONResponse{Trip: tripDetailsFromDomain(td)}, nil
}

func isMemberNotProvisioned(err error) bool {
	ae := (*members.Error)(nil)
	if errors.As(err, &ae) {
		return ae.Code == "MEMBER_NOT_PROVISIONED"
	}
	return false
}

func oasError(ctx context.Context, code string, message string, details map[string]any) oas.ErrorResponse {
	var er oas.ErrorResponse
	er.Error.Code = code
	er.Error.Message = message
	if details != nil {
		er.Error.Details = nullable.NewNullableWithValue(map[string]any(details))
	}
	if rid := middleware.GetReqID(ctx); rid != "" {
		er.Error.RequestId = nullable.NewNullableWithValue(rid)
	}
	return er
}

func tripSummaryFromDomain(t domain.TripSummary) oas.TripSummary {
	out := oas.TripSummary{
		TripId: string(t.ID),
		Status: oas.TripStatus(t.Status),
	}
	out.Name = nullableString(t.Name)
	out.StartDate = nullableDate(t.StartDate)
	out.EndDate = nullableDate(t.EndDate)
	out.CapacityRigs = nullableInt(t.CapacityRigs)
	out.AttendingRigs = nullableInt(t.AttendingRigs)
	if t.DraftVisibility != nil {
		dv := oas.DraftVisibility(*t.DraftVisibility)
		out.DraftVisibility = &dv
	}
	return out
}

func tripDetailsFromDomain(t domain.TripDetails) oas.TripDetails {
	out := oas.TripDetails{
		TripId:             string(t.ID),
		Status:             oas.TripStatus(t.Status),
		Organizers:         make([]oas.MemberSummary, 0, len(t.Organizers)),
		Artifacts:          make([]oas.TripArtifact, 0, len(t.Artifacts)),
		RsvpActionsEnabled: t.RSVPActionsEnabled,
	}

	out.Name = nullableString(t.Name)
	out.Description = nullableString(t.Description)
	out.StartDate = nullableDate(t.StartDate)
	out.EndDate = nullableDate(t.EndDate)
	out.CapacityRigs = nullableInt(t.CapacityRigs)
	out.DifficultyText = nullableString(t.DifficultyText)
	out.CommsRequirementsText = nullableString(t.CommsRequirementsText)
	out.RecommendedRequirementsText = nullableString(t.RecommendedRequirementsText)
	if t.DraftVisibility != nil {
		dv := oas.DraftVisibility(*t.DraftVisibility)
		out.DraftVisibility = &dv
	}

	if t.MeetingLocation != nil {
		out.MeetingLocation = locationFromDomain(*t.MeetingLocation)
	}

	for _, m := range t.Organizers {
		out.Organizers = append(out.Organizers, memberSummaryFromDomain(m))
	}
	for _, a := range t.Artifacts {
		out.Artifacts = append(out.Artifacts, tripArtifactFromDomain(a))
	}

	// RSVP fields are introduced in later milestones; omit for now.
	out.RsvpSummary = nil
	out.MyRsvp = nil

	return out
}

func memberSummaryFromDomain(m domain.MemberSummary) oas.MemberSummary {
	out := oas.MemberSummary{
		MemberId:    string(m.ID),
		DisplayName: m.DisplayName,
		Email:       openapi_types.Email(m.Email),
	}
	if m.GroupAliasEmail != nil {
		out.GroupAliasEmail = nullable.NewNullableWithValue(openapi_types.Email(*m.GroupAliasEmail))
	}
	return out
}

func tripArtifactFromDomain(a domain.TripArtifact) oas.TripArtifact {
	return oas.TripArtifact{
		ArtifactId: a.ArtifactID,
		Type:       oas.ArtifactType(a.Type),
		Title:      a.Title,
		Url:        a.URL,
	}
}

func locationFromDomain(l domain.Location) *oas.Location {
	out := &oas.Location{Label: l.Label}
	if l.Address != nil {
		out.Address = nullable.NewNullableWithValue(*l.Address)
	}
	if l.Latitude != nil || l.Longitude != nil {
		ll := struct {
			Latitude  *float64 `json:"latitude,omitempty"`
			Longitude *float64 `json:"longitude,omitempty"`
		}{
			Latitude:  l.Latitude,
			Longitude: l.Longitude,
		}
		out.LatitudeLongitude = nullable.NewNullableWithValue(ll)
	}
	return out
}

func nullableString(p *string) nullable.Nullable[string] {
	var out nullable.Nullable[string]
	if p != nil {
		out.Set(*p)
	}
	return out
}

func nullableInt(p *int) nullable.Nullable[int] {
	var out nullable.Nullable[int]
	if p != nil {
		out.Set(*p)
	}
	return out
}

func nullableDate(p *time.Time) nullable.Nullable[openapi_types.Date] {
	var out nullable.Nullable[openapi_types.Date]
	if p != nil {
		out.Set(openapi_types.Date{Time: p.UTC()})
	}
	return out
}

func memberProfileFromDomain(m domain.Member) oas.MemberProfile {
	out := oas.MemberProfile{
		MemberId:    string(m.ID),
		DisplayName: m.DisplayName,
		Email:       openapi_types.Email(m.Email),
	}
	if m.GroupAliasEmail != nil {
		out.GroupAliasEmail = nullable.NewNullableWithValue(openapi_types.Email(*m.GroupAliasEmail))
	}
	if m.VehicleProfile != nil {
		out.VehicleProfile = vehicleProfileFromDomain(*m.VehicleProfile)
	}
	return out
}

func vehicleProfileFromDomain(vp domain.VehicleProfile) *oas.VehicleProfile {
	out := &oas.VehicleProfile{}
	if vp.Make != nil {
		out.Make = nullable.NewNullableWithValue(*vp.Make)
	}
	if vp.Model != nil {
		out.Model = nullable.NewNullableWithValue(*vp.Model)
	}
	if vp.TireSize != nil {
		out.TireSize = nullable.NewNullableWithValue(*vp.TireSize)
	}
	if vp.LiftLockers != nil {
		out.LiftLockers = nullable.NewNullableWithValue(*vp.LiftLockers)
	}
	if vp.FuelRange != nil {
		out.FuelRange = nullable.NewNullableWithValue(*vp.FuelRange)
	}
	if vp.RecoveryGear != nil {
		out.RecoveryGear = nullable.NewNullableWithValue(*vp.RecoveryGear)
	}
	if vp.HamRadioCallSign != nil {
		out.HamRadioCallSign = nullable.NewNullableWithValue(*vp.HamRadioCallSign)
	}
	if vp.Notes != nil {
		out.Notes = nullable.NewNullableWithValue(*vp.Notes)
	}
	return out
}

func vehicleProfilePatchFromOAS(vp oas.VehicleProfile) *members.VehicleProfilePatch {
	p := &members.VehicleProfilePatch{}
	p.Make = optionalStringFromNullable(vp.Make)
	p.Model = optionalStringFromNullable(vp.Model)
	p.TireSize = optionalStringFromNullable(vp.TireSize)
	p.LiftLockers = optionalStringFromNullable(vp.LiftLockers)
	p.FuelRange = optionalStringFromNullable(vp.FuelRange)
	p.RecoveryGear = optionalStringFromNullable(vp.RecoveryGear)
	p.HamRadioCallSign = optionalStringFromNullable(vp.HamRadioCallSign)
	p.Notes = optionalStringFromNullable(vp.Notes)
	return p
}

func updateMyMemberProfileInputFromOAS(b oas.UpdateMyMemberProfileRequest) members.UpdateMyMemberProfileInput {
	out := members.UpdateMyMemberProfileInput{}

	out.DisplayName = optionalStringFromNullable(b.DisplayName)
	if b.Email != nil {
		out.Email = members.Some(strings.TrimSpace(string(*b.Email)))
	}
	// Email cannot be explicitly null in the OpenAPI schema.

	if b.GroupAliasEmail.IsSpecified() {
		if b.GroupAliasEmail.IsNull() {
			out.GroupAliasEmail = members.Null[string]()
		} else if v, err := b.GroupAliasEmail.Get(); err == nil {
			out.GroupAliasEmail = members.Some(strings.TrimSpace(string(v)))
		}
	}

	// NOTE: b.VehicleProfile is a pointer (optional) but not nullable, so we cannot represent `vehicleProfile: null`.
	if b.VehicleProfile != nil {
		out.VehicleProfile = members.Some(*vehicleProfilePatchFromOAS(*b.VehicleProfile))
	}

	return out
}

func optionalStringFromNullable(n nullable.Nullable[string]) members.Optional[string] {
	if !n.IsSpecified() {
		return members.Unspecified[string]()
	}
	if n.IsNull() {
		return members.Null[string]()
	}
	v, err := n.Get()
	if err != nil {
		return members.Unspecified[string]()
	}
	return members.Some(v)
}

func hashUpdateMyMemberProfileBody(b oas.UpdateMyMemberProfileRequest) (string, error) {
	// Canonicalize fields that have normalization semantics before hashing (UC-16).
	canon := b
	if canon.DisplayName.IsSpecified() && !canon.DisplayName.IsNull() {
		if v, err := canon.DisplayName.Get(); err == nil {
			var n nullable.Nullable[string]
			n.Set(domain.NormalizeHumanName(v))
			canon.DisplayName = n
		}
	}
	if canon.Email != nil {
		e := openapi_types.Email(strings.TrimSpace(string(*canon.Email)))
		canon.Email = &e
	}
	if canon.GroupAliasEmail.IsSpecified() && !canon.GroupAliasEmail.IsNull() {
		if v, err := canon.GroupAliasEmail.Get(); err == nil {
			var n nullable.Nullable[openapi_types.Email]
			n.Set(openapi_types.Email(strings.TrimSpace(string(v))))
			canon.GroupAliasEmail = n
		}
	}

	raw, err := json.Marshal(canon)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}


