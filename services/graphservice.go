// Package services: GraphService queries workspace language-model definitions and edges.
package services

import (
	"fmt"
	"os"
	"strings"

	"paradox-modding-tools/services/internal/langmodel"
)

// GraphService searches definitions and walks reference edges.
type GraphService struct{}

// GraphDef is a definition returned to the frontend.
type GraphDef struct {
	Type     string `json:"type"`
	Key      string `json:"key"`
	FilePath string `json:"filePath"`
	Line     int    `json:"line"`
	Col      int    `json:"col"`
	EndLine  int    `json:"endLine"`
	Summary  string `json:"summary"`
}

// GraphEdge is a reference edge between keys.
type GraphEdge struct {
	FromKey  string `json:"fromKey"`
	ToKey    string `json:"toKey"`
	EdgeType string `json:"edgeType"`
}

// CascadeNode represents a node in cascade simulation results.
type CascadeNode struct {
	Key      string   `json:"key"`
	Depth    int      `json:"depth"`
	EdgeType string   `json:"edgeType,omitempty"`
	Children []string `json:"children,omitempty"`
}

// NeighborResult is a key plus its inbound/outbound edges.
type NeighborResult struct {
	Key      string      `json:"key"`
	Defs     []GraphDef  `json:"defs"`
	Outgoing []GraphEdge `json:"outgoing"`
	Incoming []GraphEdge `json:"incoming"`
}

// SearchDefinitions finds definitions by key/summary substring and optional type.
func (g *GraphService) SearchDefinitions(
	workspaceID, query, typeFilter string,
) ([]GraphDef, error) {
	m, err := loadGraphModel(workspaceID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, nil
	}
	q := strings.ToLower(query)
	var out []GraphDef
	for _, d := range m.Definitions {
		if typeFilter != "" && !strings.EqualFold(d.Type, typeFilter) {
			continue
		}
		if q != "" {
			if !strings.Contains(strings.ToLower(d.Key), q) &&
				!strings.Contains(strings.ToLower(d.Summary), q) &&
				!strings.Contains(strings.ToLower(d.Type), q) {
				continue
			}
		}
		out = append(out, toGraphDef(d))
		if len(out) >= 200 {
			break
		}
	}
	return out, nil
}

// GetNeighbors returns definitions and edges touching key.
func (g *GraphService) GetNeighbors(workspaceID, key string) (*NeighborResult, error) {
	m, err := loadGraphModel(workspaceID)
	if err != nil {
		return nil, err
	}
	res := &NeighborResult{Key: key}
	if m == nil || key == "" {
		return res, nil
	}
	for _, d := range m.Definitions {
		if d.Key == key {
			res.Defs = append(res.Defs, toGraphDef(d))
		}
	}
	for _, e := range m.Edges {
		ge := GraphEdge{FromKey: e.FromKey, ToKey: e.ToKey, EdgeType: e.EdgeType}
		if e.FromKey == key {
			res.Outgoing = append(res.Outgoing, ge)
		}
		if e.ToKey == key {
			res.Incoming = append(res.Incoming, ge)
		}
	}
	return res, nil
}

// SimulateCascade walks outgoing edges BFS from key up to maxDepth.
func (g *GraphService) SimulateCascade(
	workspaceID, key string, maxDepth int,
) ([]CascadeNode, error) {
	m, err := loadGraphModel(workspaceID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, nil
	}
	if maxDepth <= 0 {
		maxDepth = 5
	}
	graph := map[string][]struct {
		to       string
		edgeType string
	}{}
	for _, e := range m.Edges {
		graph[e.FromKey] = append(graph[e.FromKey], struct {
			to       string
			edgeType string
		}{e.ToKey, e.EdgeType})
	}

	visited := map[string]bool{}
	var result []CascadeNode
	queue := []struct {
		key   string
		depth int
	}{{key, 0}}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		if visited[curr.key] || curr.depth > maxDepth {
			continue
		}
		visited[curr.key] = true
		node := CascadeNode{Key: curr.key, Depth: curr.depth}
		for _, edge := range graph[curr.key] {
			node.Children = append(node.Children, edge.to)
			if !visited[edge.to] {
				queue = append(queue, struct {
					key   string
					depth int
				}{edge.to, curr.depth + 1})
			}
		}
		if len(graph[curr.key]) > 0 {
			node.EdgeType = graph[curr.key][0].edgeType
		}
		result = append(result, node)
	}
	return result, nil
}

func loadGraphModel(workspaceID string) (*langmodel.Model, error) {
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace id is required")
	}
	m, err := langmodel.Load(workspaceID)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		path, _ := langmodel.ModelPath(workspaceID)
		if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
			return nil, nil
		}
		return nil, err
	}
	return m, nil
}

func toGraphDef(d langmodel.Def) GraphDef {
	return GraphDef{
		Type: d.Type, Key: d.Key, FilePath: d.FilePath,
		Line: d.Line, Col: d.Col, EndLine: d.EndLine, Summary: d.Summary,
	}
}
