package types

import "github.com/cosmos/cosmos-sdk/types/errors"

var (
	ErrFileAlreadyExists = errors.Register(ModuleName, 1, "file already exists")
	ErrInvalidAddress    = errors.Register(ModuleName, 2, "invalid address")
	ErrEmptyHash         = errors.Register(ModuleName, 3, "empty file hash")
)
