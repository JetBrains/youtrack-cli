# YouTrack CLI

[![](https://jb.gg/badges/official-plastic.svg)](https://github.com/JetBrains)
[![GitHub Release](https://img.shields.io/github/v/release/JetBrains/youtrack-cli?style=plastic)](https://github.com/JetBrains/youtrack-cli/releases/latest)

Command-line client for [YouTrack](https://www.jetbrains.com/youtrack/), covering the day-to-day — issues, commands, projects, users — plus the odd jobs: knowledge-base articles, agile boards, admin settings, and raw REST calls when nothing else fits. Built for people at a terminal and AI agents driving YouTrack in a pipeline.

![youtrack-cli demo](docs/assets/youtrack-cli.gif)

Full documentation — installation, authentication, configuration, and a command reference — lives under [docs/](docs/topics/youtrack-cli.md).

> **Security notice:** YouTrack does not support granular token scopes; any token grants full account access. See [docs/security.md](docs/security.md) before using `youtrack` in pipelines or with AI agents.

## Install

<!-- Repo went internal-only after this section shipped as public-repo instructions in tag v0.3.0.
When public again, swap the `gh release download` commands below for these:

macOS & Linux:
curl -fsSL https://github.com/JetBrains/youtrack-cli/releases/latest/download/install.sh | bash

Windows: download youtrack_<version>_windows_x86_64.zip from
https://github.com/JetBrains/youtrack-cli/releases/latest, extract, add to PATH.

Debian/Ubuntu:
curl -fsSLO https://github.com/JetBrains/youtrack-cli/releases/latest/download/youtrack_linux_amd64.deb
sudo dpkg -i youtrack_linux_amd64.deb

RHEL/Fedora:
sudo rpm -i https://github.com/JetBrains/youtrack-cli/releases/latest/download/youtrack_linux_amd64.rpm

go install github.com/JetBrains/youtrack-cli@latest
-->
> **The repository is currently internal.** The `curl`/direct-download URLs below 404 until it's public. Use [GitHub CLI](https://cli.github.com/), authenticated against this org (`gh auth login`), instead.

**macOS & Linux (install script):**
```bash
gh release download --repo JetBrains/youtrack-cli --pattern install.sh -O - | bash
```

The installer verifies `checksums.txt.asc` using the JetBrains CodeSign public
key. Obtain and import that key through a trusted JetBrains channel before
running the installer.

**Windows:** `gh release download --repo JetBrains/youtrack-cli --pattern '*windows_x86_64.zip'`, extract the archive, and add the folder to your `PATH`.

<details>
<summary>More installation methods (deb, rpm, go install)</summary>

**Debian/Ubuntu:**
```bash
gh release download --repo JetBrains/youtrack-cli --pattern '*linux_amd64.deb'
sudo dpkg -i youtrack_linux_amd64.deb
```

**RHEL/Fedora:**
```bash
gh release download --repo JetBrains/youtrack-cli --pattern '*linux_amd64.rpm'
sudo rpm -i youtrack_linux_amd64.rpm
```

Replace `amd64` with `arm64` on ARM machines.

**Build from source** (requires [Go 1.26.5+](https://go.dev/dl/) and `git`/`gh` configured for private-module access):
```bash
GOPRIVATE=github.com/JetBrains/youtrack-cli go install github.com/JetBrains/youtrack-cli@latest
```

See the [getting started guide](docs/topics/youtrack-cli-get-started.md) for the full walkthrough.

</details>

## Usage

Authenticate once and the CLI remembers the instance:

```bash
youtrack instance add
```

Then:

```bash
# your open issues
youtrack issue list --query "assignee: me State: Open"

# create an issue
youtrack issue create --project DEMO --summary "Login button unresponsive on Safari"

# move it along
youtrack issue command DEMO-42 "State In Progress"

# view an issue with its five most recent comments
youtrack issue view DEMO-42 --comments 5

# what a command would do, without applying it
youtrack issue command suggest DEMO-42 "State "

# keep the CLI itself up to date
youtrack update
```

Every command takes `--json` or `--plain` for [scripting](docs/topics/youtrack-cli-scripting.md), and `--fields` to pick table/plain columns and pull extra fields from the API. When no command covers what you need, `youtrack raw` calls the REST API directly with your stored credentials. You can register several instances and switch between them, sign in through the browser, or point at a server with environment variables — see [authentication](docs/topics/youtrack-cli-authentication.md) and [configuration](docs/topics/youtrack-cli-configuration.md).

Set `YOUTRACK_RO=1` to block every write at the client layer — a safety net for AI agents, CI read jobs, and exploration.

## Commands

Canonical verbs across every group: `view`, `list`, `create`, `update`, `delete` (`get`/`add` linger as deprecated aliases). Run `youtrack <group> --help` for the full subcommand list, or see the [command reference](docs/topics/youtrack-cli.md).

| Group | Commands |
|---|---|
| **issue** | `list`, `view`, `create`, `update`, `delete`, `count`, `command` (+ `command suggest`), `assign`/`unassign`, `watch`, `vote`, `tag`, `visibility`, `comment`, `link`, `attachment`, `field`, `reaction`, `draft`, `workitem`, `activity` |
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
- [Authentication](docs/topics/youtrack-cli-authentication.md) — tokens, browser SSO, multiple instances, credential precedence
- [Configuration](docs/topics/youtrack-cli-configuration.md) — config file, environment variables, global flags, update checks
- [Scripting and automation](docs/topics/youtrack-cli-scripting.md) — JSON/plain output, field projection, exit codes, CI
- [Security](docs/security.md) — token disclaimer, `YOUTRACK_RO`, keyring storage, command-mutability reference
- [TeamCity CLI comparison](docs/teamcity-cli-feature-comparison.md) — evaluated feature candidates and proposed implementation order

## Contributing

YouTrack CLI is open source under the Apache-2.0 license. Bug reports and pull requests are welcome — [CONTRIBUTING.md](CONTRIBUTING.md) covers how the project is built and tested.
