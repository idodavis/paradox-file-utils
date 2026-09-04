// kind.go is hover labels and short blurbs for def kinds.
// What the kind is, plus script quirks (cite form, unusual structure).
// Install _*.info / script_docs override the blurb at hover (Docs wins).
package game

import "strings"

type kindInfo struct {
	label, hint string
}

var kindMeta = map[string]kindInfo{
	// Loc / ephemeral — hint describes the value in Body plus cite form.
	"loc_key":          {"localization", "Default loc text for this key"},
	"localization":     {"localization", "Default loc text for this key"},
	"loc_value":        {"loc value", "Engine-supplied number, not a localization key"},
	"script_param":     {"script parameter", "Passed parameter/variable in the macro; caller sets `scripted_trigger/effect_key = { PARAM_NAME = … }`"},
	"script_parameter": {"script parameter", "Passed parameter/variable in the macro; caller sets `scripted_trigger/effect_key = { PARAM_NAME = … }`"},
	"saved_scope":      {"saved scope", "`scope:name` from save_scope_as — targets below"},
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
	"on_actions":        {"on action", "Pulse hook; append events, don't replace vanilla effect"},
	"event":             {"event", "Scripted event; `namespace.id`; fire with trigger_event"},
	"events":            {"event", "Scripted event; `namespace.id`; fire with trigger_event"},
	"decision":          {"decision", "Choice in the decisions list"},
	"decisions":         {"decision", "Choice in the decisions list"},
	"script_value":      {"script value", "Named number or formula; use the key as a number"},
	"script_values":     {"script value", "Named number or formula; use the key as a number"},
	"scripted_gui":      {"scripted gui", "Window whose content is driven by script"},
	"scripted_guis":     {"scripted gui", "Window whose content is driven by script"},
	"customizable_localization": {
		"customizable localization",
		"Loc line chosen by trigger (`text = { trigger / localization_key }`)",
	},
	"customizable_loc": {
		"customizable localization",
		"Loc line chosen by trigger (`text = { trigger / localization_key }`)",
	},

	"gui_type":       {"gui type", "`type Name` or `template Name` in .gui"},
	"gui":            {"gui type", "`type Name` or `template Name` in .gui"},
	"data_function":  {"data function", "GUI `[Function(...)]`; names from dump_data_types"},
	"data_types":     {"data function", "GUI `[Function(...)]`; names from dump_data_types"},
	"mod_descriptor": {"mod descriptor", "descriptor.mod keys, or .metadata/metadata.json"},
	"mod":            {"mod descriptor", "descriptor.mod keys, or .metadata/metadata.json"},
	"meta":           {"mod descriptor", "descriptor.mod keys, or .metadata/metadata.json"},
	"define":         {"define", "Engine constant; `NCategory = { KEY = value }`"},
	"defines":        {"define", "Engine constant; `NCategory = { KEY = value }`"},

	"trait":          {"trait", "Character modifier; loc `trait_<key>`"},
	"traits":         {"trait", "Character modifier; loc `trait_<key>`"},
	"culture":        {"culture", "People-group; field `culture = key`"},
	"cultures":       {"culture", "People-group; field `culture = key`"},
	"religion":       {"religion", "Faith; field `religion = key`"},
	"religions":      {"religion", "Faith; field `religion = key`"},
	"faith":          {"faith", "Faith; field `faith = key`"},
	"faiths":         {"faith", "Faith; field `faith = key`"},
	"title":          {"title", "Landed title; nested e_/k_/d_/c_/b_; `title = key`"},
	"titles":         {"title", "Landed title; nested e_/k_/d_/c_/b_; `title = key`"},
	"landed_titles":  {"title", "Landed title; nested e_/k_/d_/c_/b_; `title = key`"},
	"building":       {"building", "Constructable that employs pops or gives a local bonus"},
	"buildings":      {"building", "Constructable that employs pops or gives a local bonus"},
	"building_types": {"building", "Constructable that employs pops or gives a local bonus"},
	"building_groups": {
		"building group",
		"Family of buildings that share an employment category",
	},
	"character":           {"character", "Historical or generated person with traits and a role"},
	"characters":          {"character", "Historical or generated person with traits and a role"},
	"character_templates": {"character", "Named character used in events, history, or IGs"},
	"dynasty":             {"dynasty", "Dynasty; characters belong to one"},
	"dynasties":           {"dynasty", "Dynasty; characters belong to one"},
	"government":          {"government", "How a country is ruled"},
	"governments":         {"government", "How a country is ruled"},
	"law":                 {"law", "Enactable policy; belongs to a law group"},
	"laws":                {"law", "Enactable policy; belongs to a law group"},
	"law_groups":          {"law group", "Mutually exclusive set of laws"},
	"modifier":            {"modifier", "Named bundle of modifier keys applied to a scope"},
	"modifiers":           {"modifier", "Named bundle of modifier keys applied to a scope"},
	"modifier_types":      {"modifier type", "Declares a modifier key's value kind"},
	"doctrine":            {"doctrine", "Faith doctrine; stacked on a religion/faith"},
	"doctrines":           {"doctrine", "Faith doctrine; stacked on a religion/faith"},
	"scheme":              {"scheme", "Hostile or personal scheme type"},
	"schemes":             {"scheme", "Hostile or personal scheme type"},
	"struggle":            {"struggle", "Regional struggle with phases and catalysts"},
	"struggles":           {"struggle", "Regional struggle with phases and catalysts"},
	"lifestyle":           {"lifestyle", "Lifestyle with nested focuses"},
	"lifestyles":          {"lifestyle", "Lifestyle with nested focuses"},
	"holding":             {"holding", "Holding type on a title (castle, city, temple, …)"},
	"holdings":            {"holding", "Holding type on a title (castle, city, temple, …)"},
	"terrain":             {"terrain", "Map terrain; used by holdings and movement"},
	"casus_belli":         {"casus belli", "War justification; nested costs and targets"},
	"casus_belli_types":   {"casus belli", "War justification; nested costs and targets"},
	"war_goal":            {"war goal", "Demanded outcome of a play or war"},
	"war_goals":           {"war goal", "Demanded outcome of a play or war"},
	"war_goal_types":      {"war goal", "Demanded outcome of a play or war"},
	"coat_of_arms":        {"coat of arms", "Arms definition; field `coat_of_arms = key`"},
	"flag_definition":     {"flag definition", "Flag layout and colors"},
	"flag_definitions":    {"flag definition", "Flag layout and colors"},
	"bookmark":            {"bookmark", "Start date and playable setup"},
	"bookmarks":           {"bookmark", "Start date and playable setup"},
	"faction":             {"faction", "Faction type with demands and membership"},
	"factions":            {"faction", "Faction type with demands and membership"},
	"legend":              {"legend", "Legend type with quality and completion"},
	"legends":             {"legend", "Legend type with quality and completion"},
	"story":               {"story cycle", "Chained story events with chapters"},
	"stories":             {"story cycle", "Chained story events with chapters"},
	"story_cycles":        {"story cycle", "Chained story events with chapters"},
	"flavorization":       {"flavorization", "Dynamic name or title by trigger"},
	"travel":              {"travel", "Travel option, danger, or destination"},
	"history":             {"history", "Dated start-state blocks"},
	"map":                 {"map", "Map definition, not a script object"},
	"music":               {"music", "Music track or mood referenced by key"},
	"sound":               {"sound", "Sound effect referenced by key"},
	"ai":                  {"ai", "AI strategy weights"},
	"portrait":            {"portrait", "Portrait or gene definition"},
	"portraits":           {"portrait", "Portrait or gene definition"},
	"genes":               {"portrait", "Portrait gene ranges"},
	"regiment":            {"regiment", "Men-at-arms type"},
	"regiments":           {"regiment", "Men-at-arms type"},
	"men_at_arms":         {"regiment", "Men-at-arms type"},
	"men_at_arms_types":   {"regiment", "Men-at-arms type"},
	"unit":                {"unit", "Land or naval regiment type"},
	"units":               {"unit", "Land or naval regiment type"},
	"council":             {"council", "Council position"},
	"council_positions":   {"council", "Council position"},
	"nickname":            {"nickname", "Character nickname"},
	"nicknames":           {"nickname", "Character nickname"},
	"secret":              {"secret", "Secret type with discovery and exposure"},
	"secrets":             {"secret", "Secret type with discovery and exposure"},
	"secret_types":        {"secret", "Secret type with discovery and exposure"},
	"artifact":            {"artifact", "Artifact type or template"},
	"artifacts":           {"artifact", "Artifact type or template"},
	"activity":            {"activity", "Activity type (feast, hunt, …) with phases"},
	"activities":          {"activity", "Activity type (feast, hunt, …) with phases"},
	"character_interaction": {
		"character interaction",
		"Character-menu action with send/accept effects",
	},
	"character_interactions": {
		"character interaction",
		"Character-menu action with send/accept effects",
	},
	"game_concept":      {"game concept", "Encyclopedia entry; loc `[key|E]`"},
	"game_concepts":     {"game concept", "Encyclopedia entry; loc `[key|E]`"},
	"concept":           {"game concept", "Encyclopedia entry; loc `[key|E]`"},
	"concepts":          {"game concept", "Encyclopedia entry; loc `[key|E]`"},
	"event_theme":       {"event theme", "Event window chrome (widgets, animations)"},
	"event_themes":      {"event theme", "Event window chrome (widgets, animations)"},
	"event_background":  {"event background", "Event window background art"},
	"event_backgrounds": {"event background", "Event window background art"},
	"message":           {"message", "Interface toast; `title` is often a loc key"},
	"messages":          {"message", "Interface toast; `title` is often a loc key"},
	"message_filter": {
		"message filter",
		"Message-log filter; loc `message_filter_<key>`",
	},
	"message_filter_types": {
		"message filter",
		"Message-log filter; loc `message_filter_<key>`",
	},
	"game_rule": {
		"game rule",
		"Named game rule",
	},
	"game_rules": {
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
	"important_action":  {"important action", "Alert or suggested action in the outliner"},
	"important_actions": {"important action", "Alert or suggested action in the outliner"},
	"death_reason":      {"death reason", "Cause of death"},
	"death_reasons":     {"death reason", "Cause of death"},
	"achievement":       {"achievement", "Achievement with possible/happened triggers"},
	"achievements":      {"achievement", "Achievement with possible/happened triggers"},
	"holy_order":        {"holy order", "Holy order type"},
	"holy_orders":       {"holy order", "Holy order type"},
	"opinion":           {"opinion", "Opinion modifier applied to a character"},
	"opinions":          {"opinion", "Opinion modifier applied to a character"},
	"hook":              {"hook", "Hook type (weak/strong) on a character"},
	"hooks":             {"hook", "Hook type (weak/strong) on a character"},
	"memory":            {"memory", "Character memory type"},
	"memories":          {"memory", "Character memory type"},
	"inspiration":       {"inspiration", "Inspiration type for artifacts"},
	"inspirations":      {"inspiration", "Inspiration type for artifacts"},
	"vassal_stance":     {"vassal stance", "Vassal stance toward a liege"},
	"vassal_stances":    {"vassal stance", "Vassal stance toward a liege"},
	"scripted_list":     {"scripted list", "Named list consumed by any_/every_/random_"},
	"scripted_lists":    {"scripted list", "Named list consumed by any_/every_/random_"},
	"scripted_rule":     {"scripted rule", "Named rule the engine checks by key"},
	"scripted_rules":    {"scripted rule", "Named rule the engine checks by key"},
	"court_position":    {"court position", "Court office with aptitude and salary"},
	"court_positions":   {"court position", "Court office with aptitude and salary"},
	"focus":             {"focus", "Lifestyle or education focus"},
	"focuses":           {"focus", "Lifestyle or education focus"},
	"dna_data":          {"dna", "Portrait DNA block"},

	"country":             {"country", "Playable tag; start data is separate history/setup"},
	"countries":           {"country", "Playable tag; start data is separate history/setup"},
	"country_definitions": {"country", "Playable tag; start data is separate history/setup"},
	"country_formation":   {"country formation", "Conditions and effects to form this tag"},
	"pop":                 {"pop", "Population class with needs and employment"},
	"pops":                {"pop", "Population class with needs and employment"},
	"pop_types":           {"pop", "Population class with needs and employment"},
	"good":                {"good", "Produced or traded commodity"},
	"goods":               {"good", "Produced or traded commodity"},
	"institution":         {"institution", "Country investment or spreading innovation"},
	"institutions":        {"institution", "Country investment or spreading innovation"},
	"interest_group":      {"interest group", "Political bloc with ideologies and clout"},
	"interest_groups":     {"interest group", "Political bloc with ideologies and clout"},
	"party":               {"party", "Coalition of interest groups in government"},
	"parties":             {"party", "Coalition of interest groups in government"},
	"journal":             {"journal", "Conditional goal that fires events when completed"},
	"journal_entries":     {"journal", "Conditional goal that fires events when completed"},
	"journal_entry":       {"journal", "Conditional goal that fires events when completed"},
	"technology":          {"technology", "Research node that unlocks PMs, laws, or units"},
	"technologies":        {"technology", "Research node that unlocks PMs, laws, or units"},
	"decree":              {"decree", "Timed state edict"},
	"decrees":             {"decree", "Timed state edict"},
	"treaty":              {"treaty", "Treaty article (goods, subjects, or rights)"},
	"treaties":            {"treaty", "Treaty article (goods, subjects, or rights)"},
	"treaty_articles":     {"treaty", "Treaty article (goods, subjects, or rights)"},
	"power_bloc": {
		"power bloc",
		"Bloc identity or principle that members follow",
	},
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
	"state":         {"state", "Map region with provinces, traits, and homelands"},
	"states":        {"state", "Map region with provinces, traits, and homelands"},
	"state_regions": {"state", "Map region with provinces, traits, and homelands"},
	"subject_type":  {"subject type", "Overlord–subject contract"},
	"subject_types": {"subject type", "Overlord–subject contract"},
	"objective":     {"objective", "Guided playable goal for a country"},
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
	"advance":    {"advance", "Age unlock for buildings, laws, units, or modifiers"},
	"advances":   {"advance", "Age unlock for buildings, laws, units, or modifiers"},
	"disaster":   {"disaster", "Crisis after a threshold; fires a chain of events"},
	"disasters":  {"disaster", "Crisis after a threshold; fires a chain of events"},
	"disease":    {"disease", "Spreading outbreak in locations"},
	"diseases":   {"disease", "Spreading outbreak in locations"},
	"estate":     {"estate", "Privileged class with privileges and influence"},
	"estates":    {"estate", "Privileged class with privileges and influence"},
	"mission":    {"mission", "Tree task with trigger, reward, and follow-ups"},
	"missions":   {"mission", "Tree task with trigger, reward, and follow-ups"},
	"situation":  {"situation", "Ongoing regional or world mechanic with ticks"},
	"situations": {"situation", "Ongoing regional or world mechanic with ticks"},
	"international_organization": {
		"international organization",
		"Multinational body with members, votes, and special laws",
	},
	"international_organizations": {
		"international organization",
		"Multinational body with members, votes, and special laws",
	},
	"action":          {"action", "Cabinet or character interaction in a scope"},
	"actions":         {"action", "Cabinet or character interaction in a scope"},
	"generic_actions": {"action", "Cabinet or character interaction in a scope"},
	"setup":           {"setup", "Start-date ownership, cores, and starting buildings"},
	"art":             {"art", "Illustration, icon, or 3D asset referenced by texture"},
	"animation": {
		"animation",
		"Portrait or model clip; portraits use `animation = key`",
	},
	"animations": {
		"animation",
		"Portrait or model clip; portraits use `animation = key`",
	},
	"portrait_animations": {
		"animation",
		"Portrait clip; portraits use `animation = key`",
	},
}

// KindLabel is the hover kind line for t (underscores to spaces if unknown).
func KindLabel(t string) string {
	k := strings.ReplaceAll(CanonicalKind(t), " ", "_")
	if info, ok := kindMeta[k]; ok {
		return info.label
	}
	return strings.ReplaceAll(t, "_", " ")
}

// KindHint is the fallback blurb for kind. Empty Docs at hover time means
// this line is shown; install _*.info / script_docs replace it.
func KindHint(gameID, kind string) string {
	k := strings.ReplaceAll(CanonicalKind(kind), " ", "_")
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
		case "event", "events":
			return "Scripted event; `namespace.id`; first ID wins"
		case "building", "buildings", "building_types":
			return "Constructable; INJECT:/REPLACE: prefixes ok"
		case "on_action", "on_actions":
			return "Pulse hook; append events; not INJECT"
		case "setup":
			return "Start-date TAG/location blocks (in_game and main_menu)"
		}
	case "vic3":
		switch k {
		case "event", "events":
			return "Scripted event; `namespace.id`; fire via trigger_event (never self-fire)"
		case "building", "buildings":
			return "Constructable; production methods listed on the building"
		case "history":
			return "Dated state/building/pop blocks (not a root history/ folder)"
		}
	}
	return ""
}
