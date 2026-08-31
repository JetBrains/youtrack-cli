package auth

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/config"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

// NewRefreshTokenCmd constructs the "refresh-token" command. Exported so
// cmd/instance can register it as "update-token" without duplicating logic.
func NewRefreshTokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refresh-token [<alias>]",
		Short: "Replace the saved token for an instance",
		Long: `Replaces the saved token for a configured instance using any available
authentication method:

  1) Basic auth   — login + password via Hub API
  2) Web browser  — open the token page in a browser, then paste the token
  3) Paste token  — enter a token directly without opening a browser

The instance URL is read from the config — no URL prompt. On success the
config entry is updated in-place (alias and URL are preserved).

If <alias> is omitted, the active instance is used.

Use --json for machine-readable output.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			alias := ""
			if len(args) > 0 {
				alias = args[0]
			} else {
				alias = cfg.CurrentInstance
			}
			if alias == "" {
				return fmt.Errorf("no active instance")
			}
			inst, ok := cfg.Instances[alias]
			if !ok {
				return fmt.Errorf("instance %q not found — run: youtrack instance list", alias)
			}

			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Refreshing token for %q (%s)...\n", alias, inst.URL)

			reader := bufio.NewReader(os.Stdin)

			_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
				"\nSelect authentication method:\n  1) Basic auth (login + password)\n  2) Web browser (open token page and paste)\n  3) Paste token directly\nSelect [1-3]: ")
			choiceRaw, err := internal.ReadLineContext(cmd.Context(), reader)
			if err != nil {
				return fmt.Errorf("reading auth type: %w", err)
			}

			var token string
			var hubToken *api.HubToken
			switch strings.TrimSpace(choiceRaw) {
			case "1":
				hubToken, err = acquireTokenViaBasicAuth(cmd, reader, inst.URL)
				if err != nil {
					return err
				}
				token = hubToken.Token
			case "2":
				token, err = acquireTokenViaWeb(cmd, reader, inst.URL)
				if err != nil {
					return err
				}
				hubToken = &api.HubToken{Token: token}
			case "3":
				token, err = acquireTokenViaPaste(cmd, reader, inst.URL)
				if err != nil {
					return err
				}
				hubToken = &api.HubToken{Token: token}
			default:
				return fmt.Errorf("invalid choice %q: enter 1, 2, or 3", strings.TrimSpace(choiceRaw))
			}

			// Upsert in-place: same alias + same URL = token update, no alias collision.
			cfg.AddInstance(alias, inst.URL, token)
			if err := cfg.Save(); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}
			path, _ := config.ConfigPath()
			if !cfg.TokenInKeyring(alias) {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: OS keyring unavailable — token stored in plain text at %s\n", path)
			}
			res := getTokenResult{
				HubToken:   hubToken,
				ConfigFile: path,
				Hint:       fmt.Sprintf("Token updated for %q.", alias),
			}
			if !jsonOutput(cmd) {
				printTokenResult(cmd, res)
				return nil
			}
			return output.Print(cmd, res)
		},
	}
	cmd.Flags().String("login", "", "Hub login (basic auth; prompted interactively if omitted)")
	cmd.Flags().String("password", "", "Password (basic auth; prompted interactively if omitted — avoid in scripts)")
	cmd.Flags().String("token-name", "cli-token", "Name for the permanent token (basic auth only)")
	cmd.Flags().String("scope", "", "Comma-separated Hub service names (basic auth only; default: YouTrack + YouTrack Administration)")
	return cmd
}

// acquireTokenViaBasicAuth authenticates with Hub login/password and creates a
// new permanent token. Returns the full HubToken (includes ID for potential cleanup).
func acquireTokenViaBasicAuth(cmd *cobra.Command, reader *bufio.Reader, baseURL string) (*api.HubToken, error) {
	if flagChanged(cmd, "verbose") {
		api.SetHubVerbose(cmd.ErrOrStderr())
		defer api.SetHubVerbose(nil)
	}

	login, _ := cmd.Flags().GetString("login")
	password, _ := cmd.Flags().GetString("password")
	tokenName, _ := cmd.Flags().GetString("token-name")
	scope, _ := cmd.Flags().GetString("scope")

	if login == "" {
		if _, err := fmt.Fprint(cmd.ErrOrStderr(), "Login: "); err != nil {
			return nil, err
		}
		raw, err := internal.ReadLineContext(cmd.Context(), reader)
		if err != nil {
			return nil, fmt.Errorf("reading login: %w", err)
		}
		login = strings.TrimSpace(raw)
	}

	if password == "" {
		if _, err := fmt.Fprint(cmd.ErrOrStderr(), "Password: "); err != nil {
			return nil, err
		}
		raw, err := internal.ReadPasswordContext(cmd.Context(), int(os.Stdin.Fd()))
		if _, printErr := fmt.Fprintln(cmd.ErrOrStderr()); printErr != nil {
			return nil, printErr
		}
		if err != nil {
			return nil, fmt.Errorf("reading password: %w", err)
		}
		password = string(raw)
	}

	me, err := api.GetHubMeContext(cmd.Context(), baseURL, login, password)
	if err != nil {
		if strings.Contains(err.Error(), "basic auth is not available") {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "\nBasic auth is not available for this instance.\nFalling back to direct token input.\n\n")
			token, err := acquireTokenViaPaste(cmd, reader, baseURL)
			if err != nil {
				return nil, err
			}
			return &api.HubToken{Token: token}, nil
		}
		return nil, err
	}

	services, err := api.ListHubServicesContext(cmd.Context(), baseURL, login, password)
	if err != nil {
		return nil, fmt.Errorf("listing Hub services: %w", err)
	}

	scopeIDs, err := resolveScope(scope, services)
	if err != nil {
		return nil, err
	}

	tok, err := api.CreateHubTokenContext(cmd.Context(), baseURL, login, password, me.ID, tokenName, scopeIDs)
	if err != nil {
		return nil, fmt.Errorf("creating token: %w", err)
	}
	return tok, nil
}

// acquireTokenViaWeb opens the Hub token management page in a browser, then
// prompts the user to paste the token they created. Returns the verified token.
func acquireTokenViaWeb(cmd *cobra.Command, reader *bufio.Reader, baseURL string) (string, error) {
	if flagChanged(cmd, "verbose") {
		api.SetHubVerbose(cmd.ErrOrStderr())
		defer api.SetHubVerbose(nil)
	}

	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Detecting Hub configuration for %s...\n", baseURL)
	hubURL, isInternal, err := api.DetectHubURLContext(cmd.Context(), baseURL)
	if err != nil {
		return "", fmt.Errorf("detecting Hub: %w", err)
	}

	var tokenPageURL string
	if isInternal {
		tokenPageURL = baseURL + "/users/me?tab=account-security"
	} else {
		tokenPageURL = hubURL + "/users/me?tab=authentication"
	}

	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Token page:\n  %s\n\n", hyperlink(tokenPageURL))
	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Steps:")
	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "  1. Log in if prompted")
	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "  2. Click 'New token' (or 'Generate new token')")
	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "  3. Copy the token value and return here")

	if term.IsTerminal(int(os.Stdin.Fd())) {
		if err := openBrowser(tokenPageURL); err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  (Could not open browser: %v)\n", err)
		}
	}

	return promptAndVerifyToken(cmd, reader, baseURL)
}

// acquireTokenViaPaste shows the token-management URL and prompts the user to
// paste a token directly. Returns the verified token.
func acquireTokenViaPaste(cmd *cobra.Command, reader *bufio.Reader, baseURL string) (string, error) {
	tokenPageURL := baseURL + "/users/me?tab=account-security"
	if hubURL, isInternal, err := api.DetectHubURLContext(cmd.Context(), baseURL); err == nil && !isInternal {
		tokenPageURL = hubURL + "/users/me?tab=authentication"
	}
	if err := cmd.Context().Err(); err != nil {
		return "", err
	}
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Create a permanent token at:\n  %s\n\n", hyperlink(tokenPageURL))
	return promptAndVerifyToken(cmd, reader, baseURL)
}

// promptAndVerifyToken prompts for a token paste, verifies it against the API,
// and returns the verified token string.
func promptAndVerifyToken(cmd *cobra.Command, reader *bufio.Reader, baseURL string) (string, error) {
	if _, err := fmt.Fprint(cmd.ErrOrStderr(), "Paste your token: "); err != nil {
		return "", err
	}
	tokenRaw, err := internal.ReadLineContext(cmd.Context(), reader)
	if err != nil {
		return "", fmt.Errorf("reading token: %w", err)
	}
	token := strings.TrimSpace(tokenRaw)
	if token == "" {
		return "", fmt.Errorf("token cannot be empty")
	}

	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Verifying token...")
	verifyClient := api.NewClient(baseURL, token).WithContext(cmd.Context())
	user, err := verifyClient.GetCurrentUser()
	if err != nil {
		return "", fmt.Errorf("token verification failed: %w\nRe-run and paste the token again", err)
	}
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Authenticated as %s (%s)\n", user.FullName, user.Login)
	return token, nil
}
