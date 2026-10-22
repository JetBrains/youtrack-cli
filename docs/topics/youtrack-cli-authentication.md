[//]: # (title: Authentication)

<show-structure for="chapter" depth="2"/>

YouTrack CLI authenticates to the YouTrack REST API with either a permanent token or a short-lived OAuth token obtained through browser SSO. This page covers obtaining and storing credentials for one or more instances and how the CLI decides which credentials to use.

## Authenticating with a permanent token {id="auth-token"}

YouTrack CLI authenticates with a permanent token. A permanent token does not have an expiration date, which makes it suitable for scripts and CI/CD pipelines. A token carries a `perm:` prefix.

Create a token in the YouTrack web interface:

1. In YouTrack, open your profile and select the __Account Security__ tab.
2. Click __New token__, give it a name, and confirm.
3. Copy the token value. As soon as you close the dialog, you can't access it again.

Provide the token to the CLI as environment variables, through the interactive `auth login` command, or with the global `--token` flag.

### Setting environment variables {id="auth-env-vars"}

Export the token together with your base URL. The base URL is your instance URL; the REST API is served at `<base URL>/api`.

<tabs>
<tab title="macOS and Linux">

```Shell
export YOUTRACK_BASE_URL=https://example.youtrack.cloud
export YOUTRACK_TOKEN=perm:...
```

</tab>
<tab title="Windows">

```PowerShell
$env:YOUTRACK_BASE_URL = "https://example.youtrack.cloud"
$env:YOUTRACK_TOKEN = "perm:..."
```

</tab>
</tabs>

Verify that the credentials work with `youtrack auth status`.

> Treat the token like a password. Do not pass it as a command-line flag in scripts, where it may appear in shell history. Prefer environment variables or the config file described below.
>
{style="warning"}

## Authenticating interactively {id="interactive-login"}

The `auth login` command signs you in and saves a permanent token. Run it with no arguments:

```Shell
youtrack auth login
```

It asks for the YouTrack URL (unless you pass `--base-url`). The field is pre-filled with the URL of the active instance, or `https://youtrack.jetbrains.com` when none is configured; edit it or press Enter to accept it. The alias is derived from the URL; pass `--alias` to choose another. Then:

- By default, the CLI signs you in through the browser, as [`instance login`](#browser-sso) does, and saves a permanent token created through Hub.
- When the instance's Hub has no OAuth client for the CLI, or you pass `--no-browser`, the CLI prints the token page link, offers to open it in your browser, and asks you to paste a permanent token you created there. The same happens when stdin is not a terminal, so scripts can pipe a token in.
- With `--login` and `--password`, the CLI creates the token through Hub basic auth instead. Basic auth is deprecated: the password passes through the CLI and it does not work with SSO or two-factor authentication. If basic auth is unavailable, for example on YouTrack Cloud or SSO-only instances, the CLI falls back to pasting a token. Over a plain `http` URL to a host other than `localhost`, basic auth stops before it sends the password; use the `https` URL, or pass `--insecure` to send the password unencrypted.

The CLI verifies the token against the API before saving it, then reports the authenticated user. By default the credentials are saved and the new instance becomes active.

The token in the command output is masked in every output format, including JSON. Pass `--show-token` to print it in full, for example to hand a new token to a CI secret store:

```Shell
youtrack auth login --base-url https://example.youtrack.cloud --save=false --json --show-token | jq -r .token
```

> `auth login` always prompts for the base URL even when `YOUTRACK_BASE_URL` is set, because it is used to set up new instances. Pass `--base-url` to skip the prompt in non-interactive setups.
>
{style="note"}

### auth login flags

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

`--base-url`

</td>
<td>

Base URL of the instance. Skips the interactive URL prompt.

</td>
</tr>
<tr>
<td>

`--alias`

</td>
<td>

Name to save the instance under. Derived from the URL hostname when omitted, for example `example` from `https://example.youtrack.cloud`.

</td>
</tr>
<tr>
<td>

`--scope`

</td>
<td>

Comma-separated Hub service names for a basic-auth token. Defaults to `YouTrack` plus `YouTrack Administration`.

</td>
</tr>
<tr>
<td>

`--save`

</td>
<td>

Save credentials to `~/.youtrack-cli-config.json`. Enabled by default; pass `--save=false` to verify a token without storing it.

</td>
</tr>
</table>

## Signing in with browser SSO {id="browser-sso"}

Use `instance login` to authenticate through your existing browser session without entering a password or pasting a token:

```Shell
youtrack instance login https://example.youtrack.cloud
```

The command discovers the instance's Hub, opens its authorization page, listens only on a loopback address (`127.0.0.1`) for the OAuth callback, verifies PKCE and state, and uses the resulting access token once to create a permanent token in Hub. The CLI saves the permanent token, so you do not need to sign in again. The token appears in your Hub profile under __Authentication__ with the name `youtrack-cli` (change it with `--token-name`), and you can revoke it there. Signing in again to the same alias replaces the saved token and revokes the previous one in Hub when it carries the same name; a token with another name, for example one you created by hand, stays valid and the CLI tells you so.

To get the scope it needs for creating the token, the CLI also requests access to Hub itself. Pass `--session` to skip that and save the access token instead. It lasts about one hour and is not refreshed.

Browser SSO requires an interactive terminal. Passing a URL signs in to a new instance, passing an existing alias renews that instance's token in place, and running the command with no argument prompts for the instance to sign in to.

Browser SSO works on `hub.jetbrains.com` and other Hub versions that provide the public loopback client. If the instance's Hub has no public loopback OAuth client, the command prints a notice and falls back to the permanent-token paste flow.

The flow accepts these flags:

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

`--save`

</td>
<td>

Save the resulting credentials. Enabled by default; pass `--save=false` to sign in without storing anything.

</td>
</tr>
<tr>
<td>

`--alias`

</td>
<td>

Name to save the instance under. Derived from the URL hostname when omitted.

</td>
</tr>
<tr>
<td>

`--scope`

</td>
<td>

Comma-separated extra Hub service IDs to request in addition to the defaults.

</td>
</tr>
<tr>
<td>

`--port`

</td>
<td>

Local loopback port for the OAuth callback listener. `0`, the default, picks a free port.

</td>
</tr>
<tr>
<td>

`--client-id`

</td>
<td>

Hub OAuth client ID. Defaults to `localhost-auth`.

</td>
</tr>
<tr>
<td>

`--timeout`

</td>
<td>

How long to wait for the browser callback before giving up.

</td>
</tr>
<tr>
<td>

`--no-browser`

</td>
<td>

Print the authorization URL instead of opening a browser automatically.

</td>
</tr>
<tr>
<td>

`--token-name`

</td>
<td>

Name of the permanent token created in Hub. Default: `youtrack-cli`.

</td>
</tr>
<tr>
<td>

`--session`

</td>
<td>

Save the ~1-hour OAuth access token instead of creating a permanent token.

</td>
</tr>
</table>

The callback returns to the machine running the CLI, so remote SSH sessions require loopback port forwarding.

## Managing multiple instances {id="instances"}

YouTrack CLI can hold credentials for several instances at once, for example a production instance and a sandbox. Each instance is identified by an alias.

Add an instance the same way you log in the first time:

```Shell
youtrack instance add
```

List configured instances and confirm each token is valid. The active instance is marked:

```Shell
youtrack instance list
```

Switch the active instance by alias:

```Shell
youtrack instance use sandbox
```

Run a single command against a non-active instance with the global `--server` flag. This selects a saved profile for one invocation and does not change the active instance:

```Shell
youtrack --server sandbox issue list
```

Remove an instance and its stored token with `youtrack instance delete sandbox`.

> Aliases are derived from the URL hostname. If two instances would collide on the same alias, the CLI appends a numeric suffix, for example `work` and `work-2`.
>
{style="tip"}

## Replacing a token {id="refresh-token"}

When a token is rotated or revoked, replace the saved token without re-adding the instance. The base URL and alias are preserved:

```Shell
youtrack auth refresh-token sandbox
```

If you omit the alias, the active instance is used. The command offers the same three permanent-token methods as `auth login`. The equivalent `instance refresh-token <alias>` requires the alias explicitly.

## Checking status and logging out {id="status-logout"}

Show the authenticated user for the active instance, or for every configured instance when several are saved:

```Shell
youtrack auth status
```

`auth status` always exits 0 — it is an informational probe. Inspect the reported `status` (`authenticated`, `invalid token`, or `not configured`) rather than the exit code.

Remove stored credentials with `youtrack auth logout`. By default this logs out of the active instance; pass `--instance <alias>` to target another. Logging out removes local credentials only. The token stays valid on the server until you delete it from the __Account Security__ tab of your profile.

## How credentials are resolved {id="precedence"}

When you run a command, the CLI resolves the base URL and token independently, each from the first source that provides it, in this order:

1. The `--base-url` and `--token` flags. An explicit flag always wins.
2. The `--server <alias>` flag, which selects a saved instance profile for the current command.
3. The `YOUTRACK_BASE_URL` and `YOUTRACK_TOKEN` environment variables.
4. The active instance saved with `instance use`.

Because the two values are resolved separately, you can mix sources — for example, take the base URL from the active instance while overriding only the token with `YOUTRACK_TOKEN`.

A token saved for an instance profile is only ever sent to that profile's origin (scheme, host, and port). If `--base-url` or `YOUTRACK_BASE_URL` points somewhere else and the token would come from a saved profile, the command fails instead of forwarding the saved token. Pass `--token` or `YOUTRACK_TOKEN` for the other instance, or select its profile with `--server <alias>`.

## Where tokens are stored {id="token-storage"}

Credentials saved by `auth login`, `instance add`, `instance login`, and `auth refresh-token` are stored in the operating-system keyring when one is available:

- macOS Keychain
- Windows Credential Manager
- Linux Secret Service, including GNOME Keyring and KWallet

The keyring service is `youtrack-cli`, keyed by instance alias. A second entry under the service `youtrack-cli-url`, with the same alias, records the instance URL the token belongs to; the CLI sends the token only to that URL. The config file at `~/.youtrack-cli-config.json` then contains the instance URL and non-secret metadata but no token. Removing an instance with `instance delete` or logging out with `auth logout` deletes both keyring entries as well.

If the keyring is unavailable, for example in a headless container or with a locked keychain, the CLI falls back to writing the token to the config file with mode `0600` and prints a warning to stderr. Existing file-stored tokens migrate to the keyring on a later successful save. Set `YOUTRACK_DISABLE_KEYRING=1` to request file storage explicitly.

The `YOUTRACK_TOKEN` environment variable and the `--token` flag bypass the config file entirely. Prefer them in CI and containers, where writing a token to disk is undesirable — pass `--save=false` to `auth login` if you want to verify a token without saving it.

> If the CLI reports that it used file fallback, protect the config file the way you would protect an SSH private key. Never commit it to version control or copy it to a shared machine.
>
{style="warning"}

## Token scope and security {id="security"}

A permanent token acts on behalf of your YouTrack account. YouTrack does not currently support granular permission scoping for permanent tokens, so a token can do anything your account can do — create, update, and delete issues, projects, and users. The `--scope` flag on the basic-auth flow selects which Hub services the token can reach, not object-level permissions.

To limit exposure:

- Keep `~/.youtrack-cli-config.json` readable only by your user, or pass the token through an environment variable or a secrets manager instead of saving it. Never commit a token to version control.
- Create a dedicated service account with only the access a pipeline needs.
- Rotate tokens regularly and revoke leaked tokens from the __Account Security__ tab.
- Set `YOUTRACK_RO=1` to block all write operations when a workflow only needs to read.

<seealso>
<category ref="user-guide">
<a href="youtrack-cli-get-started.md">Getting Started</a>
<a href="youtrack-cli-scripting.md">Scripting and Automation</a>
<a href="youtrack-cli-ai-agent-integration.md">AI Agent Integration</a>
</category>
<category ref="reference">
<a href="youtrack-cli-configuration.md">Configuration</a>
<a href="youtrack-cli-administration.md">Administration</a>
</category>
</seealso>
