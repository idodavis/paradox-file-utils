// Package graph answers one product view against a live session: the event
// graph, event detail (including simulation order), override rows, loc
// coverage, and definition dependencies. Handlers read the session index and
// parse files they need; they never import lsp and never assign coordinates.
package graph
