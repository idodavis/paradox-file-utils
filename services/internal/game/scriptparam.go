// scriptparam.go detects scripted macro parameters ($NAME$) and loc engine
// value filters ($VALUE|=+0$).

package game

import (
	"strconv"
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

// IsLocEngineValue reports loc $key|filter$ where filter is a numeric format
// (engine-supplied value), not a loc-key reuse filter like U or l.
func IsLocEngineValue(key, filter string) bool {
	if filter == "" {
		return false
	}
	if !strings.EqualFold(key, "VALUE") {
		return false
	}
	return isNumericLocFilter(filter)
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

// IsScriptValueField reports assignment keys that commonly name a script_value.
func IsScriptValueField(field string) bool {
	switch strings.ToLower(field) {
	case "value", "add", "multiply", "subtract", "divide", "min", "max":
		return true
	default:
		return false
	}
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

// IsScriptValueRHS reports a scalar that could name a script_value (not a number).
func IsScriptValueRHS(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" || text == "yes" || text == "no" {
		return false
	}
	if strings.ContainsAny(text, ":.$") {
		return false
	}
	if _, err := strconv.ParseFloat(text, 64); err == nil {
		return false
	}
	return isScriptParamByte(text[0], true)
}
