[//]: # (title: Configuration)
<show-structure for="chapter" depth="2"/>

This page is the configuration reference for YouTrack CLI. It covers how the CLI finds your base URL and token, the config and keyring metadata it stores, the environment variables and global flags it reads, and the startup update notice.

## Resolution order {id="precedence"}

YouTrack CLI resolves the base URL and token independently. For each one it walks the following chain and uses the first source that supplies a value:

1. The `--base-url` and `--token` flags. An explicit per-command override always wins.
2. The `--server <alias>` flag. It selects a named profile from the config file.
3. The `YOUTRACK_BASE_URL` and `YOUTRACK_TOKEN` environment variables.
4. The active instance in the config file, set with `youtrack instance use`.

Because the two values resolve separately, you can mix sources — for example, take the base URL from the active instance while passing the token through `YOUTRACK_TOKEN`.

Saved tokens are bound to their instance: a token taken from a config profile (steps 2 and 4) is only sent when the resolved base URL has the same origin — scheme, host, and port — as that profile. Overriding just the base URL to point at a different host fails with an error rather than forwarding the saved token.

> If no base URL or token can be resolved, the command fails with a message naming the missing value. Run `youtrack auth login` to configure one, or set the environment variables.
>
{style="note"}

## The config file {id="config-file"}

YouTrack CLI stores its configuration in `$XDG_CONFIG_HOME/youtrack/config.json`, which is `~/.config/youtrack/config.json` when `XDG_CONFIG_HOME` is not set. The same location is used on macOS, Linux, and Windows. Versions before this change used `~/.youtrack-cli-config.json`; the CLI moves that file to the new location the first time it runs. The file is written with `0600` permissions, so only your user account can read it. Saved tokens use the OS keyring when available and are omitted from this file. You do not edit it by hand — the `youtrack instance` commands manage it for you.

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

- `currentInstance` — alias of the active instance. Its credentials are used when no flag or environment variable overrides them.
- `instances` — map of alias to instance. Each entry holds a `url`, plus optional OAuth metadata (`authMethod` and `tokenExpiresAt`) for browser-SSO instances. A `token` appears only when keyring storage is unavailable or explicitly disabled.
- `lastUpdateCheck` — Unix timestamp of the last update check, managed automatically. See [The startup update notice](#update-notice).

Manage the file with the `youtrack instance` commands: `auth login` signs in and saves a new instance (its alias is derived from the URL hostname unless you pass `--alias`), `instance list` shows all instances and which is active, `instance use <alias>` switches the active instance, `instance refresh-token <alias>` replaces a saved token in place, and `instance delete <alias>` removes one. See [Authentication](youtrack-cli-authentication.md) for how to obtain a permanent token and complete the `auth login` flow.

## Environment variables {id="env-vars"}

Environment variables take precedence over the config file, which makes them the preferred way to pass credentials in CI/CD pipelines.

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

Permanent token used to authenticate. It carries a `perm:` prefix.

</td>
</tr>
<tr>
<td>

`YOUTRACK_RO`

</td>
<td>

Set to `1` to enable read-only mode. Every write operation (create, update, delete) is blocked, while reads and searches still work. Useful when granting an AI agent access. See [AI Agent Integration](youtrack-cli-ai-agent-integration.md).

</td>
</tr>
<tr>
<td>

`YOUTRACK_DISABLE_KEYRING`

</td>
<td>

Set to a non-empty value to store saved tokens in the mode-`0600` config file instead of the OS keyring. Intended for headless and deterministic CI environments.

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

Export the credentials in your shell before running commands:

```Shell
export YOUTRACK_BASE_URL=https://example.youtrack.cloud
export YOUTRACK_TOKEN=perm:...
```

> Treat the token like a password. Set it through an environment variable rather than the `--token` flag, which can appear in shell history.
>
{style="warning"}

## Global flags {id="global-flags"}

These flags are available on every command. The `--base-url` and `--token` flags override both the environment variables and the config file. Run `youtrack --version` to print the CLI version.

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

Permanent token to use for this command. Overrides `YOUTRACK_TOKEN`.

</td>
</tr>
<tr>
<td>

`-s`, `--server <alias>`

</td>
<td>

Select a named instance profile from the config file. Run `youtrack instance list` to see the aliases.

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

## The startup update notice {id="update-notice"}

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
<category ref="reference">
<a href="youtrack-cli-scripting.md">Scripting and Automation</a>
<a href="youtrack-cli-administration.md">Administration</a>
</category>
<category ref="user-guide">
<a href="youtrack-cli-get-started.md">Getting Started</a>
<a href="youtrack-cli-authentication.md">Authentication</a>
</category>
</seealso>
