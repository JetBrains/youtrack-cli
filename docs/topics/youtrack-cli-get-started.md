[//]: # (title: Getting Started)

<show-structure for="chapter" depth="2"/>

YouTrack CLI lets you manage issues, projects, and users from the command line. This page walks you through installing the `youtrack` binary, connecting it to your YouTrack instance, and running your first commands.

## Installing YouTrack CLI {id="install"}

YouTrack CLI ships as a single binary for macOS, Linux, and Windows. Choose the channel that fits your platform.

<tabs>
<tab title="macOS and Linux">

Run the install script. It downloads the latest release and places the `youtrack` binary on your `PATH`:

```Shell
curl -fsSL https://github.com/JetBrains/youtrack-cli/releases/latest/download/install.sh | bash
```

By default the binary installs to `/usr/local/bin`. If that directory is not writable, the script falls back to `~/.local/bin` and prints the `export PATH` line to add.

<chapter title="Install from a .deb or .rpm package (advanced)" collapsible="true" default-state="collapsed">

On Debian or Ubuntu, download and install the `.deb` package:

```Shell
curl -fsSLO https://github.com/JetBrains/youtrack-cli/releases/latest/download/youtrack_linux_amd64.deb
sudo dpkg -i youtrack_linux_amd64.deb
```

On RHEL or Fedora, install the `.rpm` package directly:

```Shell
sudo rpm -i https://github.com/JetBrains/youtrack-cli/releases/latest/download/youtrack_linux_amd64.rpm
```

Replace `amd64` with `arm64` for ARM machines.

</chapter>

</tab>
<tab title="Windows">

Download `youtrack_<version>_windows_x86_64.zip` from the [Releases](https://github.com/JetBrains/youtrack-cli/releases/latest) page, extract it, and add the folder that contains `youtrack.exe` to your `PATH`.

</tab>
</tabs>

### Installing with go install

If you have a Go toolchain, install YouTrack CLI from source. This requires Go 1.26.5 or later:

```Shell
go install github.com/JetBrains/youtrack-cli@latest
```

The binary lands in `$(go env GOPATH)/bin`. Make sure that directory is on your `PATH`.

### Verifying the installation

Confirm the binary is available and print its version:

```Shell
youtrack --version
```

## Setting your credentials {id="credentials"}

YouTrack CLI authenticates to the YouTrack REST API with a permanent token. A permanent token does not have an expiration date, which makes it suitable for scripts and CI/CD pipelines. You can store credentials interactively or pass them through environment variables.

### Authenticating interactively

Run the setup command and follow the prompts:

```Shell
youtrack instance add
```

The command asks for your base URL (for example, `https://example.youtrack.cloud`), an alias for the instance, and an authentication method:

- __Basic auth__ — sign in with your login and password; the CLI creates a permanent token for you.
- __Web browser__ — the CLI opens the token page so you can create a token and paste it back.
- __Paste token__ — enter an existing permanent token directly.
- __Browser SSO__ — authenticate through Hub with OAuth 2.0 + PKCE and save an OAuth session that the CLI refreshes automatically. You can also run `youtrack instance login <url>` directly; add `--no-browser` over SSH.

Saved tokens use the OS keyring when available; the config file at `~/.youtrack-cli-config.json` stores the instance metadata. To manage several instances, register each under its own alias and switch between them with `youtrack instance use <alias>`. See [Configuration](youtrack-cli-configuration.md) for the full precedence and storage rules.

### Using environment variables

For CI, containers, or a single instance, export your base URL and token instead. First, create a permanent token in YouTrack:

1. In YouTrack, click your avatar and open your profile.
2. Open the __Account Security__ tab and locate the __Tokens__ section.
3. Click __New token__, give it a name, and confirm. Copy the token at this point — as soon as you close the dialog, you can't access it again.

Then export both values. The token carries a `perm:` prefix:

```Shell
export YOUTRACK_BASE_URL=https://example.youtrack.cloud
export YOUTRACK_TOKEN=perm:...
```

> Treat the token like a password. Do not pass it as a command-line flag in scripts, where it may appear in shell history. YouTrack does not offer granular token scopes, so any token grants full account access.
>
{style="warning"}

Verify that the CLI can reach your instance:

```Shell
youtrack user current
```

YouTrack returns your login and profile details. If it fails, check that the base URL and token are correct. See [Authentication](youtrack-cli-authentication.md) for troubleshooting.

## Running your first commands

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

Create an issue with `youtrack issue create`. Specify the project by its short name and provide a summary:

```Shell
youtrack issue create --project DEMO --summary "Login button misaligned"
```

YouTrack returns the new issue ID, for example `DEMO-42`. To set a description as you create the issue, add the `--description` flag.

> Use `youtrack project list` to look up a project's short name.
>
{style="tip"}

By default, commands print human-readable tables. Add the `--json` flag for machine-readable output, or `--plain` for tab-separated columns suited to `grep` and `awk`.

## Where to go next

- [Authentication](youtrack-cli-authentication.md) — tokens, multiple instances, and credential precedence.
- [Managing Issues](youtrack-cli-managing-issues.md) — search queries, fields, comments, and applying commands.
- [Configuration](youtrack-cli-configuration.md) — the config file, environment variables, and output flags.

<seealso>
<category ref="user-guide">
<a href="youtrack-cli-authentication.md">Authentication</a>
<a href="youtrack-cli-configuration.md">Configuration</a>
<a href="youtrack-cli-managing-issues.md">Managing Issues</a>
</category>
<category ref="reference">
<a href="youtrack-cli.md">YouTrack CLI overview</a>
</category>
</seealso>
