// +build clientretrieve
// +build js

package client

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-state-types/big"
	"github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-cid"
	offline "github.com/ipfs/go-ipfs-exchange-offline"
	files "github.com/ipfs/go-ipfs-files"
	ipld "github.com/ipfs/go-ipld-format"
	"github.com/ipfs/go-merkledag"
	unixfile "github.com/ipfs/go-unixfs/file"
	"github.com/ipld/go-car"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	mh "github.com/multiformats/go-multihash"
	"go.uber.org/fx"

	"github.com/filecoin-project/go-address"
	datatransfer "github.com/filecoin-project/go-data-transfer"
	"github.com/filecoin-project/go-fil-markets/discovery"
	"github.com/filecoin-project/go-fil-markets/retrievalmarket"
	rm "github.com/filecoin-project/go-fil-markets/retrievalmarket"
	"github.com/filecoin-project/go-fil-markets/shared"

	marketevents "github.com/filecoin-project/lotus/markets/loggers"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/lotus/node/modules/moduleapi"
	"github.com/filecoin-project/lotus/node/repo/importmgr"
)

var DefaultHashFunction = uint64(mh.BLAKE2B_MIN + 31)

var fileDeposits []files.Node

const dealStartBufferHours uint64 = 49

type API struct {
	fx.In

	moduleapi.ChainModuleAPI
	moduleapi.PaychModuleAPI
	moduleapi.StateModuleAPI

	RetDiscovery discovery.PeerResolver
	Retrieval    rm.RetrievalClient

	Imports dtypes.ClientImportMgr

	CombinedBstore    dtypes.ClientBlockstore // TODO: try to remove
	RetrievalStoreMgr dtypes.ClientRetrievalStoreManager
	DataTransfer      dtypes.ClientDataTransfer
	Host              host.Host
}

func init() {
	fileDeposits = make([]files.Node, 0)
}

func (a *API) imgr() *importmgr.Mgr {
	return a.Imports
}

func (a *API) transfersByID(ctx context.Context) (map[datatransfer.ChannelID]api.DataTransferChannel, error) {
	inProgressChannels, err := a.DataTransfer.InProgressChannels(ctx)
	if err != nil {
		return nil, err
	}

	dataTransfersByID := make(map[datatransfer.ChannelID]api.DataTransferChannel, len(inProgressChannels))
	for id, channelState := range inProgressChannels {
		ch := api.NewDataTransferChannel(a.Host.ID(), channelState)
		dataTransfersByID[id] = ch
	}
	return dataTransfersByID, nil
}

func (a *API) ClientHasLocal(ctx context.Context, root cid.Cid) (bool, error) {
	// TODO: check if we have the ENTIRE dag

	offExch := merkledag.NewDAGService(blockservice.New(a.Imports.Blockstore, offline.Exchange(a.Imports.Blockstore)))
	_, err := offExch.Get(ctx, root)
	if err == ipld.ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (a *API) ClientFindData(ctx context.Context, root cid.Cid, piece *cid.Cid) ([]api.QueryOffer, error) {
	peers, err := a.RetDiscovery.GetPeers(root)
	if err != nil {
		return nil, err
	}

	out := make([]api.QueryOffer, 0, len(peers))
	for _, p := range peers {
		if piece != nil && !piece.Equals(*p.PieceCID) {
			continue
		}
		out = append(out, a.makeRetrievalQuery(ctx, p, root, piece, rm.QueryParams{}))
	}

	return out, nil
}

func (a *API) ClientMinerQueryOffer(ctx context.Context, miner address.Address, root cid.Cid, piece *cid.Cid) (api.QueryOffer, error) {
	mi, err := a.StateMinerInfo(ctx, miner, types.EmptyTSK)
	if err != nil {
		return api.QueryOffer{}, err
	}
	rp := rm.RetrievalPeer{
		Address: miner,
		ID:      *mi.PeerId,
	}
	return a.makeRetrievalQuery(ctx, rp, root, piece, rm.QueryParams{}), nil
}

func (a *API) makeRetrievalQuery(ctx context.Context, rp rm.RetrievalPeer, payload cid.Cid, piece *cid.Cid, qp rm.QueryParams) api.QueryOffer {
	queryResponse, err := a.Retrieval.Query(ctx, rp, payload, qp)
	if err != nil {
		return api.QueryOffer{Err: err.Error(), Miner: rp.Address, MinerPeer: rp}
	}
	var errStr string
	switch queryResponse.Status {
	case rm.QueryResponseAvailable:
		errStr = ""
	case rm.QueryResponseUnavailable:
		errStr = fmt.Sprintf("retrieval query offer was unavailable: %s", queryResponse.Message)
	case rm.QueryResponseError:
		errStr = fmt.Sprintf("retrieval query offer errored: %s", queryResponse.Message)
	}

	return api.QueryOffer{
		Root:                    payload,
		Piece:                   piece,
		Size:                    queryResponse.Size,
		MinPrice:                queryResponse.PieceRetrievalPrice(),
		UnsealPrice:             queryResponse.UnsealPrice,
		PaymentInterval:         queryResponse.MaxPaymentInterval,
		PaymentIntervalIncrease: queryResponse.MaxPaymentIntervalIncrease,
		Miner:                   queryResponse.PaymentAddress, // TODO: check
		MinerPeer:               rp,
		Err:                     errStr,
	}
}

func (a *API) ClientRetrieve(ctx context.Context, order api.RetrievalOrder, ref *api.FileRef) (int, error) {
	events := make(chan marketevents.RetrievalEvent)
	go a.clientRetrieve(ctx, order, ref, events)

	fileDepositID := -1

	for {
		select {
		case evt, ok := <-events:
			if !ok { // done successfully
				fmt.Printf("Jim ClientRetrieve success\n")
				return fileDepositID, nil
			}

			if evt.Event == retrievalmarket.ClientEventFileDeposited {
				fileDepositID = evt.FileDepositID
			}

			if evt.Err != "" {
				return -1, xerrors.Errorf("retrieval failed: %s", evt.Err)
			}
		case <-ctx.Done():
			return -1, xerrors.Errorf("retrieval timed out")
		}
	}
}

func (a *API) ClientRetrieveWithEvents(ctx context.Context, order api.RetrievalOrder, ref *api.FileRef) (<-chan marketevents.RetrievalEvent, error) {
	events := make(chan marketevents.RetrievalEvent)
	go a.clientRetrieve(ctx, order, ref, events)
	return events, nil
}

type retrievalSubscribeEvent struct {
	event rm.ClientEvent
	state rm.ClientDealState
}

func readSubscribeEvents(ctx context.Context, dealID retrievalmarket.DealID, subscribeEvents chan retrievalSubscribeEvent, events chan marketevents.RetrievalEvent) error {
	fmt.Printf("Jim readSubscribeEvents 1\n")
	for {
		var subscribeEvent retrievalSubscribeEvent
		select {
		case <-ctx.Done():
			return xerrors.New("Retrieval Timed Out")
		case subscribeEvent = <-subscribeEvents:
			if subscribeEvent.state.ID != dealID {
				// we can't check the deal ID ahead of time because:
				// 1. We need to subscribe before retrieving.
				// 2. We won't know the deal ID until after retrieving.
				continue
			}
		}

		eventState := subscribeEvent.state
		fmt.Printf("Jim > Recv: %s, Paid %s, %s (%s)\n",
			types.SizeStr(types.NewInt(eventState.TotalReceived)),
			types.FIL(eventState.FundsSpent),
			retrievalmarket.ClientEvents[subscribeEvent.event],
			retrievalmarket.DealStatuses[eventState.Status],
		)

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

	finish := func(e error) {
		if e != nil {
			events <- marketevents.RetrievalEvent{Err: e.Error(), FundsSpent: big.Zero()}
		}
	}

	/*
		mi, err := a.StateMinerInfo(ctx, order.MinerPeer.Address, types.EmptyTSK)
		if err != nil {
			finish(err)
			return
		}

		fmt.Printf("Jim client_retrieve minerInfo %v\n", mi)
		maddrs := make([]multiaddr.Multiaddr, 0, len(mi.Multiaddrs))
		for _, a := range mi.Multiaddrs {
			maddr, err := multiaddr.NewMultiaddrBytes(a)
			if err != nil {
				finish(err)
				return
			}
			fmt.Printf("Jim maddr %v\n", maddr)
			maddrs = append(maddrs, maddr)
			//	for _, p := range maddr.Protocols() {
			//		if p.Code == multiaddr.P_WSS {
			//			useDaemon = false
			//			break
			//		}
			// 	}
		}
	*/

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
	// a.Host.Peerstore().AddAddrs(order.MinerPeer.ID, maddrs, 8*time.Hour)

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

	// Subscribe to events before retrieving to avoid losing events.
	subscribeEvents := make(chan retrievalSubscribeEvent, 1)
	subscribeCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	unsubscribe := a.Retrieval.SubscribeToEvents(func(event rm.ClientEvent, state rm.ClientDealState) {
		// We'll check the deal IDs inside readSubscribeEvents.
		if state.PayloadCID.Equals(order.Root) {
			select {
			case <-subscribeCtx.Done():
			case subscribeEvents <- retrievalSubscribeEvent{event, state}:
			}
		}
	})

	dealID, err := a.Retrieval.Retrieve(
		ctx,
		order.Root,
		params,
		order.Total,
		order.MinerPeer,
		order.Client,
		order.Miner,
		store.StoreID())

	if err != nil {
		unsubscribe()
		finish(xerrors.Errorf("Retrieve failed: %w", err))
		return
	}

	err = readSubscribeEvents(ctx, dealID, subscribeEvents, events)

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

	fmt.Printf("Jim impl client_retrieve 1\n")
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
	fmt.Printf("Jim impl client_retrieve 2\n")

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
	fileDeposits = append(fileDeposits, file)
	fileDepositID := len(fileDeposits)
	fmt.Printf("Successful retrieval, fileDepositID: %v\n", fileDepositID)
	size, err := file.Size()
	if err != nil {
		finish(xerrors.Errorf("ClientRetrieve: %w", err))
		return
	}
	fmt.Printf("Jim impl client_retrieve size %v\n", size)
	/*
		readablefile := files.ToFile(file)
		b := make([]byte, 800)
		for {
			n, err := readablefile.Read(b)
			// fmt.Printf("n = %v err = %v b = %v\n", n, err, b)
			fmt.Printf("n = %v err = %v\n", n, err)
			// fmt.Printf("b[:n] = %q\n", b[:n])
			if err == io.EOF {
				break
			}
		}
	*/
	// finish(files.WriteTo(file, ref.Path))
	fmt.Printf("Jim impl client_retrieve 3\n")
	events <- marketevents.RetrievalEvent{
		Event:         retrievalmarket.ClientEventFileDeposited,
		FileDepositID: fileDepositID,
	}
	return
}

type lenWriter int64

func (w *lenWriter) Write(p []byte) (n int, err error) {
	*w += lenWriter(len(p))
	return len(p), nil
}

func (a *API) ClientListDataTransfers(ctx context.Context) ([]api.DataTransferChannel, error) {
	inProgressChannels, err := a.DataTransfer.InProgressChannels(ctx)
	if err != nil {
		return nil, err
	}

	apiChannels := make([]api.DataTransferChannel, 0, len(inProgressChannels))
	for _, channelState := range inProgressChannels {
		apiChannels = append(apiChannels, api.NewDataTransferChannel(a.Host.ID(), channelState))
	}

	return apiChannels, nil
}

func (a *API) ClientDataTransferUpdates(ctx context.Context) (<-chan api.DataTransferChannel, error) {
	channels := make(chan api.DataTransferChannel)

	unsub := a.DataTransfer.SubscribeToEvents(func(evt datatransfer.Event, channelState datatransfer.ChannelState) {
		channel := api.NewDataTransferChannel(a.Host.ID(), channelState)
		select {
		case <-ctx.Done():
		case channels <- channel:
		}
	})

	go func() {
		defer unsub()
		<-ctx.Done()
	}()

	return channels, nil
}

func (a *API) ClientRestartDataTransfer(ctx context.Context, transferID datatransfer.TransferID, otherPeer peer.ID, isInitiator bool) error {
	selfPeer := a.Host.ID()
	if isInitiator {
		return a.DataTransfer.RestartDataTransferChannel(ctx, datatransfer.ChannelID{Initiator: selfPeer, Responder: otherPeer, ID: transferID})
	}
	return a.DataTransfer.RestartDataTransferChannel(ctx, datatransfer.ChannelID{Initiator: otherPeer, Responder: selfPeer, ID: transferID})
}

func (a *API) ClientCancelDataTransfer(ctx context.Context, transferID datatransfer.TransferID, otherPeer peer.ID, isInitiator bool) error {
	selfPeer := a.Host.ID()
	if isInitiator {
		return a.DataTransfer.CloseDataTransferChannel(ctx, datatransfer.ChannelID{Initiator: selfPeer, Responder: otherPeer, ID: transferID})
	}
	return a.DataTransfer.CloseDataTransferChannel(ctx, datatransfer.ChannelID{Initiator: otherPeer, Responder: selfPeer, ID: transferID})
}

func (a *API) ClientRetrieveTryRestartInsufficientFunds(ctx context.Context, paymentChannel address.Address) error {
	return a.Retrieval.TryRestartInsufficientFunds(paymentChannel)
}
