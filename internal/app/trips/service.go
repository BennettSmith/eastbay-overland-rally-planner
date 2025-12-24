package trips

import (
	"context"
	"errors"
	"time"

	"eastbay-overland-rally-planner/internal/domain"
	"eastbay-overland-rally-planner/internal/ports/out/memberrepo"
	"eastbay-overland-rally-planner/internal/ports/out/triprepo"
)

type Service struct {
	trips   triprepo.Repository
	members memberrepo.Repository
}

func NewService(tripsRepo triprepo.Repository, membersRepo memberrepo.Repository) *Service {
	return &Service{
		trips:   tripsRepo,
		members: membersRepo,
	}
}

func (s *Service) ListVisibleTripsForMember(ctx context.Context, _ domain.MemberID) ([]domain.TripSummary, error) {
	ts, err := s.trips.ListPublishedAndCanceled(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.TripSummary, 0, len(ts))
	for _, t := range ts {
		out = append(out, toDomainSummary(t))
	}
	return out, nil
}

func (s *Service) ListMyDraftTrips(ctx context.Context, caller domain.MemberID) ([]domain.TripSummary, error) {
	ts, err := s.trips.ListDraftsVisibleTo(ctx, caller)
	if err != nil {
		return nil, err
	}
	out := make([]domain.TripSummary, 0, len(ts))
	for _, t := range ts {
		out = append(out, toDomainSummary(t))
	}
	return out, nil
}

func (s *Service) GetTripDetails(ctx context.Context, caller domain.MemberID, tripID domain.TripID) (domain.TripDetails, error) {
	t, err := s.trips.GetByID(ctx, tripID)
	if err != nil {
		if errors.Is(err, triprepo.ErrNotFound) {
			return domain.TripDetails{}, &Error{Status: 404, Code: "TRIP_NOT_FOUND", Message: "trip not found"}
		}
		return domain.TripDetails{}, err
	}

	if !isTripVisibleToCaller(t, caller) {
		// UC-02: return 404 even if it exists.
		return domain.TripDetails{}, &Error{Status: 404, Code: "TRIP_NOT_FOUND", Message: "trip not found"}
	}

	orgs, err := s.loadOrganizerSummaries(ctx, t.OrganizerMemberIDs)
	if err != nil {
		return domain.TripDetails{}, err
	}

	d := toDomainDetails(t)
	d.Organizers = orgs
	d.Artifacts = append([]domain.TripArtifact(nil), t.Artifacts...)
	d.RSVPActionsEnabled = d.Status == domain.TripStatusPublished

	// RSVP fields are added later (Milestone 6); omit for now.
	d.RSVPSummary = nil
	d.MyRSVP = nil

	return d, nil
}

func (s *Service) loadOrganizerSummaries(ctx context.Context, ids []domain.MemberID) ([]domain.MemberSummary, error) {
	if len(ids) == 0 {
		return []domain.MemberSummary{}, nil
	}
	out := make([]domain.MemberSummary, 0, len(ids))
	for _, id := range ids {
		m, err := s.members.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, domain.MemberSummary{
			ID:          m.ID,
			DisplayName: m.DisplayName,
			Email:       m.Email,
			GroupAliasEmail: cloneStringPtr(m.GroupAliasEmail),
		})
	}
	return out, nil
}

func isTripVisibleToCaller(t triprepo.Trip, caller domain.MemberID) bool {
	switch t.Status {
	case triprepo.StatusPublished, triprepo.StatusCanceled:
		return true
	case triprepo.StatusDraft:
		switch t.DraftVisibility {
		case triprepo.DraftVisibilityPublic:
			for _, id := range t.OrganizerMemberIDs {
				if id == caller {
					return true
				}
			}
			return false
		case triprepo.DraftVisibilityPrivate:
			return t.CreatorMemberID == caller
		default:
			return false
		}
	default:
		return false
	}
}

func toDomainSummary(t triprepo.Trip) domain.TripSummary {
	out := domain.TripSummary{
		ID:     t.ID,
		Name:   cloneStringPtr(t.Name),
		Status: domain.TripStatus(t.Status),

		StartDate: cloneTimePtr(t.StartDate),
		EndDate:   cloneTimePtr(t.EndDate),

		CapacityRigs:  cloneIntPtr(t.CapacityRigs),
		AttendingRigs: cloneIntPtr(t.AttendingRigs),
	}

	if t.Status == triprepo.StatusDraft {
		dv := domain.DraftVisibility(t.DraftVisibility)
		out.DraftVisibility = &dv
	}

	return out
}

func toDomainDetails(t triprepo.Trip) domain.TripDetails {
	out := domain.TripDetails{
		TripSummary: toDomainSummary(t),

		Description:                 cloneStringPtr(t.Description),
		DifficultyText:              cloneStringPtr(t.DifficultyText),
		MeetingLocation:             cloneLocationPtr(t.MeetingLocation),
		CommsRequirementsText:       cloneStringPtr(t.CommsRequirementsText),
		RecommendedRequirementsText: cloneStringPtr(t.RecommendedRequirementsText),

		Organizers: []domain.MemberSummary{},
		Artifacts:  []domain.TripArtifact{},
	}
	return out
}

func cloneStringPtr(p *string) *string {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func cloneIntPtr(p *int) *int {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func cloneTimePtr(p *time.Time) *time.Time {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func cloneLocationPtr(p *domain.Location) *domain.Location {
	if p == nil {
		return nil
	}
	cp := *p
	cp.Address = cloneStringPtr(p.Address)
	if p.Latitude != nil {
		v := *p.Latitude
		cp.Latitude = &v
	}
	if p.Longitude != nil {
		v := *p.Longitude
		cp.Longitude = &v
	}
	return &cp
}


