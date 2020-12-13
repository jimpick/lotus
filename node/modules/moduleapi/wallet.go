package moduleapi

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
)

type WalletModuleAPI interface {
	api.WalletAPI
	WalletBalance(ctx context.Context, addr address.Address) (types.BigInt, error)
	WalletSignMessage(ctx context.Context, k address.Address, msg *types.Message) (*types.SignedMessage, error)
	WalletVerify(ctx context.Context, k address.Address, msg []byte, sig *crypto.Signature) (bool, error)
	WalletDefaultAddress(ctx context.Context) (address.Address, error)
	WalletSetDefault(ctx context.Context, addr address.Address) error
	WalletValidateAddress(ctx context.Context, str string) (address.Address, error)
}
