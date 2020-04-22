package stmgr

import (
	"context"

	amt "github.com/filecoin-project/go-amt-ipld/v2"
	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/filecoin-project/specs-actors/actors/abi/big"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	init_ "github.com/filecoin-project/specs-actors/actors/builtin/init"
	"github.com/filecoin-project/specs-actors/actors/builtin/market"
	"github.com/filecoin-project/specs-actors/actors/builtin/miner"
	"github.com/filecoin-project/specs-actors/actors/builtin/power"
	"github.com/filecoin-project/specs-actors/actors/util/adt"

	"github.com/filecoin-project/go-address"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/actors/aerrors"
	"github.com/filecoin-project/lotus/chain/state"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/vm"
	"github.com/filecoin-project/lotus/node/modules/dtypes"

	cid "github.com/ipfs/go-cid"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	cbor "github.com/ipfs/go-ipld-cbor"
	"github.com/libp2p/go-libp2p-core/peer"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/xerrors"
)

func GetNetworkName(ctx context.Context, sm *StateManager, st cid.Cid) (dtypes.NetworkName, error) {
	var state init_.State
	_, err := sm.LoadActorStateRaw(ctx, builtin.InitActorAddr, &state, st)
	if err != nil {
		return "", xerrors.Errorf("(get sset) failed to load miner actor state: %w", err)
	}

	return dtypes.NetworkName(state.NetworkName), nil
}

func GetMinerWorkerRaw(ctx context.Context, sm *StateManager, st cid.Cid, maddr address.Address) (address.Address, error) {
	var mas miner.State
	_, err := sm.LoadActorStateRaw(ctx, maddr, &mas, st)
	if err != nil {
		return address.Undef, xerrors.Errorf("(get sset) failed to load miner actor state: %w", err)
	}

	cst := cbor.NewCborStore(sm.cs.Blockstore())
	state, err := state.LoadStateTree(cst, st)
	if err != nil {
		return address.Undef, xerrors.Errorf("load state tree: %w", err)
	}

	return vm.ResolveToKeyAddr(state, cst, mas.Info.Worker)
}

func GetMinerOwner(ctx context.Context, sm *StateManager, st cid.Cid, maddr address.Address) (address.Address, error) {
	var mas miner.State
	_, err := sm.LoadActorStateRaw(ctx, maddr, &mas, st)
	if err != nil {
		return address.Undef, xerrors.Errorf("(get sset) failed to load miner actor state: %w", err)
	}

	cst := cbor.NewCborStore(sm.cs.Blockstore())
	state, err := state.LoadStateTree(cst, st)
	if err != nil {
		return address.Undef, xerrors.Errorf("load state tree: %w", err)
	}

	return vm.ResolveToKeyAddr(state, cst, mas.Info.Owner)
}

func GetPower(ctx context.Context, sm *StateManager, ts *types.TipSet, maddr address.Address) (types.BigInt, types.BigInt, error) {
	return getPowerRaw(ctx, sm, ts.ParentState(), maddr)
}

func getPowerRaw(ctx context.Context, sm *StateManager, st cid.Cid, maddr address.Address) (types.BigInt, types.BigInt, error) {
	var ps power.State
	_, err := sm.LoadActorStateRaw(ctx, builtin.StoragePowerActorAddr, &ps, st)
	if err != nil {
		return big.Zero(), big.Zero(), xerrors.Errorf("(get sset) failed to load power actor state: %w", err)
	}

	var mpow big.Int
	if maddr != address.Undef {
		var claim power.Claim
		if _, err := adt.AsMap(sm.cs.Store(ctx), ps.Claims).Get(adt.AddrKey(maddr), &claim); err != nil {
			return big.Zero(), big.Zero(), err
		}

		mpow = claim.Power
	}

	return mpow, ps.TotalNetworkPower, nil
}

func GetMinerPeerID(ctx context.Context, sm *StateManager, ts *types.TipSet, maddr address.Address) (peer.ID, error) {
	var mas miner.State
	_, err := sm.LoadActorState(ctx, maddr, &mas, ts)
	if err != nil {
		return "", xerrors.Errorf("(get sset) failed to load miner actor state: %w", err)
	}

	return mas.Info.PeerId, nil
}

func GetMinerWorker(ctx context.Context, sm *StateManager, ts *types.TipSet, maddr address.Address) (address.Address, error) {
	return GetMinerWorkerRaw(ctx, sm, sm.parentState(ts), maddr)
}

func GetMinerPostState(ctx context.Context, sm *StateManager, ts *types.TipSet, maddr address.Address) (*miner.PoStState, error) {
	var mas miner.State
	_, err := sm.LoadActorState(ctx, maddr, &mas, ts)
	if err != nil {
		return nil, xerrors.Errorf("(get eps) failed to load miner actor state: %w", err)
	}

	return &mas.PoStState, nil
}

func SectorSetSizes(ctx context.Context, sm *StateManager, maddr address.Address, ts *types.TipSet) (api.MinerSectors, error) {
	var mas miner.State
	_, err := sm.LoadActorState(ctx, maddr, &mas, ts)
	if err != nil {
		return api.MinerSectors{}, xerrors.Errorf("(get sset) failed to load miner actor state: %w", err)
	}

	blks := cbor.NewCborStore(sm.ChainStore().Blockstore())
	ss, err := amt.LoadAMT(ctx, blks, mas.Sectors)
	if err != nil {
		return api.MinerSectors{}, err
	}

	ps, err := amt.LoadAMT(ctx, blks, mas.ProvingSet)
	if err != nil {
		return api.MinerSectors{}, err
	}

	return api.MinerSectors{
		Pset: ps.Count,
		Sset: ss.Count,
	}, nil
}

func PreCommitInfo(ctx context.Context, sm *StateManager, maddr address.Address, sid abi.SectorNumber, ts *types.TipSet) (miner.SectorPreCommitOnChainInfo, error) {
	var mas miner.State
	_, err := sm.LoadActorState(ctx, maddr, &mas, ts)
	if err != nil {
		return miner.SectorPreCommitOnChainInfo{}, xerrors.Errorf("(get sset) failed to load miner actor state: %w", err)
	}

	i, ok, err := mas.GetPrecommittedSector(sm.cs.Store(ctx), sid)
	if err != nil {
		return miner.SectorPreCommitOnChainInfo{}, err
	}
	if !ok {
		return miner.SectorPreCommitOnChainInfo{}, xerrors.New("precommit not found")
	}

	return *i, nil
}

func GetMinerProvingSet(ctx context.Context, sm *StateManager, ts *types.TipSet, maddr address.Address) ([]*api.ChainSectorInfo, error) {
	return getMinerProvingSetRaw(ctx, sm, ts.ParentState(), maddr)
}

func getMinerProvingSetRaw(ctx context.Context, sm *StateManager, st cid.Cid, maddr address.Address) ([]*api.ChainSectorInfo, error) {
	var mas miner.State
	_, err := sm.LoadActorStateRaw(ctx, maddr, &mas, st)
	if err != nil {
		return nil, xerrors.Errorf("(get pset) failed to load miner actor state: %w", err)
	}

	return LoadSectorsFromSet(ctx, sm.ChainStore().Blockstore(), mas.ProvingSet)
}

func GetMinerSectorSet(ctx context.Context, sm *StateManager, ts *types.TipSet, maddr address.Address) ([]*api.ChainSectorInfo, error) {
	var mas miner.State
	_, err := sm.LoadActorState(ctx, maddr, &mas, ts)
	if err != nil {
		return nil, xerrors.Errorf("(get sset) failed to load miner actor state: %w", err)
	}

	return LoadSectorsFromSet(ctx, sm.ChainStore().Blockstore(), mas.Sectors)
}

func GetSectorsForElectionPost(ctx context.Context, sm *StateManager, ts *types.TipSet, maddr address.Address) ([]abi.SectorInfo, error) {
	sectors, err := GetMinerProvingSet(ctx, sm, ts, maddr)
	if err != nil {
		return nil, xerrors.Errorf("failed to get sector set for miner: %w", err)
	}

	var uselessOtherArray []abi.SectorInfo
	for _, s := range sectors {
		uselessOtherArray = append(uselessOtherArray, abi.SectorInfo{
			RegisteredProof: s.Info.Info.RegisteredProof,
			SectorNumber:    s.ID,
			SealedCID:       s.Info.Info.SealedCID,
		})
	}

	return uselessOtherArray, nil
}

func GetMinerSectorSize(ctx context.Context, sm *StateManager, ts *types.TipSet, maddr address.Address) (abi.SectorSize, error) {
	return getMinerSectorSizeRaw(ctx, sm, ts.ParentState(), maddr)
}

func getMinerSectorSizeRaw(ctx context.Context, sm *StateManager, st cid.Cid, maddr address.Address) (abi.SectorSize, error) {
	var mas miner.State
	_, err := sm.LoadActorStateRaw(ctx, maddr, &mas, st)
	if err != nil {
		return 0, xerrors.Errorf("(get ssize) failed to load miner actor state: %w", err)
	}

	return mas.Info.SectorSize, nil
}

func GetMinerSlashed(ctx context.Context, sm *StateManager, ts *types.TipSet, maddr address.Address) (bool, error) {
	var mas miner.State
	_, err := sm.LoadActorState(ctx, maddr, &mas, ts)
	if err != nil {
		return false, xerrors.Errorf("(get miner slashed) failed to load miner actor state")
	}

	if mas.PoStState.HasFailedPost() {
		return true, nil
	}

	var spas power.State
	_, err = sm.LoadActorState(ctx, builtin.StoragePowerActorAddr, &spas, ts)
	if err != nil {
		return false, xerrors.Errorf("(get miner slashed) failed to load power actor state")
	}

	store := sm.cs.Store(ctx)
	claims := adt.AsMap(store, spas.Claims)
	ok, err := claims.Get(power.AddrKey(maddr), nil)
	if err != nil {
		return false, err
	}
	if !ok {
		return true, nil
	}

	return false, nil
}

func GetMinerFaults(ctx context.Context, sm *StateManager, ts *types.TipSet, maddr address.Address) ([]abi.SectorNumber, error) {
	var mas miner.State
	_, err := sm.LoadActorState(ctx, maddr, &mas, ts)
	if err != nil {
		return nil, xerrors.Errorf("(get ssize) failed to load miner actor state: %w", err)
	}

	ss, lerr := amt.LoadAMT(ctx, cbor.NewCborStore(sm.cs.Blockstore()), mas.Sectors)
	if lerr != nil {
		return nil, aerrors.HandleExternalError(lerr, "could not load proving set node")
	}

	faults, err := mas.FaultSet.All(2 * ss.Count)
	if err != nil {
		return nil, xerrors.Errorf("reading fault bit set: %w", err)
	}

	out := make([]abi.SectorNumber, len(faults))
	for i, fault := range faults {
		out[i] = abi.SectorNumber(fault)
	}

	return out, nil
}

func GetStorageDeal(ctx context.Context, sm *StateManager, dealId abi.DealID, ts *types.TipSet) (*api.MarketDeal, error) {
	var state market.State
	if _, err := sm.LoadActorState(ctx, builtin.StorageMarketActorAddr, &state, ts); err != nil {
		return nil, err
	}

	da, err := amt.LoadAMT(ctx, cbor.NewCborStore(sm.ChainStore().Blockstore()), state.Proposals)
	if err != nil {
		return nil, err
	}

	var dp market.DealProposal
	if err := da.Get(ctx, uint64(dealId), &dp); err != nil {
		return nil, err
	}

	sa := market.AsDealStateArray(sm.ChainStore().Store(ctx), state.States)
	st, err := sa.Get(dealId)
	if err != nil {
		return nil, err
	}

	return &api.MarketDeal{
		Proposal: dp,
		State:    *st,
	}, nil
}

func ListMinerActors(ctx context.Context, sm *StateManager, ts *types.TipSet) ([]address.Address, error) {
	var state power.State
	if _, err := sm.LoadActorState(ctx, builtin.StoragePowerActorAddr, &state, ts); err != nil {
		return nil, err
	}

	var miners []address.Address
	err := adt.AsMap(sm.cs.Store(ctx), state.Claims).ForEach(nil, func(k string) error {
		a, err := address.NewFromBytes([]byte(k))
		if err != nil {
			return err
		}
		miners = append(miners, a)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return miners, nil
}

func LoadSectorsFromSet(ctx context.Context, bs blockstore.Blockstore, ssc cid.Cid) ([]*api.ChainSectorInfo, error) {
	a, err := amt.LoadAMT(ctx, cbor.NewCborStore(bs), ssc)
	if err != nil {
		return nil, err
	}

	var sset []*api.ChainSectorInfo
	if err := a.ForEach(ctx, func(i uint64, v *cbg.Deferred) error {
		var oci miner.SectorOnChainInfo
		if err := cbor.DecodeInto(v.Raw, &oci); err != nil {
			return err
		}
		sset = append(sset, &api.ChainSectorInfo{
			Info: oci,
			ID:   abi.SectorNumber(i),
		})
		return nil
	}); err != nil {
		return nil, err
	}

	return sset, nil
}

func ComputeState(ctx context.Context, sm *StateManager, height abi.ChainEpoch, msgs []*types.Message, ts *types.TipSet) (cid.Cid, []*api.InvocResult, error) {
	if ts == nil {
		ts = sm.cs.GetHeaviestTipSet()
	}

	base, trace, err := sm.ExecutionTrace(ctx, ts)
	if err != nil {
		return cid.Undef, nil, err
	}

	fstate, err := sm.handleStateForks(ctx, base, height, ts.Height())
	if err != nil {
		return cid.Undef, nil, err
	}

	r := store.NewChainRand(sm.cs, ts.Cids(), height)
	vmi, err := vm.NewVM(fstate, height, r, builtin.SystemActorAddr, sm.cs.Blockstore(), sm.cs.VMSys())
	if err != nil {
		return cid.Undef, nil, err
	}

	for i, msg := range msgs {
		// TODO: Use the signed message length for secp messages
		ret, err := vmi.ApplyMessage(ctx, msg)
		if err != nil {
			return cid.Undef, nil, xerrors.Errorf("applying message %s: %w", msg.Cid(), err)
		}
		if ret.ExitCode != 0 {
			log.Infof("compute state apply message %d failed (exit: %d): %s", i, ret.ExitCode, ret.ActorErr)
		}
	}

	root, err := vmi.Flush(ctx)
	if err != nil {
		return cid.Undef, nil, err
	}

	return root, trace, nil
}

func MinerGetBaseInfo(ctx context.Context, sm *StateManager, tsk types.TipSetKey, maddr address.Address) (*api.MiningBaseInfo, error) {
	ts, err := sm.ChainStore().LoadTipSet(tsk)
	if err != nil {
		return nil, xerrors.Errorf("failed to load tipset for mining base: %w", err)
	}

	st, _, err := sm.TipSetState(ctx, ts)
	if err != nil {
		return nil, err
	}

	provset, err := getMinerProvingSetRaw(ctx, sm, st, maddr)
	if err != nil {
		return nil, xerrors.Errorf("failed to get proving set: %w", err)
	}

	mpow, tpow, err := getPowerRaw(ctx, sm, st, maddr)
	if err != nil {
		return nil, xerrors.Errorf("failed to get power: %w", err)
	}

	worker, err := GetMinerWorkerRaw(ctx, sm, st, maddr)
	if err != nil {
		return nil, xerrors.Errorf("failed to get miner worker: %w", err)
	}

	ssize, err := getMinerSectorSizeRaw(ctx, sm, st, maddr)
	if err != nil {
		return nil, xerrors.Errorf("failed to get miner sector size: %w", err)
	}

	prev, err := sm.ChainStore().GetLatestBeaconEntry(ts)
	if err != nil {
		return nil, xerrors.Errorf("failed to get latest beacon entry: %w", err)
	}

	return &api.MiningBaseInfo{
		MinerPower:      mpow,
		NetworkPower:    tpow,
		Sectors:         provset,
		Worker:          worker,
		SectorSize:      ssize,
		PrevBeaconEntry: *prev,
	}, nil
}
