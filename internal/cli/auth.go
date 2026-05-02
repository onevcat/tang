package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	"tangled.org/onev.cat/tang/internal/atproto"
	"tangled.org/onev.cat/tang/internal/auth"
	"tangled.org/onev.cat/tang/internal/tangled"
)

func newAuthCommand(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate with a Tangled PDS",
	}
	cmd.AddCommand(newAuthLoginCommand(opts))
	cmd.AddCommand(newAuthLogoutCommand())
	cmd.AddCommand(newAuthRefreshCommand())
	cmd.AddCommand(newAuthTokenCommand(opts))
	cmd.AddCommand(newAuthStatusCommand(opts))
	return cmd
}

func newAuthLoginCommand(opts *RootOptions) *cobra.Command {
	var handle string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in with a handle and app password",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if handle == "" {
				var err error
				handle, err = promptLine(cmd, "Handle: ")
				if err != nil {
					return err
				}
			}
			pds := strings.TrimRight(opts.PDS, "/")
			if pds == "" {
				ident, err := atproto.ResolveHandle(ctx, handle)
				if err != nil {
					return err
				}
				pds = ident.PDS
			}
			password, err := promptPassword(cmd, "App password: ")
			if err != nil {
				return err
			}
			client := tangled.NewAnonymousPDSClient(pds, http.DefaultClient)
			out, err := client.CreateSession(ctx, handle, password)
			if err != nil {
				return err
			}
			session, err := auth.NewSession(out.Did, out.Handle, pds, out.AccessJwt, out.RefreshJwt)
			if err != nil {
				return err
			}
			if err := auth.Save(session); err != nil {
				return err
			}
			if rendered, err := renderJSONIfRequested(cmd, opts, session); rendered || err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Logged in as %s\n", session.Handle)
			return err
		},
	}
	cmd.Flags().StringVar(&handle, "handle", "", "Handle to authenticate")
	return cmd
}

func newAuthLogoutCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Clear the stored session",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := auth.Clear(); err != nil {
				return err
			}
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "Logged out")
			return err
		},
	}
}

func newAuthRefreshCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh",
		Short: "Refresh the stored session token",
		RunE: func(cmd *cobra.Command, _ []string) error {
			session, err := auth.Load()
			if err != nil {
				return err
			}
			client := tangled.NewPDSClient(session, http.DefaultClient)
			refreshed, err := client.RefreshSession(cmd.Context(), session)
			if err != nil {
				return err
			}
			if err := auth.Save(refreshed); err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "Token refreshed")
			return err
		},
	}
}

func newAuthTokenCommand(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "token",
		Short: "Print the current access token",
		RunE: func(cmd *cobra.Command, _ []string) error {
			session, err := auth.Load()
			if err != nil {
				return err
			}
			if rendered, err := renderJSONIfRequested(cmd, opts, map[string]string{"token": session.AccessJWT}); rendered || err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), session.AccessJWT)
			return err
		},
	}
}

func newAuthStatusCommand(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show authentication status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return printAuthStatus(cmd, opts)
		},
	}
}

func promptLine(cmd *cobra.Command, label string) (string, error) {
	_, _ = fmt.Fprint(cmd.OutOrStdout(), label)
	reader := bufio.NewReader(cmd.InOrStdin())
	line, err := reader.ReadString('\n')
	if err != nil && (!errors.Is(err, io.EOF) || line == "") {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func promptPassword(cmd *cobra.Command, label string) (string, error) {
	_, _ = fmt.Fprint(cmd.OutOrStdout(), label)
	if term.IsTerminal(int(os.Stdin.Fd())) {
		raw, err := term.ReadPassword(int(os.Stdin.Fd()))
		_, _ = fmt.Fprintln(cmd.OutOrStdout())
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(raw)), nil
	}
	return promptLine(cmd, "")
}

func loadSessionOrNil() (*auth.Session, error) {
	session, err := auth.Load()
	if errors.Is(err, auth.ErrNotFound) {
		return nil, nil
	}
	return session, err
}
