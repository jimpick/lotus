package modules

import (
	"context"
	"time"

	"github.com/filecoin-project/go-multistore"
	"golang.org/x/xerrors"

	"go.uber.org/fx"

	discoveryimpl "github.com/filecoin-project/go-fil-markets/discovery/impl"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	storageimpl "github.com/filecoin-project/go-fil-markets/storagemarket/impl"
	"github.com/filecoin-project/go-fil-markets/storagemarket/impl/funds"
	"github.com/filecoin-project/go-fil-markets/storagemarket/impl/requestvalidation"
	smnet "github.com/filecoin-project/go-fil-markets/storagemarket/network"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"
	"github.com/libp2p/go-libp2p-core/host"

	"github.com/filecoin-project/lotus/journal"
	"github.com/filecoin-project/lotus/lib/blockstore"
	"github.com/filecoin-project/lotus/markets"
	marketevents "github.com/filecoin-project/lotus/markets/loggers"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/lotus/node/repo"
	"github.com/filecoin-project/lotus/node/repo/importmgr"
)

func ClientMultiDatastore(lc fx.Lifecycle, r repo.LockedRepo) (dtypes.ClientMultiDstore, error) {
	ds, err := r.Datastore("/client")
	if err != nil {
		return nil, xerrors.Errorf("getting datastore out of reop: %w", err)
	}

	mds, err := multistore.NewMultiDstore(ds)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return mds.Close()
		},
	})

	return mds, nil
}

func ClientImportMgr(mds dtypes.ClientMultiDstore, ds dtypes.MetadataDS) dtypes.ClientImportMgr {
	return importmgr.New(mds, namespace.Wrap(ds, datastore.NewKey("/client")))
}

func ClientBlockstore(imgr dtypes.ClientImportMgr) dtypes.ClientBlockstore {
	// in most cases this is now unused in normal operations -- however, it's important to preserve for the IPFS use case
	return blockstore.WrapIDStore(imgr.Blockstore)
}

// RegisterClientValidator is an initialization hook that registers the client
// request validator with the data transfer module as the validator for
// StorageDataTransferVoucher types
func RegisterClientValidator(crv dtypes.ClientRequestValidator, dtm dtypes.ClientDataTransfer) {
	if err := dtm.RegisterVoucherType(&requestvalidation.StorageDataTransferVoucher{}, (*requestvalidation.UnifiedRequestValidator)(crv)); err != nil {
		panic(err)
	}
}

// NewClientDatastore creates a datastore for the client to store its deals
func NewClientDatastore(ds dtypes.MetadataDS) dtypes.ClientDatastore {
	return namespace.Wrap(ds, datastore.NewKey("/deals/client"))
}

type ClientDealFunds funds.DealFunds

func NewClientDealFunds(ds dtypes.MetadataDS) (ClientDealFunds, error) {
	return funds.NewDealFunds(ds, datastore.NewKey("/marketfunds/client"))
}

func StorageClient(lc fx.Lifecycle, h host.Host, ibs dtypes.ClientBlockstore, mds dtypes.ClientMultiDstore, r repo.LockedRepo, dataTransfer dtypes.ClientDataTransfer, discovery *discoveryimpl.Local, deals dtypes.ClientDatastore, scn storagemarket.StorageClientNode, dealFunds ClientDealFunds, j journal.Journal) (storagemarket.StorageClient, error) {
	net := smnet.NewFromLibp2pHost(h)
	c, err := storageimpl.NewClient(net, ibs, mds, dataTransfer, discovery, deals, scn, dealFunds, storageimpl.DealPollingInterval(time.Second))
	if err != nil {
		return nil, err
	}
	c.OnReady(marketevents.ReadyLogger("storage client"))
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			c.SubscribeToEvents(marketevents.StorageClientLogger)

			evtType := j.RegisterEventType("markets/storage/client", "state_change")
			c.SubscribeToEvents(markets.StorageClientJournaler(j, evtType))

			return c.Start(ctx)
		},
		OnStop: func(context.Context) error {
			return c.Stop()
		},
	})
	return c, nil
}
