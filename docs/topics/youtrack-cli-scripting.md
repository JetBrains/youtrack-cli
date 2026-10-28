[//]: # (title: Scripting and Automation)

<show-structure for="chapter" depth="2"/>

YouTrack CLI is built to drop into scripts, pipelines, and agents. This page covers its machine-readable output modes, field projections, exit codes, and the `raw` command for calling any REST endpoint.

## Machine-readable output {id="json-output"}

Every command detects whether its output is going to a terminal. When stdout is not a terminal — a pipe, a redirect, or a CI job — YouTrack CLI switches to JSON automatically. You do not have to pass a flag.

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

> Successful data is written to stdout. In JSON mode, a failure is emitted as a structured `{"error":{...}}` object on stderr and the process exits non-zero. This keeps stdout safe for `jq` and prevents error text from contaminating a data pipeline.
>
{style="note"}

An error object looks like this:

```json
{
  "error": {
    "code": "not_found",
    "message": "issue DEMO-999 not found",
    "suggestion": "Resource not found — verify the ID"
  }
}
```

## Selecting fields with `--fields` {id="fields"}

The `--fields` flag (short `-f`) takes a comma-separated projection. In table and plain output it selects which columns are shown; for every format it is merged into the field set the CLI requests from YouTrack, so you can pull fields the default projection omits.

```Shell
youtrack issue list --query "project: DEMO" --fields "idReadable,summary,reporter.login"
```

Reach into a nested object with a dot path, or select nested fields with YouTrack's parenthesis syntax:

```Shell
youtrack issue view DEMO-42 --fields "idReadable,summary,project.shortName"
youtrack issue list --query "for: me" --fields "idReadable,project(shortName)"
```

On `issue list`, an entry that is not an issue attribute is read as a custom field name, so `--fields "idReadable,Priority,State"` shows those custom fields as columns (see [Listing and searching issues](youtrack-cli-managing-issues.md#list)).

In table and plain output, each projected field becomes a column. JSON output is **not** trimmed to the projection — the full response is printed (the projection only widens what the server returns), so narrow JSON with a tool like `jq` when you need just a few keys.

## Plain output for grep and awk {id="plain"}

The `--plain` flag prints tab-separated rows with no headers, colors, or box drawing. Combine it with `--fields` to feed line-oriented tools:

```Shell
youtrack issue list --query "State: Open" --fields "idReadable,summary" --plain \
  | awk -F '\t' '{ print $1 }'
```

Add `--quiet` to suppress tips, hints, and the startup update notice, all of which are written to stderr:

```Shell
youtrack issue list --query "for: me" --plain --quiet
```

## Piping to jq {id="jq"}

Because a pipe already selects JSON, you can send output straight to `jq`. Extract the IDs of your open issues:

```Shell
youtrack issue list --query "for: me State: Open" | jq -r '.[].idReadable'
```

Capture the ID of a freshly created issue:

```Shell
id=$(youtrack issue create --project DEMO --summary "Nightly build failed" | jq -r '.idReadable')
youtrack issue comment create "$id" --text "Filed automatically by CI"
```

## Exit codes {id="exit-codes"}

Scripts should branch on the exit code, not on parsing text. YouTrack CLI returns `0` on success and a non-zero code on failure.

| Exit code | Meaning |
|---|---|
| `0` | The command succeeded. |
| non-zero | The command failed — invalid flags, missing credentials, an unknown command, or an API error. |

Branch on the code rather than scraping output:

```Shell
if youtrack issue view DEMO-42 --quiet >/dev/null; then
  echo "issue exists"
else
  echo "lookup failed"
fi
```

## Making raw API requests {id="raw"}

The `raw` command sends an authenticated request to any endpoint under `<base URL>/api`. Use it to reach parts of the REST API that do not have a dedicated command. It reuses your configured credentials, so you never handle the token yourself.

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

All list commands with `--limit` (`issue list`, `article list`, `user list`, `tag list`, and `tag issue list`) require non-negative values; `0` returns an empty list. Use `raw --paginate` to fetch every page. The `issue list --offset` value must also be non-negative.

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
</table>

## Using YouTrack CLI in CI {id="ci"}

In a CI job there is no terminal, so output is JSON without any flag. Provide credentials through the [environment variables](youtrack-cli-authentication.md) and add `--quiet` to keep stderr free of hints and the update notice:

```Shell
export YOUTRACK_BASE_URL=https://example.youtrack.cloud
export YOUTRACK_TOKEN=perm:...

youtrack issue list --query "assignee: me State: Open" --quiet \
  | jq -r '.[].idReadable'
```

> Treat the token like a password. Set it from a CI secret and never pass it as a `--token` flag, where it can land in build logs or shell history.
>
{style="warning"}

To see the underlying HTTP request and response while debugging a script, add `--verbose`. It writes the method, URL, and status to stderr, leaving stdout clean.

<seealso>
<category ref="user-guide">
<a href="youtrack-cli-managing-issues.md">Managing Issues</a>
<a href="youtrack-cli-ai-agent-integration.md">AI Agent Integration</a>
<a href="youtrack-cli-authentication.md">Authentication</a>
</category>
<category ref="reference">
<a href="youtrack-cli-configuration.md">Configuration</a>
<a href="youtrack-cli-get-started.md">Getting Started</a>
</category>
</seealso>
