[//]: # (title: Getting Started)

<show-structure for="chapter" depth="2"/>

YouTrack CLI lets you manage issues, projects, and users from the command line. This page shows how to install the `youtrack` binary, connect it to your YouTrack instance, and run your first commands. Do the steps in order: install, sign in, verify, then run commands.

## Install the single binary for your platform {id="install"}

YouTrack CLI ships as a single binary for macOS, Linux, and Windows. Choose the channel that fits your platform. The install scripts verify the downloaded archive against `checksums.txt`. If `gpg` is installed, the scripts also verify the signature against the pinned JetBrains key.

<tabs>
<tab title="macOS and Linux">

**Homebrew (recommended):**

```Shell
brew install jetbrains/utils/youtrack
```

**Install script:**

Run the install script. It downloads the latest release and places the `youtrack` binary on your `PATH`:

```Shell
curl -fsSL https://jb.gg/youtrack-cli/install | bash
```

New installs use `~/.local/bin` without sudo. If you run the script again, it updates an existing unmanaged installation in place. The script adds missing Bash and Zsh `PATH` entries automatically. Restart your shell, or run the `PATH` command that the script prints.

To choose a version and directory explicitly, pass them as arguments. Replace `<version>` with a release tag, for example from the [Releases](https://github.com/JetBrains/youtrack-cli/releases) page:

```Shell
curl -fsSL https://jb.gg/youtrack-cli/install | bash -s -- <version> "$HOME/.local/bin"
```

<chapter title="Install from a .deb or .rpm package (advanced)" collapsible="true" default-state="collapsed">

For ARM machines, replace `amd64` with `arm64` in the commands below.

On Debian or Ubuntu, download and install the `.deb` package:

```Shell
curl -fsSLO https://github.com/JetBrains/youtrack-cli/releases/latest/download/youtrack_linux_amd64.deb
sudo dpkg -i youtrack_linux_amd64.deb
```

On RHEL or Fedora, install the `.rpm` package directly:

```Shell
curl -fsSLO https://github.com/JetBrains/youtrack-cli/releases/latest/download/youtrack_linux_amd64.rpm
sudo rpm -i youtrack_linux_amd64.rpm
```

</chapter>

</tab>
<tab title="Windows">

**Winget (recommended):**

```PowerShell
winget install JetBrains.YouTrackCLI
```

**Chocolatey:**

```PowerShell
choco install youtrackcli
```

**Scoop:**

```PowerShell
scoop bucket add jetbrains https://github.com/JetBrains/scoop-utils
scoop install youtrack
```

Package-manager releases can lag behind GitHub releases. If a package is unavailable, use an install script.

**PowerShell (install script):**

```PowerShell
irm https://jb.gg/youtrack-cli/install.ps1 | iex
```

**CMD (install script):**

```Shell
curl -fsSL https://jb.gg/youtrack-cli/install.cmd -o install.cmd && install.cmd && del install.cmd
```

The installers detect x86-64 or ARM64, install to `$env:USERPROFILE\.local\bin`, and add the directory to your user `PATH`. Restart your terminal afterward. If you want a specific version or destination in PowerShell, set `YOUTRACK_INSTALL_RELEASE` and `YOUTRACK_INSTALL_DIR`. CMD accepts the same choices as positional arguments: `install.cmd [version] [directory]`.

As an alternative, download the matching `youtrack_<version>_windows_x86_64.zip` or `youtrack_<version>_windows_arm64.zip` from the [Releases](https://github.com/JetBrains/youtrack-cli/releases/latest) page, extract it, and add the folder that contains `youtrack.exe` to your `PATH`.

</tab>
</tabs>

### Install from source with go install

If you have a Go toolchain (Go 1.26 or later), install YouTrack CLI from source:

```Shell
go install github.com/JetBrains/youtrack-cli@latest
```

The binary lands in `$(go env GOPATH)/bin`. Make sure that directory is on your `PATH`.

### Check that the binary runs

Print the version:

```Shell
youtrack --version
```

A release build prints its version, its commit, and its build date, for example `youtrack 0.8.2 (<commit>, <date>)`. A `go install` build carries no version data and prints `youtrack dev (unknown, unknown)`.

### Upgrade with youtrack update {id="upgrade"}

To upgrade, run:

```Shell
youtrack update
```

The command checks GitHub for a newer release and upgrades through the same install method, for example Homebrew or the install script. If the CLI cannot upgrade that install method safely (apt, rpm, AUR), it prints manual instructions. Update other package-managed installations through their package manager.

## Sign in through the browser, or use a token for CI {id="credentials"}

YouTrack CLI authenticates to the YouTrack REST API with an OAuth session or a permanent token. An OAuth session is a browser sign-in based on the OAuth 2.0 standard. It renews itself and suits interactive use. A permanent token does not expire, so it suits scripts and continuous integration (CI) pipelines. You can sign in interactively or pass a token through environment variables.

### Sign in through the browser

Run the sign-in command and follow the prompts:

```Shell
youtrack auth login
```

The command asks for your YouTrack URL, pre-filled with the active instance's URL or `https://youtrack.jetbrains.com`. Edit the URL or press Enter to accept it. The command then opens your browser. You sign in through Hub, the JetBrains service that manages YouTrack users and logins. The CLI saves an OAuth session and renews it automatically. Hub limits the session to 90 days by default, and the CLI tells you when to sign in again. If the browser is on another device, for example over Secure Shell (SSH), add `--no-browser` and paste the redirect URL back.

To create and save a permanent token instead, add `--permanent`. If the instance's Hub has no OAuth client for the CLI, or you pass `--no-browser --permanent`, the CLI prints the token page link, offers to open it, and asks you to paste a permanent token you created there.

Saved tokens use the OS keyring when it is available. The config file at `~/.config/youtrack/config.json` stores the instance metadata. To manage several instances, register each under its own alias and switch between them with `youtrack instance use <alias>`. See [Configuration](youtrack-cli-configuration.md) for the full precedence and storage rules.

### Use environment variables in CI and containers

For CI, containers, or a single instance, export your base URL and token instead.

> Treat the token like a password. Do not pass it as a command-line flag in scripts, where it can appear in shell history. YouTrack does not offer granular token scopes, so any token grants full account access. See [Security](youtrack-cli-security.md).
>
{style="warning"}

First, create a permanent token in YouTrack:

1. In YouTrack, click your avatar and open your profile.
2. Open the __Account Security__ tab and locate the __Tokens__ section.
3. Click __New token__, give it a name, and confirm. Copy the token at this point — as soon as you close the dialog, you can't access it again.

Then export both values. The token carries a `perm:` prefix:

```Shell
export YOUTRACK_BASE_URL=https://example.youtrack.cloud
export YOUTRACK_TOKEN=perm:...
```

## Verify the connection {id="verify"}

After you sign in or export the variables, check the saved credentials and the connection:

```Shell
youtrack auth status
youtrack user current
```

`auth status` shows the active instance and the token state. `user current` returns your login and profile details. If a command fails, check the base URL and the token. See [Authentication](youtrack-cli-authentication.md) for troubleshooting.

## Run your first commands

Most commands take an issue ID in the form `PROJECT-123`, for example `DEMO-42`.

Search for issues with `youtrack issue list`. The `--query` flag takes a YouTrack search query and is required:

```Shell
youtrack issue list --query "assignee: me"
```

To narrow the search to one project, filter by its short name:

```Shell
youtrack issue list --query "project: DEMO" --limit 20
```

View a single issue by its ID:

```Shell
youtrack issue view DEMO-1
```

Add `--comments 5` to include the five most recent comments.

> To look up a project's short name, run `youtrack project list`.
>
{style="tip"}

Create an issue with `youtrack issue create`. Specify the project by its short name and provide a summary:

```Shell
youtrack issue create --project DEMO --summary "Login button misaligned"
```

YouTrack returns the new issue ID, for example `DEMO-42`. To set a description as you create the issue, add the `--description` flag.

By default, commands print human-readable tables. Add the `--json` flag for machine-readable output, or `--plain` for tab-separated columns suited to `grep` and `awk`.

## Next steps

- [Authentication](youtrack-cli-authentication.md) — tokens, multiple instances, and credential precedence.
- [Managing Issues](youtrack-cli-managing-issues.md) — search queries, fields, comments, and applying commands.
- [Configuration](youtrack-cli-configuration.md) — the config file, environment variables, and output flags.

<seealso>
<category ref="04">
<a href="youtrack-cli-authentication.md">Authentication</a>
<a href="youtrack-cli-configuration.md">Configuration</a>
<a href="youtrack-cli-managing-issues.md">Managing Issues</a>
</category>
<category ref="06">
<a href="youtrack-cli.md">YouTrack CLI overview</a>
</category>
</seealso>
