package api

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/ipfs/go-cid"
)

type Retrieve interface {
	ClientRetrieve(ctx context.Context, order RetrievalOrder, ref *FileRef) error
}

type StateForRetrieval interface {
	StateNetworkName(ctx context.Context) (dtypes.NetworkName, error)
	StateMinerInfo(ctx context.Context, actor address.Address, tsk types.TipSetKey) (MinerInfo, error)
}

type PaychForRetrieval interface {
	PaychGet(ctx context.Context, from, to address.Address, amt types.BigInt) (*ChannelInfo, error)
	PaychGetWaitReady(context.Context, cid.Cid) (address.Address, error)
	PaychAllocateLane(ctx context.Context, ch address.Address) (uint64, error)
	PaychVoucherCreate(context.Context, address.Address, types.BigInt, uint64) (*VoucherCreateResult, error)
	PaychAvailableFunds(ctx context.Context, ch address.Address) (*ChannelAvailableFunds, error)
}

type ChainForRetrieval interface {
	ChainHead(context.Context) (*types.TipSet, error)
}
