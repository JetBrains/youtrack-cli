package internal

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
)

type clientProviderKey struct{}

// WithClientProvider scopes an optional client source to command execution.
func WithClientProvider(ctx context.Context, provider func(*cobra.Command) (*api.Client, error)) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, clientProviderKey{}, provider)
}

func providedClient(cmd *cobra.Command) (*api.Client, error) {
	ctx := cmd.Context()
	if ctx == nil {
		return nil, nil
	}
	provider, _ := ctx.Value(clientProviderKey{}).(func(*cobra.Command) (*api.Client, error))
	if provider == nil {
		return nil, nil
	}
	return provider(cmd)
}

func configureClient(cmd *cobra.Command, source *api.Client) *api.Client {
	// Keep the provider's client reusable across commands and execution contexts.
	c := *source
	c.WithContext(cmd.Context())
	flags := cmd.Root().PersistentFlags()
	c.Verbose, _ = flags.GetBool("verbose")
	fields, _ := flags.GetString("fields")
	if fields = expandFields(fields); fields != "" {
		if c.Fields != "" {
			c.Fields += ","
		}
		c.Fields += fields
	}
	if c.UserAgent == "" || c.UserAgent == "youtrack-cli" {
		c.UserAgent = "youtrack-cli/" + Version
	}
	return &c
}
