# Observed game syntax

What CK3, Vic3 and EU5 actually do, as measured against real installs — not what
the engine assumes. **When a game update breaks PMT, start here:** re-run the
measurement in the row that broke and compare it against the recorded value.

Every claim below has a reproduction. Run them with the install env vars set:

```
PMT_TEST_INSTALL_CK3   PMT_TEST_DOCS_CK3     (…/Documents/Paradox Interactive/Crusader Kings III/logs)
PMT_TEST_INSTALL_VIC3  PMT_TEST_DOCS_VIC3    (…/Victoria 3/docs)
PMT_TEST_INSTALL_EU5   PMT_TEST_DOCS_EU5     (…/Europa Universalis V/docs)
```

Measured 2026-09 against CK3 1.19.x, Vic3 and EU5 current on Steam.

---

## 1. The type system is declared, not inferred

Each game ships its own in `script_docs`. Same semantics everywhere; only the
block syntax differs.

| Dump | Declares | CK3 | EU5 | Vic3 |
|---|---|---|---|---|
| `event_scopes.log` | the scope-type universe | 70 | ~120 | ~100 |
| `event_targets.log` | scope links, `Input Scopes` → `Output Scopes`; `Global Link`+`Requires Data` = the citable prefix table | 286 | 334 | 353 |
| `effects.log` | effects, `Supported Scopes` / `Supported Targets` | 1905 | 1534 | 3152 |
| `triggers.log` | triggers, same shape | 1785 | 1798 | 1816 |
| `on_actions.log` | `Expected Scope` per on_action | 902 | yes | yes |
| `modifiers.log` | modifier tokens, use area / mask | 742 | yes | yes |

**Block syntax per game** — three splitters in `catalog/schema.go`:

| Form | Used by | Looks like |
|---|---|---|
| Classic | CK3 | blocks separated by `----` |
| Markdown | EU5, Vic3 | `##` / `###` headings |
| Bare | `event_scopes.log` everywhere; Vic3 `modifiers.log` | indented `name:` / `Tag: name` / `key: name` |

**A link name is declared twice** — once global (`culture:english`), once
contextual (`character.culture`). Entries must **merge**; overwriting loses the
whole prefix table. Output types were checked across all three games and never
disagree.

**Vic3 `s:` scopes to a state _region_,** not a state. The game says so.

**`event` is not a scope type in any game.** So `trigger_event` has no declared
target type; CK3 documents it in prose only. This is why fire keys stay derived.

---

## 2. Folder rules

| Fact | Detail |
|---|---|
| Kind comes from the parent folder | `common/cultures/` → kind `cultures`. `game.MatchExtract`. |
| EU5 nests under a stage root | Script lives in `game/in_game/...`, not `game/...`. Walking `game/events` finds nothing on EU5. |
| script_docs land in different folders | CK3 `logs/`, Vic3 and EU5 `docs/`, all under user data. |
| The console command and launch option are the **same** in all three | `-debug_mode` and `script_docs`. Verified by grepping each game binary (`ck3.exe`, `victoria3.exe`, `eu5.exe`); all three also carry the longer alias `script_documentation`. Only the output folder differs, which `ScriptDocsSubdir` already encodes — so the walkthrough needs no new per-game data. |

### Where the install's version and timestamp are (and are not)

| Source | CK3 | Vic3 | EU5 |
|---|---|---|---|
| `launcher/launcher-settings.json` | present but **11 days stale** vs the real update | accurate | **does not exist** — no launcher directory at all |
| Newest mtime in `binaries/` | 2026-06-05 02:25:14 | 2026-08-19 15:27:13 | 2026-07-16 14:08:11 |
| Steam `appmanifest_<id>.acf` `LastUpdated` | 02:25:21 | 15:27:15 | 14:08:16 |

**EU5 publishes its version nowhere in the install.** Anything that needs "which
build is this" must not depend on a version string.

`game.InstallUpdatedAt` therefore uses the **newest mtime among `binaries/`**,
falling back to the install root. It works for all three, needs nothing outside
the install folder (so non-Steam copies work), and answers the question that
actually matters — *did the files change since these dumps were written*.

Two properties worth keeping in mind:

- The exe mtime lands **2–7 seconds before** the appmanifest's `LastUpdated` on
  every game, which shows Steam stamps files as it writes them rather than
  carrying a build's own timestamps. So switching to an older build through
  Steam's game-version settings — a **downgrade** — still moves the mtime
  forward, and is correctly treated as invalidating dumps from the other build.
  A version comparison would miss that.
- Copying or moving an install preserves mtimes. That can only make staleness
  detection quieter, never falsely loud.
| Loose `.txt` at the script root is not script | `checksum_manifest.txt`, `credits.txt`. Parsing the manifest as Jomini produced fire edges for `common`, `events`, `history`. |
| CK3 faiths have **no folder** | They exist only nested under `religion_types`, so `faith:` binds to nothing without a nested harvest. Verified: no `faiths/` directory exists. |
| EU5 `production_methods` **does** have a folder | `in_game/common/production_methods`. A nested shape for it is redundant. |

---

## 3. Shapes that look identical but are not

This section is the one that keeps costing us. Each row is a case where two
different things are **syntactically indistinguishable**, so the discriminator
has to come from outside the syntax.

### 3a. Setup wrapper vs. scripted macro

```
# Vic3 common/history/countries.txt — a wrapper; COUNTRIES is not an object
COUNTRIES = { c:USA ?= { ... } }

# CK3 common/scripted_triggers/… — a definition; the name IS the object
hegemon_favors_advancement_trigger = { title:h_china.holder ?= { ... } }
```

Character for character the same shape, `?=` included. **No body-shape rule can
separate them.** The discriminators:

- `game.DeclaresNamedBodies(kind)` — a `scripted_*` / `script_value*` folder has
  already declared that its top-level keys are definitions.
- `keywordDeclared` — `scripted_effect foo = { }` inside an events file is a
  declaration; the keyword states it.

Cost of getting this wrong: 33 CK3, 30 EU5 and 20 Vic3 vanilla macros deleted
from the model, breaking go-to-definition on every one.

### 3b. Nested database vs. repeated inner key

`deriveNestedShapes` looks for `prefix:id` citations whose id turns up as a
block child of some other def. Without a filter it treats **any** repeated inner
key as a database:

| Game | Shapes before filter | Phantom defs minted |
|---|---|---|
| CK3 | 133 | **342,138** under kinds `SKILL`, `TIER`, `NUMBER`, `LEVEL`, `BUILDING_LEVEL`, `$SKILL$` — script *parameter* names |
| EU5 | 183 | 35,458 under `CALL_FOR_PEACE_WARSCORE_LIMIT`, `var:chinese_expedition_opinion` |
| Vic3 | 134 | 5,231 under scripted constants like `@portrait_box_height` |

On CK3 that was **64% of the entire model**. The discriminator is
`Schema.Declares(childKind)` — the child must be a scope type or scope link the
game names. That leaves CK3 6 shapes, Vic3 4, EU5 15, including
`religion_types -> faith`.

Root cause worth remembering: `game.typedKindOK` accepts **any** `word:word` as
a typed cite, with only a small denylist. The lexer is deliberately permissive;
everything downstream must therefore re-check against the schema.

### 3c. `random_list` weight vs. fire key — *not* a bug

```
# CK3 common/on_action/army_on_actions.txt
random_events = {
    chance_of_no_event = 97
    100 = bp2_yearly.4030
    1   = epidemic_events.1120
}
```

Roughly half of each game's fire table is bare integers (CK3 37 of 71, Vic3
22/29, EU5 20/27). **These are correct.** The numeric key is a weight and the
RHS is a real event, so the on_action → event edges in the Event Graph depend on
them. Do not "clean" them.

---

## 3d. "Unreferenced in vanilla" is never evidence

Vanilla defines a great deal that vanilla itself never references: the engine
consumes it directly, or it exists for mods to reference. So the absence of a
citation says nothing about whether an object is real.

This is the reason IDE diagnostics are suppressed in install files, and it is
easy to violate somewhere else by accident. It has already cost twice:

- **Option databases.** `NestedShape.SkipKeys` was first computed as *"every
  inner key the field never referenced"*. That swept in every genuine row
  vanilla happens not to cite — CK3's `feudal_elective_succession_law` among
  them, which then showed up as a dangling reference on a real mod. The
  discriminator is not citation but **repetition**: an ordinary field appears in
  most instances of the parent (`categories` in 81 of 81 game rules, `flag` in
  24 of 27 law groups), a row appears in one.
- **Field typing** and **fire keys** both counted only the values that resolved,
  so a single coincidental match typed the whole field or key. See §3e.

**Rule: derive membership from how a thing is written, not from whether anything
in vanilla points at it.**

## 3e. Unanimity is not coverage

Three separate passes resolved conflicts by unanimity while silently dropping
values that resolved to nothing. One coincidental hit was therefore enough to
claim a fact, and every later use of it emitted a reference that could never
resolve. Measured on vanilla:

| Pass | Worst case | Floor added |
|---|---|---|
| `deriveFieldValueKinds` | `has_cultural_parameter` typed `ep_3` on **1 of 432** values; `remove_variable` typed `achievements` on 2 of 392 | `fieldKindCoverage` 50% |
| `resolveFireKinds` | Vic3 `alert`, `character` became on_action fire sites — **4,033** false rows on one mod | `fireKeyCoverage` 50%, but only once a key carries `minFireNames` (8) names |
| `deriveNestedShapes` | 342,138 phantom defs (§3b) | `Schema.Declares` |

The fire-key floor needs the extra minimum because there a *low* hit rate is
often the defect worth reporting: a mod whose `trigger_event` targets mostly do
not exist has broken references, and vetoing the key would hide exactly what
Workspace Health is for.

Effect on real mods, dangling references reported:

| Mod | Before | After |
|---|---|---|
| More Historicity (CK3) | 1,086 | **3** |
| Fix Trade Companies (EU5) | 86 | **4** |
| Morgenroete (Vic3) | 698 | 139 |

## 4. Reference forms

An object is not always cited as `prefix:id`. Anything keyed only off typed
cites will miss these.

| Form | Example | Where it appears |
|---|---|---|
| Typed cite | `culture:english`, `title:k_x` | everywhere; drives `bindKinds` and nested shapes |
| Dotted chain | `character.culture`, `title:h_china.holder` | scope walks |
| Field RHS | `has_game_rule = suf_quieter` | **not a typed cite** — handled by `deriveFieldValueKinds` |
| Keyword pair | `scripted_effect foo = { }` | inline macros, GUI `type`/`template` |
| Saved scope | `save_scope_as = x` then `scope:x` | per-file, ephemeral |
| Script param | `$SKILL$` inside a `scripted_*` body | ephemeral, owner-scoped |

---

## 5. Field-referenced nested options — an unindexed class

A recurring shape: a database whose rows live **inside** another def's block, and
which is referenced by a **field RHS** rather than a `prefix:id` cite. The
nested-database pass is cite-driven, so it is structurally blind to all of them.

| Case | Defined in | Referenced as |
|---|---|---|
| Game-rule options | `common/game_rules/`, inside a rule block | `has_game_rule = suf_quieter` |
| Cultural parameters | inside culture tradition blocks | `has_cultural_parameter = has_access_to_shieldmaidens` |

Measured on real mods, these are the largest remaining source of false
"dangling reference" rows in Workspace Health — `events_karling_decline_rule`
×17 and `has_access_to_shieldmaidens` ×10 on *More Historicity - CK3*, both of
which are defined, just not indexed.

**Solved by `catalog/options.go`.** `deriveNestedOptions` joins each field
against the nested blocks of the install: when nearly every value a field takes
is a child of one particular block, that block holds the database. It is a
deterministic join over many citations, like `bindKinds` — not a shape guess.
Three filters, tuned against all three games:

| Filter | Value | Why |
|---|---|---|
| `minOptionRefs` | 8 distinct names | below this, coincidence dominates |
| `optionCoverage` | 95% of the field's values | real cases 96–100%, noise 80–92% |
| `optionOwnerShare` | 50% of the block's inner keys | EU5 `start` matched 6,009 of 38,182 keys — not an option database |

What it finds, per game:

| Game | Option databases |
|---|---|
| CK3 (12) | `game_rule`, `realm_law`, `cultural_parameter`, `doctrine_parameter`, `domicile_parameter`, `religion` (faiths), `current_phase`, `participant_group_type`, `task_contract`, `track` |
| EU5 (4) | `game_rule`, `language` (dialects), `country_color`, `target` |
| Vic3 (3) | `game_rule`, `secret_goal`, `target` |

**The shape stores the complement.** `NestedShape.SkipKeys` lists the block's
*ordinary fields* (`categories`, `default`), not its rows — so anything else
under the parent is a row, and a mod's own new option is picked up without
re-deriving. Storing the row names instead froze mods out, which is what left
`events_karling_decline_rule` dangling on *More Historicity - CK3*.

`applyOptionShape` also ignores the value's shape: a game-rule option is a
block, a cultural parameter is `name = yes`, and both are equally real.

### Game rules (CK3), worked through

```
# common/game_rules/zz_quieter_events_game_rules.txt
secret_unbeliever_found_rule = {
    categories = { quieter_events }
    default    = suf_quieter          # the rule names one of its own options
    suf_vanilla = { flag = GG_can_change_rule }
    suf_quieter = { flag = GG_can_change_rule }
}
```

- The **rule** is a def (folder kind `game_rules`). PMT indexes it.
- The **options** (`suf_vanilla`, `suf_quieter`) are real named objects. PMT
  does **not** index them — 7 rules, 0 options on a real mod.
- They are referenced as `has_game_rule = suf_quieter` — a field RHS, never
  `game_rule_setting:suf_quieter`. Nothing in vanilla CK3 or in any mod checked
  writes that prefix, which is why the cite-driven nested pass cannot see them.
- Localization convention: `rule_<rule_id>` for the rule, `setting_<option_id>`
  and `setting_<option_id>_desc` for each option — a **prefixed** loc pattern,
  which `LocConventions` (`id` / `id_desc` / `kind_id`) does not model.

Three things must line up to support these: harvest the options as defs, let
`has_game_rule` resolve to them via field-value kinds, and teach loc conventions
the `setting_<id>` prefix form. Tracked in the plan.

---

## 6. Where these are asserted

Real-install tests, skipped when the env var is absent:

| Fact | Test |
|---|---|
| Schema counts, prefix binding coverage | `catalog/bind_test.go`, `schema_test.go` |
| Keyword pair is not a wrapper | `TestKeywordDeclaredBlockIsNotAWrapper` |
| Named-body folder is not a wrapper | `TestNamedBodyFolderIsNeverAWrapper` |
| Nested child must be declared | `TestDeriveNestedShapesNeedsDeclaredChild` |
| Cache size ceiling | `catalog/scansize_test.go` |
| Scope walk, zero false positives on vanilla | `session/scopereal_test.go` |

**Rule for adding to this file:** record the measurement and the reproduction,
not the conclusion alone. Every wrong turn in this engine came from reasoning
about what the games *should* do instead of sampling what they *do*.
