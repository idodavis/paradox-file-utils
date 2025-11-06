package object_parsers

import (
	"fmt"

	"paradox-file-utils/parser"
)

// TODO: Look into each field type, likely need to start with smallest construct INFO files first, then work way up to EVENTS.

// EventsFile wraps the generic parser with Event-specific types
type EventsFile struct {
	base   *parser.ParadoxFile
	Events map[string]*Event
}

// Event represents a single Event definition
type Event struct {
	Cooldown            *parser.Block `pdx:"cooldown"`              // Default to character, what type of event this is Optional, defaults to character, will make this event fire with a different expected root scope. none can be used to have no scopes set up. Optional, what special event file and widget in gui/event_windows should this event use, only used for character events If you have a cooldown, the recipient will get a variable saved with that duration. The variable name will be the event ID Anything that checks that an event is legal to fire will also check that the recipient doesn't have that variable
	ContentSource       *string       `pdx:"content_source"`        // DLC or Mod that this Event belongs to, shown in Event Window if set.
	LeftPortrait        *string       `pdx:"left_portrait"`         // Specify a character portrait to appear in the event on the specified position.
	RightPortrait       *string       `pdx:"right_portrait"`        //
	LowerLeftPortrait   *string       `pdx:"lower_left_portrait"`   // not used in all event types
	LowerCenterPortrait *string       `pdx:"lower_center_portrait"` //
	LowerRightPortrait  *string       `pdx:"lower_right_portrait"`  //
	Character           *string       `pdx:"character"`             // for letter events, required X can be one of:
	Camera              *string       `pdx:"camera"`                // Override camera to be used instead of the normal event ones
	Artifact            *parser.Block `pdx:"artifact"`              // Specifies outfit tags for this portrait in ascending priority (i.e. tag2 will "override" tag1 here if anything with tag2 is found in a specific portrait modifier category) If set to yes, portrait modifier categories in which nothing matches any of the event tags will be disabled completely (no by default) If set to yes, only the portrait will be shown, with no identifiable elements (no CoA, tooltips, clicks...) (no by default) Specify an artifact to appear in the event on the specified position
	OnTriggerFail       *parser.Block `pdx:"on_trigger_fail"`       // Can't be in the same position as a portrait Optional, as for character portraits This will be run if a queued event (or one triggered immediately from script) does not fulfil its trigger Events failing to trigger from an on-action will *not* run this
	Widgets             *parser.Block `pdx:"widgets"`               // Specify custom widgets to embed in the event. See section about Custom Widgets below.
	Trigger             *parser.Block `pdx:"trigger"`               // Theme to use in the event. For a list, check: 00_event_themes.txt A background that can be shown when the event pops up. This overrides the theme one. In case that there are multiples the first one that fits the trigger will be the one selected. In case none fits the ones inthe theme will be checked after. Receives the event scope to check if it's valid. Path to the texture A transition that can be shown when the event pops up, before the event options and backgrounds. This overrides the theme one. In case that there are multiples the first one that fits the trigger will be the one selected. In case none fits the ones inthe theme will be checked after. Receives the event scope to check if it's valid. Path to the texture A 2d effect that can be put on top of the background. This overrides the theme one. In case that there are multiples the first one that fits the trigger will be the one selected. In case none fits the ones inthe theme will be checked after. Receives the event scope to check if it's valid. key to the effect An icon that can be shown when the event pops up. This overrides the theme one. In case that there are multiples the first one that fits the trigger will be the one selected. In case none fits the ones inthe theme will be checked after. Receives the event scope to check if it's valid. Path to the texture The header asset located behind the event icon. This overrides the header asset defined by the theme. If there are multiples defined here, the first one that passes its trigger will be selected. If none are valid, then the theme's header asset will be used
	// Raw block for fields not yet typed
	RawBlock *parser.Block
}

// Cooldown represents the cooldown block
type Cooldown struct {
	RawBlock *parser.Block
}

// TriggeredAnimation represents the triggered_animation block
type Triggered_animation struct {
	Trigger           *string `pdx:"trigger"`
	Animation         *string `pdx:"animation"`
	ScriptedAnimation *string `pdx:"scripted_animation"`
	Camera            *string `pdx:"camera"`
	RawBlock          *parser.Block
}

// TriggeredOutfit represents the triggered_outfit block
type Triggered_outfit struct {
	Trigger             *string `pdx:"trigger"`
	OutfitTags          *string `pdx:"outfit_tags"`
	RemoveDefaultOutfit *string `pdx:"remove_default_outfit"`
	HideInfo            *string `pdx:"hide_info"`
	RawBlock            *parser.Block
}

// Artifact represents the artifact block
type Artifact struct {
	Target   *string `pdx:"target"`
	Position *string `pdx:"position"`
	RawBlock *parser.Block
}

// OnTriggerFail represents the on_trigger_fail block
type On_trigger_fail struct {
	RawBlock *parser.Block
}

// Widgets represents the widgets block
type Widgets struct {
	Widget            *parser.Block `pdx:"widget"`
	IsShown           *parser.Block `pdx:"is_shown"`
	Gui               *string       `pdx:"gui"`
	Container         *string       `pdx:"container"`
	Controller        *string       `pdx:"controller"`
	SetupScope        *parser.Block `pdx:"setup_scope"`
	Name              *string       `pdx:"name"`
	Trigger           *parser.Block `pdx:"trigger"`
	ShowAsUnavailable *parser.Block `pdx:"show_as_unavailable"`
	HighlightPortrait *string       `pdx:"highlight_portrait"`
	Base              *string       `pdx:"base"`
	Modifier          *parser.Block `pdx:"modifier"`
	Add               *string       `pdx:"add"`
	Modifier          *parser.Block `pdx:"modifier"`
	Factor            *string       `pdx:"factor"`
	Base              *string       `pdx:"base"`
	If                *parser.Block `pdx:"if"`
	Limit             *parser.Block `pdx:"limit"`
	Add               *string       `pdx:"add"`
	ElseIf            *parser.Block `pdx:"else_if"`
	Limit             *parser.Block `pdx:"limit"`
	Multiply          *string       `pdx:"multiply"`
	RawBlock          *parser.Block
}

// Trigger represents the trigger block
type Trigger struct {
	Reference *string `pdx:"reference"`
	RawBlock  *parser.Block
}

// ParseEventsFile parses a Event file using the base parser
func ParseEventsFile(filename string) (*EventsFile, error) {
	base, err := parser.ParseFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}
	return convertToEventsFile(base)
}

// convertToEventsFile converts generic ParadoxFile to EventsFile
func convertToEventsFile(base *parser.ParadoxFile) (*EventsFile, error) {
	Events := make(map[string]*Event)

	// Extract Events from base.Lines
	for _, line := range base.Lines {
		if line.Statement != nil && line.Statement.Assignment != nil {
			assign := line.Statement.Assignment
			key := parser.GetKeyText(assign.Key)

			// Check if this is an Event key (format: namespace.number)
			if assign.Value != nil && assign.Value.Block != nil {
				item := &Event{
					RawBlock: assign.Value.Block,
				}

				// Extract fields from the block
				extractEventFields(item, assign.Value.Block)

				Events[key] = item
			}
		}
	}

	return &EventsFile{
		base:   base,
		Events: Events,
	}, nil
}

// extractEventFields extracts typed fields from a block
func extractEventFields(item *Event, block *parser.Block) {
	for _, blockItem := range block.Items {
		if blockItem.Statement != nil && blockItem.Statement.Assignment != nil {
			assign := blockItem.Statement.Assignment
			key := parser.GetKeyText(assign.Key)

			switch key {
			case "cooldown":
				if assign.Value != nil && assign.Value.Block != nil {
					event.Cooldown = assign.Value.Block
				}
			case "content_source":
				if assign.Value != nil && assign.Value.Atom != nil {
					if val := assign.Value.Atom; val != nil {
						if val.Str != nil {
							s := *val.Str
							event.ContentSource = &s
						} else if val.ID != nil {
							s := *val.ID
							event.ContentSource = &s
						}
					}
				}
			case "left_portrait":
				if assign.Value != nil && assign.Value.Atom != nil {
					if val := assign.Value.Atom; val != nil {
						if val.Str != nil {
							s := *val.Str
							event.LeftPortrait = &s
						} else if val.ID != nil {
							s := *val.ID
							event.LeftPortrait = &s
						}
					}
				}
			case "right_portrait":
				if assign.Value != nil && assign.Value.Atom != nil {
					if val := assign.Value.Atom; val != nil {
						if val.Str != nil {
							s := *val.Str
							event.RightPortrait = &s
						} else if val.ID != nil {
							s := *val.ID
							event.RightPortrait = &s
						}
					}
				}
			case "lower_left_portrait":
				if assign.Value != nil && assign.Value.Atom != nil {
					if val := assign.Value.Atom; val != nil {
						if val.Str != nil {
							s := *val.Str
							event.LowerLeftPortrait = &s
						} else if val.ID != nil {
							s := *val.ID
							event.LowerLeftPortrait = &s
						}
					}
				}
			case "lower_center_portrait":
				if assign.Value != nil && assign.Value.Atom != nil {
					if val := assign.Value.Atom; val != nil {
						if val.Str != nil {
							s := *val.Str
							event.LowerCenterPortrait = &s
						} else if val.ID != nil {
							s := *val.ID
							event.LowerCenterPortrait = &s
						}
					}
				}
			case "lower_right_portrait":
				if assign.Value != nil && assign.Value.Atom != nil {
					if val := assign.Value.Atom; val != nil {
						if val.Str != nil {
							s := *val.Str
							event.LowerRightPortrait = &s
						} else if val.ID != nil {
							s := *val.ID
							event.LowerRightPortrait = &s
						}
					}
				}
			case "character":
				if assign.Value != nil && assign.Value.Atom != nil {
					if val := assign.Value.Atom; val != nil {
						if val.Str != nil {
							s := *val.Str
							event.Character = &s
						} else if val.ID != nil {
							s := *val.ID
							event.Character = &s
						}
					}
				}
			case "triggered_animation":
				if assign.Value != nil && assign.Value.Block != nil {
					if event.TriggeredAnimation == nil {
						event.TriggeredAnimation = []*parser.Block{}
					}
					event.TriggeredAnimation = append(event.TriggeredAnimation, assign.Value.Block)
				}
			case "triggered_outfit":
				if assign.Value != nil && assign.Value.Block != nil {
					if event.TriggeredOutfit == nil {
						event.TriggeredOutfit = []*parser.Block{}
					}
					event.TriggeredOutfit = append(event.TriggeredOutfit, assign.Value.Block)
				}
			case "camera":
				if assign.Value != nil && assign.Value.Atom != nil {
					if val := assign.Value.Atom; val != nil {
						if val.Str != nil {
							s := *val.Str
							event.Camera = &s
						} else if val.ID != nil {
							s := *val.ID
							event.Camera = &s
						}
					}
				}
			case "artifact":
				if assign.Value != nil && assign.Value.Block != nil {
					event.Artifact = assign.Value.Block
				}
			case "on_trigger_fail":
				if assign.Value != nil && assign.Value.Block != nil {
					event.OnTriggerFail = assign.Value.Block
				}
			case "widgets":
				if assign.Value != nil && assign.Value.Block != nil {
					event.Widgets = assign.Value.Block
				}
			case "trigger":
				if assign.Value != nil && assign.Value.Block != nil {
					event.Trigger = assign.Value.Block
				}
			}
		}
	}
}

// GetBase returns the underlying generic parser result
func (f *EventsFile) GetBase() *parser.ParadoxFile {
	return f.base
}

// GetEvent returns a specific Event by key
func (f *EventsFile) GetEvent(key string) (*Event, bool) {
	Event, ok := f.Events[key]
	return Event, ok
}

// GetAllEvents returns all Events
func (f *EventsFile) GetAllEvents() map[string]*Event {
	return f.Events
}
