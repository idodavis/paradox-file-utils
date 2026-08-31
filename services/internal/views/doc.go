// Package views answers one product view against a live session: the event
// graph, event detail, override rows, and loc coverage. Handlers use the
// session query API; they never import lsp, never persist, and never assign
// coordinates.
package views
