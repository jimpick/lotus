package api

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
)

type StateForRetrieval interface {
	StateNetworkName(ctx context.Context) (dtypes.NetworkName, error)

	StateMinerInfo(ctx context.Context, actor address.Address, tsk types.TipSetKey) (MinerInfo, error)
}
