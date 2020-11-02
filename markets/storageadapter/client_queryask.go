package storageadapter

// this file implements storagemarket.StorageClientNode

import (
	"context"

	"github.com/filecoin-project/go-address"

	"github.com/filecoin-project/go-fil-markets/shared"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/lib/sigs"
	"github.com/filecoin-project/lotus/node/modules/moduleapi"
)

type ClientNodeAdapter struct {
	moduleapi.StateModuleAPI
	moduleapi.ChainModuleAPI
}

func NewClientNodeAdapter(stateapi moduleapi.StateModuleAPI, chain moduleapi.ChainModuleAPI) storagemarket.StorageClientNode {
	return &ClientNodeAdapter{
		StateModuleAPI: stateapi,
		ChainModuleAPI: chain,
	}
}

func (c *ClientNodeAdapter) VerifySignature(ctx context.Context, sig crypto.Signature, addr address.Address, input []byte, encodedTs shared.TipSetToken) (bool, error) {
	addr, err := c.StateAccountKey(ctx, addr, types.EmptyTSK)
	if err != nil {
		return false, err
	}

	err = sigs.Verify(&sig, addr, input)
	return err == nil, err
}

func (c *ClientNodeAdapter) GetChainHead(ctx context.Context) (shared.TipSetToken, abi.ChainEpoch, error) {
	head, err := c.ChainHead(ctx)
	if err != nil {
		return nil, 0, err
	}

	return head.Key().Bytes(), head.Height(), nil
}

var _ storagemarket.StorageClientNode = &ClientNodeAdapter{}
