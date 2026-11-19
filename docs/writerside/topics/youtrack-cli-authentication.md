# Authentication

<show-structure for="chapter" depth="2"/>

YouTrack CLI signs in to YouTrack with an OAuth session from browser sign-in or with a permanent token. For interactive use, run `youtrack auth login`: it opens the browser and saves an OAuth session that the CLI refreshes automatically. For scripts and continuous integration and delivery (CI/CD) pipelines, use a [permanent token](#permanent-tokens-suit-scripts-and-ci-pipelines).

The CLI signs in through Hub, the JetBrains service that manages YouTrack users and authentication. A YouTrack instance has a built-in Hub at `<base URL>/hub`, or it connects to an external Hub.

## auth login signs you in through the browser by default

Run the command with no arguments:

```Shell
youtrack auth login
```

The command asks for the YouTrack URL. The field shows the URL of the active instance, or `https://youtrack.jetbrains.com` when no instance is configured. Edit the URL or press Enter. If you pass a URL or an alias as the argument, or pass `--base-url`, the command does not ask. The command asks for the URL even when `YOUTRACK_BASE_URL` is set, because it sets up new instances.

The command then signs you in by one of these methods:

- **Browser sign-in (default).** The CLI opens the browser and saves an OAuth session. See [Browser sign-in](#browser-sign-in-saves-an-auto-refreshing-oauth-session).
- **Browser sign-in with `--permanent`.** The CLI signs you in through the browser, creates a permanent token in Hub, and saves that token.
- **Pasted permanent token.** The CLI prints the link to the token page, offers to open it, and asks you to paste a permanent token. The CLI uses this method in these cases:
  - you pass `--no-browser --permanent`;
  - the Hub has no OAuth client for the CLI;
  - stdin is not a terminal and you do not pass `--web`. A script can pipe a token to stdin.
- **Basic auth with `--login` and `--password`.** The CLI creates a permanent token through the Hub API. See [Basic auth](#basic-auth-is-deprecated).

The CLI verifies the credentials against the API, then shows the authenticated user. By default, the CLI saves the credentials and makes the instance active. The alias comes from the URL hostname; pass `--alias` to choose another alias.

`auth login`, `auth get-token`, `instance login`, and `instance add` are the same command with the same flags.

To verify the connection, run:

```Shell
youtrack auth status
```

### Basic auth is deprecated

<warning>

Basic auth sends your password through the CLI. It does not work with single sign-on (SSO) or two-factor authentication. Use browser sign-in when you can.

</warning>

With `--login` and `--password`, the CLI creates a permanent token through the Hub API. If you omit `--password`, the CLI asks for it. The token gets the Hub services `YouTrack` and `YouTrack Administration`; pass `--scope` to choose other services by name. If basic auth is not available, for example on YouTrack Cloud or on an instance that uses only SSO, the CLI asks you to paste a token.

If the URL uses plain `http` and the host is not `localhost`, basic auth stops before it sends the password. Use the `https` URL. Pass `--insecure` only if you accept that the password goes over the network unencrypted.

## Browser sign-in saves an auto-refreshing OAuth session

```Shell
youtrack auth login https://example.youtrack.cloud
```

The command finds the Hub of the instance and opens its authorization page in the browser. The CLI listens for the OAuth callback only on the loopback address `127.0.0.1`. It checks the `state` value and uses Proof Key for Code Exchange (PKCE), which protects the authorization code. The browser then shows a static page that tells you to return to the terminal. To cancel, press `Ctrl+C`; the CLI releases the loopback port at once.

The CLI saves an OAuth session: a short-lived access token and a refresh token. Every command renews the access token automatically (see [Automatic refresh](#automatic-refresh)). The authorization request uses `access_type=offline`, so Hub issues the refresh token. If Hub issues no refresh token, the command tells you so; sign in again when the access token expires.

The argument selects the instance:

- A URL signs in to that instance.
- An existing alias, for example `youtrack auth login work`, renews the credentials of that instance in place.
- No argument makes the command ask for the URL.

Browser sign-in needs an interactive terminal. If no terminal is available, pass `--web` (see [Signing in from an app or script](#signing-in-from-an-app-or-script)) or `--no-browser` (see [Signing in without a local browser](#signing-in-without-a-local-browser)).

The instance URL and the Hub URL must use `https`. Plain `http` works only for a local instance on `http://localhost`, `http://127.0.0.1`, or `http://[::1]`. The CLI refuses remote `http` URLs.

### Browser sign-in with --permanent creates a permanent token

With `--permanent` or `--token-name`, the CLI uses the access token once to create a permanent token in Hub, then saves that token. To create the token, the CLI also requests access to the Hub service. The token gets the name `youtrack-cli`; pass `--token-name` to choose another name. You can see and revoke the token in your Hub profile (see [Where to revoke a token](#where-to-revoke-a-token)).

If you sign in again to the same alias, the CLI replaces the saved token. If the old token has the same name, the CLI revokes it in Hub. A token with another name, for example a token that you created manually, stays valid, and the CLI tells you so.

A permanent token does not expire. It works when an administrator disables the OAuth client.

### Automatic refresh

The access token is valid for about one hour. If the access token has expired or expires within 60 seconds, the command refreshes it first and then continues. If Hub returns a new refresh token, the CLI replaces the stored refresh token. If several CLI processes run at the same time, a lock file next to the config file makes sure that only one process refreshes. `auth status` also refreshes an access token that is about to expire.

By default, Hub limits a refresh token to 90 days (the Hub "remember me" interval). Hub does not tell the CLI the real lifetime. The CLI records when Hub issued the refresh token (`refreshIssuedAt` in the config file) and calculates an estimated end date. `auth status` shows this date ("sign in again by about YYYY-MM-DD"; `sessionExpiresAt` in JSON, as an RFC 3339 timestamp such as `2026-12-01T10:00:00Z`). From 7 days before that date, every command prints a tip to stderr. `--quiet` suppresses the tip. To sign in again, run:

```Shell
youtrack auth login <alias>
```

> An administrator can renew active sessions automatically. Set the Hub Java Virtual Machine (JVM) parameter `jetbrains.hub.refreshToken.daysBeforeExpiryToAutoRenew=N`. Hub then issues a new refresh token when a refresh occurs within `N` days of the expiry.

### Refresh errors tell you what to do

<table>
<tr>
<td>

Error

</td>
<td>

Cause and action

</td>
</tr>
<tr>
<td>

`the OAuth session for "<alias>" has expired or was revoked`

</td>
<td>

Hub rejected the refresh token (`invalid_grant` or `expired_refresh_token`). For example, the session was revoked, or an administrator disabled the OAuth client. The CLI deletes the stale tokens and keeps the instance entry. Run `youtrack auth login <alias>`.

</td>
</tr>
<tr>
<td>

`refreshing the OAuth session for "<alias>": ...`

</td>
<td>

A network error or a Hub error occurred. The CLI keeps the tokens. Run the command again.

</td>
</tr>
<tr>
<td>

`cannot read the OAuth session for "<alias>" from the OS keyring`

</td>
<td>

The keyring is locked. Unlock the keyring and run the command again. You do not have to sign in again.

</td>
</tr>
<tr>
<td>

`the saved session for "<alias>" was removed or replaced by another youtrack process`

</td>
<td>

Another CLI process signed in again or removed the instance while this command ran. Run the command again.

</td>
</tr>
</table>

For scripts and CI/CD pipelines, use a permanent token. A permanent token does not depend on the OAuth client or on a refresh.

### Registering an OAuth client

The CLI signs in as the OAuth client with the Client ID `youtrack-cli`. On `hub.jetbrains.com`, the CLI uses the existing client `localhost-auth`. If the instance has no such client, a system administrator registers one. From YouTrack 2026.2, you manage OAuth clients in YouTrack. You need the __Low-level Admin Write__ permission.

1. Open __Administration__ > __Access Management__ > __OAuth Clients__.
2. Click __New OAuth client__, enter the name `YouTrack CLI`, and click __Create__.
3. On the __Authentication__ tab, set __Client ID__ to `youtrack-cli`.
4. Enable __Authorization Code flow__ and __Require PKCE__. Keep the other flows disabled.
5. Under __Redirect URIs__, add `http://127.0.0.1`. A loopback URI matches any port, so the CLI can use a free port. If your policy requires a fixed port, register `http://127.0.0.1:<port>` and tell users to pass `--port <port>`.
6. Do not set a __Client secret__. The CLI runs on the computers of users and cannot keep a secret. PKCE protects the code exchange.
7. Click __Save__. Users can then sign in without extra flags.

If your YouTrack Server connects to an external Hub, YouTrack does not show the __OAuth Clients__ page. Register the client as a service in that Hub, with the same settings, and pass `--hub-url`.

If your organization has a client with another Client ID for this flow, pass that ID with `--client-id`.

If the client is missing or rejects the loopback redirect, `auth login` tells you so and asks you to paste a permanent token. If Hub reports `invalid_client` or `unauthorized_client` during the code exchange, check the __Authentication__ tab of the client: __Authorization Code flow__ and __Require PKCE__ must be enabled, and no __Client secret__ must be required.

### Signing in without a local browser

If the browser is on another device, for example over Secure Shell (SSH), in Windows Subsystem for Linux (WSL) without a browser bridge, or on a headless computer, use `--no-browser`. The CLI also selects this mode when it finds no browser.

```Shell
youtrack auth login https://example.youtrack.cloud --no-browser
```

1. The CLI prints the authorization URL. Open the URL in a browser on any device and sign in.
2. The browser goes to `http://127.0.0.1:<port>/?code=...&state=...`. This page does not load on that device.
3. Copy the full URL from the address bar of the browser.
4. Paste the URL into the terminal where the CLI runs, or pipe it to stdin.

The CLI accepts the URL only if it points to the expected loopback address and has the `state` of this sign-in attempt. The CLI refuses a code without the URL.

To paste a permanent token instead, add `--permanent`.

### Signing in from an app or script

An app that runs the CLI without a terminal, such as a desktop app that manages your CLI tools, can sign in through the browser with `--web`:

```Shell
youtrack auth login https://example.youtrack.cloud --web --json
```

With `--web`, the CLI does not ask questions. Pass the instance URL or a configured alias as the argument or with `--base-url`; if you do not, the command fails at once. If browser sign-in is not available, for example because the Hub has no OAuth client for the CLI, the command fails with exit code 1. It does not ask you to paste a token.

The CLI opens the browser, waits for the callback, and saves the credentials. It writes the authorization URL to stderr on its own line, so the caller can open the URL if the browser does not open. With JSON output (`--json`, or the default when stdout is not a terminal), the CLI does not print progress messages, and the URL arrives on stderr as a JSON line:

```JSON
{"authorizationUrl": "https://example.youtrack.cloud/hub/api/rest/oauth2/auth?..."}
```

The result goes to stdout. You cannot use `--web` together with `--no-browser`, `--login`, or `--password`.

## auth login flags

If you pass a flag that does not apply to the selected method, the command usually fails with an error that names the flag, so that the flag is not ignored silently.

<table>
<tr>
<td>

Flag

</td>
<td>

Applies to

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

All methods

</td>
<td>

Base URL of the instance. The command does not ask for the URL.

</td>
</tr>
<tr>
<td>

`--alias`

</td>
<td>

All methods

</td>
<td>

Name to save the instance under. If you omit it, the CLI uses the URL hostname, for example `example` from `https://example.youtrack.cloud`. Ignored with `--save=false`.

</td>
</tr>
<tr>
<td>

`--save`

</td>
<td>

All methods

</td>
<td>

Save the credentials to the [config file](youtrack-cli-configuration.md#the-config-file-has-no-secrets-when-a-keyring-is-available) and the OS keyring. Enabled by default. With `--save=false`, the CLI stores nothing locally. A permanent token that the command creates (with `--permanent`, `--token-name`, or `--login`) stays valid in Hub until you revoke it.

</td>
</tr>
<tr>
<td>

`--show-token`

</td>
<td>

All methods

</td>
<td>

Print the full token. Without this flag, the CLI masks the token in every output format, including JSON.

</td>
</tr>
<tr>
<td>

`--permanent`

</td>
<td>

Browser sign-in

</td>
<td>

Create and save a permanent token, not an OAuth session. With `--no-browser`, paste a permanent token. The CLI accepts the hidden flag `--session`, which selects the default OAuth session.

</td>
</tr>
<tr>
<td>

`--token-name`

</td>
<td>

Browser sign-in, basic auth

</td>
<td>

Name of the permanent token that the CLI creates in Hub. Default: `youtrack-cli` for browser sign-in, `cli-token` for basic auth. Implies `--permanent`.

</td>
</tr>
<tr>
<td>

`--scope`

</td>
<td>

Browser sign-in, basic auth

</td>
<td>

Comma-separated Hub services to include besides YouTrack. For browser sign-in, use service IDs. For basic auth, use service names; the default is `YouTrack` and `YouTrack Administration`.

</td>
</tr>
<tr>
<td>

`--client-id`

</td>
<td>

Browser sign-in

</td>
<td>

Hub OAuth client ID. Default: `localhost-auth` on `hub.jetbrains.com`, `youtrack-cli` on every other Hub.

</td>
</tr>
<tr>
<td>

`--hub-url`

</td>
<td>

Browser sign-in

</td>
<td>

Hub base URL, for an external Hub. By default, the CLI gets it from the instance, for example `<base URL>/hub`. If the instance reports a Hub on another origin, the CLI asks you to confirm that Hub. Without a terminal or with `--web`, the CLI stops and asks for `--hub-url`. The CLI does not ask you to confirm a Hub that you give with `--hub-url`.

</td>
</tr>
<tr>
<td>

`--port`

</td>
<td>

Browser sign-in

</td>
<td>

Fixed loopback port for the OAuth callback. `0`, the default, selects a free port. Use it only if your client registration requires a fixed port.

</td>
</tr>
<tr>
<td>

`--timeout`

</td>
<td>

Browser sign-in

</td>
<td>

Maximum time for the browser callback or the pasted URL. Default: 5 minutes.

</td>
</tr>
<tr>
<td>

`--no-browser`

</td>
<td>

Browser sign-in

</td>
<td>

Print the authorization URL and read the redirect URL that you paste. With `--permanent`, paste a permanent token. See [Signing in without a local browser](#signing-in-without-a-local-browser).

</td>
</tr>
<tr>
<td>

`--web`

</td>
<td>

Browser sign-in

</td>
<td>

Sign in through the browser without questions, also when stdin is not a terminal. See [Signing in from an app or script](#signing-in-from-an-app-or-script).

</td>
</tr>
<tr>
<td>

`--login`

</td>
<td>

Basic auth

</td>
<td>

Hub login. Selects basic auth.

</td>
</tr>
<tr>
<td>

`--password`

</td>
<td>

Basic auth

</td>
<td>

Password. If you omit it, the CLI asks for it. Do not use this flag in scripts, because the password can appear in shell history.

</td>
</tr>
<tr>
<td>

`--insecure`

</td>
<td>

Basic auth

</td>
<td>

Send the password over plain `http` to a host other than `localhost`. The password goes over the network unencrypted.

</td>
</tr>
</table>

## Permanent tokens suit scripts and CI pipelines

A permanent token has no expiration date and does not need a refresh. A permanent token starts with `perm:`.

<warning>

A permanent token can do everything that its user can do. For automation, create the token for a dedicated user that has only the permissions the automation needs. If the token leaks, it then exposes as little as possible.

</warning>

To create a token in the YouTrack web interface:

1. In YouTrack, open your profile and select the __Account Security__ tab.
2. Click __New token__, give it a name, and confirm.
3. Copy the token value. After you close the dialog, you cannot see the value again.

Give the token to the CLI in one of these ways:

- Set [environment variables](#setting-environment-variables). Use this way in CI/CD pipelines.
- Save the token: run `youtrack auth login --base-url <url> --no-browser --permanent` and paste the token. A script can pipe the token to stdin.
- Pass the global `--token` flag for one command.

To create a permanent token for a CI secret store from the command line, sign in through the browser with `--permanent` and print the token:

```Shell
youtrack auth login --base-url https://example.youtrack.cloud --permanent --save=false --json --show-token | jq -r .token
```

<warning>

Add `--permanent` to this command. Without `--permanent`, the command prints the OAuth access token, which expires in about one hour. With `--save=false`, the CLI does not store the refresh token.

</warning>

### Setting environment variables

<warning>

Treat the token like a password. Do not pass it with the `--token` flag in scripts, because the flag can appear in shell history. Use environment variables or the saved credentials.

</warning>

Export the token together with the base URL. The base URL is your instance URL; the REST API is at `<base URL>/api`.

**macOS and Linux:**

```Shell
export YOUTRACK_BASE_URL=https://example.youtrack.cloud
export YOUTRACK_TOKEN=perm:...
```

**Windows:**

```PowerShell
$env:YOUTRACK_BASE_URL = "https://example.youtrack.cloud"
$env:YOUTRACK_TOKEN = "perm:..."
```

To verify the credentials, run `youtrack auth status`.

## Managing multiple instances

YouTrack CLI can keep credentials for several instances, for example a production instance and a sandbox. An alias identifies each instance.

To add an instance, sign in to it:

```Shell
youtrack auth login https://sandbox.youtrack.cloud
```

If two instances get the same alias from their hostnames, the CLI adds a number, for example `work` and `work-2`.

To check every configured instance and its credentials, run `youtrack auth status`. The output marks the active instance. `youtrack instance list` shows one row for each instance, which suits scripts.

To change the active instance, run:

```Shell
youtrack instance use sandbox
```

To run one command against another instance, use the global `--server` flag. The flag does not change the active instance:

```Shell
youtrack --server sandbox issue list
```

`youtrack instance delete sandbox` removes an instance and its stored credentials. The command asks for confirmation; pass `--force` in scripts.

## auth refresh-token saves a new permanent token

`auth refresh-token` replaces the saved credentials of an instance with a permanent token. The alias and the URL stay the same. The command does not refresh an OAuth session.

<note>

If the instance has an OAuth session, `auth refresh-token` replaces it with a permanent token. To renew an OAuth session, run `youtrack auth login <alias>`.

</note>

```Shell
youtrack auth refresh-token sandbox
```

If you omit the alias, the command uses the active instance. The command asks how to get the new token: basic auth, the token page in a browser, or a pasted token. `instance refresh-token <alias>` does the same, but the alias is required.

## Checking status and logging out

To show the authenticated user, run:

```Shell
youtrack auth status
```

`auth status` shows one of two reports:

- If you set `--base-url`, `--token`, `--server`, `YOUTRACK_BASE_URL`, or `YOUTRACK_TOKEN`, it checks the credentials that a normal command would use. The report also shows where the token comes from (`source`).
- Otherwise, it checks every saved instance.

For each instance, `auth status` shows the auth method (`token` or `oauth`), the access-token expiry, whether a refresh token is stored, and, for an OAuth session, the estimated date to sign in again. The JSON fields are `authMethod`, `expiresAt`, `hasRefreshToken`, and `sessionExpiresAt`. The command never prints tokens.

`auth status` always exits with code 0. Check the reported `status` (`authenticated`, `invalid token`, or `not configured`), not the exit code.

To remove stored credentials, run `youtrack auth logout`. The command asks for confirmation; pass `--force` in scripts. By default, it logs out of the active instance; pass `--instance <alias>` for another instance.

`auth logout` removes credentials only on your computer:

- A permanent token stays valid on the server until you revoke it.
- The refresh token of an OAuth session stays valid in Hub until it expires. Hub gives the CLI no way to revoke it, so the CLI prints a warning.

### Where to revoke a token

Open your profile and revoke the token or session there. The tab name depends on the Hub:

- Built-in Hub: the __Account Security__ tab of your YouTrack profile.
- External Hub: the __Authentication__ tab of your Hub profile.

## How credentials are resolved

The CLI resolves the base URL and the token separately. The order of sources and the rules for OAuth sessions are in [Configuration: Resolution order](youtrack-cli-configuration.md#flags-win-over-environment-variables-which-win-over-the-active-instance).

## Where tokens are stored

`auth login`, `instance add`, `instance login`, and `auth refresh-token` store credentials in the operating-system keyring when one is available:

- macOS Keychain
- Windows Credential Manager
- Linux Secret Service, including GNOME Keyring and KWallet

The CLI writes these keyring entries:

<table>
<tr>
<td>

Service

</td>
<td>

Account

</td>
<td>

Content

</td>
</tr>
<tr>
<td>

`youtrack-cli`

</td>
<td>

`<alias>`

</td>
<td>

The permanent token.

</td>
</tr>
<tr>
<td>

`youtrack-cli-oauth`

</td>
<td>

`<alias>/<version>`

</td>
<td>

For an OAuth session: the access token, the refresh token, and the Hub token endpoint from the sign-in. The CLI sends the refresh token only to this endpoint. Each save writes a new version; the config file names the current version (`oauthVersion`). If the config file cannot be written, the CLI deletes the new version, and the previous session stays usable.

</td>
</tr>
<tr>
<td>

`youtrack-cli-url`

</td>
<td>

`<alias>`

</td>
<td>

The instance URL that the token belongs to. The CLI sends the token only to this URL.

</td>
</tr>
</table>

The config file `~/.config/youtrack/config.json` then has the instance URL and metadata without secrets, but no token. `instance delete` and `auth logout` also delete all keyring entries of the instance.

If the keyring is not available, for example in a headless container or with a locked keychain, the CLI writes the tokens to the config file with mode `0600` and prints a warning to stderr. Each write goes to a new temporary file, which the CLI then renames, so a crash cannot leave a half-written file. If the file system cannot limit the file to its owner, the CLI does not write tokens to it. The next successful save moves tokens from the file to the keyring. To use file storage on purpose, set `YOUTRACK_DISABLE_KEYRING` to any non-empty value, for example `1`.

<warning>

If the CLI tells you that it stores tokens in the file, protect the config file like a private SSH key. Do not commit it to version control or copy it to a shared computer.

</warning>

The `YOUTRACK_TOKEN` environment variable and the `--token` flag do not use the config file. Use them in CI and containers, where a token on disk is a risk. `auth login --save=false` stores nothing locally, but a permanent token that it creates stays valid in Hub.

## Tokens have the full permissions of their user

A token acts for your YouTrack user. YouTrack does not support permission scopes for permanent tokens, so a token can do everything that your user can do: create, update, and delete issues, projects, and users. The `--scope` flag selects the Hub services that the token can reach. It does not limit permissions on objects.

To limit the risk:

- Keep `~/.config/youtrack/config.json` readable only by your user, or give the token through an environment variable or a secrets manager. Do not commit a token to version control.
- Create a dedicated user with only the permissions that a pipeline needs.
- Rotate tokens regularly. Revoke a leaked token at once (see [Where to revoke a token](#where-to-revoke-a-token)).
- If a workflow only reads data, set `YOUTRACK_RO=1` to block write operations. Read-only mode is a safety net, not a security boundary. See [Security](youtrack-cli-security.md#read-only-mode-is-a-safety-net-not-a-security-boundary).

<seealso>
<category ref="04">
<a href="youtrack-cli-get-started.md">Getting Started</a>
<a href="youtrack-cli-scripting.md">Scripting and Automation</a>
<a href="youtrack-cli-ai-agent-integration.md">AI Agent Integration</a>
</category>
<category ref="06">
<a href="youtrack-cli-configuration.md">Configuration</a>
<a href="youtrack-cli-security.md">Security</a>
<a href="youtrack-cli-administration.md">Administration</a>
</category>
</seealso>
