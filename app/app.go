package app

import (
	dbm "github.com/cometbft/cometbft-db"
	log "github.com/cometbft/cometbft/libs/log"
	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authmodule "github.com/cosmos/cosmos-sdk/x/auth"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	"github.com/cosmos/cosmos-sdk/x/bank"
	consensusmodule "github.com/cosmos/cosmos-sdk/x/consensus"
	consensuskeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	consensustypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	genutilmodule "github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"io"
	"path/filepath"

	// Cosmos SDK
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	bankmodule "github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	paramsmodule "github.com/cosmos/cosmos-sdk/x/params"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"google.golang.org/grpc"

	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	// 내 모듈
	filehashmodule "doctorium/x/filehash"
	filehashkeeper "doctorium/x/filehash/keeper"
	filehashtypes "doctorium/x/filehash/types"
)

const (
	AppName = "doctorium"
)

var (
	DefaultNodeHome = ".doctoriumd"

	// 모듈 계정 퍼미션 (auth module account permissions)
	maccPerms = map[string][]string{
		authtypes.FeeCollectorName:     nil,
		filehashtypes.ModuleName:       {authtypes.Minter, authtypes.Burner},
		stakingtypes.NotBondedPoolName: nil, // <–– 추가
		stakingtypes.BondedPoolName:    nil, // <–– 추가
	}
)

var ModuleBasics = module.NewBasicManager(
	auth.AppModuleBasic{},
	bank.AppModuleBasic{},
	staking.AppModuleBasic{},
	genutilmodule.AppModuleBasic{},
	paramsmodule.AppModuleBasic{},
	filehashmodule.AppModuleBasic{},
	consensusmodule.AppModuleBasic{},
	// + custom modules (예: filehash)
)

// EncodingConfig 묶음: 앱에서 사용할 Codec/Amino/InterfaceRegistry/TxConfig
type EncodingConfig struct {
	InterfaceRegistry codectypes.InterfaceRegistry
	Marshaler         codec.Codec
	Amino             *codec.LegacyAmino
	TxConfig          client.TxConfig
}

type App struct {
	*baseapp.BaseApp

	// filehash 모듈 keeper
	FileHashKeeper filehashkeeper.Keeper
	ModuleManager  *module.Manager

	// (원한다면 auth, bank keeper 등도 여기에)
}

func (a *App) RegisterAPIRoutes(server *api.Server, config config.APIConfig) {
	a.RegisterAPIRoutes(server, config)
}

func (a *App) RegisterTxService(context client.Context) {
	a.RegisterTxService(context)
}

func (a *App) RegisterTendermintService(context client.Context) {
	a.RegisterTendermintService(context)
}

func (a *App) RegisterNodeService(context client.Context) {
	a.RegisterNodeService(context)
}

// NewApp 생성자
func NewDoctoriumApp(logger log.Logger, db dbm.DB, traceStore io.Writer, loadLatest bool, opts servertypes.AppOptions) servertypes.Application {
	encodingConfig := MakeEncodingConfig()
	appCodec := encodingConfig.Marshaler

	homeDir, ok := opts.Get(flags.FlagHome).(string)
	if !ok || homeDir == "" {
		panic("app home directory not set")
	}

	var chainID string
	genPath := filepath.Join(homeDir, "config", "genesis.json")
	if doc, err := tmtypes.GenesisDocFromFile(genPath); err == nil {
		chainID = doc.ChainID
	}

	// 1) BaseApp 생성
	bApp := baseapp.NewBaseApp(
		AppName,
		logger,
		db,
		encodingConfig.TxConfig.TxDecoder(),
		baseapp.SetChainID(chainID),
	)
	bApp.SetInterfaceRegistry(encodingConfig.InterfaceRegistry)

	// 2) 스토어 키 정의
	keys := sdk.NewKVStoreKeys(
		authtypes.StoreKey,
		banktypes.StoreKey,
		stakingtypes.StoreKey,
		paramstypes.StoreKey, // ← 파라미터 스토어 키
		filehashtypes.StoreKey,
		consensustypes.StoreKey,
	)

	tkeys := sdk.NewTransientStoreKeys(
		paramstypes.TStoreKey, // ← 파라미터 트랜지언트 스토어 키
	)

	// 3) Params Keeper
	paramsKeeper := paramskeeper.NewKeeper(
		appCodec,
		encodingConfig.Amino,
		keys[paramstypes.StoreKey],
		tkeys[paramstypes.TStoreKey],
	)
	paramsModule := paramsmodule.NewAppModule(paramsKeeper)

	consensusKeeper := consensuskeeper.NewKeeper(
		encodingConfig.Marshaler,
		keys[consensustypes.StoreKey],
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// 8) BaseApp에 파라미터 저장소(ConsensusParams)로 등록
	bApp.SetParamStore(&consensusKeeper)

	bApp.MountKVStores(keys)
	bApp.MountTransientStores(tkeys)

	if loadLatest {
		if err := bApp.LoadLatestVersion(); err != nil {
			panic(err)
		}
	}

	// 4) Auth Keeper
	accountKeeper := authkeeper.NewAccountKeeper(
		appCodec,                 // 1) codec
		keys[authtypes.StoreKey], // 2) KVStoreService
		func() authtypes.AccountI {
			return &authtypes.BaseAccount{}
		}, // 3) 계정 생성 함수
		maccPerms,               // 4) 모듈 계정 권한
		sdk.Bech32PrefixAccAddr, // 5) Bech32 계정 주소 접두사
		authtypes.ModuleName,    // 6) 권한자(authority)
	)

	blockedAddrs := make(map[string]bool)
	// 5) Bank Keeper
	bankKeeper := bankkeeper.NewBaseKeeper(
		appCodec,                 // codec
		keys[banktypes.StoreKey], // store key
		accountKeeper,            // auth keeper
		blockedAddrs,             // blocked module accounts
		authtypes.NewModuleAddress(filehashtypes.ModuleName).String(), // authority
	)

	stakingkeeper := stakingkeeper.NewKeeper(
		appCodec,                    // codec
		keys[stakingtypes.StoreKey], // store key
		accountKeeper,               // auth keeper
		nil,                         // blocked module accounts
		authtypes.NewModuleAddress(filehashtypes.ModuleName).String(), // authority
	)

	// 6) FileHash Keeper
	fileHashKeeper := filehashkeeper.NewKeeper(
		appCodec,
		keys[filehashtypes.StoreKey],
		bankKeeper,
	)

	// 7) ModuleManager 설정
	mm := module.NewManager(
		// x/auth 모듈
		authmodule.NewAppModule(
			appCodec,                       // 1) codec.Codec
			accountKeeper,                  // 2) keeper.AccountKeeper
			authsims.RandomGenesisAccounts, // 3) RandomGenesisAccountsFn
			paramsKeeper.Subspace(authtypes.ModuleName), // 4) Subspace
		),

		// x/bank 모듈 (예시)
		bankmodule.NewAppModule(
			appCodec,
			bankKeeper,
			accountKeeper,
			paramsKeeper.Subspace(banktypes.ModuleName),
		),

		staking.NewAppModule(
			appCodec,
			stakingkeeper,
			accountKeeper,
			bankKeeper, // staking 모듈은 bankKeeper, accountKeeper 필요
			paramsKeeper.Subspace(stakingtypes.ModuleName),
		),
		consensusmodule.NewAppModule(appCodec, consensusKeeper),
		paramsModule,

		genutilmodule.NewAppModule(accountKeeper, stakingkeeper, bApp.DeliverTx, encodingConfig.TxConfig),

		filehashmodule.NewAppModule(fileHashKeeper),
	)

	mm.SetOrderBeginBlockers(
		authtypes.ModuleName,
		banktypes.ModuleName,
		stakingtypes.ModuleName,
		genutiltypes.ModuleName,
		consensustypes.ModuleName,
		paramstypes.ModuleName,
		filehashtypes.ModuleName,
	)
	mm.SetOrderEndBlockers(
		authtypes.ModuleName,
		banktypes.ModuleName,
		stakingtypes.ModuleName,
		genutiltypes.ModuleName,
		consensustypes.ModuleName,
		paramstypes.ModuleName,
		filehashtypes.ModuleName,
	)
	mm.SetOrderInitGenesis(
		authtypes.ModuleName,
		banktypes.ModuleName,
		stakingtypes.ModuleName,
		genutiltypes.ModuleName,
		consensustypes.ModuleName,
		paramstypes.ModuleName,
		filehashtypes.ModuleName,
	)

	// 10) BaseApp 반환
	return &App{
		BaseApp:        bApp,
		FileHashKeeper: fileHashKeeper,
		ModuleManager:  mm,
	}

}
func (a *App) RegisterGRPCServices(grpcSrv *grpc.Server) {
	// Msg 서비스 등록
	filehashtypes.RegisterMsgServer(grpcSrv, a.FileHashKeeper)
	// Query 서비스 등록
	filehashtypes.RegisterQueryServer(grpcSrv, a.FileHashKeeper)
}
