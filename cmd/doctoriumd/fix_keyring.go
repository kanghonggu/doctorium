package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/spf13/cobra"

	"doctorium/app"
)

func newFixKeyringCmd() *cobra.Command {
	var (
		home   string
		force  bool
		backup bool
	)

	cmd := &cobra.Command{
		Use:   "fix-keyring",
		Short: "Detect and repair a corrupted keyring-file (dangerous: may delete local keys)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// 1) 홈 디렉터리 결정
			clientCtx := client.GetClientContextFromCmd(cmd)
			if home == "" {
				if clientCtx.HomeDir != "" {
					home = clientCtx.HomeDir
				} else {
					home = app.DefaultNodeHome
				}
			}
			keyringDir := filepath.Join(home, "keyring-file")

			// 2) file backend keyring 인스턴스 열기
			kr, err := keyring.New("doctorium", keyring.BackendFile, home, cmd.InOrStdin(), clientCtx.Codec)
			if err != nil {
				return fmt.Errorf("open keyring: %w", err)
			}

			// 3) 리스트 시도 → 손상 탐지
			if _, err := kr.List(); err == nil {
				fmt.Fprintf(cmd.OutOrStdout(), "keyring is healthy at %s\n", keyringDir)
				return nil
			} else {
				errStr := strings.ToLower(err.Error())
				isCorrupt := strings.Contains(errStr, "bytes left over") ||
					strings.Contains(errStr, "unmarshalbinarylengthprefixed") ||
					strings.Contains(errStr, "unmarshal") ||
					strings.Contains(errStr, "invalid character") ||
					strings.Contains(errStr, "unexpected eof") ||
					strings.Contains(errStr, "cipher")

				if !isCorrupt {
					return fmt.Errorf("keyring error (not auto-fixable): %w", err)
				}
			}

			// 4) 파괴적 조치 경고/확인
			if !force {
				fmt.Fprintf(cmd.OutOrStdout(),
					"Detected corrupted keyring at %s\nThis will REMOVE the directory. Type 'yes' to continue: ",
					keyringDir,
				)
				reader := bufio.NewReader(cmd.InOrStdin())
				line, _ := reader.ReadString('\n')
				if strings.TrimSpace(strings.ToLower(line)) != "yes" {
					fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
					return nil
				}
			}

			// 5) 백업 옵션
			if backup {
				bak := keyringDir + ".bak-" + time.Now().Format("20060102-150405")
				if _, err := os.Stat(keyringDir); err == nil {
					if err := os.Rename(keyringDir, bak); err != nil {
						return fmt.Errorf("backup keyring: %w", err)
					}
					fmt.Fprintf(cmd.OutOrStdout(), "Backed up corrupted keyring to %s\n", bak)
				}
			} else {
				if err := os.RemoveAll(keyringDir); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("remove corrupted keyring: %w", err)
				}
			}

			// 6) 깨끗한 디렉터리 재생성(0700)
			if err := os.MkdirAll(keyringDir, 0o700); err != nil {
				return fmt.Errorf("recreate keyring dir: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Recreated clean keyring directory at %s\n", keyringDir)
			fmt.Fprintln(cmd.OutOrStdout(), "Done. Re-add your keys with `doctoriumd keys add ...`.")
			return nil
		},
	}

	cmd.Flags().StringVar(&home, "home", app.DefaultNodeHome, "node home directory")
	cmd.Flags().BoolVar(&force, "force", false, "do not prompt for confirmation")
	cmd.Flags().BoolVar(&backup, "backup", true, "backup the corrupted keyring directory before deleting it")

	return cmd
}
