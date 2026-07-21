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

> If no base URL or token can be resolved, the command fails with a message naming the missing value. Run `youtrack instance add` to configure one, or set the environment variables.
>
{style="note"}

## The config file {id="config-file"}

YouTrack CLI stores its configuration in `~/.youtrack-cli-config.json`. The file is written with `0600` permissions, so only your user account can read it. Saved tokens use the OS keyring when available and are omitted from this file. You do not edit it by hand — the `youtrack instance` commands manage it for you.

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

Manage the file with the `youtrack instance` commands: `instance add` authenticates and saves a new instance (its alias is derived from the URL hostname unless you pass `--alias`), `instance list` shows all instances and which is active, `instance use <alias>` switches the active instance, `instance refresh-token <alias>` replaces a saved token in place, and `instance delete <alias>` removes one. See [Authentication](youtrack-cli-authentication.md) for how to obtain a permanent token and complete the `instance add` flow.

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
</table>

For how to combine `--json`, `--plain`, and `--fields` in scripts, see [Scripting and Automation](youtrack-cli-scripting.md).

## The startup update notice {id="update-notice"}

On startup, a released build of YouTrack CLI checks GitHub at most once every 24 hours for a newer release. It shows a notice only when the latest release is strictly newer than the running version by semantic-version comparison, printing a single informational line to stderr after the command finishes:

```Shell
ℹ  youtrack <tag> is available (you have <version>) — see https://github.com/JetBrains/youtrack-cli/releases
```

The check runs in the background, writes nothing to stdout, and records the time of the last attempt in `lastUpdateCheck` to throttle itself to once every 24 hours. Command exit waits no more than 500 milliseconds for the notice. Development builds, `--quiet`, CI, non-interactive stderr, `YOUTRACK_NO_UPDATE`, and `DO_NOT_TRACK` skip the network check. To upgrade, reinstall using any method in [Getting Started](youtrack-cli-get-started.md).

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
