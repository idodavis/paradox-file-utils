// intern.go collapses the absolute file path repeated on every Def, Ref and
// Edge into one table plus indices.
//
// A CK3 install produces millions of records and each one carried its own copy
// of a ~110-byte install path, which dominated the on-disk model. Interning
// happens only at the save/load boundary: Path holds a real path everywhere
// else, so no consumer changes.
//
// Loading also hands every record with the same path the one string from the
// table, so the saving shows up in RAM as well as on disk.

package catalog

import "strconv"

// internPaths returns a copy of c whose Def/Ref/Edge paths are decimal indices
// into the returned Paths table. The receiver is left untouched: a live session
// may be holding it while the scan writes.
func internPaths(c *VanillaCache) *VanillaCache {
	if c == nil {
		return nil
	}
	out := *c
	table := make([]string, 0, 4096)
	index := make(map[string]int, 4096)
	id := func(path string) string {
		if path == "" {
			return ""
		}
		i, ok := index[path]
		if !ok {
			i = len(table)
			index[path] = i
			table = append(table, path)
		}
		return strconv.Itoa(i)
	}

	out.Defs = make([]Def, len(c.Defs))
	for i, d := range c.Defs {
		d.Path = id(d.Path)
		out.Defs[i] = d
	}
	out.LocRefs = internRefs(c.LocRefs, id)
	out.CallRefs = internRefs(c.CallRefs, id)
	out.Edges = make([]Edge, len(c.Edges))
	for i, e := range c.Edges {
		e.Path = id(e.Path)
		out.Edges[i] = e
	}
	out.Paths = table
	return &out
}

func internRefs(refs []Ref, id func(string) string) []Ref {
	if refs == nil {
		return nil
	}
	out := make([]Ref, len(refs))
	for i, r := range refs {
		r.Path = id(r.Path)
		out[i] = r
	}
	return out
}

// expandPaths turns interned indices back into paths, in place. The cache has
// just been decoded and is not shared yet. A cache written before interning has
// no table and is left alone.
func expandPaths(c *VanillaCache) {
	if c == nil || len(c.Paths) == 0 {
		return
	}
	table := c.Paths
	at := func(s string) string {
		if s == "" {
			return ""
		}
		i, err := strconv.Atoi(s)
		if err != nil || i < 0 || i >= len(table) {
			// Not an index: a pre-interning file, or a corrupt one. Keep the
			// value rather than losing the location.
			return s
		}
		return table[i]
	}
	for i := range c.Defs {
		c.Defs[i].Path = at(c.Defs[i].Path)
	}
	for i := range c.LocRefs {
		c.LocRefs[i].Path = at(c.LocRefs[i].Path)
	}
	for i := range c.CallRefs {
		c.CallRefs[i].Path = at(c.CallRefs[i].Path)
	}
	for i := range c.Edges {
		c.Edges[i].Path = at(c.Edges[i].Path)
	}
	c.Paths = nil
}
