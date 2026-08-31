// viewsservice.go is the Wails RPC shell for event graph, conflicts, and loc
// coverage. It holds a pointer to SessionService and delegates to package views.

package services

import (
	"paradox-modding-tools/services/internal/session"
	"paradox-modding-tools/services/internal/views"
)

// ViewsService exposes non-IDE session views over Wails.
type ViewsService struct {
	Session *SessionService
}

// GetEventGraph returns the defs-first event graph (no coordinates).
func (s *ViewsService) GetEventGraph(workspaceID string, params views.EventGraphParams) (views.EventGraph, error) {
	return withSession(s.Session, workspaceID, func(sess *session.Session) views.EventGraph {
		return views.Graph(sess, params)
	})
}

// GetEventDetail returns inspector content for one event id.
func (s *ViewsService) GetEventDetail(workspaceID, eventID string) (*views.EventDetail, error) {
	return withSession(s.Session, workspaceID, func(sess *session.Session) *views.EventDetail {
		return views.Detail(sess, eventID)
	})
}

// GetOverrides returns FIOS/LIOS override rows for the conflict monitor.
func (s *ViewsService) GetOverrides(workspaceID string) ([]views.OverrideRow, error) {
	return withSession(s.Session, workspaceID, func(sess *session.Session) []views.OverrideRow {
		return views.OverrideRows(sess)
	})
}

// GetLocCoverage returns per-language localization health for workspace mods.
func (s *ViewsService) GetLocCoverage(workspaceID string) ([]views.LocCoverage, error) {
	return withSession(s.Session, workspaceID, func(sess *session.Session) []views.LocCoverage {
		return views.Coverage(sess)
	})
}

// LookupLoc returns the english loc text and site for key.
func (s *ViewsService) LookupLoc(workspaceID, key string) (*views.LocLookup, error) {
	return withSession(s.Session, workspaceID, func(sess *session.Session) *views.LocLookup {
		return views.Lookup(sess, key)
	})
}
