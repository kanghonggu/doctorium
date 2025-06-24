package filehash

import (
	"context"
	"encoding/json"

	runtime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	abci "cosmossdk.io/api/tendermint/abci"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	keeper "doctorium/x/filehash/keeper"
	types "doctorium/x/filehash/types"
)

var (
	_ module.AppModuleBasic = AppModuleBasic{}
	_ module.AppModule      = AppModule{}

	ModuleName   = types.ModuleName
	RouterKey    = types.RouterKey
	QuerierRoute = types.QuerierRoute
)

// AppModuleBasic defines the basic application module used by the filehash module.
type AppModuleBasic struct{}

// Name returns the filehash module's name.
func (AppModuleBasic) Name() string {
	return ModuleName
}

// RegisterLegacyAminoCodec registers the module's types for the legacy Amino codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)
}

// RegisterInterfaces registers protobuf interfaces for the module.
func (AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(registry)
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the filehash module.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(
	clientCtx client.Context,
	mux *runtime.ServeMux, // v1 패키지 기준 ServeMux
) {
	types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx))
	types.RegisterMsgHandlerClient(context.Background(), mux, types.NewMsgClient(clientCtx))
}

// GetTxCmd returns the root tx command for the filehash module.
func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return nil
}

// GetQueryCmd returns the root query command for the filehash module.
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return nil
}

// DefaultGenesis returns initial genesis state as raw JSON for the filehash module.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	gs := &types.GenesisState{Files: []*types.FileData{}}
	return cdc.MustMarshalJSON(gs)
}

// ValidateGenesis performs genesis state validation.
func (AppModuleBasic) ValidateGenesis(
	cdc codec.JSONCodec,
	txConfig client.TxConfig,
	bz json.RawMessage,
) error {
	var gs types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &gs); err != nil {
		return err
	}
	return types.ValidateGenesis(&gs)
}

// AppModule implements the AppModule interface for the filehash module.
type AppModule struct {
	AppModuleBasic
	keeper keeper.Keeper
}

// NewAppModule creates a new AppModule instance for the filehash module.
func NewAppModule(k keeper.Keeper) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{},
		keeper:         k,
	}
}

// Name returns the filehash module's name.
func (am AppModule) Name() string {
	return ModuleName
}

// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), am.keeper)
	types.RegisterQueryServer(cfg.QueryServer(), am.keeper)
}

// InitGenesis initializes the module's state from genesis.
func (am AppModule) InitGenesis(
	ctx sdk.Context,
	cdc codec.JSONCodec,
	data json.RawMessage,
) []abci.ValidatorUpdate {
	var gs types.GenesisState
	_ = cdc.UnmarshalJSON(data, &gs)
	return []abci.ValidatorUpdate{}
}

// ExportGenesis exports the module's state to genesis.
func (am AppModule) ExportGenesis(
	ctx sdk.Context,
	cdc codec.JSONCodec,
) json.RawMessage {
	gs := &types.GenesisState{Files: []*types.FileData{}}
	return cdc.MustMarshalJSON(gs)
}

// RegisterInvariants registers module invariants.
func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

// BeginBlock executes block begin logic.
func (am AppModule) BeginBlock(
	ctx sdk.Context,
	req abci.RequestBeginBlock,
) {
}

// EndBlock executes block end logic.
func (am AppModule) EndBlock(
	ctx sdk.Context,
	req abci.RequestEndBlock,
) []abci.ValidatorUpdate {
	return []abci.ValidatorUpdate{}
}
