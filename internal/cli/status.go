package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"tangled.org/onev.cat/tang/internal/config"
	"tangled.org/onev.cat/tang/internal/repo"
)

type statusResult struct {
	Authentication authStatus     `json:"authentication"`
	Repository     repoStatus     `json:"repository"`
	Services       []serviceCheck `json:"services"`
}

type authStatus struct {
	Authenticated bool      `json:"authenticated"`
	Handle        string    `json:"handle,omitempty"`
	DID           string    `json:"did,omitempty"`
	PDS           string    `json:"pds,omitempty"`
	ExpiresAt     time.Time `json:"expiresAt,omitempty"`
}

type repoStatus struct {
	Detected   bool   `json:"detected"`
	Identifier string `json:"identifier,omitempty"`
	Knot       string `json:"knot,omitempty"`
	Remote     string `json:"remote,omitempty"`
	URLKind    string `json:"urlKind,omitempty"`
}

type serviceCheck struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	Reachable bool   `json:"reachable"`
	Error     string `json:"error,omitempty"`
}

func newStatusCommand(opts *RootOptions) *cobra.Command {
	var section string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show auth, repository, and service status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if section == "auth" {
				return printAuthStatus(cmd, opts)
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			result := collectStatus(cmd.Context(), cfg)
			if rendered, err := renderJSONIfRequested(cmd, opts, result); rendered || err != nil {
				return err
			}
			return printStatus(cmd, result)
		},
	}
	cmd.Flags().StringVar(&section, "section", "", "Limit status output to a section")
	return cmd
}

func collectStatus(ctx context.Context, cfg *config.Config) statusResult {
	session, _ := loadSessionOrNil()
	authOut := authStatus{}
	if session != nil {
		authOut = authStatus{
			Authenticated: true,
			Handle:        session.Handle,
			DID:           session.DID,
			PDS:           session.PDS,
			ExpiresAt:     session.ExpiresAt,
		}
	}

	repoOut := repoStatus{}
	if cwd, err := os.Getwd(); err == nil {
		if context, err := repo.Resolve(ctx, cwd, cfg); err == nil {
			repoOut = repoStatus{
				Detected:   true,
				Identifier: context.Identifier(),
				Knot:       context.Knot,
				Remote:     context.RemoteName,
				URLKind:    string(context.URLKind),
			}
		}
	}

	return statusResult{
		Authentication: authOut,
		Repository:     repoOut,
		Services: []serviceCheck{
			checkService(ctx, "Constellation", cfg.Constellation.URL),
			checkService(ctx, "AppView", cfg.AppView.URL),
		},
	}
}

func printAuthStatus(cmd *cobra.Command, opts *RootOptions) error {
	session, err := loadSessionOrNil()
	if err != nil {
		return err
	}
	out := authStatus{}
	if session != nil {
		out = authStatus{
			Authenticated: true,
			Handle:        session.Handle,
			DID:           session.DID,
			PDS:           session.PDS,
			ExpiresAt:     session.ExpiresAt,
		}
	}
	if rendered, err := renderJSONIfRequested(cmd, opts, out); rendered || err != nil {
		return err
	}
	if !out.Authenticated {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "Authentication\n  (not authenticated)\n  Run: tang auth login")
		return err
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Authentication\n  Logged in as %s (%s)\n  PDS: %s\n  Token: expires %s\n", out.Handle, out.DID, out.PDS, formatExpiry(out.ExpiresAt))
	return err
}

func printStatus(cmd *cobra.Command, result statusResult) error {
	if result.Authentication.Authenticated {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Authentication\n  Logged in as %s (%s)\n  PDS: %s\n  Token: expires %s\n\n", result.Authentication.Handle, result.Authentication.DID, result.Authentication.PDS, formatExpiry(result.Authentication.ExpiresAt))
	} else {
		_, _ = fmt.Fprint(cmd.OutOrStdout(), "Authentication\n  (not authenticated)\n  Run: tang auth login\n\n")
	}
	if result.Repository.Detected {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Repository\n  %s\n  Knot: %s (via remote %q, %s URL)\n\n", result.Repository.Identifier, result.Repository.Knot, result.Repository.Remote, result.Repository.URLKind)
	} else {
		_, _ = fmt.Fprint(cmd.OutOrStdout(), "Repository\n  (not in a Tangled repository)\n  Add a remote whose host is listed in tang config key knot.hosts.\n\n")
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Services")
	for _, service := range result.Services {
		state := "reachable"
		if !service.Reachable {
			state = "unreachable"
		}
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  %s: %s (%s)\n", service.Name, service.URL, state); err != nil {
			return err
		}
	}
	return nil
}

func checkService(ctx context.Context, name, serviceURL string) serviceCheck {
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(checkCtx, http.MethodHead, serviceURL, nil)
	if err != nil {
		return serviceCheck{Name: name, URL: serviceURL, Error: err.Error()}
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return serviceCheck{Name: name, URL: serviceURL, Error: err.Error()}
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	return serviceCheck{Name: name, URL: serviceURL, Reachable: resp.StatusCode < 500}
}

func formatExpiry(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	d := time.Until(t).Round(time.Minute)
	if d < 0 {
		return "expired"
	}
	return "in " + strings.TrimSuffix(d.String(), "0s")
}
