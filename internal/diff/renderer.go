package diff

import (
	"fmt"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// DiffFormat represents different diff output formats
type DiffFormat string

const (
	FormatUnified DiffFormat = "unified"   // Unified diff format
	FormatHTML    DiffFormat = "html"      // HTML formatted diff
	FormatSideBySide DiffFormat = "sidebyside" // Side-by-side comparison
)

// RenderDiff renders diffs in the specified format
func RenderDiff(diffs []diffmatchpatch.Diff, format DiffFormat) string {
	switch format {
	case FormatUnified:
		return renderUnified(diffs)
	case FormatHTML:
		return renderHTML(diffs)
	case FormatSideBySide:
		return renderSideBySide(diffs)
	default:
		return renderUnified(diffs)
	}
}

// renderUnified renders a unified diff format
func renderUnified(diffs []diffmatchpatch.Diff) string {
	var result strings.Builder
	var lineNumA, lineNumB int
	
	for _, diff := range diffs {
		lines := strings.Split(diff.Text, "\n")
		
		for i, line := range lines {
			if i == len(lines)-1 && line == "" {
				// Skip trailing empty line
				continue
			}
			
			switch diff.Type {
			case diffmatchpatch.DiffDelete:
				result.WriteString(fmt.Sprintf("-%d: %s\n", lineNumA, line))
				lineNumA++
			case diffmatchpatch.DiffInsert:
				result.WriteString(fmt.Sprintf("+%d: %s\n", lineNumB, line))
				lineNumB++
			case diffmatchpatch.DiffEqual:
				result.WriteString(fmt.Sprintf(" %d: %s\n", lineNumA, line))
				lineNumA++
				lineNumB++
			}
		}
	}
	
	return result.String()
}

// renderHTML renders an HTML formatted diff
func renderHTML(diffs []diffmatchpatch.Diff) string {
	var result strings.Builder
	result.WriteString("<div class=\"diff\">\n")
	
	for _, diff := range diffs {
		escaped := escapeHTML(diff.Text)
		lines := strings.Split(escaped, "\n")
		
		for _, line := range lines {
			if line == "" {
				continue
			}
			
			switch diff.Type {
			case diffmatchpatch.DiffDelete:
				result.WriteString(fmt.Sprintf("<div class=\"diff-delete\">-%s</div>\n", line))
			case diffmatchpatch.DiffInsert:
				result.WriteString(fmt.Sprintf("<div class=\"diff-insert\">+%s</div>\n", line))
			case diffmatchpatch.DiffEqual:
				result.WriteString(fmt.Sprintf("<div class=\"diff-equal\"> %s</div>\n", line))
			}
		}
	}
	
	result.WriteString("</div>\n")
	return result.String()
}

// renderSideBySide renders a side-by-side comparison
func renderSideBySide(diffs []diffmatchpatch.Diff) string {
	var result strings.Builder
	var leftLines, rightLines []string
	
	for _, diff := range diffs {
		lines := strings.Split(diff.Text, "\n")
		
		for _, line := range lines {
			switch diff.Type {
			case diffmatchpatch.DiffDelete:
				leftLines = append(leftLines, "- "+line)
				rightLines = append(rightLines, "")
			case diffmatchpatch.DiffInsert:
				leftLines = append(leftLines, "")
				rightLines = append(rightLines, "+ "+line)
			case diffmatchpatch.DiffEqual:
				leftLines = append(leftLines, "  "+line)
				rightLines = append(rightLines, "  "+line)
			}
		}
	}
	
	// Pad to same length
	maxLen := len(leftLines)
	if len(rightLines) > maxLen {
		maxLen = len(rightLines)
	}
	
	for len(leftLines) < maxLen {
		leftLines = append(leftLines, "")
	}
	for len(rightLines) < maxLen {
		rightLines = append(rightLines, "")
	}
	
	result.WriteString("Left (A)                    | Right (B)\n")
	result.WriteString(strings.Repeat("-", 50) + "\n")
	
	for i := 0; i < maxLen; i++ {
		left := leftLines[i]
		right := rightLines[i]
		
		// Truncate if too long
		if len(left) > 25 {
			left = left[:22] + "..."
		}
		if len(right) > 25 {
			right = right[:22] + "..."
		}
		
		result.WriteString(fmt.Sprintf("%-25s | %s\n", left, right))
	}
	
	return result.String()
}

// escapeHTML escapes HTML special characters
func escapeHTML(text string) string {
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	text = strings.ReplaceAll(text, "\"", "&quot;")
	text = strings.ReplaceAll(text, "'", "&#39;")
	return text
}

