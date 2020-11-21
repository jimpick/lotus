package client

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/markets/utils"
	"github.com/libp2p/go-libp2p-core/peer"
	multiaddr "github.com/multiformats/go-multiaddr"
	"golang.org/x/xerrors"
)

func (a *API) ClientQueryAsk(ctx context.Context, p peer.ID, miner address.Address) (*storagemarket.StorageAsk, error) {
	mi, err := a.StateMinerInfo(ctx, miner, types.EmptyTSK)
	if err != nil {
		return nil, xerrors.Errorf("failed getting miner info: %w", err)
	}
	fmt.Printf("Jim ClientQueryAsk multiaddrs:\n")
	for _, maddrBytes := range mi.Multiaddrs {
		maddr, err := multiaddr.NewMultiaddrBytes(maddrBytes)
		if err != nil {
			fmt.Printf("  Error %v\n", err)
		} else {
			fmt.Printf("  %v\n", maddr.String())
		}
	}
	var maddrs [][]byte
	if miner.String() == "f02620" {
		maddrs = make([][]byte, 0)
		// relayMaddr, err := multiaddr.NewMultiaddr("/ip4/80.82.17.10/tcp/9999")
		// relayMaddr, err := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/2011/p2p/QmS4Dcz8L7tH6YFU2V9SBufmuNUAj6696TjXmK7tMiVPKA/p2p-circuit")
		// relayPeerID := "QmUnkHyKFkrKQBNmnn12mBhUNWm9oQMjeMq9y3FBC8CnVc"
		// relayMaddr, err := multiaddr.NewMultiaddr("/dns4/libp2p-caddy-relay.localhost/tcp/9058/wss/p2p/" + relayPeerID + "/p2p-circuit")
		// relayMaddr, err := multiaddr.NewMultiaddr("/dns4/libp2p-caddy-relay.localhost/tcp/9058/wss/p2p/" + relayPeerID + "/p2p-circuit/ip4/80.82.17.10/tcp/9999/p2p/12D3KooWDMpcct12Vb6jPXwjvLQHA2hoP8XKGbUZ2tpue1ydoZUm")
		// relayMaddr, err := multiaddr.NewMultiaddr("/dns4/libp2p-caddy-relay.localhost/tcp/9058/wss/p2p/" + relayPeerID + "/p2p-circuit/ip4/80.82.17.10/tcp/9999")
		// relayMaddr, err := multiaddr.NewMultiaddr("/dns4/libp2p-caddy-relay.localhost/tcp/9058/wss/p2p-circuit/ip4/80.82.17.10/tcp/9999")

		// jpimac: ipfs p2p listen /x/miner/f02620/1.0 /ip4/80.82.17.10/tcp/9999
		// jpmbp2: ipfs p2p forward /x/miner/f02620/1.0 /ip4/0.0.0.0/tcp/9999 /p2p/QmScdku7gc3VvfZZvT8kHU77bt6bnH3PnGXkyFRZ17g9EG
		relayMaddr, err := multiaddr.NewMultiaddr("/ip4/10.0.1.11/tcp/9999")
		if err != nil {
			panic(err)
		}
		maddrs = append(maddrs, relayMaddr.Bytes())
		fmt.Printf("Jim ClientQueryAsk new multiaddrs:\n")
		for _, maddrBytes := range maddrs {
			maddr, err := multiaddr.NewMultiaddrBytes(maddrBytes)
			if err != nil {
				fmt.Printf("  Error %v\n", err)
			} else {
				fmt.Printf("  %v\n", maddr.String())
			}
		}
	} else {
		maddrs = mi.Multiaddrs
	}

	// info := utils.NewStorageProviderInfo(miner, mi.Worker, mi.SectorSize, p, mi.Multiaddrs)
	info := utils.NewStorageProviderInfo(miner, mi.Worker, mi.SectorSize, p, maddrs)
	fmt.Printf("Jim ClientQueryAsk info %v\n", info)
	ask, err := a.SMDealClient.GetAsk(ctx, info)
	if err != nil {
		return nil, err
	}
	return ask, nil
}
