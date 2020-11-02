package modules

import (
	"context"

	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"
	"go.uber.org/fx"

	discoveryimpl "github.com/filecoin-project/go-fil-markets/discovery/impl"

	"github.com/filecoin-project/lotus/journal"
	"github.com/filecoin-project/lotus/lib/peermgr"
	marketevents "github.com/filecoin-project/lotus/markets/loggers"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/lotus/node/modules/helpers"
	"github.com/filecoin-project/lotus/node/repo"
)

func protosContains(protos []string, search string) bool {
	for _, p := range protos {
		if p == search {
			return true
		}
	}
	return false
}

func RunPeerMgr(mctx helpers.MetricsCtx, lc fx.Lifecycle, pmgr *peermgr.PeerMgr) {
	go pmgr.Run(helpers.LifecycleCtx(mctx, lc))
}

func NewLocalDiscovery(lc fx.Lifecycle, ds dtypes.MetadataDS) (*discoveryimpl.Local, error) {
	local, err := discoveryimpl.NewLocal(namespace.Wrap(ds, datastore.NewKey("/deals/local")))
	if err != nil {
		return nil, err
	}
	local.OnReady(marketevents.ReadyLogger("discovery"))
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return local.Start(ctx)
		},
	})
	return local, nil
}

func OpenFilesystemJournal(lr repo.LockedRepo, lc fx.Lifecycle, disabled journal.DisabledEvents) (journal.Journal, error) {
	jrnl, err := journal.OpenFSJournal(lr, disabled)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error { return jrnl.Close() },
	})

	return jrnl, err
}
