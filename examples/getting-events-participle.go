package main

// This is an example program that demonstrates how to use the Participle-based
// Paradox Script parser to extract event blocks from a Paradox event file.
//
// Usage: `go run getting-events-participle.go <input_file>`
// The program will print out all event keys and their fields found in the specified file.
// It also prints the total number of events found.
//
import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"paradox-file-utils/parser"
)

var EVENT_KEY_PATTERN = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*\.\d+$`)

type Event struct {
	Key    string
	Fields map[string]string
}

type EventCollector struct {
	Events []Event
}

// CollectEvents collects all events from a parsed ParadoxFile
func (eventCollector *EventCollector) CollectEvents(file *parser.ParadoxFile) {
	for _, entry := range file.Entries {

		if entry.Assignment == nil {
			continue
		}

		// Only Entries with assignments to objects
		if entry.Assignment.Object != nil {
			if EVENT_KEY_PATTERN.MatchString(entry.Assignment.Key) {
				eventCollector.Events = append(eventCollector.Events, Event{
					Key:    entry.Assignment.Key,
					Fields: map[string]string{"value": entry.Assignment.Object.ToString()},
				})
			}
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ./getting-events-participle <input_file>")
		return
	}
	inputFile := os.Args[1]

	// Parse using Participle parser
	parsedFile, err := parser.ParseFile(inputFile)
	if err != nil {
		fmt.Printf("Error parsing file: %v\n", err)
		return
	}

	collector := &EventCollector{}
	collector.CollectEvents(parsedFile)

	for _, event := range collector.Events {
		fmt.Printf("Event: %s\n", event.Key)
		for k, v := range event.Fields {
			fmt.Printf("  %s: %s\n", k, v)
		}
		fmt.Println(strings.Repeat("-", 40))
	}
	fmt.Printf("\nTotal Event Count: %d\n", len(collector.Events))
}
