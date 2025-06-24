package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	"doctorium/x/filehash/types"
)

// Keeper handles storage and business logic for the filehash module.
type Keeper struct {
	types.UnimplementedMsgServer
	types.UnimplementedQueryServer

	storeKey   storetypes.StoreKey
	cdc        codec.BinaryCodec
	bankKeeper bankkeeper.Keeper
}

// NewKeeper creates a new Keeper instance.
func NewKeeper(cdc codec.BinaryCodec, key storetypes.StoreKey, bankKeeper bankkeeper.Keeper) Keeper {
	return Keeper{storeKey: key, cdc: cdc, bankKeeper: bankKeeper}
}

// StoreFileHash saves a file hash under the creator address.
func (k Keeper) StoreFileHash(ctx sdk.Context, creator, hash string) {
	store := ctx.KVStore(k.storeKey)
	store.Set([]byte(hash), []byte(creator))
}

// HasFileHash checks if a file hash already exists.
func (k Keeper) HasFileHash(ctx sdk.Context, hash string) bool {
	return ctx.KVStore(k.storeKey).Has([]byte(hash))
}

func (k Keeper) GetAllFiles(ctx sdk.Context, req *types.QueryFileListRequest) (*types.QueryFileListResponse, error) {
	store := ctx.KVStore(k.storeKey)
	resp := &types.QueryFileListResponse{}
	pageRes, err := query.Paginate(store, req.Pagination, func(key []byte, value []byte) error {
		// value, key 로 만든 FileData 값을 포인터로 만들어 슬라이스에 추가
		resp.Files = append(resp.Files, &types.FileData{
			Creator:  string(value),
			FileHash: string(key),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	resp.Pagination = pageRes
	return resp, nil
}

// UploadFile processes a file upload message and mints a reward.
func (k Keeper) UploadFile(goCtx context.Context, msg *types.MsgUploadFile) (*types.MsgUploadFileResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Prevent duplicate uploads
	if k.HasFileHash(ctx, msg.FileHash) {
		return nil, sdkerrors.Wrap(types.ErrFileAlreadyExists, msg.FileHash)
	}

	// Store the hash
	k.StoreFileHash(ctx, msg.Creator, msg.FileHash)

	// Mint and send reward coins
	coins := sdk.NewCoins(sdk.NewInt64Coin("drt", 10))
	// Mint into module account
	if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, coins); err != nil {
		return nil, err
	}
	// Send from module to user
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, sdk.AccAddress(msg.Creator), coins); err != nil {
		return nil, err
	}

	return &types.MsgUploadFileResponse{Success: true}, nil
}
