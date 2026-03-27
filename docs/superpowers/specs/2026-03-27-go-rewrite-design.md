# Go Rewrite Design — youtrack-cli

**Date:** 2026-03-27
**Status:** Approved

## Motivation

The current Node.js CLI requires a Node.js runtime. The Go rewrite produces a single self-contained binary with no runtime dependency, enabling cross-platform distribution (macOS, Linux, Windows) and ~15x faster startup time.

## Approach: API-client-first (layered architecture)

Clean separation between HTTP/API logic and CLI presentation, following the JetBrains/teamcity-cli reference:

- `api/` — pure Go SDK: HTTP client, typed structs, domain methods
- `cmd/` — CLI layer: cobra commands, output formatting

## Project Structure

```
go/
├── main.go                  # Entry point: cmd.Execute()
├── go.mod                   # module youtrack-cli
├── go.sum
├── api/
│   ├── client.go            # HTTP client (base URL, token, net/http)
│   ├── types.go             # All request/response structs
│   ├── issues.go            # GetIssue(), AddIssue(), ListIssues(), ...
│   ├── articles.go
│   ├── projects.go
│   ├── users.go
│   ├── groups.go
│   ├── sprints.go
│   ├── boards.go
│   ├── tags.go
│   ├── fields.go
│   ├── bundles.go
│   ├── workflows.go
│   ├── admin.go
│   └── *_test.go            # Unit tests via httptest.NewServer
└── cmd/
    ├── root.go              # Root cobra command, global flags
    ├── env.go               # Load ~/.youtrack-cli-env
    ├── issue.go             # issue subcommand group
    ├── issue_get.go
    ├── issue_add.go
    ├── issue_list.go
    ├── issue_update.go
    ├── issue_delete.go
    ├── issue_comment.go     # issue comment add/list/delete
    ├── issue_link.go
    ├── issue_command.go     # issue command (apply-command equivalent)
    ├── article.go
    ├── project.go
    ├── user.go
    ├── sprint.go
    ├── board.go
    ├── tag.go
    ├── field.go
    ├── bundle.go
    ├── workflow.go
    ├── admin.go
    └── output/
        ├── json.go          # json.MarshalIndent → stdout
        └── table.go         # text/tabwriter (stdlib, no extra deps)
```

## CLI Interface

### Design rules

- **Positional argument** only for entity ID when it is the sole required parameter
- **Named flags** for everything else, using YouTrack API field names verbatim (`--shortName`, `--name`, `--summary`)
- No renaming: what appears in YouTrack docs appears in the CLI

### Command hierarchy

```
youtrack
├── issue
│   ├── get <issueId>
│   ├── list --query "..." [--limit N]
│   ├── add --project <shortName> --summary "..." [--description "..."] [--asUser <login>]
│   ├── update <issueId> [--summary "..."] [--description "..."]
│   ├── delete <issueId>
│   ├── command <issueId> "State Fixed"        # apply-command equivalent
│   ├── link --from <id> --to <id> --type "..."
│   └── comment
│       ├── add <issueId> --text "..." [--asUser <login>]
│       ├── list <issueId>
│       └── delete <issueId> <commentId>
├── article
│   ├── get <articleId>
│   ├── list [--project <shortName>]
│   ├── add --project <shortName> --title "..." [--content "..."] [--asUser <login>]
│   ├── update <articleId> [--title "..."] [--content "..."]
│   ├── delete <articleId>
│   └── comment
│       ├── add <articleId> --text "..."
│       └── delete <articleId> <commentId>
├── project
│   ├── list
│   ├── get <shortName>
│   ├── add --shortName FOO --name "..." --leader <login>
│   └── delete <shortName>
├── user
│   ├── list [--query "..."]
│   ├── get <login>
│   ├── add --login <login> --name "..." --email "..."
│   └── current
├── sprint
│   ├── list --board <name>
│   ├── add --board <name> --name "..."
│   └── update <sprintId> [--name "..."] [--start "..."] [--finish "..."]
├── board
│   ├── list
│   └── get <boardId>
├── tag
│   ├── list
│   ├── add --name "..."
│   └── delete <tagId>
├── field
│   ├── list [--project <shortName>]
│   └── attach --project <shortName> --field <name>
├── bundle
│   ├── list
│   ├── add --name "..." --type <enum|user|...>
│   └── value add --bundle <name> --type <...> --value "..."
├── workflow
│   └── list
└── admin
    ├── settings get
    └── feature-flags list
```

### Global flags

```
--format json|table    JSON by default (for agents), table for humans
--verbose              Print HTTP request/response to stderr
--base-url <url>       Override YOUTRACK_BASE_URL
--token <token>        Override YOUTRACK_TOKEN
```

## API Layer

### `api/client.go`

```go
type Client struct {
    BaseURL    string
    Token      string
    HTTPClient *http.Client
    Verbose    bool
}

func NewClient(baseURL, token string) *Client
func (c *Client) Get(path string, result any) error
func (c *Client) Post(path string, body, result any) error
func (c *Client) Put(path string, body, result any) error
func (c *Client) Delete(path string) error
func (c *Client) Multipart(path, filePath, fileName, mimeType string, result any) error
```

### `api/types.go`

Typed structs matching YouTrack API fields. Field names match JSON tags exactly:

```go
type Issue struct {
    ID          string   `json:"id"`
    IDReadable  string   `json:"idReadable"`
    Summary     string   `json:"summary"`
    Description string   `json:"description,omitempty"`
    Project     *Project `json:"project,omitempty"`
    Reporter    *User    `json:"reporter,omitempty"`
    Resolved    *int64   `json:"resolved,omitempty"`
    Created     int64    `json:"created,omitempty"`
}

type Project struct {
    ID        string `json:"id"`
    ShortName string `json:"shortName"`
    Name      string `json:"name"`
}

type User struct {
    ID    string `json:"id"`
    Login string `json:"login"`
    Name  string `json:"fullName,omitempty"`
}
// ... (one struct per YouTrack entity)
```

### Domain files

Each domain file exposes typed methods. The `fields=` query parameter is hardcoded per method — callers don't manage field selection:

```go
// api/issues.go
func (c *Client) GetIssue(id string) (*Issue, error)
func (c *Client) ListIssues(query string, limit int) ([]Issue, error)
func (c *Client) AddIssue(req AddIssueRequest) (*Issue, error)
func (c *Client) UpdateIssue(id string, req UpdateIssueRequest) (*Issue, error)
func (c *Client) DeleteIssue(id string) error
```

## CMD Layer

Thin wrappers: parse flags → call api method → print result:

```go
// cmd/issue_get.go
var issueGetCmd = &cobra.Command{
    Use:   "get <issueId>",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        issue, err := getClient(cmd).GetIssue(args[0])
        if err != nil { return err }
        return output.Print(cmd, issue)
    },
}
```

`output.Print()` reads `--format` flag:
- `json` (default): `json.MarshalIndent` → stdout
- `table`: `text/tabwriter` with column headers

## Configuration

Same env vars as Node.js version:
```
YOUTRACK_BASE_URL   https://your-instance.youtrack.cloud
YOUTRACK_TOKEN      perm:...
```

Auto-loaded from `~/.youtrack-cli-env` on each invocation (same format as current).

## Dependencies

| Package | Purpose |
|---|---|
| `spf13/cobra` | CLI framework |
| `stretchr/testify` | Test assertions (dev only) |

All HTTP, JSON, and table formatting via Go stdlib (`net/http`, `encoding/json`, `text/tabwriter`).

## Testing

### Unit tests (`api/*_test.go`)
- `httptest.NewServer` — no real YouTrack needed
- Run in CI: `go test ./...`
- Cover: HTTP error handling, JSON parsing, field mapping

### Integration tests (`go/integration_test.go`)
- Real YouTrack instance (`YOUTRACK_BASE_URL` + `YOUTRACK_TOKEN`)
- Same pattern as current `test.js`
- Run with: `cd go && go test -run Integration ./...`
- Clean up all created resources in `TestMain` teardown

## Build and Distribution

```bash
# Local development
cd go && go build -o youtrack-cli .

# Cross-compile
GOOS=linux   GOARCH=amd64 go build -o dist/youtrack-cli-linux-amd64 .
GOOS=darwin  GOARCH=arm64 go build -o dist/youtrack-cli-darwin-arm64 .
GOOS=windows GOARCH=amd64 go build -o dist/youtrack-cli-windows-amd64.exe .
```

Both the Node.js version (root) and Go version (`go/`) coexist in the same repository during the transition period.

## Coexistence with Node.js version

During migration both versions live in the same repo:
- Node.js: `youtrack-cli.js` (root) — current, unchanged
- Go: `go/` — new version being built

Once Go version reaches feature parity, Node.js version is removed.
