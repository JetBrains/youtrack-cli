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
		err = cmd.RenderError(root, err)
		os.Exit(1)
	}
}
```

`WithClientProvider` scopes the provider to the supplied execution context. It
runs before the normal flags, environment, and saved-instance credential
resolution. Returning `(nil, nil)` uses that normal resolution; returning an
error stops the command. The provider decides when to honor application sessions
or defer to explicitly supplied CLI credentials.

The CLI shallow-copies a supplied client before applying the command context and
`--verbose`. It appends `--fields` to the client's existing additional `Fields`
and sets a versioned CLI User-Agent unless the client already has a custom one.
The provider's original client is unchanged. Its `HTTPClient` and transport are
shared, preserving their timeout, redirect, and credential-handling policies.
Configure those policies for the application's authentication model.

This provider applies to commands using the normal client resolver. Authentication
flows and saved-instance management can create clients for specific instances;
they keep their existing behavior. An embedding application can remove or replace
those commands when it manages authentication itself.

`api.NewClient` copies the deprecated package-level `api.UserAgent` default when
constructing a client. Prefer `client.UserAgent` for independent clients; changing
one client does not alter other clients or the process-wide HTTP transport.

For additional commands, use `cmd.PrintOutput(command, data)` to honor `--json`,
`--plain`, `--fields`, automatic format selection, and `command.OutOrStdout()`.
Use `cmd.OutputFormat(command)` to inspect the selected format and
`cmd.RenderError(command, err)` to write the standard error envelope or terminal
message to `command.ErrOrStderr()`. The error wrapper preserves JSON error codes,
suggestions, and `--quiet` behavior, using the root command's name in canned hints.

Client setup copies the supplied client before applying command options. When
write controls are available, the same setup applies their flags to that copy;
the host can pass preview errors to `RenderError` and receives nil after a
successful preview. The provider's original client remains unchanged.
