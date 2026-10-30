// Command embedding exercises the public embedding API as a compiled CLI.
package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/cmd"
)

func main() {
	if err := execute(); err != nil {
		os.Exit(1)
	}
}

func execute() error {
	var skills embed.FS
	root := cmd.NewRootCmd(skills)
	root.Use = "embedded-youtrack"
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	root.AddCommand(&cobra.Command{
		Use: "embedded-output", GroupID: "util",
		RunE: func(c *cobra.Command, _ []string) error {
			return cmd.PrintOutput(c, map[string]string{"format": cmd.OutputFormat(c)})
		},
	})
	client := api.NewClient(os.Getenv("EMBED_BASE_URL"), os.Getenv("EMBED_TOKEN"))
	// The command's context must replace a client's earlier lifecycle.
	previousContext, cancelPrevious := context.WithCancel(context.Background())
	cancelPrevious()
	client.WithContext(previousContext)
	client.Fields = "id"
	client.Verbose = true
	client.UserAgent = "embedding-fixture"
	client.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	if os.Getenv("EMBED_MODE") == "timeout" {
		client.HTTPClient.Timeout = time.Nanosecond
	}
	defaultTransport, defaultUserAgent := http.DefaultTransport, api.NewClient("", "").UserAgent
	var traceMu sync.Mutex
	var sentUserAgent string
	// Observe the actual request to the live server without replacing transport.
	traceContext := httptrace.WithClientTrace(context.Background(), &httptrace.ClientTrace{
		WroteHeaderField: func(key string, values []string) {
			if strings.EqualFold(key, "User-Agent") {
				traceMu.Lock()
				sentUserAgent = strings.Join(values, ",")
				traceMu.Unlock()
			}
		},
	})
	ctx := cmd.WithClientProvider(traceContext, func(*cobra.Command) (*api.Client, error) {
		switch os.Getenv("EMBED_MODE") {
		case "fallback":
			return nil, nil
		case "error":
			return nil, errors.New("embedded provider failed")
		default:
			return client, nil
		}
	})
	if os.Getenv("EMBED_MODE") == "canceled" {
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		cancel()
	}
	err := root.ExecuteContext(ctx)
	if client.Fields != "id" || !client.Verbose || client.UserAgent != "embedding-fixture" {
		err = errors.New("provider client was mutated")
	}
	if http.DefaultTransport != defaultTransport || api.NewClient("", "").UserAgent != defaultUserAgent {
		err = errors.New("embedding changed process-wide HTTP defaults")
	}
	traceMu.Lock()
	if err == nil && sentUserAgent != "" {
		if os.Getenv("EMBED_MODE") == "fallback" {
			if !strings.HasPrefix(sentUserAgent, "youtrack-cli/") {
				err = fmt.Errorf("fallback request has unexpected User-Agent %q", sentUserAgent)
			}
		} else if sentUserAgent != client.UserAgent {
			err = fmt.Errorf("provider request has unexpected User-Agent %q", sentUserAgent)
		}
	}
	traceMu.Unlock()
	if err != nil {
		return cmd.RenderError(root, err)
	}
	if os.Getenv("EMBED_MODE") == "isolated" {
		// A separate execution must use its own credentials, not the first
		// execution's provider. The test supplies an invalid fallback token.
		second := cmd.NewRootCmd(skills)
		second.Use = "embedded-youtrack"
		second.SetOut(os.Stdout)
		second.SetErr(os.Stderr)
		second.SetArgs([]string{"user", "current", "--json"})
		if err := second.ExecuteContext(context.Background()); err != nil {
			cmd.RenderError(second, err)
			return err
		}
		return fmt.Errorf("separate execution unexpectedly authenticated")
	}
	return nil
}
