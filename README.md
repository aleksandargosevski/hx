# hx

> **Note:** This project is entirely vibe-coded for personal use. It works for my workflow but may have rough edges. Use at your own risk, PRs welcome.

A fast shell history manager that replaces `Ctrl+R` with a fuzzy-search TUI backed by SQLite. Search, edit, delete, and templatize your command history.

- Single binary, no dependencies
- Fuzzy search across your full history (100k+ entries in milliseconds)
- **Recency-first ranking** -- results ordered by most recent, with optional frecency toggle (`Ctrl+R`)
- **Directory filter** -- instantly scope results to the current working directory
- **Secrets filter** -- automatically skips recording commands containing API keys, tokens, and passwords
- Edit or delete history entries in-place
- Command templates with tab-stop placeholders (`${1:param}`)
- Real-time recording via zsh hooks (command, directory, exit code, duration)
- Pre-populated search -- type a partial command then hit `Ctrl+R` to search with it
- YAML config file for version-controllable templates

## Installation

### Homebrew (macOS / Linux)

```bash
brew install aleksandargosevski/tap/hx
```

### Script

```bash
curl -fsSL https://raw.githubusercontent.com/aleksandargosevski/hx/main/install.sh | bash
```

Installs to `/usr/local/bin` by default. Override with `HX_INSTALL_DIR`:

```bash
curl -fsSL https://raw.githubusercontent.com/aleksandargosevski/hx/main/install.sh | HX_INSTALL_DIR=~/.local/bin bash
```

### From source

```bash
git clone https://github.com/aleksandargosevski/hx.git
cd hx
make install
```

## Setup

Add this line to your `~/.zshrc`:

```zsh
eval "$(hx init zsh)"
```

This does four things:

1. **Configures zsh history** -- sets `HISTSIZE`/`SAVEHIST` to 100k, enables `EXTENDED_HISTORY`, deduplication, and `SHARE_HISTORY`
2. **Hooks into your shell** -- records every command to hx's SQLite database via `preexec`/`precmd` hooks, capturing the command, working directory, exit code, and duration
3. **Filters sensitive commands** -- automatically detects and skips recording commands containing API keys, tokens, passwords, and other secrets
4. **Binds Ctrl+R** -- replaces the default reverse-i-search with the hx fuzzy finder

### Import existing history

To bootstrap hx with your existing shell history:

```bash
hx import
```

This reads `~/.zsh_history` and imports all entries into the database, skipping duplicates. You can also specify a custom file:

```bash
hx import --file /path/to/history
```

## Usage

### Fuzzy search (Ctrl+R)

Press `Ctrl+R` in your terminal (or run `hx` / `hx search` directly). A fuzzy finder appears with your full command history. Type to filter, arrow keys to navigate, Enter to select.

If you've already started typing a command, pressing `Ctrl+R` pre-populates the search with what you've typed. For example, type `docker` then hit `Ctrl+R` and the search opens pre-filtered to docker commands.

Results are ranked by **most recent first** by default. Press `Ctrl+R` to toggle **frecency** ranking -- a combination of how often and how recently you've used each command.

The selected command is placed on your shell prompt, ready to execute or edit.

```
  docker exec -it api-prod bash                          2h ago  ~/proj
  docker build -t myapp .                                1d ago  ~/proj
> docker compose up -d                                   3h ago  ~/proj
  docker compose logs -f api                             3h ago  ~/proj

  4/2847           ^d:del  ^e:edit  ^r:freq  ^f:ok  ^g:cwd  ^t:tmpl  tab:templates
> docker_
```

#### Keybindings

| Key                 | Action                                             |
| ------------------- | -------------------------------------------------- |
| Type                | Fuzzy filter results                               |
| `Up` / `Down`       | Navigate results                                   |
| `Ctrl+P` / `Ctrl+N` | Navigate results (alternative)                     |
| `Enter`             | Select command and place on prompt                 |
| `Ctrl+D`            | Delete selected entry                              |
| `Ctrl+Z`            | Undo last delete                                   |
| `Ctrl+E`            | Edit selected entry inline                         |
| `Ctrl+R`            | Toggle sort: recent (default) / frecency           |
| `Ctrl+F`            | Toggle filter: all / successful only (exit code 0) |
| `Ctrl+G`            | Toggle filter: all / current directory only        |
| `Ctrl+T`            | Create template from selected entry                |
| `Tab`               | Switch to template search mode                     |
| `Esc` / `Ctrl+C`    | Cancel                                             |

#### Editing (Ctrl+E)

Opens an inline editor for the selected command. Standard readline-style bindings work:

| Key      | Action                         |
| -------- | ------------------------------ |
| `Ctrl+A` | Move cursor to start           |
| `Ctrl+E` | Move cursor to end             |
| `Ctrl+K` | Kill text from cursor to end   |
| `Ctrl+U` | Kill text from cursor to start |
| `Enter`  | Save changes                   |
| `Esc`    | Cancel                         |

### Templates

Templates are reusable command patterns with named placeholders. Placeholders use the syntax `${N:label}` where `N` is the tab-stop order and `label` is a descriptive name. The label text acts as the default value -- it appears pre-selected in the buffer so you can accept it with `Ctrl+]` or type to replace it.

Example: `docker exec -it ${1:container} ${2:command}`

When you select a template, hx expands it directly onto your shell prompt with the first placeholder selected. You fill it in using your shell's native features — including **Tab completion for files, directories, and anything else your shell knows about**. Press `Ctrl+]` to jump to the next placeholder.

This means templates work with all your existing shell completions out of the box — file paths, git branches, docker containers, kubectl resources, SSH hosts, etc.

#### Create templates

**From the TUI:** Press `Ctrl+T` on any history entry. This opens an editor where you can replace parts of the command with `${N:label}` placeholders, give it a name, and save.

**From the CLI:**

```bash
hx template add \
  --name "docker-exec" \
  --command 'docker exec -it ${1:container} ${2:command}' \
  --description "Execute command in running container"
```

**From a config file** (`~/.config/hx/templates.yaml`):

```yaml
templates:
  - name: docker-exec
    command: "docker exec -it ${1:container} ${2:command}"
    description: "Execute command in running container"
  - name: ssh-connect
    command: "ssh ${1:user}@${2:host} -p ${3:port}"
    description: "SSH into a server"
  - name: git-rebase
    command: "git rebase -i HEAD~${1:count}"
    description: "Interactive rebase last N commits"
```

Templates from the config file are synced into the database on each search launch, so you can version-control your templates alongside your dotfiles.

#### Use templates

Press `Tab` in the search TUI to switch to template mode. Fuzzy-search your templates, select one with `Enter`. The template is expanded onto your prompt with the first placeholder selected:

```
$ docker exec -it [container] command
                  ^^^^^^^^^^^ selected — type to replace, Tab to complete files
```

Type a replacement value (shell Tab completion works here), then press `Ctrl+]` to jump to the next placeholder:

```
$ docker exec -it api-prod [command]
                           ^^^^^^^^^ now selected
```

Fill in the last placeholder and press `Enter` to execute.

| Key      | Action                               |
| -------- | ------------------------------------ |
| Type     | Replace the selected placeholder     |
| `Tab`    | Shell completion (files, dirs, etc.) |
| `Ctrl+]` | Jump to next placeholder             |
| `Enter`  | Execute the command                  |

#### Manage templates

```bash
hx template list                    # List all templates
hx template add --name ... --command ... --description ...
hx template remove --name ...       # Remove a template
```

### Stats

Run `hx stats` to see analytics about your shell history:

```bash
hx stats        # Show all stats (top 10 per section)
hx stats -n 20  # Show top 20 per section
```

Sections include:

- **Overview** — total commands, unique commands, directories, avg commands/day
- **Most Used Commands** — your top commands ranked by frequency with bar chart
- **Most Active Directories** — where you spend most of your time
- **Activity by Hour** — heatmap of when you're most active
- **Most Failing Commands** — commands with the highest error rates (min 3 runs)
- **Slowest Commands** — commands with the highest average duration

## CLI Reference

| Command                      | Description                                            |
| ---------------------------- | ------------------------------------------------------ |
| `hx`                         | Launch fuzzy search (alias for `hx search`)            |
| `hx search`                  | Launch the fuzzy search TUI                            |
| `hx search --query "docker"` | Launch with a pre-filled search query                  |
| `hx search --cwd /path`      | Launch with directory filter context                   |
| `hx init zsh`                | Print the zsh integration snippet                      |
| `hx import`                  | Import history from `~/.zsh_history`                   |
| `hx import -f FILE`          | Import from a specific history file                    |
| `hx record`                  | Record a command (called automatically by shell hooks) |
| `hx template list`           | List all templates                                     |
| `hx template add`            | Add a new template                                     |
| `hx template remove`         | Remove a template by name                              |
| `hx stats`                   | Show history analytics and usage patterns              |
| `hx stats -n 20`             | Show top 20 entries per section                        |
| `hx version`                 | Print version                                          |

## How it works

- **Storage:** All history and templates are stored in a SQLite database at `~/.config/hx/hx.db` with WAL mode enabled for fast concurrent reads.
- **Recording:** The zsh `preexec` hook fires `hx record` as a background job (`&!`) on every command, so it never slows down your shell. Each record includes the command text, working directory, timestamp, exit code, and duration.
- **Secrets filter:** Before recording, commands are checked against patterns for common secrets (AWS keys, GitHub PATs, Slack/Stripe tokens, `--password`, `--token`, `API_KEY=`, etc.). Matching commands are silently skipped and never written to the database.
- **Ranking:** By default, results are sorted by most recent first. Press `Ctrl+R` to toggle frecency ranking, which scores results by `frequency * (1 / (1 + days_since_last_use))` -- a command used 10 times yesterday scores higher than one used twice today. When you type a search query, fuzzy matching determines relevance, with the active sort order used as a tiebreaker.
- **Search:** On launch, hx loads history into memory and uses [sahilm/fuzzy](https://github.com/sahilm/fuzzy) (the same algorithm as Sublime Text / VS Code) for sub-millisecond fuzzy matching.
- **Ctrl+R integration:** The TUI renders on `/dev/tty` while the selected command is printed to stdout, which the zsh widget captures and places into your shell's edit buffer. If you had text on the command line before pressing `Ctrl+R`, it becomes the initial search query.
- **Directory filter:** Press `Ctrl+G` to scope results to commands that were run in your current working directory. Useful for project-specific recall.
- **Duration display:** Commands that took more than a second show their duration in the search results metadata (e.g. `3s`, `2m15s`).
- **Undo delete:** Pressing `Ctrl+Z` in the search TUI restores the last deleted entry (supports multiple undos within a session).
- **Deduplication:** History is deduplicated on display (most recent occurrence of each unique command), but all occurrences are kept in the database for accurate frecency scoring.
- **Deletion:** Entries are soft-deleted (flagged, not removed), so the underlying data is recoverable if needed.

## Project structure

```
hx/
├── main.go                        # Entry point
├── Makefile                       # build, install, test, clean
├── cmd/
│   ├── root.go                    # CLI root command (cobra)
│   ├── search.go                  # hx search
│   ├── init_zsh.go                # hx init zsh (shell integration + secrets filter)
│   ├── import.go                  # hx import
│   ├── record.go                  # hx record
│   ├── expand.go                  # hx expand (template expansion, used by zsh widget)
│   └── template.go                # hx template [list|add|remove]
└── internal/
    ├── config/config.go           # YAML config loader
    ├── db/
    │   ├── db.go                  # SQLite connection + migrations
    │   ├── history.go             # History CRUD
    │   └── template.go            # Template CRUD
    ├── history/parser.go          # Zsh EXTENDED_HISTORY parser
    ├── template/template.go       # ${N:label} placeholder engine
    └── tui/
        ├── search.go              # Main TUI model (bubbletea)
        └── styles.go              # Terminal styling (lipgloss)
```

## Built with

- [bubbletea](https://github.com/charmbracelet/bubbletea) -- TUI framework
- [lipgloss](https://github.com/charmbracelet/lipgloss) -- Terminal styling
- [sahilm/fuzzy](https://github.com/sahilm/fuzzy) -- Fuzzy matching
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) -- Pure Go SQLite (no CGo)
- [cobra](https://github.com/spf13/cobra) -- CLI framework

## License

MIT
