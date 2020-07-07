// +build debug 8m

package build

import (
	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/filecoin-project/specs-actors/actors/abi/big"
	"github.com/filecoin-project/specs-actors/actors/builtin/miner"
	"github.com/filecoin-project/specs-actors/actors/builtin/power"
	"github.com/filecoin-project/specs-actors/actors/builtin/verifreg"
)

func init() {
	power.ConsensusMinerMinPower = big.NewInt(8388608)
	miner.SupportedProofTypes = map[abi.RegisteredSealProof]struct{}{
		abi.RegisteredSealProof_StackedDrg8MiBV1: {},
	}
	verifreg.MinVerifiedDealSize = big.NewInt(256)

	BuildType |= Build8m
}

// Seconds
const BlockDelaySecs = uint64(10)

const PropagationDelaySecs = uint64(5)

// SlashablePowerDelay is the number of epochs after ElectionPeriodStart, after
// which the miner is slashed
//
// Epochs
const SlashablePowerDelay = 400

// Epochs
const InteractivePoRepConfidence = 200
