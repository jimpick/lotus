package modules

import (
	"go.uber.org/fx"

	"github.com/filecoin-project/go-fil-markets/storagemarket"
	storageimpl "github.com/filecoin-project/go-fil-markets/storagemarket/impl"
	smnet "github.com/filecoin-project/go-fil-markets/storagemarket/network"
	"github.com/libp2p/go-libp2p-core/host"
)

func StorageClient(lc fx.Lifecycle, h host.Host, scn storagemarket.StorageClientNode) (storagemarket.StorageClient, error) {
	net := smnet.NewFromLibp2pHost(h)
	c, err := storageimpl.NewClient(net, scn)
	if err != nil {
		return nil, err
	}
	return c, nil
}
