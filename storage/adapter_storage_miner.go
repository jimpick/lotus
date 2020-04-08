package storage

import (
	"bytes"
	"context"

	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/filecoin-project/specs-actors/actors/abi/big"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	"github.com/filecoin-project/specs-actors/actors/builtin/market"
	"github.com/filecoin-project/specs-actors/actors/builtin/miner"
	"github.com/filecoin-project/specs-actors/actors/crypto"
	"github.com/filecoin-project/specs-actors/actors/util/adt"

	"github.com/filecoin-project/lotus/api/apibstore"
	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/storage/sealing"
)

var _ sealing.SealingAPI = new(SealingAPIAdapter)

type SealingAPIAdapter struct {
	delegate storageMinerApi
}

func NewSealingAPIAdapter(api storageMinerApi) SealingAPIAdapter {
	return SealingAPIAdapter{delegate: api}
}

func (s SealingAPIAdapter) StateMinerSectorSize(ctx context.Context, maddr address.Address, tok sealing.TipSetToken) (abi.SectorSize, error) {
	tsk, err := types.TipSetKeyFromBytes(tok)
	if err != nil {
		return 0, xerrors.Errorf("failed to unmarshal TipSetToken to TipSetKey: %w", err)
	}

	return s.delegate.StateMinerSectorSize(ctx, maddr, tsk)
}

func (s SealingAPIAdapter) StateWaitMsg(ctx context.Context, mcid cid.Cid) (sealing.MsgLookup, error) {
	wmsg, err := s.delegate.StateWaitMsg(ctx, mcid)
	if err != nil {
		return sealing.MsgLookup{}, err
	}

	return sealing.MsgLookup{
		Receipt: sealing.MessageReceipt{
			ExitCode: wmsg.Receipt.ExitCode,
			Return:   wmsg.Receipt.Return,
			GasUsed:  wmsg.Receipt.GasUsed,
		},
		TipSetTok: wmsg.TipSet.Key().Bytes(),
		Height:    wmsg.TipSet.Height(),
	}, nil
}

func (s SealingAPIAdapter) StateComputeDataCommitment(ctx context.Context, maddr address.Address, sectorType abi.RegisteredProof, deals []abi.DealID, tok sealing.TipSetToken) (cid.Cid, error) {
	tsk, err := types.TipSetKeyFromBytes(tok)
	if err != nil {
		return cid.Undef, xerrors.Errorf("failed to unmarshal TipSetToken to TipSetKey: %w", err)
	}

	ccparams, err := actors.SerializeParams(&market.ComputeDataCommitmentParams{
		DealIDs:    deals,
		SectorType: sectorType,
	})
	if err != nil {
		return cid.Undef, xerrors.Errorf("computing params for ComputeDataCommitment: %w", err)
	}

	ccmt := &types.Message{
		To:       builtin.StorageMarketActorAddr,
		From:     maddr,
		Value:    types.NewInt(0),
		GasPrice: types.NewInt(0),
		GasLimit: 9999999999,
		Method:   builtin.MethodsMarket.ComputeDataCommitment,
		Params:   ccparams,
	}
	r, err := s.delegate.StateCall(ctx, ccmt, tsk)
	if err != nil {
		return cid.Undef, xerrors.Errorf("calling ComputeDataCommitment: %w", err)
	}
	if r.MsgRct.ExitCode != 0 {
		return cid.Undef, xerrors.Errorf("receipt for ComputeDataCommitment had exit code %d", r.MsgRct.ExitCode)
	}

	var c cbg.CborCid
	if err := c.UnmarshalCBOR(bytes.NewReader(r.MsgRct.Return)); err != nil {
		return cid.Undef, xerrors.Errorf("failed to unmarshal CBOR to CborCid: %w", err)
	}

	return cid.Cid(c), nil
}

func (s SealingAPIAdapter) StateSectorPreCommitInfo(ctx context.Context, maddr address.Address, sectorNumber abi.SectorNumber, tok sealing.TipSetToken) (*miner.SectorPreCommitOnChainInfo, error) {
	tsk, err := types.TipSetKeyFromBytes(tok)
	if err != nil {
		return nil, xerrors.Errorf("failed to unmarshal TipSetToken to TipSetKey: %w", err)
	}

	act, err := s.delegate.StateGetActor(ctx, maddr, tsk)
	if err != nil {
		return nil, xerrors.Errorf("handleSealFailed(%d): temp error: %+v", sectorNumber, err)
	}

	st, err := s.delegate.ChainReadObj(ctx, act.Head)
	if err != nil {
		return nil, xerrors.Errorf("handleSealFailed(%d): temp error: %+v", sectorNumber, err)
	}

	var state miner.State
	if err := state.UnmarshalCBOR(bytes.NewReader(st)); err != nil {
		return nil, xerrors.Errorf("handleSealFailed(%d): temp error: unmarshaling miner state: %+v", sectorNumber, err)
	}

	var pci miner.SectorPreCommitOnChainInfo
	precommits := adt.AsMap(store.ActorStore(ctx, apibstore.NewAPIBlockstore(s.delegate)), state.PreCommittedSectors)
	if _, err := precommits.Get(adt.UIntKey(uint64(sectorNumber)), &pci); err != nil {
		return nil, err
	}

	return &pci, nil
}

func (s SealingAPIAdapter) StateMarketStorageDeal(ctx context.Context, dealID abi.DealID, tok sealing.TipSetToken) (market.DealProposal, market.DealState, error) {
	tsk, err := types.TipSetKeyFromBytes(tok)
	if err != nil {
		return market.DealProposal{}, market.DealState{}, err
	}

	deal, err := s.delegate.StateMarketStorageDeal(ctx, dealID, tsk)
	if err != nil {
		return market.DealProposal{}, market.DealState{}, err
	}

	return deal.Proposal, deal.State, nil
}

func (s SealingAPIAdapter) SendMsg(ctx context.Context, from, to address.Address, method abi.MethodNum, value, gasPrice big.Int, gasLimit int64, params []byte) (cid.Cid, error) {
	msg := types.Message{
		To:       to,
		From:     from,
		Value:    value,
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Method:   method,
		Params:   params,
	}

	smsg, err := s.delegate.MpoolPushMessage(ctx, &msg)
	if err != nil {
		return cid.Undef, err
	}

	return smsg.Cid(), nil
}

func (s SealingAPIAdapter) ChainHead(ctx context.Context) (sealing.TipSetToken, abi.ChainEpoch, error) {
	head, err := s.delegate.ChainHead(ctx)
	if err != nil {
		return nil, 0, err
	}

	return head.Key().Bytes(), head.Height(), nil
}

func (s SealingAPIAdapter) ChainGetRandomness(ctx context.Context, tok sealing.TipSetToken, personalization crypto.DomainSeparationTag, randEpoch abi.ChainEpoch, entropy []byte) (abi.Randomness, error) {
	tsk, err := types.TipSetKeyFromBytes(tok)
	if err != nil {
		return nil, err
	}

	return s.delegate.ChainGetRandomness(ctx, tsk, personalization, randEpoch, entropy)
}

func (s SealingAPIAdapter) ChainReadObj(ctx context.Context, ocid cid.Cid) ([]byte, error) {
	return s.delegate.ChainReadObj(ctx, ocid)
}
