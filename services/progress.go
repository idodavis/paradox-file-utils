// Package services: shared progress events for long-running Go jobs.
package services

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

// ProgressEvent is a phase/progress update for language-model/semantics jobs.
type ProgressEvent struct {
	Job         string  `json:"job"`
	Phase       string  `json:"phase"`
	Done        int     `json:"done"`
	Total       int     `json:"total"`
	Percent     float64 `json:"percent"`
	Message     string  `json:"message,omitempty"`
	WorkspaceID string  `json:"workspaceId,omitempty"`
}

const (
	eventSemanticsProgress = "semantics:progress"
	eventLangModelProgress = "langmodel:progress"
)

// emitProgress sends a progress event to the frontend when the app is running.
func emitProgress(name string, ev ProgressEvent) {
	app := application.Get()
	if app == nil {
		return
	}
	if ev.Total > 0 {
		ev.Percent = float64(ev.Done) / float64(ev.Total) * 100
	}
	app.Event.Emit(name, ev)
}
