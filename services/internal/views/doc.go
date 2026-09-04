// Package views answers one product view against a live session: the event
// graph, event detail, and workspace health. Handlers use the session query
// API; they never import lsp, never persist, and never assign coordinates.
package views
