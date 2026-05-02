package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	core "tangled.org/core/api/tangled"
	"tangled.org/onev.cat/tang/internal/auth"
	"tangled.org/onev.cat/tang/internal/tangled"
)

type sshKeyRow struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Key       string `json:"key"`
	CreatedAt string `json:"createdAt"`
	URI       string `json:"uri"`
}

func newSSHKeyCommand(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssh-key",
		Short: "Manage Tangled SSH keys",
	}
	cmd.AddCommand(newSSHKeyAddCommand(opts))
	cmd.AddCommand(newSSHKeyListCommand(opts))
	cmd.AddCommand(newSSHKeyDeleteCommand())
	return cmd
}

func newSSHKeyAddCommand(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "add <path>",
		Short: "Add an SSH public key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			session, err := auth.Load()
			if err != nil {
				return err
			}
			data, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			record := &core.PublicKey{
				LexiconTypeID: core.PublicKeyNSID,
				Name:          filepath.Base(args[0]),
				Key:           strings.TrimSpace(string(data)),
				CreatedAt:     time.Now().UTC().Format(time.RFC3339),
			}
			client := tangled.NewPDSClient(session, nil)
			out, err := client.CreateRecord(cmd.Context(), session.DID, core.PublicKeyNSID, record, nil)
			if err != nil {
				return err
			}
			row := sshKeyRow{ID: rkeyFromURI(out.Uri), Name: record.Name, Key: record.Key, CreatedAt: record.CreatedAt, URI: out.Uri}
			if rendered, err := renderJSONIfRequested(cmd, opts, row); rendered || err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Added SSH key %s\n", row.ID)
			return err
		},
	}
}

func newSSHKeyListCommand(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List SSH public keys",
		RunE: func(cmd *cobra.Command, _ []string) error {
			session, err := auth.Load()
			if err != nil {
				return err
			}
			client := tangled.NewPDSClient(session, nil)
			out, err := client.ListRecords(cmd.Context(), session.DID, core.PublicKeyNSID, 100, "")
			if err != nil {
				return err
			}
			rows := make([]sshKeyRow, 0, len(out.Records))
			for _, record := range out.Records {
				key, ok := record.Value.Val.(*core.PublicKey)
				if !ok {
					continue
				}
				rows = append(rows, sshKeyRow{
					ID:        rkeyFromURI(record.Uri),
					Name:      key.Name,
					Key:       key.Key,
					CreatedAt: key.CreatedAt,
					URI:       record.Uri,
				})
			}
			if rendered, err := renderJSONIfRequested(cmd, opts, rows); rendered || err != nil {
				return err
			}
			for _, row := range rows {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", row.ID, row.Name, row.CreatedAt); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func newSSHKeyDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an SSH public key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			session, err := auth.Load()
			if err != nil {
				return err
			}
			client := tangled.NewPDSClient(session, nil)
			if err := client.DeleteRecord(cmd.Context(), session.DID, core.PublicKeyNSID, args[0]); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Deleted SSH key %s\n", args[0])
			return err
		},
	}
}

func rkeyFromURI(uri string) string {
	parts := strings.Split(uri, "/")
	if len(parts) == 0 {
		return uri
	}
	return parts[len(parts)-1]
}
