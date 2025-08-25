package main

import (
	"context"
	"encoding/json"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/spf13/cobra"
	"io"
	"os"

	// Cosmos SDK
	"github.com/cosmos/cosmos-sdk/client"
	sdkserver "github.com/cosmos/cosmos-sdk/server"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"

	keyscli "github.com/cosmos/cosmos-sdk/client/keys"

	// CometBFT
	tmdb "github.com/cometbft/cometbft-db"
	cmtcfg "github.com/cometbft/cometbft/config"
	tmlog "github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtypes "github.com/cometbft/cometbft/types"

	// Your app
	"doctorium/app"
)

func main() {
	// 1) Encoding (TxConfig 포함)
	enc := app.MakeEncodingConfig()

	// 2) 기본 client.Context (너희 포크는 WithViper(prefix string))
	initClientCtx := client.Context{}.
		WithCodec(enc.Marshaler).
		WithInterfaceRegistry(enc.InterfaceRegistry).
		WithTxConfig(enc.TxConfig).
		WithLegacyAmino(enc.Amino).
		WithInput(os.Stdin).
		WithHomeDir(app.DefaultNodeHome).
		WithViper("DOCTORIUM")

	// rootCmd := &cobra.Command{ ... }
	rootCmd := &cobra.Command{
		Use:   "doctoriumd",
		Short: "Doctorium Network Daemon",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SetOut(cmd.OutOrStdout())

			clientCtx, err := client.ReadPersistentCommandFlags(initClientCtx, cmd.Flags())
			if err != nil {
				return err
			}
			// 순서: setter 먼저 → handler
			if err := client.SetCmdClientContext(cmd, clientCtx); err != nil {
				return err
			}
			if err := client.SetCmdClientContextHandler(clientCtx, cmd); err != nil {
				return err
			}
			// 그 다음에 InterceptConfigsPreRunHandler
			tmCfg := cmtcfg.DefaultConfig()
			tmCfg.RootDir = clientCtx.HomeDir
			return sdkserver.InterceptConfigsPreRunHandler(cmd, "", serverconfig.DefaultConfig(), tmCfg)
		},
	}
	rootCmd.SetContext(context.Background())

	// ValidateGenesis (커스텀 하나만 등록)
	balIter := banktypes.GenesisBalancesIterator{}

	valCmd := genutilcli.ValidateGenesisCmd(app.ModuleBasics)
	valCmd.PreRunE = func(cmd *cobra.Command, _ []string) error {
		clientCtx, err := client.ReadPersistentCommandFlags(initClientCtx, cmd.Flags())
		if err != nil {
			return err
		}
		// ★ 여기서도 둘 다 호출
		if err := client.SetCmdClientContextHandler(clientCtx, cmd); err != nil {
			return err
		}
		if err := client.SetCmdClientContext(cmd, clientCtx); err != nil {
			return err
		}
		return nil
	}

	rootCmd.AddCommand(
		genutilcli.InitCmd(app.ModuleBasics, app.DefaultNodeHome),
		genutilcli.GenTxCmd(app.ModuleBasics, enc.TxConfig, balIter, app.DefaultNodeHome),
		genutilcli.CollectGenTxsCmd(balIter, app.DefaultNodeHome, genutiltypes.DefaultMessageValidator),
		valCmd, // ← 이것만
		genutilcli.AddGenesisAccountCmd(app.DefaultNodeHome),
	)

	// 5) 키/유틸
	rootCmd.AddCommand(
		keyscli.Commands(app.DefaultNodeHome),
		newFixKeyringCmd(), // 별도 파일의 복구 커맨드(중복 정의 금지)
	)

	// 6) Tendermint run/export 커맨드
	sdkserver.AddCommands(
		rootCmd,
		app.DefaultNodeHome,

		// AppCreator
		func(
			logger tmlog.Logger,
			db tmdb.DB,
			trace io.Writer,
			opts servertypes.AppOptions,
		) servertypes.Application {
			if logger == nil {
				logger = tmlog.NewNopLogger()
			}
			if db == nil {
				db = tmdb.NewMemDB()
			}
			return app.NewDoctoriumApp(logger, db, trace, true, opts)
		},

		// AppExporter
		func(
			logger tmlog.Logger,
			db tmdb.DB,
			trace io.Writer,
			height int64,
			forZeroHeight bool,
			_ []string,
			opts servertypes.AppOptions,
			_ []string,
		) (servertypes.ExportedApp, error) {
			raw := app.NewDoctoriumApp(logger, db, trace, false, opts).(*app.App)
			ctx := raw.BaseApp.NewContext(forZeroHeight, tmproto.Header{Height: height})
			state := raw.ModuleManager.ExportGenesis(ctx, enc.Marshaler)

			bz, err := json.MarshalIndent(state, "", "  ")
			if err != nil {
				return servertypes.ExportedApp{}, err
			}
			return servertypes.ExportedApp{
				AppState:        bz,
				Validators:      []tmtypes.GenesisValidator{},
				Height:          height,
				ConsensusParams: &tmproto.ConsensusParams{},
			}, nil
		},

		// addModuleInitFlags (필요 없으면 no-op)
		func(cmd *cobra.Command) {},
	)

	// 7) 실행
	if err := svrcmd.Execute(rootCmd, "DOCTORIUM", app.DefaultNodeHome); err != nil {
		os.Exit(1)
	}
}
