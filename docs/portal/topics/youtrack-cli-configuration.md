# Configuration
<show-structure for="chapter" depth="2"/>

This page is the configuration reference for YouTrack CLI. It describes how the CLI finds the base URL and the token, what the config file stores, and which environment variables and global flags the CLI reads. You do not have to edit any file: `youtrack auth login` and the `youtrack instance` commands manage the configuration.

## Flags win over environment variables, which win over the active instance

YouTrack CLI resolves the base URL and the token separately. For each value, the CLI uses the first source in this list that has a value:

1. The `--base-url` and `--token` flags.
2. The `--server <alias>` flag. It selects a saved instance for one command.
3. The `YOUTRACK_BASE_URL` and `YOUTRACK_TOKEN` environment variables.
4. The active instance in the config file, set with `youtrack instance use`.

Because the two values resolve separately, you can mix sources. For example, the base URL can come from the active instance and the token from `YOUTRACK_TOKEN`.

These rules apply to saved OAuth sessions:

- The `--token` flag always wins over a saved OAuth session. The CLI does not refresh the session.
- `YOUTRACK_TOKEN` wins over the active instance, but not over an instance that you select with `--server`. If that instance has saved credentials, the CLI uses them and refreshes the session when necessary.
- With `--server`, the CLI never uses the token of the active instance.

A saved token goes only to the origin (scheme, host, and port) of its instance. If the token comes from a saved instance (sources 2 and 4) and the base URL points to another origin, the command fails. It does not send the saved token. For another instance, pass `--token` or `YOUTRACK_TOKEN`, or select its saved instance with `--server <alias>`.

<note>

If the CLI cannot resolve the base URL or the token, the command fails with a message that names the missing value. Run `youtrack auth login`, or set the environment variables.

</note>

## The config file has no secrets when a keyring is available

YouTrack CLI stores its configuration in `$XDG_CONFIG_HOME/youtrack/config.json`, which is `~/.config/youtrack/config.json` when `XDG_CONFIG_HOME` is not set. The location is the same on macOS, Linux, and Windows. If the CLI finds a config file at `~/.youtrack-cli-config.json`, it moves the file to this location. The CLI writes the file with `0600` permissions, so only your user can read it. When the OS keyring is available, the CLI stores tokens in the keyring, not in this file. Do not edit the file manually; the `youtrack auth` and `youtrack instance` commands manage it.

A config file with two instances looks like this:

```json
{
  "currentInstance": "work",
  "instances": {
    "work": {
      "url": "https://example.youtrack.cloud"
    },
    "sandbox": {
      "url": "https://sandbox.youtrack.cloud",
      "authMethod": "oauth",
      "tokenExpiresAt": 1783698120
    }
  },
  "lastUpdateCheck": 1752144000
}
```

The top-level keys are:

- `currentInstance` — alias of the active instance. The CLI uses its credentials when no flag or environment variable overrides them.
- `instances` — map of alias to instance. See the instance keys below.
- `lastUpdateCheck` — Unix timestamp of the last update check.
- `latestVersion` — the latest release that the update check found.
- `dismissedVersion` — the release for which you selected __Skip until next version__.

The CLI manages `lastUpdateCheck`, `latestVersion`, and `dismissedVersion`. See [The startup update notice](#the-startup-update-notice).

Each instance entry has these keys:

- `url` — the instance URL. Always present.
- `authMethod` — `oauth` for an OAuth session from browser sign-in. Absent for a permanent token.
- `tokenExpiresAt` — Unix timestamp when the access token of the OAuth session expires.
- `refreshIssuedAt` — Unix timestamp when Hub issued the current refresh token. The CLI uses it to estimate when the session ends.
- `oauthVersion` — the current version of the OAuth session entry in the keyring.
- `sessionId` — identifies one sign-in. The CLI uses it to detect a new sign-in by another process.
- `clientId` — the OAuth client ID that the session uses.
- `tokenEndpoint` — the Hub token endpoint from the sign-in.
- `token` and `refreshToken` — present only when the keyring is not available or `YOUTRACK_DISABLE_KEYRING` is set.

The file `config.json.lock` next to the config file serializes every config update between CLI processes: token refreshes, sign-ins, `instance use`, `instance delete`, and `auth logout`. So one process never overwrites a refresh token that another process has just replaced.

These commands manage the file:

- `auth login` signs in and saves an instance. The alias comes from the URL hostname, unless you pass `--alias`.
- `instance list` shows all instances and marks the active instance.
- `instance use <alias>` changes the active instance.
- `instance refresh-token <alias>` replaces the saved credentials with a permanent token.
- `instance delete <alias>` removes an instance.

See [Authentication](youtrack-cli-authentication.md) for the sign-in methods.

## Environment variables suit CI pipelines

Environment variables win over the active instance in the config file. Use them to give credentials to continuous integration and delivery (CI/CD) pipelines.

<table>
<tr>
<td>

Variable

</td>
<td>

Description

</td>
</tr>
<tr>
<td>

`YOUTRACK_BASE_URL`

</td>
<td>

Base URL of your YouTrack instance, for example `https://example.youtrack.cloud`. The REST API is served at `<base URL>/api`.

</td>
</tr>
<tr>
<td>

`YOUTRACK_TOKEN`

</td>
<td>

Token for authentication, usually a permanent token that starts with `perm:`.

</td>
</tr>
<tr>
<td>

`YOUTRACK_RO`

</td>
<td>

Set to `1` to enable read-only mode. Only the value `1` enables it; `true` or `yes` do not. The CLI blocks write operations (create, update, delete); reads and searches work. Read-only mode is a safety net, not a security boundary. See [Security](youtrack-cli-security.md#read-only-mode-is-a-safety-net-not-a-security-boundary).

</td>
</tr>
<tr>
<td>

`YOUTRACK_DISABLE_KEYRING`

</td>
<td>

Set to any non-empty value, for example `1`, to store saved tokens in the config file (mode `0600`), not in the OS keyring. Every non-empty value disables the keyring, also `0`. Use it in headless and CI environments.

</td>
</tr>
<tr>
<td>

`YOUTRACK_NO_UPDATE`

</td>
<td>

Set to a non-empty value to disable the startup update check entirely. `DO_NOT_TRACK` and `CI` also disable it, and the check is skipped automatically when stderr is not a terminal.

</td>
</tr>
</table>

<warning>

Treat the token like a password. Set it through an environment variable, not with the `--token` flag, because the flag can appear in shell history.

</warning>

To export the credentials in your shell, run:

```Shell
export YOUTRACK_BASE_URL=https://example.youtrack.cloud
export YOUTRACK_TOKEN=perm:...
```

## Global flags work on every command

The `--base-url` and `--token` flags override the environment variables and the config file. Run `youtrack --version` to print the CLI version.

<table>
<tr>
<td>

Flag

</td>
<td>

Description

</td>
</tr>
<tr>
<td>

`--base-url <url>`

</td>
<td>

Base URL to use for this command. Overrides `YOUTRACK_BASE_URL`.

</td>
</tr>
<tr>
<td>

`--token <token>`

</td>
<td>

Token to use for this command. Overrides `YOUTRACK_TOKEN` and saved OAuth sessions; the CLI does not refresh it.

</td>
</tr>
<tr>
<td>

`-s`, `--server <alias>`

</td>
<td>

Select a saved instance by alias for this command. Run `youtrack instance list` to see the aliases.

</td>
</tr>
<tr>
<td>

`-f`, `--fields <list>`

</td>
<td>

Comma-separated field projection, for example `id,summary,project.shortName`. Controls which fields are fetched and shown.

</td>
</tr>
<tr>
<td>

`-j`, `--json`

</td>
<td>

Output as JSON. Enabled automatically when stdout is not a terminal.

</td>
</tr>
<tr>
<td>

`--plain`

</td>
<td>

Tab-separated output with no headers or colors, for `grep` and `awk` pipelines.

</td>
</tr>
<tr>
<td>

`--quiet`

</td>
<td>

Suppress tips, hints, and the update notice. Useful for scripting.

</td>
</tr>
<tr>
<td>

`--verbose`

</td>
<td>

Print the HTTP request and response to stderr. Useful for debugging.

</td>
</tr>
<tr>
<td>

`--dry-run`

</td>
<td>

Print the first write request (method, URL, and JSON body) to stdout instead of sending it, and exit 0. Password, token, and secret fields are masked recursively. Reads still run, so a command that looks something up before writing shows the real request. The command stops at that first write: later steps that depend on its result are not shown, and nothing is changed. Delete confirmations are skipped, since nothing is sent. The `auth`, `instance`, `skill`, and `update` commands change local state rather than YouTrack data and reject `--dry-run`.

</td>
</tr>
<tr>
<td>

`--mute-notifications`

</td>
<td>

Add `muteUpdateNotifications=true` to every POST, PUT, and DELETE request, including tags and cleanup deletes. `issue command` also sends `silent: true`. This needs the *Apply Commands Silently* permission in every affected project. Without that permission, YouTrack refuses the write with HTTP 403 and nothing changes; it does not fall back to notifying watchers. Retry without the flag only if notifications are acceptable. Workflow rules still run and notifications sent by workflows are not muted. Changes and their history entries are kept.

</td>
</tr>
</table>

For how to combine `--json`, `--plain`, and `--fields` in scripts, see [Scripting and Automation](youtrack-cli-scripting.md).

## The startup update notice

On startup, a released build of YouTrack CLI checks GitHub at most once every 24 hours for a newer release. When the latest release is strictly newer than the running version by semantic-version comparison, it prints one line to stderr after the command finishes:

```Shell
ℹ  YouTrack CLI v<latest> is available (you have <version>) — run 'youtrack update' to update
```

The check runs in the background and writes nothing to stdout. It records the time of the last attempt in `lastUpdateCheck` and the latest release in `latestVersion`. Command exit waits no more than 500 milliseconds for the notice. Development builds, `--quiet`, CI, non-interactive stderr, `YOUTRACK_NO_UPDATE`, and `DO_NOT_TRACK` skip the network check.

On a later run, when `latestVersion` is newer than the running version, the CLI asks before the command runs:

```Shell
Update available! v<version> → v<latest>
Release notes: https://github.com/JetBrains/youtrack-cli/releases/tag/v<latest>

  1. Update now (runs `<upgrade command>`)
  2. Skip
  3. Skip until next version
```

- **1** runs the same upgrade as `youtrack update`, then runs your command on the current version.
- **2** runs your command. The CLI asks again on the next run.
- **3** records the release in `dismissedVersion`. The CLI does not ask again until a newer release is out.

For `youtrack` without arguments, the CLI shows only the prompt: no logo, next steps, or tips. Run `youtrack` again to see the start screen.

The CLI asks only when stdin, stdout, and stderr are terminals and the output format is a table, so scripts and AI agents never see the prompt. To upgrade from a script or an app, run `youtrack update --yes`: it upgrades without a terminal and exits with an error when the upgrade fails or the install method has no automated upgrade. With JSON output, the package manager's output goes to stderr and stdout gets the status with `"upgraded": true`. It does not ask for `youtrack update`, `help`, or shell completion, for `--quiet`, or when the install method has no automated upgrade (apt, rpm, AUR, or an undetected install). The same settings that skip the network check also turn off the prompt.

To silence the notice, pass `--quiet`, which also suppresses tips and hints:

```Shell
youtrack issue list --query "for: me" --quiet
```

<seealso>
<category ref="06">
<a href="youtrack-cli-scripting.md">Scripting and Automation</a>
<a href="youtrack-cli-administration.md">Administration</a>
</category>
<category ref="04">
<a href="youtrack-cli-get-started.md">Getting Started</a>
<a href="youtrack-cli-authentication.md">Authentication</a>
</category>
</seealso>
