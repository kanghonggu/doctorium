// cmd/doctoriumd/main.go
package main

import (
	"context"
	"encoding/json"
	"io"
	"os"

	"github.com/spf13/cobra"

	// Cosmos SDK
	"github.com/cosmos/cosmos-sdk/client"
	sdkserver "github.com/cosmos/cosmos-sdk/server"     // ← 반드시 여기
	servercmd "github.com/cosmos/cosmos-sdk/server/cmd" // Execute 용
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	"github.com/cosmos/cosmos-sdk/server/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"

	// CometBFT
	tmdb "github.com/cometbft/cometbft-db"
	cmtcfg "github.com/cometbft/cometbft/config"
	tmLog "github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtypes "github.com/cometbft/cometbft/types"

	keyscli "github.com/cosmos/cosmos-sdk/client/keys"
	// your app
	"doctorium/app"
)

func main() {
	// 1) 한 번만 생성할 인코딩 설정
	encCfg := app.MakeEncodingConfig()

	// 2) client.Context 준비

	initClientCtx := client.Context{}.
		WithCodec(encCfg.Marshaler).
		WithInterfaceRegistry(encCfg.InterfaceRegistry).
		WithTxConfig(encCfg.TxConfig).
		WithLegacyAmino(encCfg.Amino).
		WithInput(os.Stdin).
		WithHomeDir(os.ExpandEnv("$HOME/" + app.DefaultNodeHome)).
		WithViper("DOCTORIUM")

	// 3) rootCmd 정의 (PersistentPreRunE에서 설정 생성)

	rootCmd := &cobra.Command{
		Use:   "doctoriumd",
		Short: "Doctorium Network Daemon",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SetOut(cmd.ErrOrStderr())
			clientCtx, err := client.ReadPersistentCommandFlags(initClientCtx, cmd.Flags())
			if err != nil {
				return err
			}
			if err := client.SetCmdClientContext(cmd, clientCtx); err != nil {
				return err
			}
			tmCfg := cmtcfg.DefaultConfig()
			tmCfg.RootDir = clientCtx.HomeDir
			if err := sdkserver.InterceptConfigsPreRunHandler(
				cmd,
				"",                           // custom app.toml 템플릿 없으면 빈 문자열
				serverconfig.DefaultConfig(), // *serverconfig.Config (포인터)
				tmCfg,                        // *cmtcfg.Config      (포인터)
			); err != nil {
				return err
			}
			return nil
		},
	}
	rootCmd.SetContext(context.Background())

	// 4) genesis 계열 서브커맨드 등록

	balIter := banktypes.GenesisBalancesIterator{}
	rootCmd.AddCommand(

		//authcli.AddGenesisAccountCmd(app.DefaultNodeHome),
		genutilcli.InitCmd(app.ModuleBasics, app.DefaultNodeHome),
		genutilcli.GenTxCmd(app.ModuleBasics, encCfg.TxConfig, balIter, app.DefaultNodeHome),
		genutilcli.CollectGenTxsCmd(balIter, app.DefaultNodeHome, genutiltypes.DefaultMessageValidator),
		genutilcli.ValidateGenesisCmd(app.ModuleBasics),

		genutilcli.AddGenesisAccountCmd(
			app.DefaultNodeHome,
		),
	)

	rootCmd.AddCommand(
		keyscli.Commands(app.DefaultNodeHome),
		newFixKeyringCmd(),
	)

	// 5) tendermint init, start, unsafe-reset-all 등 노드 실행 커맨드 등록
	sdkserver.AddCommands(
		rootCmd,
		app.DefaultNodeHome,

		// AppCreator
		func(
			logger tmLog.Logger,
			db tmdb.DB,
			traceStore io.Writer,
			opts types.AppOptions,
		) types.Application {
			if logger == nil {
				logger = tmLog.NewNopLogger()
			}
			if db == nil {
				db = tmdb.NewMemDB()
			}

			return app.NewDoctoriumApp(logger, db, traceStore, true, opts)
		},

		// AppExporter: 직접 ExportGenesis → JSON 직렬화
		func(
			logger tmLog.Logger,
			db tmdb.DB,
			traceStore io.Writer,
			height int64,
			forZeroHeight bool,
			_ []string,
			opts types.AppOptions,
			_ []string,
		) (types.ExportedApp, error) {
			// loadLatest=false
			rawApp := app.NewDoctoriumApp(logger, db, traceStore, false, opts).(*app.App)
			ctx := rawApp.BaseApp.NewContext(forZeroHeight, tmproto.Header{Height: height})
			state := rawApp.ModuleManager.ExportGenesis(ctx, encCfg.Marshaler)
			bz, err := json.MarshalIndent(state, "", "  ")
			if err != nil {
				return types.ExportedApp{}, err
			}
			return types.ExportedApp{
				AppState:        bz,
				Validators:      []tmtypes.GenesisValidator{},
				Height:          height,
				ConsensusParams: &tmproto.ConsensusParams{},
			}, nil
		},

		// no-op for module init flags

		func(cmd *cobra.Command) {},
	)
	// 6) Execute: servercmd.Execute 로 Cobra+SDK wrapper 함께 실행
	if err := servercmd.Execute(rootCmd, "DOCTORIUM", app.DefaultNodeHome); err != nil {
		os.Exit(1)
	}

}
