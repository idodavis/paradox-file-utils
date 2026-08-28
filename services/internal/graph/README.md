# graph

Answers one product view against a live `*session.Session`. One file per view.
This package must not import `lsp`. Layout (x/y) is frontend/dagre, not Go.

## Files

| File             | View |
| ---------------- | ---- |
| `types.go`       | DTOs + shared parse/block/loc helpers |
| `eventgraph.go`  | nodes/edges/via hops/suggestions (no coordinates) |
| `eventdetail.go` | read-only inspector + `simSteps` |
| `overrides.go`   | FIOS/LIOS rows from `model.Overrides` |
| `loccoverage.go` | missing / orphan / untranslated + loc lookup |
| `dependencies.go`| dependents + inner references of one def |
