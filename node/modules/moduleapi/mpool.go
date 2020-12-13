package moduleapi

import (
	"context"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	cid "github.com/ipfs/go-cid"
)

type MpoolModuleAPI interface {
	MpoolPush(ctx context.Context, smsg *types.SignedMessage) (cid.Cid, error)
	MpoolPushMessage(ctx context.Context, msg *types.Message, spec *api.MessageSendSpec) (*types.SignedMessage, error)
}
