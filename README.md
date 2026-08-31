**My Discord User:** https://discord.com/users/168438474905616386

# Paradox Modding Tools

Cross-platform desktop utilities for **Paradox Interactive** game modders. The app is built with **[Wails v3](https://v3.wails.io/)** (**Go** backend, **Vue** frontend) and licensed under **GPL-3.0-or-later**. **Crusader Kings III**, **Europa Universalis V** (partial), and **Victoria 3** are supported today; the direction is to grow coverage and workflows across Paradox titles.

## Download / Use The Tool! (testing)

Builds are published under [GitHub Releases](https://github.com/idodavis/paradox-modding-tools/releases) for manual download while the modding community tries them out. An in-app updater is planned so testers do not need to fetch every build from Releases; until that ships, **Releases remain the source of binaries**. Release packaging is manual for now; **GitHub Actions** automation for builds and uploads is planned.

## What's in the app

- **Workspace Library** — Manage modding workspaces tied to specific game installs and mod folders. Create, switch, and organize your modding projects.
- **Workspace IDE** — File browser across game, mod, and staging roots with integrated editor/diff viewer. Compare files, switch themes, and navigate to patching tools.
- **Patch Center** — Browse wiki patch notes for each game version and import game error.log files for analysis.
- **Mod Patcher** — Update mods between game versions. Preview changes, accept/skip files, resolve conflicts with the merge editor.
- **Event Graph** — Explore definitions and event flows harvested into the live workspace session (maps rebuilt on open; no persisted mod index).
- **Ad-hoc Merge** — Merge two files or directories without a workspace using the script merger.
- **Settings** — App configuration in a JSON config store (`config.json`; a format-version mismatch wipes).

### Paradox script parser (Go)

A hand-written **Go CST** parses typical Paradox `.txt` / `.gui` script for compare/merge, the workspace IDE, and related features. Implementation lives under `services/internal/parser/jomini` (`.txt` and `.gui` share the CST) and `services/internal/parser/loc`. The live workspace session (`services/internal/session`) holds VanillaCache plus RAM harvest maps rebuilt when a workspace opens.

## Prerequisites (from source)

- **[Go](https://go.dev/dl/)** (see `go.mod` for the required version)
- **[Node.js](https://nodejs.org/)** and **npm** (for the Vue frontend)
- **[Task](https://taskfile.dev/installation/)** (Taskfile v3)
- **[Wails v3 CLI](https://v3.wails.io/)** (`wails3`), aligned with the `github.com/wailsapp/wails/v3` version in `go.mod`

Install the Wails CLI following the official v3 docs so `wails3` is on your `PATH`.

## Develop (hot reload)

From the repository root:

```bash
task dev
```

This runs `wails3 dev` with `./build/config.yml` (frontend build, binding generation, and the app in development mode). Override the Vite port if needed, for example:

```bash
WAILS_VITE_PORT=9246 task dev
```

## Build a production binary

OS-specific tasks are selected automatically (`windows`, `darwin`, `linux`):

```bash
task build
```

The executable is written under `bin/` (e.g. `bin/paradox-modding-tools.exe` on Windows, `bin/paradox-modding-tools` on macOS/Linux).

## Run the built binary

After a successful `task build`:

```bash
task run
```

## Package installers (optional)

```bash
task package
```

Uses the platform's configured format (e.g. NSIS on Windows). You need the extra tooling each format expects (see `build/windows/Taskfile.yml` and sibling platform Taskfiles).

## Other useful tasks

| Task | Purpose |
|------|--------|
| `task setup:docker` | Docker image for cross-compilation / CGO workflows |
| `task build:server` / `task run:server` | Server-style build without the desktop shell (see Taskfiles) |

## Project layout (high level)

| Path | Role |
|------|------|
| `main.go` | Wails app entry, services, embedded `frontend/dist` |
| `services/` | Go services exposed to the UI |
| `frontend/` | Vue + Vite UI |
| `build/` | Wails build config, icons, platform Taskfiles |

## License

This project is licensed under the **GNU General Public License v3.0 or later** (GPL-3.0-or-later). See the [LICENSE](LICENSE) file for details.

---

## Other Paradox projects

### CK3 Quieter Events Mod

Successor to **Less Event Spam**: turns several full-screen or intrusive vanilla events into smaller toasts or messages.

Find it [here](https://github.com/idodavis/ck3-quieter-events).
