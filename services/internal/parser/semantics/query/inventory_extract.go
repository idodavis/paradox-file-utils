package query

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"paradox-modding-tools/services/internal/parser"
	"paradox-modding-tools/services/internal/parser/ast"
)

type SourceSpan struct {
	LineStart int
	LineEnd   int
}

type EntityRelationship struct {
	RelationKind string
	TargetKey    string
	TargetKind   string
	FilePath     string
	Span         SourceSpan
	Confidence   float64
	Evidence     string
}

type EntityRecord struct {
	ID            string
	Key           string
	Kind          string
	FilePath      string
	Span          SourceSpan
	RawText       string
	Attributes    map[string]bool
	PotentialRefs []string
	Outgoing      []EntityRelationship
	Incoming      []EntityRelationship
}

type inventoryExtractVisitor struct {
	ast.NoopVisitor
	filePath    string
	objectTypes []string
	seen        map[string]map[string]bool
	onItem      func(EntityRecord)
}

func (v *inventoryExtractVisitor) VisitExpression(expr *parser.Expression, ctx *ast.Context) {
	if expr == nil || expr.Key == "" {
		return
	}
	inline := ctx.Depth > 0
	typeName, ok := selectKindForPath(v.filePath, v.objectTypes)
	if !ok {
		return
	}
	displayKey := expr.Key

	var attrs map[string]bool
	if expr.Object != nil {
		attrs = ast.TopLevelKeys(expr.Object)
	}
	_ = inline // usage form captured by caller if needed later
	if v.seen[typeName][displayKey] {
		return
	}
	v.seen[typeName][displayKey] = true

	lineStart := 1
	if expr.Pos.Line > 0 {
		lineStart = expr.Pos.Line
	}
	raw := expr.GetRawText()
	lineEnd := ast.LineEnd(lineStart, raw)
	potentialRefs := ast.CollectIdentifiers(expr, displayKey)

	v.onItem(EntityRecord{
		ID:            entityID(typeName, displayKey, v.filePath, lineStart),
		Key:           displayKey,
		Kind:          typeName,
		FilePath:      v.filePath,
		Span:          SourceSpan{LineStart: lineStart, LineEnd: lineEnd},
		RawText:       raw,
		PotentialRefs: potentialRefs,
		Attributes:    attrs,
	})
}

func entityID(kind, key, filePath string, lineStart int) string {
	return fmt.Sprintf("%s|%s|%s|%d", kind, key, filePath, lineStart)
}

// ExtractEntityRecordsFromFile parses one file and extracts semantic entity records for selected object types.
func ExtractEntityRecordsFromFile(path string, game string, objectTypes []string) ([]EntityRecord, error) {
	if len(objectTypes) == 0 {
		return nil, nil
	}
	_ = game

	tree, err := parser.ParseFile(path)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]map[string]bool)
	for _, t := range objectTypes {
		seen[t] = make(map[string]bool)
	}

	var items []EntityRecord
	v := &inventoryExtractVisitor{
		filePath:    path,
		objectTypes: objectTypes,
		seen:        seen,
		onItem:      func(it EntityRecord) { items = append(items, it) },
	}
	ast.Walk(tree, v)

	return items, nil
}

func selectKindForPath(path string, objectTypes []string) (string, bool) {
	pathKind := canonicalFolderKindForPath(path)
	if pathKind != "" && slices.Contains(objectTypes, pathKind) {
		return pathKind, true
	}
	if len(objectTypes) == 0 {
		return "", false
	}
	return objectTypes[0], true
}

func canonicalFolderKindForPath(path string) string {
	p := filepath.ToSlash(path)
	parts := strings.Split(p, "/")
	for i, part := range parts {
		if part == "common" && i+1 < len(parts) {
			return "folder:" + parts[i+1]
		}
	}
	for _, part := range parts {
		if part != "" {
			return "folder:" + part
		}
	}
	return ""
}

func EnrichEntityRelationships(items []EntityRecord) {
	itemIndex := make(map[string]int)
	for i := range items {
		itemIndex[items[i].Key] = i
	}

	for index := range items {
		for _, potentialRef := range items[index].PotentialRefs {
			rel := EntityRelationship{
				RelationKind: "identifier_reference",
				TargetKey:    potentialRef,
				TargetKind:   items[index].Kind,
				FilePath:     items[index].FilePath,
				Span:         items[index].Span,
				Confidence:   0.5,
				Evidence:     potentialRef,
			}
			items[index].Outgoing = append(items[index].Outgoing, rel)
			if j, ok := itemIndex[potentialRef]; ok {
				items[j].Incoming = append(items[j].Incoming, EntityRelationship{
					RelationKind: "identifier_referred_by",
					TargetKey:    items[index].Key,
					TargetKind:   items[index].Kind,
					FilePath:     items[index].FilePath,
					Span:         items[index].Span,
					Confidence:   0.5,
					Evidence:     items[index].Key,
				})
			}
		}
	}
}

// ExtractInventoryItemsFromFile is a compatibility adapter for service-layer inventory projections.
func ExtractInventoryItemsFromFile(path string, game string, objectTypes []string) ([]InventoryItem, error) {
	entities, err := ExtractEntityRecordsFromFile(path, game, objectTypes)
	if err != nil {
		return nil, err
	}
	items := make([]InventoryItem, 0, len(entities))
	for _, e := range entities {
		items = append(items, InventoryItem{
			Key:           e.Key,
			Type:          e.Kind,
			FilePath:      e.FilePath,
			LineStart:     e.Span.LineStart,
			LineEnd:       e.Span.LineEnd,
			RawText:       e.RawText,
			PotentialRefs: e.PotentialRefs,
			Attributes:    e.Attributes,
		})
	}
	return items, nil
}

// EnrichInventoryReferences is a compatibility adapter for service-layer inventory projections.
func EnrichInventoryReferences(items []InventoryItem) {
	entities := make([]EntityRecord, 0, len(items))
	for _, it := range items {
		entities = append(entities, EntityRecord{
			Key:           it.Key,
			Kind:          it.Type,
			FilePath:      it.FilePath,
			Span:          SourceSpan{LineStart: it.LineStart, LineEnd: it.LineEnd},
			PotentialRefs: it.PotentialRefs,
		})
	}
	EnrichEntityRelationships(entities)
	for idx := range entities {
		for _, rel := range entities[idx].Outgoing {
			items[idx].References = append(items[idx].References, InventoryReference{
				Key:       rel.TargetKey,
				Type:      entities[idx].Kind,
				FilePath:  rel.FilePath,
				LineStart: rel.Span.LineStart,
				LineEnd:   rel.Span.LineEnd,
			})
		}
		for _, rel := range entities[idx].Incoming {
			items[idx].Referrers = append(items[idx].Referrers, InventoryReference{
				Key:       rel.TargetKey,
				Type:      rel.TargetKind,
				FilePath:  rel.FilePath,
				LineStart: rel.Span.LineStart,
				LineEnd:   rel.Span.LineEnd,
			})
		}
	}
}

type InventoryItem struct {
	Key           string
	Type          string
	FilePath      string
	LineStart     int
	LineEnd       int
	RawText       string
	PotentialRefs []string
	References    []InventoryReference
	Referrers     []InventoryReference
	Attributes    map[string]bool
}

type InventoryReference struct {
	Key       string
	Type      string
	FilePath  string
	LineStart int
	LineEnd   int
}
