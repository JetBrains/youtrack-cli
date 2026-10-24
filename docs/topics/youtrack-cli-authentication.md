[//]: # (title: Authentication)

<show-structure for="chapter" depth="2"/>

YouTrack CLI authenticates to the YouTrack REST API with either a permanent token or an OAuth session obtained through browser SSO, which the CLI refreshes automatically. This page covers obtaining and storing credentials for one or more instances and how the CLI decides which credentials to use.

## Authenticating with a permanent token {id="auth-token"}

YouTrack CLI authenticates with a permanent token. A permanent token does not have an expiration date, which makes it suitable for scripts and CI/CD pipelines. A token carries a `perm:` prefix.

For interactive use, prefer [browser SSO](#browser-sso). For scripts and automation, use a permanent token, ideally one that belongs to a dedicated account with only the permissions the automation needs, so a leaked token exposes as little as possible.

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

The `auth login` command signs you in and saves a permanent token. By default it creates the token through browser sign-in; you can also paste one or create it through basic auth. Run it with no arguments:

```Shell
youtrack auth login
```

It asks for the YouTrack URL (unless you pass `--base-url`). The field is pre-filled with the URL of the active instance, or `https://youtrack.jetbrains.com` when none is configured; edit it or press Enter to accept it. The alias is derived from the URL; pass `--alias` to choose another. Then:

- By default, the CLI signs you in through the browser, as described in [Signing in with the browser](#browser-sso), and saves a permanent token created through Hub.
- When the instance's Hub has no OAuth client for the CLI, or you pass `--no-browser`, the CLI prints the token page link, offers to open it in your browser, and asks you to paste a permanent token you created there. The same happens when stdin is not a terminal, so scripts can pipe a token in.
- With `--login` and `--password`, the CLI creates the token through Hub basic auth instead. Basic auth is deprecated: the password passes through the CLI and it does not work with SSO or two-factor authentication. If basic auth is unavailable, for example on YouTrack Cloud or SSO-only instances, the CLI falls back to pasting a token. Over a plain `http` URL to a host other than `localhost`, basic auth stops before it sends the password; use the `https` URL, or pass `--insecure` to send the password unencrypted.
- With `--session`, the CLI saves an auto-refreshing OAuth session instead of a permanent token; see [Automatic refresh](#oauth-refresh).

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

Comma-separated Hub services to include in the token besides YouTrack: service IDs, or service names with `--login`. A basic-auth token defaults to `YouTrack` plus `YouTrack Administration`.

</td>
</tr>
<tr>
<td>

`--save`

</td>
<td>

Save credentials to the [config file](youtrack-cli-configuration.md#config-file) (`~/.config/youtrack/config.json`). Enabled by default; pass `--save=false` to verify a token without storing it.

</td>
</tr>
</table>

## Signing in with the browser {id="browser-sso"}

`auth login` authenticates through your existing browser session without entering a password or pasting a token:

```Shell
youtrack auth login https://example.youtrack.cloud
```

`auth login`, `instance login`, and `instance add` are the same command with the same flags; use whichever reads best.

The command discovers the instance's Hub, opens its authorization page, listens only on a loopback address (`127.0.0.1`) for the OAuth callback, verifies PKCE and state, and uses the resulting access token once to create a permanent token in Hub. The CLI saves the permanent token, so you do not need to sign in again. The token appears in your Hub profile under __Authentication__ with the name `youtrack-cli` (change it with `--token-name`), and you can revoke it there. Signing in again to the same alias replaces the saved token and revokes the previous one in Hub when it carries the same name; a token with another name, for example one you created by hand, stays valid and the CLI tells you so.

To get the scope it needs for creating the token, the CLI also requests access to Hub itself. Pass `--session` to skip that and save an OAuth session instead: a short-lived access token plus a refresh token, which every command renews automatically (see [Automatic refresh](#oauth-refresh)). The authorization request then uses `access_type=offline`, so Hub issues the refresh token.

Browser sign-in requires an interactive terminal, except `--session --no-browser`. Passing a URL signs in to that instance, passing an existing alias (for example `youtrack auth login work`) renews that instance's token in place, and running the command with no argument asks for the URL.

The browser shows a static page that tells you to return to the terminal. Passing a URL signs in to a new instance, and passing an existing alias renews that instance's session in place. With no argument, the command prompts for the instance. Press `Ctrl+C` to cancel; the loopback port is released immediately.

The instance and Hub URLs must use `https`. Plain `http` works only for a local development instance on `http://localhost`, `http://127.0.0.1` or `http://[::1]`. Remote `http` URLs are always refused.

### Automatic refresh {id="oauth-refresh"}

The access token lasts about one hour. Any command that uses a saved OAuth session refreshes the access token when it has expired or will expire within 60 seconds, then continues. If Hub returns a new refresh token, it replaces the stored one. When several CLI processes run at once, a lock file (the config file's `.lock` sibling) makes sure that only one of them refreshes.

If Hub rejects the refresh token, for example because the session was revoked, the CLI deletes the stale tokens, keeps the instance entry, and asks you to sign in again:

```Shell
youtrack auth login --session <alias>
```

### Registering an OAuth client {id="oauth-client"}

The CLI signs in as the OAuth client with the Client ID `youtrack-cli`. On `hub.jetbrains.com`, it uses the existing `localhost-auth` client instead. If the instance has no such client, a system administrator registers one. Since YouTrack 2026.2, OAuth clients are managed directly in YouTrack (this needs the __Low-level Admin Write__ permission):

1. Open __Administration__ > __Access Management__ > __OAuth Clients__, click __New OAuth client__, enter the name `YouTrack CLI`, and click __Create__.
2. On the __Authentication__ tab, set __Client ID__ to `youtrack-cli`. Enable __Authorization Code flow__ and __Require PKCE__. Leave the other flows disabled.
3. Under __Redirect URIs__, add `http://127.0.0.1`. A registered loopback URI matches any port, so the CLI can use a free one. If your policy requires a fixed port, register `http://127.0.0.1:<port>` and pass `--port <port>`.
4. Leave __Client secret__ unset. The CLI runs on users' machines and cannot keep a secret; PKCE protects the code exchange instead.
5. Click __Save__. Users can now sign in without extra flags.

If your YouTrack Server is connected to an external Hub, the __OAuth Clients__ page is not available in YouTrack: register the client as a service in that Hub, with the same Client ID, flow, PKCE, and redirect URI settings, and pass `--hub-url`.

If your organization already has a client with another Client ID for this flow, pass that ID with `--client-id`.

If the client is missing or rejects the loopback redirect, `auth login --session` says so and falls back to the permanent-token paste flow. If Hub reports `invalid_client` or `unauthorized_client` during the exchange, check on the client's __Authentication__ tab that __Authorization Code flow__ and __Require PKCE__ are enabled and that no __Client secret__ is required.

### Signing in without a local browser {id="oauth-headless"}

Over SSH, in WSL without a browser bridge, or on a headless machine, use `--session --no-browser`. The CLI also chooses this mode on its own when it detects no browser.

```Shell
youtrack auth login https://example.youtrack.cloud --session --no-browser
```

The CLI prints the authorization URL. Open it in a browser on any device and sign in. The browser then goes to `http://127.0.0.1:<port>/?code=...&state=...`, a page that fails to load on that device. Copy the full URL from the address bar and paste it into the terminal. The CLI accepts it only if it points at the expected loopback address and carries the `state` of this sign-in attempt. It refuses a bare code.

Browser sign-in also accepts these flags. Combined with `--no-browser`, `--login`, or piped stdin, the sign-in-only flags are rejected rather than ignored:

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

Save the resulting session. Enabled by default; pass `--save=false` to only verify that sign-in works — nothing is stored and no token is printed.

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

`--client-id`

</td>
<td>

Hub OAuth client ID. Defaults to `localhost-auth` on `hub.jetbrains.com` and to `youtrack-cli` on every other Hub. Pass it only for a client with another ID.

</td>
</tr>
<tr>
<td>

`--hub-url`

</td>
<td>

Hub base URL, for a standalone Hub. By default the CLI discovers it from the instance (for a built-in Hub, `<base URL>/hub`).

</td>
</tr>
<tr>
<td>

`--scope`

</td>
<td>

Comma-separated extra Hub service IDs to request and include in the token, in addition to YouTrack.

</td>
</tr>
<tr>
<td>

`--port`

</td>
<td>

Fixed loopback port for the OAuth callback listener. `0`, the default, picks a free port. Use it only if your client registration requires a fixed port.

</td>
</tr>
<tr>
<td>

`--timeout`

</td>
<td>

How long to wait for the browser callback or a pasted URL. Defaults to 5 minutes.

</td>
</tr>
<tr>
<td>

`--no-browser`

</td>
<td>

Skip browser sign-in and paste a permanent token instead. With `--session`, print the authorization URL and accept the redirect URL pasted back into the terminal instead (no browser and no terminal needed on this machine); the paste-back is also offered automatically when no browser is detected, for example over SSH.

</td>
</tr>
<tr>
<td>

`--token-name`

</td>
<td>

Name of the permanent token created in Hub. Default: `youtrack-cli` (`cli-token` for basic auth).

</td>
</tr>
<tr>
<td>

`--session`

</td>
<td>

Save an auto-refreshing OAuth session (access + refresh token) instead of creating a permanent token.

</td>
</tr>
<tr>
<td>

`--insecure`

</td>
<td>

With `--login`, allow basic auth over plain `http` to a host other than `localhost` (the password is sent unencrypted). Browser sign-in rejects it: a local `http` instance works without it, and remote `http` URLs are always refused.

</td>
</tr>
</table>


## Managing multiple instances {id="instances"}

YouTrack CLI can hold credentials for several instances at once, for example a production instance and a sandbox. Each instance is identified by an alias.

Add an instance the same way you log in the first time:

```Shell
youtrack auth login
```

Check every configured instance and confirm each token is valid. The active instance is marked:

```Shell
youtrack auth status
```

`youtrack instance list` prints the same instances as one row each, which suits scripts.

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

If you omit the alias, the active instance is used. The command asks how to get the new token: basic auth, the token page in a browser, or pasting one. The equivalent `instance refresh-token <alias>` requires the alias explicitly. To get the new token through browser sign-in instead, run `youtrack auth login <alias>`.

## Checking status and logging out {id="status-logout"}

Show the authenticated user for the active instance, or for every configured instance when several are saved:

```Shell
youtrack auth status
```

`auth status` always exits 0 — it is an informational probe. Inspect the reported `status` (`authenticated`, `invalid token`, or `not configured`) rather than the exit code.

For each instance, `auth status` also reports the auth method (`token` or `oauth`), the access-token expiry, and whether a refresh token is stored (`authMethod`, `expiresAt`, and `hasRefreshToken` in JSON). With `--token`, `YOUTRACK_TOKEN`, or `--server`, the single-instance report also says where the token came from (`source`). It never prints the tokens.

Remove stored credentials with `youtrack auth logout`. By default this logs out of the active instance; pass `--instance <alias>` to target another. For a permanent token, logging out removes local credentials only; the token stays valid on the server until you delete it from the __Account Security__ tab of your profile. For an OAuth session, logout also removes the tokens only locally. Hub gives the CLI no way to revoke them, so the refresh token stays valid at Hub until it expires, and the CLI prints a warning. To revoke it, open your profile in Hub (__Authentication__ or __Account Security__).

## How credentials are resolved {id="precedence"}

When you run a command, the CLI resolves the base URL and token independently, each from the first source that provides it, in this order:

1. The `--base-url` and `--token` flags. An explicit flag always wins.
2. The `--server <alias>` flag, which selects a saved instance profile for the current command.
3. The `YOUTRACK_BASE_URL` and `YOUTRACK_TOKEN` environment variables.
4. The active instance saved with `instance use`.

An explicitly provided token (`--token`, or `YOUTRACK_TOKEN`) always takes precedence over a saved OAuth session, and never triggers a refresh of it. If nothing provides a token, the command fails and tells you how to sign in.

Because the two values are resolved separately, you can mix sources — for example, take the base URL from the active instance while overriding only the token with `YOUTRACK_TOKEN`.

A token saved for an instance profile is only ever sent to that profile's origin (scheme, host, and port). If `--base-url` or `YOUTRACK_BASE_URL` points somewhere else and the token would come from a saved profile, the command fails instead of forwarding the saved token. Pass `--token` or `YOUTRACK_TOKEN` for the other instance, or select its profile with `--server <alias>`.

## Where tokens are stored {id="token-storage"}

Credentials saved by `auth login`, `instance add`, `instance login`, and `auth refresh-token` are stored in the operating-system keyring when one is available:

- macOS Keychain
- Windows Credential Manager
- Linux Secret Service, including GNOME Keyring and KWallet

The keyring service is `youtrack-cli`, keyed by instance alias. For an OAuth session, a second entry under the service `youtrack-cli-oauth`, with the same alias, holds the refresh token and the Hub token endpoint recorded at sign-in; the CLI sends the refresh token only to that endpoint. A third entry under the service `youtrack-cli-url`, with the same alias, records the instance URL the token belongs to; the CLI sends the token only to that URL. The config file at `~/.config/youtrack/config.json` then contains the instance URL and non-secret metadata but no token. Removing an instance with `instance delete` or logging out with `auth logout` deletes all its keyring entries as well.

If the keyring is unavailable, for example in a headless container or with a locked keychain, the CLI falls back to writing the tokens to the config file with mode `0600` and prints a warning to stderr. Every write goes to a new temporary file that is renamed into place, so a crash can't leave a half-written file. If the file system can't restrict the file to its owner, the CLI refuses to write tokens there. Existing file-stored tokens migrate to the keyring on a later successful save. Set `YOUTRACK_DISABLE_KEYRING=1` to request file storage explicitly.

The `YOUTRACK_TOKEN` environment variable and the `--token` flag bypass the config file entirely. Prefer them in CI and containers, where writing a token to disk is undesirable — pass `--save=false` to `auth login` if you want to verify a token without saving it.

> If the CLI reports that it used file fallback, protect the config file the way you would protect an SSH private key. Never commit it to version control or copy it to a shared machine.
>
{style="warning"}

## Token scope and security {id="security"}

A permanent token acts on behalf of your YouTrack account. YouTrack does not currently support granular permission scoping for permanent tokens, so a token can do anything your account can do — create, update, and delete issues, projects, and users. The `--scope` flag on the basic-auth flow selects which Hub services the token can reach, not object-level permissions.

To limit exposure:

- Keep `~/.config/youtrack/config.json` readable only by your user, or pass the token through an environment variable or a secrets manager instead of saving it. Never commit a token to version control.
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
