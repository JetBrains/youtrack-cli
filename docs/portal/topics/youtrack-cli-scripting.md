# Scripting and Automation

<show-structure for="chapter" depth="2"/>

YouTrack CLI works in scripts, pipelines, and agents without extra setup. This page describes the machine-readable output modes, field projections, exit codes, and the `raw` command for any REST endpoint. Read it before you parse CLI output in a script.

## Output is JSON when stdout is not a terminal

Every command detects whether its output is going to a terminal. If stdout is not a terminal, for example a pipe, a redirect, or a continuous integration (CI) job, YouTrack CLI switches to JSON (JavaScript Object Notation) automatically. You do not have to pass a flag.

```Shell
youtrack issue list --query "assignee: me"        # a terminal: table
youtrack issue list --query "assignee: me" | cat  # a pipe: JSON
```

To force JSON while still attached to a terminal, add `--json` (or `-j`):

```Shell
youtrack issue view DEMO-42 --json
```

Read commands print the entity or list as JSON. Write commands behave in one of two ways:

- `issue create` and `issue update` return the created or updated issue object, including its `idReadable`.
- `issue delete` and other confirmations print a short success line, such as `deleted issue DEMO-42` (prefixed with a green `✓` in a terminal).

Declining a confirmation exits with code `0`. In JSON mode, destructive commands such as `issue delete` report `{"status":"canceled","message":"Canceled."}` on stdout; table and plain modes print `Canceled.`. Check the status to distinguish a canceled operation from a completed one.

<note>

The CLI writes successful data to stdout. In JSON mode, the CLI writes a failure as a structured `{"error":{...}}` object to stderr and exits with code `1`. Stdout stays safe for `jq`, and error text does not get into a data pipeline.

</note>

An error object for an API error looks like this. The message starts with the HTTP status, followed by the server message:

```json
{
  "error": {
    "code": "not_found",
    "message": "HTTP 404: ...",
    "suggestion": "Resource not found — verify the ID"
  }
}
```

API errors use the codes `unauthorized`, `forbidden`, `not_found`, `unavailable`, and `api_error`. A response that the CLI cannot decode uses `unexpected_response`. All other errors use `internal_error`, for example a missing required flag.

## Select fields with `--fields`

The `--fields` flag (short `-f`) takes a comma-separated projection. In table and plain output, the projection selects the columns. In every format, the CLI adds the projection to the field set that it requests from YouTrack, so you can get fields that the default projection omits.

```Shell
youtrack issue list --query "project: DEMO" --fields "idReadable,summary,reporter.login"
```

Reach into a nested object with a dot path, or select nested fields with YouTrack's parenthesis syntax:

```Shell
youtrack issue view DEMO-42 --fields "idReadable,summary,project.shortName"
youtrack issue list --query "for: me" --fields "idReadable,project(shortName)"
```

On `issue list`, an entry that is not an issue attribute is read as a custom field name, so `--fields "idReadable,Priority,State"` shows those custom fields as columns (see [Find issues with a search query](youtrack-cli-managing-issues.md#find-issues-with-a-search-query)).

In table and plain output, each projected field becomes a column. The CLI does **not** trim JSON output to the projection: it prints the full response, and the projection only widens what the server returns. If you need only a few keys, filter the JSON with a tool like `jq`.

## Plain output feeds grep and awk

The `--plain` flag prints tab-separated rows with no headers, colors, or box drawing. Combine it with `--fields` to feed line-oriented tools:

```Shell
youtrack issue list --query "State: Open" --fields "idReadable,summary" --plain \
  | awk -F '\t' '{ print $1 }'
```

Add `--quiet` to suppress tips, hints, and the startup update notice, all of which are written to stderr:

```Shell
youtrack issue list --query "for: me" --plain --quiet
```

## Pipe JSON to jq without flags

Because a pipe already selects JSON, you can send output straight to `jq`. Extract the IDs of your open issues:

```Shell
youtrack issue list --query "for: me State: Open" | jq -r '.[].idReadable'
```

Capture the ID of a freshly created issue:

```Shell
id=$(youtrack issue create --project DEMO --summary "Nightly build failed" | jq -r '.idReadable')
youtrack issue comment create "$id" --text "Filed automatically by CI"
```

## Exit code 0 means success, 1 means failure

Scripts must branch on the exit code, not on parsed text. YouTrack CLI returns `0` on success and `1` on every failure.

<table style="none">
<tr><td>Exit code</td><td>Meaning</td></tr>
<tr><td>

`0`
</td><td>

The command succeeded.
</td></tr>
<tr><td>

`1`
</td><td>

The command failed: invalid flags, missing credentials, an unknown command, an API error, or an internal crash.
</td></tr>
</table>

Branch on the code rather than scraping output:

```Shell
if youtrack issue view DEMO-42 --quiet >/dev/null; then
  echo "issue exists"
else
  echo "lookup failed"
fi
```

## Call any endpoint with `raw`

The `raw` command sends an authenticated request to any endpoint under `<base URL>/api`. Use it to reach parts of the REST API that do not have a dedicated command. It uses your configured credentials, so you do not handle the token yourself.

Pass the path starting with `/api`. A GET request is the default:

```Shell
youtrack raw /api/issues/DEMO-42 --fields "idReadable,summary,customFields(name,value(name))"
```

The `--fields` flag on `raw` is appended to the URL as a `fields=` query parameter. To write, set `--method` and supply a JSON `--body`:

```Shell
youtrack raw /api/issues/DEMO-42/comments \
  --method POST \
  --body '{"text":"Deployed to staging"}'
```

For long bodies, read the JSON from a file with `--body-file`, or from stdin with `--body-file -`. This also avoids the command-line length limit and quote escaping on Windows:

```Shell
youtrack raw /api/issues/DEMO-42 --method POST --body-file update.json
jq -n --arg d "$(cat notes.md)" '{description: $d}' | youtrack raw /api/issues/DEMO-42 --method POST --body-file -
```

Raw responses are arbitrary JSON, so pass `--json` or pipe the output when you want to parse it.

To keep a response exactly as the server sent it — an image, a file, or JSON you do not want reformatted — pass `--output` with a file path, or `-` for stdout. The body is not parsed as JSON:

```Shell
youtrack raw "/api/files/1-2?sign=..." --output screenshot.png
```

Without an explicit `$top`, YouTrack normally returns 42 items per collection page. When a list response has exactly 42 items and the path sets no `$top`, `raw` prints a warning on stderr.

For collection endpoints, `--paginate` follows `$skip`/`$top` pages and emits
one combined JSON array, or `[]` when the collection is empty. `--page-size` defaults to 100:

```Shell
youtrack raw /api/issues --fields "idReadable,summary" --paginate --page-size 100
```

The `--limit` flag of `issue list`, `article list`, `user list`, `tag list`, and `tag issue list` takes a non-negative value, and `0` returns an empty list. The `issue list --offset` value must also be non-negative. `issue field list-values --limit` defaults to 50 and must be greater than `0`. To fetch every page, use `raw --paginate`.

Other list commands send one request with `$top=500`. If a list command gets exactly 500 items, it writes a warning to stderr, because the server can have more items. The warning does not change stdout or the exit code. `--quiet` suppresses the warning. To fetch every item, use `raw --paginate`.

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

`--method`
</td>
<td>

HTTP method: `GET` (default), `POST`, `PUT`, or `DELETE`.
</td>
</tr>
<tr>
<td>

`--body`
</td>
<td>

JSON request body, sent with `POST` and `PUT`.
</td>
</tr>
<tr>
<td>

`--body-file`
</td>
<td>

Read the JSON request body from a file, or from stdin with `-`. Cannot be combined with `--body`.
</td>
</tr>
<tr>
<td>

`--output`, `-o`
</td>
<td>

Write the response body unchanged to a file, or to stdout with `-`, without parsing it as JSON.
</td>
</tr>
<tr>
<td>

`--fields`
</td>
<td>

Field projection appended to the URL as `fields=`.
</td>
</tr>
<tr>
<td>

`--paginate`
</td>
<td>

Fetch every page of a JSON array response and print one combined array.
</td>
</tr>
<tr>
<td>

`--page-size`
</td>
<td>

Items per request with `--paginate`. The default is 100.
</td>
</tr>
</table>

## Use environment variables and `--quiet` in CI

<warning>

Treat the token like a password. Set it from a CI secret. Do not pass it as a `--token` flag, where it can get into build logs or shell history. Any token grants full account access. See [Security](youtrack-cli-security.md).

</warning>

A CI job has no terminal, so output is JSON without any flag. Provide credentials through the [environment variables](youtrack-cli-authentication.md#setting-environment-variables). Add `--quiet` to keep hints and the update notice out of stderr:

```Shell
export YOUTRACK_BASE_URL=https://example.youtrack.cloud
export YOUTRACK_TOKEN=perm:...

youtrack issue list --query "assignee: me State: Open" --quiet \
  | jq -r '.[].idReadable'
```

To see the underlying HTTP request and response while debugging a script, add `--verbose`. It writes the method, URL, and status to stderr, leaving stdout clean.

To check a write before you send it, add `--dry-run`. Reads still run. The CLI prints the first write to stdout instead of sending it, and exits with code `0` without changing anything. If stdout is not a terminal, the write is printed as JSON (`dryRun`, `method`, `url`, `body`). Commands that write more than once stop at the first write.

For bulk edits that would otherwise email every watcher, add `--mute-notifications`. It adds the parameter to every POST, PUT, and DELETE request and needs the *Apply Commands Silently* permission in every affected project. Without that permission, the write fails with HTTP 403 and nothing changes. If notifications are acceptable, retry without the flag. The flag does not mute workflow notifications.

<seealso>
<category ref="04">
<a href="youtrack-cli-managing-issues.md">Managing Issues</a>
<a href="youtrack-cli-ai-agent-integration.md">AI Agent Integration</a>
<a href="youtrack-cli-authentication.md">Authentication</a>
</category>
<category ref="06">
<a href="youtrack-cli-configuration.md">Configuration</a>
<a href="youtrack-cli-get-started.md">Getting Started</a>
<a href="youtrack-cli-security.md">Security</a>
</category>
</seealso>
