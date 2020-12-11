package moduleapi

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/actors/builtin/paych"
	"github.com/filecoin-project/lotus/chain/types"
	cid "github.com/ipfs/go-cid"
)

type PaychModuleAPI interface {
	PaychGet(ctx context.Context, from, to address.Address, amt types.BigInt) (*api.ChannelInfo, error)
	PaychAvailableFunds(ctx context.Context, ch address.Address) (*api.ChannelAvailableFunds, error)
	PaychAvailableFundsByFromTo(ctx context.Context, from, to address.Address) (*api.ChannelAvailableFunds, error)
	PaychGetWaitReady(ctx context.Context, sentinel cid.Cid) (address.Address, error)
	PaychAllocateLane(ctx context.Context, ch address.Address) (uint64, error)
	PaychNewPayment(ctx context.Context, from, to address.Address, vouchers []api.VoucherSpec) (*api.PaymentInfo, error)
	PaychList(ctx context.Context) ([]address.Address, error)
	PaychStatus(ctx context.Context, pch address.Address) (*api.PaychStatus, error)
	PaychSettle(ctx context.Context, addr address.Address) (cid.Cid, error)
	PaychCollect(ctx context.Context, addr address.Address) (cid.Cid, error)
	PaychVoucherCheckValid(ctx context.Context, ch address.Address, sv *paych.SignedVoucher) error
	PaychVoucherCheckSpendable(ctx context.Context, ch address.Address, sv *paych.SignedVoucher, secret []byte, proof []byte) (bool, error)
	PaychVoucherAdd(ctx context.Context, ch address.Address, sv *paych.SignedVoucher, proof []byte, minDelta types.BigInt) (types.BigInt, error)
	PaychVoucherCreate(ctx context.Context, pch address.Address, amt types.BigInt, lane uint64) (*api.VoucherCreateResult, error)
	PaychVoucherList(ctx context.Context, pch address.Address) ([]*paych.SignedVoucher, error)
}
