# Embedding the CLI

Go applications can construct the command tree with `cmd.NewRootCmd`, supply an
API client for an execution, and reuse the CLI's output and error formatting.

```go
package main

import (
	"context"
	"embed"
	"os"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/cmd"
)

func main() {
	var skills embed.FS
	root := cmd.NewRootCmd(skills)
	root.Use = "my-youtrack"
	ctx := cmd.WithClientProvider(context.Background(), func(command *cobra.Command) (*api.Client, error) {
		client := api.NewClient(os.Getenv("MY_APP_YOUTRACK_URL"), os.Getenv("MY_APP_YOUTRACK_TOKEN"))
		client.UserAgent = "my-app/1.0"
		// Set client.HTTPClient or its Transport here to use an application's
		// proxy or session without replacing http.DefaultTransport.
		return client, nil
	})
	if err := root.ExecuteContext(ctx); err != nil {
		if err := cmd.RenderError(root, err); err != nil {
			os.Exit(1)
		}
	}
}
```

`WithClientProvider` runs before normal credential resolution. Return `(nil, nil)`
to fall back to CLI flags, environment, and saved instances, or an error to stop.
Authentication and instance-management commands may create their own clients.

The CLI copies the supplied client, applies the execution context, appends
`--fields`, and sets a versioned User-Agent unless the client has a custom one.
Explicit `--verbose` (including `--verbose=false`) overrides provider verbosity;
otherwise it is preserved. Write-control flags apply to the copy. The original
client stays unchanged; its HTTP client and transport remain shared.

For added commands, `cmd.PrintOutput(command, data)` honors format and projection
flags and the configured output writer. `cmd.OutputFormat(command)` returns the
effective format; automatic detection uses the output writer, so buffers default
to JSON.

`cmd.RenderError(command, err)` writes the usual errors and uses the host
command name in canned hints. Check its return value before choosing an exit status:
dry-run previews return nil on success; cancellation returns the original error
and prints only `Canceled.` in non-JSON formats.
