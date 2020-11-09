package client

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/markets/utils"
	"github.com/libp2p/go-libp2p-core/peer"
	"golang.org/x/xerrors"
	// "github.com/filecoin-project/lotus/chain/types"
	// "github.com/filecoin-project/lotus/markets/utils"
)

func (a *API) ClientQueryAsk(ctx context.Context, p peer.ID, miner address.Address) (*storagemarket.StorageAsk, error) {
	fmt.Printf("Jim node impl ClientQueryAsk %v %v\n", p, miner)
	mi, err := a.StateMinerInfo(ctx, miner, types.EmptyTSK)
	if err != nil {
		return nil, xerrors.Errorf("failed getting miner info: %w", err)
	}
	fmt.Printf("Jim node impl ClientQueryAsk minerinfo %v\n", mi)
	// return nil, xerrors.Errorf("jim1 - clientqueryask")

	info := utils.NewStorageProviderInfo(miner, mi.Worker, mi.SectorSize, p, mi.Multiaddrs)
	fmt.Printf("Jim ClientQueryAsk info %v\n", info)
	ask, err := a.SMDealClient.GetAsk(ctx, info)
	if err != nil {
		return nil, err
	}
	return ask, nil
}
