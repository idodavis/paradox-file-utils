// loccoverage_test.go covers strict-only missing loc, broad-field skips,
// game-rule / message-filter orphans, and descriptor/metadata name refs.
package views

import (
	"strings"
	"testing"

	"paradox-modding-tools/services/internal/catalog"
)

func coverageIssue(rows []LocCoverage, lang, kind, key string) bool {
	for _, row := range rows {
		if row.Language != lang {
			continue
		}
		for _, m := range row.Issues {
			if m.Kind == kind && m.Key == key {
				return true
			}
		}
	}
	return false
}

func TestCoverageStrictAndSkips(t *testing.T) {
	s := buildSession(t, "ck3", nil, &catalog.VanillaLoc{
		Sites: map[string]catalog.LocEntry{"vanilla_key": {Value: "Hi"}},
	}, []catalog.ModInput{oneMod(t, "ck3", map[string]string{
		"events/x.txt": `namespace = ns
ns.1 = {
	type = character_event
	title = missing_key
	name = some_var
	desc = used_key
}
`,
		"localization/english/a_l_english.yml": "" +
			"l_english:\n used_key:0 \"Hi\"\n orphan_key:0 \"Bye\"\n",
		"localization/english/game_rules_l_english.yml": "" +
			"l_english:\n setting_rule:0 \"Rule\"\n",
		"localization/english/message_filter_l_english.yml": "" +
			"l_english:\n filter_key:0 \"Filter\"\n",
	})})
	cov := Coverage(s)
	if !coverageIssue(cov, "english", "missing", "missing_key") {
		t.Fatalf("missing_key not reported: %+v", cov)
	}
	if coverageIssue(cov, "english", "missing", "some_var") {
		t.Fatal("loc-broad name= should not be missing")
	}
	if coverageIssue(cov, "english", "orphaned", "setting_rule") {
		t.Fatal("game_rules loc should not be orphaned")
	}
	if coverageIssue(cov, "english", "orphaned", "filter_key") {
		t.Fatal("message_filter loc should not be orphaned")
	}
	if !coverageIssue(cov, "english", "orphaned", "orphan_key") {
		t.Fatal("unused loc should stay orphaned")
	}
}

func TestCoverageConventionLocsInGenericFile(t *testing.T) {
	s := buildSession(t, "ck3", nil, nil, []catalog.ModInput{oneMod(t, "ck3", map[string]string{
		"common/game_rules/00.txt": `my_rule = {
	default = opt_a
	opt_a = { }
}
`,
		"common/messages/00.txt": `my_msg = { }
`,
		"localization/english/mod_l_english.yml": "" +
			"l_english:\n rule_my_rule:0 \"R\"\n setting_opt_a:0 \"A\"\n my_msg:0 \"M\"\n unused_key:0 \"U\"\n",
	})})
	cov := Coverage(s)
	if coverageIssue(cov, "english", "orphaned", "rule_my_rule") {
		t.Fatal("rule_<id> should not be orphaned")
	}
	if coverageIssue(cov, "english", "orphaned", "setting_opt_a") {
		t.Fatal("game_rules option setting should not be orphaned")
	}
	if !coverageIssue(cov, "english", "orphaned", "my_msg") {
		t.Fatal("message def id is not a loc key")
	}
	if !coverageIssue(cov, "english", "orphaned", "unused_key") {
		t.Fatal("unused loc should stay orphaned")
	}
}

func TestCoverageIgnoresVanillaCacheRefs(t *testing.T) {
	s := buildSession(t, "ck3", &catalog.VanillaCache{
		LocRefs: []catalog.Ref{{
			Key: "vanilla_title", Kind: "loc", Path: "game/events/x.txt", Line: 1,
		}},
	}, nil, []catalog.ModInput{oneMod(t, "ck3", map[string]string{
		"events/x.txt": `namespace = ns
ns.1 = { title = event_message_title }
`,
		"common/messages/zz.txt": `quieter_events_neutral = { title = event_message_title }
`,
		"localization/english/zz_l_english.yml": "" +
			"l_english:\n event_message_title:0 \"T\"\n unused_mod:0 \"U\"\n",
	})})
	cov := Coverage(s)
	if coverageIssue(cov, "english", "missing", "vanilla_title") {
		t.Fatal("vanilla cache loc refs must not be missing")
	}
	if coverageIssue(cov, "english", "missing", "quieter_events_neutral") {
		t.Fatal("message id must not be invented as missing")
	}
	if coverageIssue(cov, "english", "missing", "event_message_title") {
		t.Fatal("defined title loc should not be missing")
	}
	if !coverageIssue(cov, "english", "orphaned", "unused_mod") {
		t.Fatal("unused mod loc should stay orphaned")
	}
}

func TestCoverageSkipsDescriptorName(t *testing.T) {
	s := buildSession(t, "ck3", nil, nil, []catalog.ModInput{oneMod(t, "ck3", map[string]string{
		"events/x.txt": `namespace = ns
ns.1 = { type = character_event title = used_key }
`,
		"localization/english/a_l_english.yml": "l_english:\n used_key:0 \"Hi\"\n",
	})})
	for _, row := range Coverage(s) {
		for _, m := range row.Issues {
			if strings.Contains(strings.ToLower(m.File), "descriptor.mod") ||
				m.Key == "t" && m.Kind == "missing" {
				t.Fatalf("descriptor name leaked: %+v", m)
			}
		}
	}
}

func TestCoverageSkipsMetadataName(t *testing.T) {
	s := buildSession(t, "vic3", nil, nil, []catalog.ModInput{oneMod(t, "vic3", map[string]string{
		"events/x.txt": `namespace = ns
ns.1 = { type = character_event title = used_key }
`,
		"localization/english/a_l_english.yml": "l_english:\n used_key:0 \"Hi\"\n",
	})})
	for _, row := range Coverage(s) {
		for _, m := range row.Issues {
			if strings.Contains(strings.ToLower(m.File), "metadata.json") {
				t.Fatalf("metadata leaked: %+v", m)
			}
		}
	}
}
