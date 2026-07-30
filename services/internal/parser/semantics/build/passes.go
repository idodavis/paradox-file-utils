package build

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"paradox-modding-tools/services/internal/parser"
	"paradox-modding-tools/services/internal/parser/semantics/model"
)

const (
	maxObservedFolders = 64
	maxObservedFiles   = 32
	maxAssociatedInfo  = 16
	maxMacroTokens     = 2000
)

func runPasses(meta *model.SemanticMetadata, game string, files []parsedTxt) error {
	macroSet := map[string]struct{}{}
	for _, pf := range files {
		rel := filepath.ToSlash(pf.relPath)
		folder := firstFolder(rel)
		containment := detectContainment(rel, pf.tree)

		for _, expr := range topLevelExpressions(pf.tree) {
			kind := classifyKind(game, rel, expr.Key, expr.Object != nil)
			if kind == "" {
				continue
			}
			spec := meta.Types[kind]
			if spec.UsageForms == nil {
				spec.UsageForms = map[string]int64{}
			}
			spec.UsageForms["top_level_definition"]++
			spec.ContainmentPattern = setContainmentIfEmpty(spec.ContainmentPattern, containment)
			spec.ObservedFolders = addUniqueCapped(spec.ObservedFolders, folder, maxObservedFolders)
			spec.ObservedFiles = addUniqueCapped(spec.ObservedFiles, rel, maxObservedFiles)
			spec.AssociatedInfo = addUniqueCapped(spec.AssociatedInfo, guessInfoPath(rel), maxAssociatedInfo)
			if len(spec.AttributeNames) == 0 && expr.Object != nil {
				spec.AttributeNames = extractAttributeNames(expr.Object)
			}
			meta.Types[kind] = spec
		}
		collectMacroVarTokens(pf.absPath, macroSet)
	}

	if meta.LanguageFacts == nil {
		meta.LanguageFacts = map[string]any{}
	}
	meta.LanguageFacts["macro_and_var_tokens"] = sortedTokensCapped(macroSet, maxMacroTokens)
	meta.LanguageFacts["model_focus"] = "high_level_structure_only"
	return nil
}

func detectContainment(rel string, tree *parser.ParadoxFile) string {
	rel = filepath.ToSlash(rel)
	switch {
	case strings.Contains(rel, "/history/") || strings.HasPrefix(rel, "history/"):
		if majorityTopLevelKeysMatch(tree, dateKeyRE) {
			return "date_keyed_history"
		}
	case strings.Contains(rel, "/events/") || strings.HasPrefix(rel, "events/"):
		if fileNamespace(tree) != "" {
			return "events_namespaced"
		}
	case strings.Contains(rel, "/settings/") || strings.Contains(rel, "/game_rules/"):
		return "namespaced_settings"
	}
	if majorityTopLevelKeysMatch(tree, numericKeyRE) {
		return "numeric_id_keyed"
	}
	if majorityTopLevelKeysMatch(tree, dateKeyRE) {
		return "date_keyed_history"
	}
	if singleTopLevelObjectBlock(tree) {
		return "wrapper_keyed"
	}
	if looksTwoLevelTree(tree) {
		return "two_level"
	}
	if maxDepthFromFile(tree) > 10 {
		return "recursive"
	}
	return "flat"
}

func fileNamespace(tree *parser.ParadoxFile) string {
	if tree == nil {
		return ""
	}
	for _, e := range tree.Entries {
		if e.Namespace != nil && e.Namespace.Value != "" {
			return e.Namespace.Value
		}
	}
	return ""
}

func topLevelExpressions(tree *parser.ParadoxFile) []*parser.Expression {
	if tree == nil {
		return nil
	}
	var out []*parser.Expression
	for _, e := range tree.Entries {
		if e.Expression != nil && e.Expression.Key != "" {
			out = append(out, e.Expression)
		}
	}
	return out
}

func majorityTopLevelKeysMatch(tree *parser.ParadoxFile, re *regexp.Regexp) bool {
	exprs := topLevelExpressions(tree)
	if len(exprs) < 2 {
		return false
	}
	n := 0
	for _, x := range exprs {
		if re.MatchString(x.Key) {
			n++
		}
	}
	return n*2 >= len(exprs)
}

func singleTopLevelObjectBlock(tree *parser.ParadoxFile) bool {
	exprs := topLevelExpressions(tree)
	if len(exprs) != 1 {
		return false
	}
	return exprs[0].Object != nil
}

func looksTwoLevelTree(tree *parser.ParadoxFile) bool {
	exprs := topLevelExpressions(tree)
	if len(exprs) < 2 {
		return false
	}
	ok := 0
	for _, x := range exprs {
		if x.Object == nil {
			continue
		}
		innerObjs, total := 0, 0
		for _, oe := range x.Object.Entries {
			if oe.Expression != nil && oe.Expression.Key != "" {
				total++
				if oe.Expression.Object != nil {
					innerObjs++
				}
			}
		}
		if total > 0 && innerObjs == total {
			ok++
		}
	}
	return ok >= 2
}

func maxDepthFromFile(tree *parser.ParadoxFile) int {
	maxd := 0
	for _, e := range tree.Entries {
		if e.Expression != nil && e.Expression.Object != nil {
			if d := objectDepth(e.Expression.Object); d > maxd {
				maxd = d
			}
		}
	}
	return maxd
}

func objectDepth(o *parser.Object) int {
	if o == nil {
		return 0
	}
	maxd := 1
	for _, oe := range o.Entries {
		if oe.Object != nil {
			if d := 1 + objectDepth(oe.Object); d > maxd {
				maxd = d
			}
		}
		if oe.Expression != nil && oe.Expression.Object != nil {
			if d := 1 + objectDepth(oe.Expression.Object); d > maxd {
				maxd = d
			}
		}
	}
	return maxd
}
var (
	macroRe           = regexp.MustCompile(`\$[A-Za-z0-9_]+\$`)
	varTokenRe        = regexp.MustCompile(`@[A-Za-z0-9_\[\].$:'&|%/\\-]+`)
	dateKeyRE         = regexp.MustCompile(`^\d+\.\d+\.\d+(\.\d+)?$`)
	numericKeyRE      = regexp.MustCompile(`^-?\d+$`)
)

func classifyKind(game, relPath, key string, hasObject bool) string {
	folder := canonicalFolderKind(relPath)
	if folder == "" {
		return "unknown"
	}
	return folder
}

func canonicalFolderKind(relPath string) string {
	relPath = filepath.ToSlash(relPath)
	parts := strings.Split(relPath, "/")
	if len(parts) == 0 {
		return ""
	}
	for i, p := range parts {
		if p == "common" && i+1 < len(parts) {
			return "folder:" + parts[i+1]
		}
	}
	for _, p := range parts {
		if p != "" {
			return "folder:" + p
		}
	}
	return ""
}

func firstFolder(relPath string) string {
	relPath = filepath.ToSlash(relPath)
	parts := strings.Split(relPath, "/")
	if len(parts) == 0 {
		return ""
	}
	for _, p := range parts {
		if p != "" {
			return p
		}
	}
	return ""
}

func setContainmentIfEmpty(existing, inferred string) string {
	if existing != "" {
		return existing
	}
	return inferred
}

func addUniqueCapped(in []string, value string, capN int) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return in
	}
	for _, v := range in {
		if v == value {
			return in
		}
	}
	if len(in) >= capN {
		return in
	}
	return append(in, value)
}

func guessInfoPath(relPath string) string {
	dir := filepath.ToSlash(filepath.Dir(relPath))
	if dir == "." || dir == "" {
		return ""
	}
	base := filepath.Base(dir)
	if base == "" || base == "." {
		return ""
	}
	return dir + "/_" + base + ".info"
}

func extractAttributeNames(obj *parser.Object) []string {
	if obj == nil {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, 32)
	for _, e := range obj.Entries {
		if e.Expression == nil || e.Expression.Key == "" {
			continue
		}
		k := e.Expression.Key
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func collectMacroVarTokens(absPath string, tokenSet map[string]struct{}) {
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return
	}
	text := string(raw)
	for _, m := range macroRe.FindAllString(text, -1) {
		tokenSet[m] = struct{}{}
	}
	for _, m := range varTokenRe.FindAllString(text, -1) {
		tokenSet[m] = struct{}{}
	}
}

func sortedTokensCapped(tokenSet map[string]struct{}, capN int) []any {
	if len(tokenSet) == 0 {
		return nil
	}
	tokens := make([]string, 0, len(tokenSet))
	for t := range tokenSet {
		tokens = append(tokens, t)
	}
	sort.Strings(tokens)
	if len(tokens) > capN {
		tokens = tokens[:capN]
	}
	out := make([]any, len(tokens))
	for i, t := range tokens {
		out[i] = t
	}
	return out
}
