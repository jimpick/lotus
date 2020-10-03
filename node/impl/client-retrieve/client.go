package client

import (
	"context"
	"fmt"
	"os"

	"github.com/filecoin-project/go-state-types/big"
	"golang.org/x/xerrors"

	"github.com/ipfs/go-cid"
	files "github.com/ipfs/go-ipfs-files"
	unixfile "github.com/ipfs/go-unixfs/file"
	"github.com/ipld/go-car"
	"go.uber.org/fx"

	"github.com/filecoin-project/go-fil-markets/retrievalmarket"
	rm "github.com/filecoin-project/go-fil-markets/retrievalmarket"
	"github.com/filecoin-project/go-fil-markets/shared"

	marketevents "github.com/filecoin-project/lotus/markets/loggers"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
)

type API struct {
	fx.In

	Retrieval rm.RetrievalClient

	RetrievalStoreMgr dtypes.ClientRetrievalStoreManager
}

func (a *API) ClientRetrieve(ctx context.Context, order api.RetrievalOrder, ref *api.FileRef) error {
	events := make(chan marketevents.RetrievalEvent)
	go a.clientRetrieve(ctx, order, ref, events)

	for {
		select {
		case evt, ok := <-events:
			if !ok { // done successfully
				return nil
			}

			if evt.Err != "" {
				return xerrors.Errorf("retrieval failed: %s", evt.Err)
			}
		case <-ctx.Done():
			return xerrors.Errorf("retrieval timed out")
		}
	}
}

type retrievalSubscribeEvent struct {
	event rm.ClientEvent
	state rm.ClientDealState
}

func readSubscribeEvents(ctx context.Context, subscribeEvents chan retrievalSubscribeEvent, events chan marketevents.RetrievalEvent) error {
	for {
		var subscribeEvent retrievalSubscribeEvent
		select {
		case <-ctx.Done():
			return xerrors.New("Retrieval Timed Out")
		case subscribeEvent = <-subscribeEvents:
		}

		select {
		case <-ctx.Done():
			return xerrors.New("Retrieval Timed Out")
		case events <- marketevents.RetrievalEvent{
			Event:         subscribeEvent.event,
			Status:        subscribeEvent.state.Status,
			BytesReceived: subscribeEvent.state.TotalReceived,
			FundsSpent:    subscribeEvent.state.FundsSpent,
		}:
		}

		state := subscribeEvent.state
		switch state.Status {
		case rm.DealStatusCompleted:
			return nil
		case rm.DealStatusRejected:
			return xerrors.Errorf("Retrieval Proposal Rejected: %s", state.Message)
		case
			rm.DealStatusDealNotFound,
			rm.DealStatusErrored:
			return xerrors.Errorf("Retrieval Error: %s", state.Message)
		}
	}
}

func (a *API) clientRetrieve(ctx context.Context, order api.RetrievalOrder, ref *api.FileRef, events chan marketevents.RetrievalEvent) {
	defer close(events)

	fmt.Printf("Jim clientRetrieve2 order %v\n", order)
	fmt.Printf("Jim clientRetrieve2 ref %v\n", ref)
	finish := func(e error) {
		if e != nil {
			events <- marketevents.RetrievalEvent{Err: e.Error(), FundsSpent: big.Zero()}
		}
	}

	/*
		if order.MinerPeer.ID == "" {
			mi, err := a.StateMinerInfo(ctx, order.Miner, types.EmptyTSK)
			if err != nil {
				finish(err)
				return
			}

			order.MinerPeer = retrievalmarket.RetrievalPeer{
				ID:      *mi.PeerId,
				Address: order.Miner,
			}
		}
	*/

	if order.Size == 0 {
		finish(xerrors.Errorf("cannot make retrieval deal for zero bytes"))
		return
	}

	/*id, st, err := a.imgr().NewStore()
	if err != nil {
		return err
	}
	if err := a.imgr().AddLabel(id, "source", "retrieval"); err != nil {
		return err
	}*/

	var dealID retrievalmarket.DealID
	subscribeEvents := make(chan retrievalSubscribeEvent, 1)
	subscribeCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	unsubscribe := a.Retrieval.SubscribeToEvents(func(event rm.ClientEvent, state rm.ClientDealState) {
		fmt.Printf("Jim retrieval event %v, %v, %v\n", event, rm.DealStatuses[state.Status], state)
		if state.PayloadCID.Equals(order.Root) && state.ID == dealID {
			select {
			case <-subscribeCtx.Done():
			case subscribeEvents <- retrievalSubscribeEvent{event, state}:
			}
		}
	})

	ppb := types.BigDiv(order.Total, types.NewInt(order.Size))

	params, err := rm.NewParamsV1(ppb, order.PaymentInterval, order.PaymentIntervalIncrease, shared.AllSelector(), order.Piece, order.UnsealPrice)
	if err != nil {
		finish(xerrors.Errorf("Error in retrieval params: %s", err))
		return
	}

	store, err := a.RetrievalStoreMgr.NewStore()
	if err != nil {
		finish(xerrors.Errorf("Error setting up new store: %w", err))
		return
	}

	defer func() {
		_ = a.RetrievalStoreMgr.ReleaseStore(store)
	}()

	dealID, err = a.Retrieval.Retrieve(
		ctx,
		order.Root,
		params,
		order.Total,
		order.MinerPeer,
		order.Client,
		order.Miner,
		store.StoreID())

	if err != nil {
		finish(xerrors.Errorf("Retrieve failed: %w", err))
		return
	}

	err = readSubscribeEvents(ctx, subscribeEvents, events)

	unsubscribe()
	if err != nil {
		finish(xerrors.Errorf("Retrieve: %w", err))
		return
	}

	// If ref is nil, it only fetches the data into the configured blockstore.
	if ref == nil {
		finish(nil)
		return
	}

	rdag := store.DAGService()

	if ref.IsCAR {
		f, err := os.OpenFile(ref.Path, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			finish(err)
			return
		}
		err = car.WriteCar(ctx, rdag, []cid.Cid{order.Root}, f)
		if err != nil {
			finish(err)
			return
		}
		finish(f.Close())
		return
	}

	nd, err := rdag.Get(ctx, order.Root)
	if err != nil {
		finish(xerrors.Errorf("ClientRetrieve: %w", err))
		return
	}
	file, err := unixfile.NewUnixfsFile(ctx, rdag, nd)
	if err != nil {
		finish(xerrors.Errorf("ClientRetrieve: %w", err))
		return
	}
	finish(files.WriteTo(file, ref.Path))
	return
}
