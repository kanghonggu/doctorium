package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/spf13/cobra"
)

// newFixKeyringCmd returns a command that detects a corrupted keyring-file and
// recreates it. This is useful when operations like `keys add` or `keys show`
// fail with "Bytes left over in UnmarshalBinaryLengthPrefixed" errors due to
// malformed data.
func newFixKeyringCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fix-keyring",
		Short: "Detect and remove a corrupted keyring-file",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			kr, err := keyring.New("doctorium", keyring.BackendFile, clientCtx.HomeDir, cmd.InOrStdin(), clientCtx.Codec)
			if err != nil {
				return err
			}
			if _, err := kr.List(); err != nil {
				if !strings.Contains(err.Error(), "Bytes left over") &&
					!strings.Contains(err.Error(), "UnmarshalBinaryLengthPrefixed") &&
					!strings.Contains(strings.ToLower(err.Error()), "unmarshal") {
					return err
				}
				keyringDir := filepath.Join(clientCtx.HomeDir, "keyring-file")
				if rmErr := os.RemoveAll(keyringDir); rmErr != nil {
					return rmErr
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Removed corrupted keyring at %s\n", keyringDir)
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), "keyring is healthy")
			return nil
		},
	}
	return cmd
}
