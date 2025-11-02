package main

// This is a simple example program that demonstrates how to use the Paradox Script parser
// to extract event blocks from a Paradox event file.
//
// Usage: `go run getting-events.go <input_file>`
// The program will print out all event keys and their fields found in the specified file.
// It also prints the total number of events found.
// You'll have to build a struct for event fields as needed for your use case if you wish to do more with them.
//
import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"paradox-file-utils/parser"

	"github.com/antlr4-go/antlr/v4"
)

var EVENT_KEY_PATTERN = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*\.\d+$`)

type Event struct {
	Key      string
	Fields   map[string]interface{}
	RawBlock string // Optional: the raw text of the block
}

type EventCollector struct {
	Events []Event
}

func (v *EventCollector) VisitFile(ctx *parser.FileContext) interface{} {
	for _, line := range ctx.AllLine() {
		if stmt := line.Statement(); stmt != nil {
			if assign := stmt.Assignment(); assign != nil {
				key := assign.Key().GetText()
				val := assign.Value()
				// Only collect events with block values && matching key pattern
				if val.Block() != nil && EVENT_KEY_PATTERN.MatchString(key) {
					fields := collectFields(val.Block())
					v.Events = append(v.Events, Event{Key: key, Fields: fields})
				}
			}
		}
	}
	return nil
}

// Recursively collect fields from a block
func collectFields(block parser.IBlockContext) map[string]interface{} {
	fields := make(map[string]interface{})
	for _, content := range block.AllBlockContent() {
		if stmt := content.Statement(); stmt != nil {
			if assign := stmt.Assignment(); assign != nil {
				key := assign.Key().GetText()
				val := assign.Value()
				if val.Block() != nil {
					fields[key] = collectFields(val.Block())
				} else {
					fields[key] = val.GetText()
				}
			}
		}
	}
	return fields
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ./getting-events <input_file>")
		return
	}
	inputFile := os.Args[1]

	input, err := antlr.NewFileStream(inputFile)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}
	lexer := parser.NewParadoxLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewParadoxParser(stream)
	tree := p.File()

	collector := &EventCollector{}
	collector.VisitFile(tree.(*parser.FileContext))

	for _, event := range collector.Events {
		fmt.Printf("Event: %s\n", event.Key)
		for k, v := range event.Fields {
			fmt.Printf("  %s: %v\n", k, v)
		}
		fmt.Println(strings.Repeat("-", 40))
	}
	fmt.Printf("\nTotal Event Count: %d", len(collector.Events))
}
