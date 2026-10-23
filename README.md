# YouTrack CLI

[![](https://jb.gg/badges/official-plastic.svg)](https://github.com/JetBrains)
[![GitHub Release](https://img.shields.io/github/v/release/JetBrains/youtrack-cli?style=plastic)](https://github.com/JetBrains/youtrack-cli/releases/latest)

Command-line client for [YouTrack](https://www.jetbrains.com/youtrack/), covering the day-to-day — issues, commands, projects, users — plus the odd jobs: knowledge-base articles, agile boards, admin settings, and raw REST calls when nothing else fits. Built for people at a terminal and AI agents driving YouTrack in a pipeline.

![youtrack-cli demo](docs/assets/youtrack-cli.gif)

Full documentation — installation, authentication, configuration, and a command reference — lives under [docs/](docs/topics/youtrack-cli.md).

> **Security notice:** YouTrack does not support granular token scopes; any token grants full account access. See [docs/security.md](docs/security.md) before using `youtrack` in pipelines or with AI agents.

## Install

**macOS & Linux (install script):**
```bash
curl -fsSL https://jb.gg/youtrack-cli/install | bash
```

New installs use `~/.local/bin` without sudo; rerunning the script updates an existing unmanaged installation in place. Missing Bash/Zsh PATH entries are added automatically. Restart your shell afterward, or use the printed PATH command. Package-managed installations should be updated through their package manager.

The installers verify the downloaded archive against `checksums.txt` and, when `gpg` is installed, verify its signature against the pinned JetBrains key.

**Homebrew (macOS & Linux):**
```bash
brew install jetbrains/utils/youtrack
```

**Scoop (Windows):**
```powershell
scoop bucket add jetbrains https://github.com/JetBrains/scoop-utils
scoop install youtrack
```

`youtrack update` upgrades the CLI through the method that installed it. In a terminal, the CLI also offers the update before a command runs when a newer release is out. Set `YOUTRACK_NO_UPDATE=1` to turn off update checks.

**Windows (PowerShell):**
```powershell
irm https://jb.gg/youtrack-cli/install.ps1 | iex
```

**Windows (CMD):**
```cmd
curl -fsSL https://jb.gg/youtrack-cli/install.cmd -o install.cmd && install.cmd && del install.cmd
```

Windows installers use `%USERPROFILE%\.local\bin` and add it to the user PATH. Set `YOUTRACK_INSTALL_RELEASE` or `YOUTRACK_INSTALL_DIR` to choose a version or destination in PowerShell; CMD accepts `install.cmd [version] [directory]`.

Alternatively, download the matching `*windows_x86_64.zip` or `*windows_arm64.zip` release archive, extract it, and add the folder to your `PATH`.

<details>
<summary>More installation methods (deb, rpm, go install)</summary>

**Debian/Ubuntu:**
```bash
curl -fsSLO https://github.com/JetBrains/youtrack-cli/releases/latest/download/youtrack_linux_amd64.deb
sudo dpkg -i youtrack_linux_amd64.deb
```

**RHEL/Fedora:**
```bash
curl -fsSLO https://github.com/JetBrains/youtrack-cli/releases/latest/download/youtrack_linux_amd64.rpm
sudo rpm -i youtrack_linux_amd64.rpm
```

Replace `amd64` with `arm64` on ARM machines.

**Build from source** (requires [Go 1.26+](https://go.dev/dl/)):
```bash
go install github.com/JetBrains/youtrack-cli@latest
```

See the [getting started guide](docs/topics/youtrack-cli-get-started.md) for the full walkthrough.

</details>

## Usage

Authenticate once and the CLI remembers the instance:

```bash
youtrack auth login
```

It asks for the YouTrack URL (pre-filled with `https://youtrack.jetbrains.com` or your active instance), opens your browser to sign in through Hub, and saves a permanent token. Pass `--no-browser` to paste a token instead.

Or save an OAuth session that refreshes itself instead of a permanent token. `--no-browser` works over SSH:

```bash
youtrack auth login https://example.youtrack.cloud --session
youtrack auth status      # auth method, expiry, refresh token present — never the token itself
```

Auth commands mask the token in their output, JSON included. Pass `--show-token` when a script needs the full value, for example `youtrack auth login --save=false --json --show-token | jq -r .token`.

Then:

```bash
# your open issues
youtrack issue list --query "assignee: me State: Open"

# pick columns, mixing issue attributes and custom fields, like the issue list UI
youtrack issue list --query "project: DEMO #Unresolved" --fields "idReadable,summary,Priority,State,Assignee"

# create an issue
youtrack issue create --project DEMO --summary "Login button unresponsive on Safari"

# move it along
youtrack issue command DEMO-42 "State In Progress"

# view an issue with its five most recent comments
youtrack issue view DEMO-42 --comments 5

# what a command would do, without applying it
youtrack issue command suggest DEMO-42 "State "

# link a commit to an issue, and list what is linked
youtrack issue vcs attach DEMO-42 1a2b3c4d5e6f7081
youtrack issue vcs list DEMO-42

# keep the CLI itself up to date
youtrack update
```

Every command takes `--json` or `--plain` for [scripting](docs/topics/youtrack-cli-scripting.md), and `--fields` to pick table/plain columns and pull extra fields from the API. When no command covers what you need, `youtrack raw` calls the REST API directly with your stored credentials. You can register several instances and switch between them, sign in through the browser, or point at a server with environment variables — see [authentication](docs/topics/youtrack-cli-authentication.md) and [configuration](docs/topics/youtrack-cli-configuration.md).

Set `YOUTRACK_RO=1` to block every write at the client layer — a safety net for AI agents, CI read jobs, and exploration.

## Commands

Canonical verbs across every group: `view`, `list`, `create`, `update`, `delete` (`get`/`add` linger as deprecated aliases). Run `youtrack <group> --help` for the full subcommand list, or see the [command reference](docs/topics/youtrack-cli.md).

| Group | Commands |
|---|---|
| **issue** | `list`, `view`, `create`, `update`, `delete`, `count`, `command` (+ `command suggest`), `assign`/`unassign`, `watch`, `vote`, `tag`, `visibility`, `comment`, `link`, `attachment`, `field`, `reaction`, `draft`, `workitem`, `activity`, `vcs` |
| **project** | `list`, `view`, `create`, `update`, `delete`, `archive`, `team`, `field`, `template`, `timetrack`, `workflow` |
| **article** | `list`, `view`, `create`, `update`, `delete`, `comment`, `attachment`, `tag` |
| **board** | `list`, `view`, `create`, `update`, `delete`, `sprint` |
| **tag** | `list`, `create`, `delete`, `issue` |
| **search** | `list`, `view`, `create`, `delete`, `share` |
| **user** | `list`, `view`, `current`, `create`, `delete`, `ban`/`unban`, `drafts`, `locale`, `timezone`, `profile` |
| **group** | `list`, `view`, `create`, `delete`, `member`, `role` |
| **role** | `list`, `view`, `create`, `update`, `delete`, `grant`, `revoke`, `permission` |
| **org** | `list`, `create`, `delete` |
| **field** | `list`, `create`, `delete`, `bundle`, `prototypes` |
| **workflow** | `list`, `create`, `delete`, `rule` |
| **settings** | `view`, `feature` |
| **app** | `list`, `plugin`, `deactivate`, `delete` |
| **dashboard** | `list`, `create`, `update`, `delete` |
| **gantt** | `list`, `update`, `delete`, `settings`, `member` |
| **whiteboard** | `list`, `view`, `create`, `delete`, `node`, `link` |
| **auth** | `login` (`get-token`), `logout`, `status`, `refresh-token` |
| **instance** | `add`, `login`, `list`, `use`, `version`, `delete`, `refresh-token` |
| **raw** | Authenticated request to any REST endpoint |
| **update** | Check for a newer CLI release and upgrade through the detected install method |
| **skill** | `list`, `install`, `update`, `remove` |

## For AI agents

The CLI ships with an [Agent Skill](https://agentskills.io) that teaches coding agents (Claude Code, Cursor, and others) how to drive `youtrack`:

```bash
youtrack skill install    # install the bundled skill for detected agents
youtrack skill update     # update to the version bundled with this release
youtrack skill remove     # uninstall
```

or specifically for **Claude Code** via the GitHub CLI:
```bash
gh skill install JetBrains/youtrack-cli youtrack-cli --allow-hidden-dirs --agent claude-code
```

See [AI agent integration](docs/topics/youtrack-cli-ai-agent-integration.md) for read-only mode, JSON output, and skill details.

## Learn more

- [Getting started](docs/topics/youtrack-cli-get-started.md) — install, authenticate, and run your first commands
- [Authentication](docs/topics/youtrack-cli-authentication.md) — tokens, browser SSO (OAuth client registration, headless sign-in, automatic refresh), multiple instances, credential precedence
- [Configuration](docs/topics/youtrack-cli-configuration.md) — config file, environment variables, global flags, update checks
- [Scripting and automation](docs/topics/youtrack-cli-scripting.md) — JSON/plain output, field projection, exit codes, CI
- [Security](docs/security.md) — token disclaimer, `YOUTRACK_RO`, keyring storage, command-mutability reference

## Contributing

YouTrack CLI is open source under the Apache-2.0 license. Bug reports and pull requests are welcome — [CONTRIBUTING.md](CONTRIBUTING.md) covers how the project is built and tested.
