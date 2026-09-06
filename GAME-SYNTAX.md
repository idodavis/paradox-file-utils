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

`minAffixDefs` (8 definitions must share a localization naming shape) and
`minLocFieldHits` (8 values of a property must be keys, at 80% coverage) are the
same rule applied to the two localization derivations.

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

## 3f. The same spelling means different things in different slots

The games reuse names across unrelated kinds constantly, and the engine is never
confused because it always knows which slot it is reading. A resolver that looks
a name up across every index has no such context, and breaking the tie by load
order silently lets load order decide *what kind of thing a name is*.

Measured 2026-09-06, all four found as wrong hover/peek cards:

| Written | Also spelled the same | Wrong card | What actually decides |
|---|---|---|---|
| `has_realm_law = x` | the loc key `has_realm_law` | the caption "Realm Law" | it is **left of an `=`** — `probe.inKey` |
| `culture = french` | the culture's own loc key | the caption | a value naming an object is the object — `resolveNonLoc` |
| `add_trait = conqueror` | `namespace = conqueror` in the mod | the namespace, because mod beats vanilla | a namespace is not an object — `jomini.IsObjectKind` |
| `random_list = { 95 = … }` | province `95` in `map_data/positions.txt` | the town Amiens | the **file** that defines numeric rows — `isLocalDefFile` |

The last one is worth stating plainly because it is a whole class: **a bare
number on the left of an `=` is a weight** (§3c says the same of fire tables),
and vanilla CK3 keys map positions the same way. Nothing distinguishes the two
by name; the file does.

A weight's share is computed from its siblings, never assumed to be out of 100 —
`{ 10 = {…} 30 = {…} }` is a quarter and three quarters. It is only claimed
where the branches are **bodies**: `random_events = { 100 = some.event }` (§3c)
and a Vic3 gene block's `20 = empty` both key scalars, and nothing separates a
weight from a lookup index there, so the card says nothing rather than guessing.

**The rule:** load order decides which *definition of the same kind* wins,
nothing else. Every discriminator above is a structural fact from the parse tree
— left of `=`, which file, whether the branch is a body — so none of them is a
list of names that a game update can invalidate.

`jomini.IsObjectKind` is the predicate for "a thing script can point at". Its
complement is three kinds, one per bug above: ephemerals, `loc_key`, and
`namespace`. Use it when resolving a name to a thing; not when asking whether a
name is declared at all, because a namespace genuinely is.

## 3g. A property holds a loc key — but a value is not always one

`parser/loc` splits loc-key properties into **strict** (an unresolved value is a
defect worth a diagnostic) and **broad** (a resolved value gets a hint, an
unresolved one is silent). Measured over vanilla, where nothing is missing by
definition, the split was wrong in two ways.

**`localization_key` is not strict.** Its value is a stem that customizable
localization completes at runtime — `localization_key = CustomLoc_BR_male_`
ends in an underscore because the engine appends to it. Unresolved values on
vanilla:

| Game | `localization_key` | All other strict props | `localization_key` share |
|---|---|---|---|
| CK3 | 2,607 | 730 | 78% |
| Vic3 | 11,597 | 271 | 98% |
| EU5 | **46,481** | 139 | **99.7%** |

Now broad: still harvested, so the key counts as used and hovers with its text;
it just no longer asserts a key the game builds itself is missing.

**Quoting does not tell you anything.** The obvious guess — quoted means display
text, bare means a key reference — is wrong. Of quoted strict-property values on
vanilla, **86% (CK3), 99% (Vic3), 92% (EU5)** resolve to a real key: Paradox
quotes keys freely (`war_name = "INDEPENDENCE_WAR_NAME"`). Do not use quoting as
a discriminator.

**Whitespace does.** A localization file is `key:0 "value"`, so a key is a
single token and can never contain a space. `desc = "Always make coronations!"`
is display text the game shows verbatim, and demanding a key by that name was
537 of A Game of Thrones' 618 `missing-required-loc` findings. `LooksLikeKey`
now rejects any value containing whitespace.

What remains after both fixes is genuinely ambiguous and vanilla shares it:
`desc = "base"`, `desc = "Pyke"`, `desc = "Cowardly"` are single tokens
indistinguishable from keys, and CK3 vanilla itself has 233 of them. No
structural rule separates these; do not invent a heuristic for them.

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
  and `setting_<option_id>_desc` for each option.

Counted on vanilla CK3: **78 rules, 77 with a `rule_<id>` key** (99%), and **380
options with 590 `setting_*` keys**.

The loc side is **done**, and it did not need the options to be definitions.
`deriveLocAffixes` covers `rule_<id>`, because a rule is a def. `setting_<id>`
is covered by `deriveLocMemberAffixes`, which runs the same derivation over
**block-member key names** instead of def keys — the option names were already
harvested into `Structures` and then discarded. `BuildIndex` now carries them
too, so a mod's *new* option is recognised even though it exists in no install.

The two maps are kept apart on purpose. Member names include ordinary field
names (`default`, `categories`), and letting a def-keyed convention such as
`<id>_desc` match one of those would explain away real orphans.

Still open, and unaffected by the above: harvest the options as **defs**, and
let `has_game_rule` resolve to them via field-value kinds. That is what
go-to-definition and completion on an option need. Tracked in the plan.

---

## 6. Total-conversion scale is the measurement

Everything above is about what the games contain. This section is about what
that volume does to code that reads it, because the two are the same subject:
every entry here was correct on vanilla and only wrong on a mod big enough.

Whenever a loop does expensive work and *then* decides whether it was needed,
that is a bug waiting for a big enough mod. Five instances so far, one shape:

| Where | What it did | Cost at total-conversion scale |
|---|---|---|
| `scriptParamItems` | took the first 80 script params, then kept this macro's | returned nothing, across 2,536 CK3 macros |
| `defsOfType` | `FindDefs` with no kind, then filtered by kind | returned nothing with an empty prefix |
| `views.refRows` | built a site (file read + line split) per reference, then called `Resolve` | hundreds of thousands of discarded reads |
| `views.Coverage` | built a site per loc issue per language, then applied `healthLocCap` | ~99% of the work discarded |
| `catalog.ConventionOwner` | built its visitor closure per *span* of a key rather than per key | 22 allocations and 448ns to answer one key; 4 and 155ns after |

**The rule:** filter before the limit, and decide before you materialise. Where a
budget exists, spend it on the *expensive* half — count cheaply and uncapped so
KPIs stay honest, materialise only what is kept. `FindMacroDefs` and
`Session.FindParamsOf` document the filter-before-limit half; `views.Coverage`
documents the count-then-materialise half.

The same rule decides where a derived fact is *stored*. A localization naming
convention could be expanded forwards — one reference per definition per
convention, hundreds of thousands of rows on CK3 — or applied backwards by
`ConventionOwner` only to the few keys that still look orphaned. It is applied
backwards.

### Scheduling must not decide anything

The corpus walk is parallel, and twice that let worker completion order decide a
fact about the game. Both were found by scanning one install twice and diffing
the caches, which is now `TestHealthIsDeterministic`:

| Where | What order decided | Symptom |
|---|---|---|
| `accum.merge` → `MergeLoc` | which value survives when two files of one language define the same key | `untranslated` moved by 1–2 between identical runs |
| `deriveNestedOptions` | which field names an option database, because the greedy loop consumes values as it assigns owners | two scans disagreed on whether CK3's `genes` database was `template` or `positive_mirror` — **352 defs under a different kind**, and `orphaned` differed by 30 |

Both are now ordered: localization merges under **load order** (the file's index
in the walk list, which is mod order then walk order — the same rule that
decides every other override), and the option pass iterates fields, owners and
values sorted.

The second one is the one to remember. It was not a display bug: the *kind* of
352 definitions depended on thread scheduling, so `FieldValueKinds`,
go-to-definition and every reference to them changed between runs of the same
workspace. Parallelism may decide when work happens, never what anything is.

**Reproduction:** `task test:sweep`. Every Workspace Health check against every
workshop mod on the machine, reported per game with hover and completion answer
rates, diagnostics per code, and live heap. A Game of Thrones (21,499 files) is
the acceptance case, and the run includes the determinism guard — a count that
moves between runs cannot be read as a regression either way.

## 7. Where these are asserted

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
