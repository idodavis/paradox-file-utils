// scriptparam.go detects scripted macro parameters ($NAME$) and loc engine
// value filters ($VALUE|=+0$).

package game

import (
	"strings"
	"unicode"
)

// ScriptParamSpan reports when off lies inside a $NAME$ macro span in src.
func ScriptParamSpan(src string, off int) (name string, start, end int, ok bool) {
	if off < 0 || off > len(src) {
		return "", 0, 0, false
	}
	sliceEnd := off + 1
	if sliceEnd > len(src) {
		sliceEnd = len(src)
	}
	open := strings.LastIndexByte(src[:sliceEnd], '$')
	if open < 0 {
		return "", 0, 0, false
	}
	i := open + 1
	nameStart := i
	for i < len(src) && isScriptParamByte(src[i], i == nameStart) {
		i++
	}
	if i == nameStart || i >= len(src) || src[i] != '$' {
		return "", 0, 0, false
	}
	name = src[nameStart:i]
	if off < open || off >= i+1 {
		return "", 0, 0, false
	}
	return name, open, i + 1, true
}

func isScriptParamByte(c byte, first bool) bool {
	if first {
		return c == '_' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
	}
	return c == '_' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9')
}

// IsLocEngineValue reports a loc $key$ / $key|filter$ that is engine data,
// not a loc-key reuse ($other_key$ / $INDEPENDENCE_WAR_NAME$ / $key|U$).
func IsLocEngineValue(key, filter string) bool {
	if isLocEngineToken(key) {
		return true
	}
	return strings.EqualFold(key, "VALUE") && isNumericLocFilter(filter)
}

// isLocEngineToken reports a single ALL_CAPS token ($ORDER$, $VALUE$, $NAME$).
// SNAKE_CASE ($INDEPENDENCE_WAR_NAME$) is loc-key reuse.
func isLocEngineToken(key string) bool {
	if key == "" || strings.Contains(key, "_") {
		return false
	}
	for i := 0; i < len(key); i++ {
		c := key[i]
		if c < 'A' || c > 'Z' {
			return false
		}
	}
	return true
}

func isNumericLocFilter(filter string) bool {
	if filter == "" {
		return false
	}
	for _, r := range filter {
		switch r {
		case '=', '+', '-', '%', '.':
		default:
			if !unicode.IsDigit(r) {
				return false
			}
		}
	}
	return true
}

// MessageTypeParent reports blocks where type= names a message def.
func MessageTypeParent(parentKey string) bool {
	switch strings.ToLower(parentKey) {
	case "send_interface_message", "send_interface_toast":
		return true
	default:
		return false
	}
}
