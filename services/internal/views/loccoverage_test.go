// loccoverage_test.go covers strict-only missing loc, broad-field skips,
// game-rule / message-filter orphans, and descriptor/metadata name refs.
package views

import (
	"strings"
	"testing"

	"paradox-modding-tools/services/internal/catalog"
	"paradox-modding-tools/services/internal/session"
)

func coverageIssue(rows []HealthRow, lang, kind, key string) bool {
	for _, r := range rows {
		if r.Language == lang && r.Type == kind && r.Name == key {
			return true
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
			"l_english:\n used_key:0 \"Hi\"\n orphan_key:0 \"Bye\"\n" +
			" war:0 \"$ORDER$ $no_such_key$\"\n",
		"localization/english/game_rules_l_english.yml": "" +
			"l_english:\n setting_rule:0 \"Rule\"\n",
		"localization/english/message_filter_l_english.yml": "" +
			"l_english:\n filter_key:0 \"Filter\"\n",
	})})
	_, cov := Coverage(s)
	if !coverageIssue(cov, "english", "missing", "missing_key") {
		t.Fatalf("missing_key not reported: %+v", cov)
	}
	if lu := Lookup(s, "used_key"); lu == nil || lu.Text != "Hi" {
		t.Fatalf("Lookup used_key = %+v", lu)
	}
	if lu := Lookup(s, "vanilla_key"); lu == nil || lu.Origin != "vanilla" {
		t.Fatalf("Lookup vanilla_key = %+v", lu)
	}
	if coverageIssue(cov, "english", "missing", "some_var") {
		t.Fatal("loc-broad name= should not be missing")
	}
	if !coverageIssue(cov, "english", "orphaned", "setting_rule") {
		t.Fatal("unused game_rules loc is orphaned without a convention vote")
	}
	if !coverageIssue(cov, "english", "orphaned", "filter_key") {
		t.Fatal("unused message_filter loc is orphaned without a convention vote")
	}
	if !coverageIssue(cov, "english", "orphaned", "orphan_key") {
		t.Fatal("unused loc should stay orphaned")
	}
	if coverageIssue(cov, "english", "missing", "ORDER") {
		t.Fatal("$ORDER$ is engine data, not a missing loc key")
	}
	if !coverageIssue(cov, "english", "missing", "no_such_key") {
		t.Fatal("$no_such_key$ reuse should stay missing")
	}
}

func TestCoverageConventionLocsInGenericFile(t *testing.T) {
	s := buildSession(t, "ck3", &catalog.VanillaCache{
		LocConventions: map[string]string{
			"decisions":  "id_desc",
			"game_rules": "kind_id",
		},
	}, nil, []catalog.ModInput{oneMod(t, "ck3", map[string]string{
		"common/game_rules/00.txt": `my_rule = {
	default = opt_a
	opt_a = { }
	exists = game_rule_setting:opt_a
}
`,
		"common/decisions/00.txt": `ai_mogyer_adopt_christianity = { }
`,
		"common/casus_belli_types/cb.txt": "hre_conquest = { war_name = \"HRE_CONQUEST_WAR_NAME\" " +
			"cb_name = \"HRE_CONQUEST_DUCHY_CB_NAME\" }\n",
		"common/character_interactions/i.txt": "demand_x = { notification_text = DEMAND_IMPERIAL_SUBJUGATION_NOTIFICATION }\n",
		"common/flavorization/f.txt":          "hungarian_empire = { type = empire }\n",
		"common/messages/00.txt": `my_msg = { }
`,
		"localization/english/mod_l_english.yml": "" +
			"l_english:\n game_rules_my_rule:0 \"R\"\n rule_my_rule:0 \"R\"\n setting_opt_a:0 \"A\"\n my_msg:0 \"M\"\n" +
			" ai_mogyer_adopt_christianity:0 \"A\"\n" +
			" ai_mogyer_adopt_christianity_desc:0 \"D\"\n" +
			" ai_mogyer_adopt_christianity_tooltip:0 \"T\"\n" +
			" ai_mogyer_adopt_christianity_confirm:0 \"C\"\n" +
			" HRE_CONQUEST_WAR_NAME:0 \"War\"\n" +
			" HRE_CONQUEST_DUCHY_CB_NAME:0 \"CB\"\n" +
			" DEMAND_IMPERIAL_SUBJUGATION_NOTIFICATION:0 \"N\"\n" +
			" cn_hungarian_empire_adj:0 \"Hungarian\"\n unused_key:0 \"U\"\n",
	})})
	_, cov := Coverage(s)
	if coverageIssue(cov, "english", "orphaned", "game_rules_my_rule") &&
		coverageIssue(cov, "english", "orphaned", "game_rule_my_rule") {
		t.Fatal("voted game_rules kind_id loc should not be orphaned")
	}
	if !coverageIssue(cov, "english", "orphaned", "rule_my_rule") {
		t.Fatal("encyclopedia rule_<id> is unused without a vote")
	}
	if !coverageIssue(cov, "english", "orphaned", "setting_opt_a") {
		t.Fatal("encyclopedia setting_ key is unused without a vote")
	}
	if coverageIssue(cov, "english", "orphaned", "ai_mogyer_adopt_christianity") {
		t.Fatal("decision id loc should not be orphaned")
	}
	if !coverageIssue(cov, "english", "orphaned", "ai_mogyer_adopt_christianity_tooltip") {
		t.Fatal("encyclopedia <id>_tooltip is unused without a vote")
	}
	if !coverageIssue(cov, "english", "orphaned", "ai_mogyer_adopt_christianity_confirm") {
		t.Fatal("encyclopedia <id>_confirm is unused without a vote")
	}
	if coverageIssue(cov, "english", "orphaned", "HRE_CONQUEST_WAR_NAME") {
		t.Fatal("war_name cite should not be orphaned")
	}
	if coverageIssue(cov, "english", "orphaned", "HRE_CONQUEST_DUCHY_CB_NAME") {
		t.Fatal("cb_name cite should not be orphaned")
	}
	if coverageIssue(cov, "english", "orphaned", "DEMAND_IMPERIAL_SUBJUGATION_NOTIFICATION") {
		t.Fatal("notification_text cite should not be orphaned")
	}
	if coverageIssue(cov, "english", "orphaned", "ai_mogyer_adopt_christianity_desc") {
		t.Fatal("decision id_desc loc should not be orphaned")
	}
	if !coverageIssue(cov, "english", "orphaned", "cn_hungarian_empire_adj") {
		t.Fatal("encyclopedia flavorization loc is unused without a vote")
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
	_, cov := Coverage(s)
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

func TestCoverageMissingNotSuppressedByDefaultLang(t *testing.T) {
	s := session.NewWithLoc("ws", "ck3", "english", nil, nil, []catalog.ModInput{
		oneMod(t, "ck3", map[string]string{
			"events/x.txt": `namespace = ns
ns.1 = { type = character_event title = used_key }
`,
			"localization/english/a_l_english.yml": "l_english:\n used_key:0 \"Hi\"\n",
			"localization/french/a_l_french.yml":   "l_french:\n other:0 \"Bonjour\"\n",
		}),
	})
	_, cov := Coverage(s)
	if !coverageIssue(cov, "french", "missing", "used_key") {
		t.Fatalf("french missing used_key (english has it): %+v", cov)
	}
	if coverageIssue(cov, "english", "missing", "used_key") {
		t.Fatal("english defines used_key")
	}
}

func TestCoverageSkipsDescriptorName(t *testing.T) {
	s := buildSession(t, "ck3", nil, nil, []catalog.ModInput{oneMod(t, "ck3", map[string]string{
		"events/x.txt": `namespace = ns
ns.1 = { type = character_event title = used_key }
`,
		"localization/english/a_l_english.yml": "l_english:\n used_key:0 \"Hi\"\n",
	})})
	_, rows := Coverage(s)
	for _, m := range rows {
		path := m.Rel
		if len(m.Sites) > 0 {
			path = m.Sites[0].Path
		}
		if strings.Contains(strings.ToLower(path), "descriptor.mod") ||
			m.Name == "t" && m.Type == "missing" {
			t.Fatalf("descriptor name leaked: %+v", m)
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
	_, rows := Coverage(s)
	for _, m := range rows {
		path := m.Rel
		if len(m.Sites) > 0 {
			path = m.Sites[0].Path
		}
		if strings.Contains(strings.ToLower(path), "metadata.json") {
			t.Fatalf("metadata leaked: %+v", m)
		}
	}
}
