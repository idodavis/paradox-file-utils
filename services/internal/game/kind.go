// kind.go is hover labels and short blurbs for def kinds.
// What the kind is, plus script quirks (cite form, unusual structure).
// Install _*.info / script_docs override the blurb at hover (Docs wins).
package game

import "strings"

type kindInfo struct {
	label, hint string
}

// kindMeta is keyed by CanonicalKind. Also: ephemeral names, RefFieldKind
// results, and a few hover-only kinds that are not folders.
var kindMeta = map[string]kindInfo{
	"loc_key":      {"localization", "Default loc text for this key"},
	"loc_value":    {"loc value", "Engine-supplied substitution, not a localization key"},
	"script_param": {"script parameter", "Passed parameter/variable in the macro; caller sets `scripted_trigger/effect_key = { PARAM_NAME = … }`"},
	"saved_scope":  {"saved scope", "`scope:name` from save_scope_as — targets below"},
	"character_flag": {
		"character flag",
		"add/has/remove_character_flag — flag name; values below",
	},
	"variable": {
		"variable",
		"`var:name` from set_variable — values below",
	},
	"global_variable": {
		"global variable",
		"`global_var:name` from set_global_variable — values below",
	},
	"local_variable": {
		"local variable",
		"`local_var:name` from set_local_variable — values below",
	},
	"dead_character_variable": {
		"dead character variable",
		"set/has/remove_dead_character_variable — values below",
	},

	"effect":  {"effect", "Changes state; `key = value` or a nested block"},
	"trigger": {"trigger", "True/false check; `key = value` or a nested block"},
	"scope":   {"scope", "Jump: `foo = { }` or `scope:name`"},
	"field":   {"field", "Assignment key; _*.info / script_docs fill Docs when present"},

	"scripted_trigger":  {"scripted trigger", "Reusable trigger; called with `key = yes` OR `{ SCRIPT_PARAM = … }`"},
	"scripted_effect":   {"scripted effect", "Reusable effect; called with `key = yes` OR `{ SCRIPT_PARAM = … }`"},
	"scripted_modifier": {"scripted modifier", "Reusable modifier; used as a modifier key"},
	"on_action":         {"on action", "Pulse hook; append events, don't replace vanilla effect"},
	"event":             {"event", "Scripted event; `namespace.id`; fire with trigger_event"},
	"namespace": {
		"event namespace",
		"Event file declaration; events are `id.N`. Shared across files; not an override.",
	},
	"decision":      {"decision", "Choice in the decisions list"},
	"script_value":  {"script value", "Named number or formula; use the key as a number"},
	"scripted_guis": {"scripted gui", "Window whose content is driven by script"},
	"customizable_localization": {
		"customizable localization",
		"Loc line chosen by trigger (`text = { trigger / localization_key }`)",
	},

	"gui_type":       {"gui type", "`type Name` or `template Name` in .gui"},
	"data_function":  {"data function", "GUI `[Function(...)]`; names from dump_data_types"},
	"mod_descriptor": {"mod descriptor", "descriptor.mod keys, or .metadata/metadata.json"},
	"define":         {"define", "Engine constant; `NCategory = { KEY = value }`"},
	"defines":        {"define", "Engine constant; `NCategory = { KEY = value }`"},

	"traits":         {"trait", "Character modifier; loc `trait_<key>`"},
	"culture":        {"culture", "People-group; field `culture = key`"},
	"religion":       {"religion", "Faith; field `religion = key`"},
	"faith":          {"faith", "Faith; field `faith = key`"},
	"title":          {"title", "Landed title; cite `titles:e_hre` or `title:k_france`"},
	"buildings":      {"building", "Constructable that employs pops or gives a local bonus"},
	"building_types": {"building", "Constructable that employs pops or gives a local bonus"},
	"building_groups": {
		"building group",
		"Family of buildings that share an employment category",
	},
	"characters":          {"character", "Historical or generated person with traits and a role"},
	"character_templates": {"character", "Named character used in events, history, or IGs"},
	"dynasties":           {"dynasty", "Dynasty; characters belong to one"},
	"governments":         {"government", "How a country is ruled"},
	"laws":                {"law", "Enactable policy; belongs to a law group"},
	"law_groups":          {"law group", "Mutually exclusive set of laws"},
	"modifiers":           {"modifier", "Named bundle of modifier keys applied to a scope"},
	"schemes":             {"scheme", "Hostile or personal scheme type"},
	"struggles":           {"struggle", "Regional struggle with phases and catalysts"},
	"lifestyles":          {"lifestyle", "Lifestyle with nested focuses"},
	"holdings":            {"holding", "Holding type on a title (castle, city, temple, …)"},
	"terrain":             {"terrain", "Map terrain; used by holdings and movement"},
	"casus_belli":         {"casus belli", "War justification; nested costs and targets"},
	"casus_belli_types":   {"casus belli", "War justification; nested costs and targets"},
	"war_goal_types":      {"war goal", "Demanded outcome of a play or war"},
	"coat_of_arms":        {"coat of arms", "Arms definition; field `coat_of_arms = key`"},
	"flag_definition":     {"flag definition", "Flag layout and colors"},
	"bookmarks":           {"bookmark", "Start date and playable setup"},
	"factions":            {"faction", "Faction type with demands and membership"},
	"story_cycles":        {"story cycle", "Chained story events with chapters"},
	"flavorization":       {"flavorization", "Dynamic name or title by trigger"},
	"ai":                  {"ai", "AI strategy weights"},
	"genes":               {"portrait", "Portrait gene ranges"},
	"men_at_arms_types":   {"regiment", "Men-at-arms type"},
	"council_positions":   {"council", "Council position"},
	"nicknames":           {"nickname", "Character nickname"},
	"secret_types":        {"secret", "Secret type with discovery and exposure"},
	"activities":          {"activity", "Activity type (feast, hunt, …) with phases"},
	"character_interactions": {
		"character interaction",
		"Character-menu action with send/accept effects",
	},
	"game_concepts":     {"game concept", "Encyclopedia entry; loc `[key|E]`"},
	"event_themes":      {"event theme", "Event window chrome (widgets, animations)"},
	"event_backgrounds": {"event background", "Event window background art"},
	"message":           {"message", "Interface toast; `title` is often a loc key"},
	"message_filter_types": {
		"message filter",
		"Message-log filter; loc `message_filter_<key>`",
	},
	"game_rule": {
		"game rule",
		"Named game rule",
	},
	"game_rule_setting": {
		"game rule setting",
		"Named game rule setting",
	},
	"game_rule_category": {
		"game rule category",
		"Group of game rules, defined in rule block, allows filtering in game rule window",
	},
	"important_actions": {"important action", "Alert or suggested action in the outliner"},
	"death_reason":      {"death reason", "Cause of death"},
	"achievements":      {"achievement", "Achievement with possible/happened triggers"},
	"inspirations":      {"inspiration", "Inspiration type for artifacts"},
	"vassal_stances":    {"vassal stance", "Vassal stance toward a liege"},
	"scripted_lists":    {"scripted list", "Named list consumed by any_/every_/random_"},
	"scripted_rules":    {"scripted rule", "Named rule the engine checks by key"},
	"focuses":           {"focus", "Lifestyle or education focus"},
	"dna_data":          {"dna", "Portrait DNA block"},

	"countries":           {"country", "Playable tag; start data is separate history/setup"},
	"country_definitions": {"country", "Playable tag; start data is separate history/setup"},
	"country_formation":   {"country formation", "Conditions and effects to form this tag"},
	"pops":                {"pop", "Population class with needs and employment"},
	"pop_types":           {"pop", "Population class with needs and employment"},
	"goods":               {"good", "Produced or traded commodity"},
	"institution":         {"institution", "Country investment or spreading innovation"},
	"institutions":        {"institution", "Country investment or spreading innovation"},
	"interest_groups":     {"interest group", "Political bloc with ideologies and clout"},
	"parties":             {"party", "Coalition of interest groups in government"},
	"journal_entries":     {"journal", "Conditional goal that fires events when completed"},
	"technologies":        {"technology", "Research node that unlocks PMs, laws, or units"},
	"decrees":             {"decree", "Timed state edict"},
	"treaties":            {"treaty", "Treaty article (goods, subjects, or rights)"},
	"treaty_articles":     {"treaty", "Treaty article (goods, subjects, or rights)"},
	"power_blocs": {
		"power bloc",
		"Bloc identity or principle that members follow",
	},
	"power_bloc_identities": {
		"power bloc",
		"Bloc identity that members follow",
	},
	"power_bloc_principles": {
		"power bloc",
		"Principle slotted into a bloc identity",
	},
	"states":        {"state", "Map region with provinces, traits, and homelands"},
	"subject_types": {"subject type", "Overlord–subject contract"},
	"objectives":    {"objective", "Guided playable goal for a country"},
	"diplomacy":     {"diplomacy", "Bilateral action or play that can escalate to war"},
	"diplomatic_actions": {
		"diplomacy",
		"Bilateral action (improve relations, alliance, …)",
	},
	"diplomatic_plays": {
		"diplomacy",
		"Escalating confrontation that can end in a treaty or war",
	},
	"diplomatic_catalysts": {
		"diplomacy",
		"Event that shifts relations or play escalation",
	},
	"production_methods": {
		"production method",
		"How a building employs pops and converts goods; listed on the building",
	},
	"production_method_groups": {
		"production method group",
		"Mutually exclusive PMs on a building",
	},
	"advances":   {"advance", "Age unlock for buildings, laws, units, or modifiers"},
	"disasters":  {"disaster", "Crisis after a threshold; fires a chain of events"},
	"diseases":   {"disease", "Spreading outbreak in locations"},
	"estates":    {"estate", "Privileged class with privileges and influence"},
	"missions":   {"mission", "Tree task with trigger, reward, and follow-ups"},
	"situations": {"situation", "Ongoing regional or world mechanic with ticks"},
	"international_organizations": {
		"international organization",
		"Multinational body with members, votes, and special laws",
	},
	"generic_actions": {"action", "Cabinet or character interaction in a scope"},
	"animation": {
		"animation",
		"Portrait or model clip; portraits use `animation = key`",
	},
}

// KindLabel is the hover kind line for t (underscores to spaces if unknown).
func KindLabel(t string) string {
	k := CanonicalKind(t)
	if info, ok := kindMeta[k]; ok {
		return info.label
	}
	return strings.ReplaceAll(t, "_", " ")
}

// KindHint is the fallback blurb for kind. Empty Docs at hover time means
// this line is shown; install _*.info / script_docs replace it.
func KindHint(gameID, kind string) string {
	k := CanonicalKind(kind)
	if h := gameKindHint(gameID, k); h != "" {
		return h
	}
	if info, ok := kindMeta[k]; ok {
		return info.hint
	}
	if strings.HasSuffix(k, "_key") {
		base := strings.ReplaceAll(strings.TrimSuffix(k, "_key"), "_", " ")
		return "Field on this " + base
	}
	return ""
}

// gameKindHint overlays script quirks that differ by title.
func gameKindHint(gameID, k string) string {
	switch gameID {
	case "eu5":
		switch k {
		case "event":
			return "Scripted event; `namespace.id`; first ID wins"
		case "buildings", "building_types":
			return "Constructable; INJECT:/REPLACE: prefixes ok"
		case "on_action":
			return "Pulse hook; append events; not INJECT"
		case "setup":
			return "Start-date TAG/location blocks (in_game and main_menu)"
		}
	case "vic3":
		switch k {
		case "event":
			return "Scripted event; `namespace.id`; fire via trigger_event (never self-fire)"
		case "buildings":
			return "Constructable; production methods listed on the building"
		case "history":
			return "Dated state/building/pop blocks (not a root history/ folder)"
		}
	}
	return ""
}
