// +build js

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/filecoin-project/lotus/chain/actors/builtin"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multihash"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/filecoin-project/lotus/api"
	lapi "github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
)

var stateCmd = &cli.Command{
	Name:  "state",
	Usage: "Interact with and query filecoin chain state",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "tipset",
			Usage: "specify tipset to call method on (pass comma separated array of cids)",
		},
	},
	Subcommands: []*cli.Command{
		statePowerCmd,
		stateSectorsCmd,
		stateActiveSectorsCmd,
		stateListActorsCmd,
		stateListMinersCmd,
		stateCircSupplyCmd,
		stateSectorCmd,
		stateGetActorCmd,
		stateLookupIDCmd,
		stateSectorSizeCmd,
		stateReadStateCmd,
		stateListMessagesCmd,
		stateGetDealSetCmd,
		stateSearchMsgCmd,
		stateMinerInfo,
		stateMarketCmd,
		stateExecTraceCmd,
		stateNtwkVersionCmd,
	},
}

var stateMinerInfo = &cli.Command{
	Name:      "miner-info",
	Usage:     "Retrieve miner information",
	ArgsUsage: "[minerAddress]",
	Action: func(cctx *cli.Context) error {
		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		if !cctx.Args().Present() {
			return fmt.Errorf("must specify miner to get information for")
		}

		addr, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return err
		}

		ts, err := LoadTipSet(ctx, cctx, api)
		if err != nil {
			return err
		}

		mi, err := api.StateMinerInfo(ctx, addr, ts.Key())
		if err != nil {
			return err
		}

		availableBalance, err := api.StateMinerAvailableBalance(ctx, addr, ts.Key())
		if err != nil {
			return xerrors.Errorf("getting miner available balance: %w", err)
		}
		fmt.Printf("Available Balance: %s\n", types.FIL(availableBalance))
		fmt.Printf("Owner:\t%s\n", mi.Owner)
		fmt.Printf("Worker:\t%s\n", mi.Worker)
		for i, controlAddress := range mi.ControlAddresses {
			fmt.Printf("Control %d: \t%s\n", i, controlAddress)
		}

		fmt.Printf("PeerID:\t%s\n", mi.PeerId)
		fmt.Printf("Multiaddrs:\t")
		for _, addr := range mi.Multiaddrs {
			a, err := multiaddr.NewMultiaddrBytes(addr)
			if err != nil {
				return xerrors.Errorf("undecodable listen address: %w", err)
			}
			fmt.Printf("%s ", a)
		}
		fmt.Println()
		fmt.Printf("Consensus Fault End:\t%d\n", mi.ConsensusFaultElapsed)

		fmt.Printf("SectorSize:\t%s (%d)\n", types.SizeStr(types.NewInt(uint64(mi.SectorSize))), mi.SectorSize)
		pow, err := api.StateMinerPower(ctx, addr, ts.Key())
		if err != nil {
			return err
		}

		rpercI := types.BigDiv(types.BigMul(pow.MinerPower.RawBytePower, types.NewInt(1000000)), pow.TotalPower.RawBytePower)
		qpercI := types.BigDiv(types.BigMul(pow.MinerPower.QualityAdjPower, types.NewInt(1000000)), pow.TotalPower.QualityAdjPower)

		fmt.Printf("Byte Power:   %s / %s (%0.4f%%)\n",
			color.BlueString(types.SizeStr(pow.MinerPower.RawBytePower)),
			types.SizeStr(pow.TotalPower.RawBytePower),
			float64(rpercI.Int64())/10000)

		fmt.Printf("Actual Power: %s / %s (%0.4f%%)\n",
			color.GreenString(types.DeciStr(pow.MinerPower.QualityAdjPower)),
			types.DeciStr(pow.TotalPower.QualityAdjPower),
			float64(qpercI.Int64())/10000)

		fmt.Println()

		cd, err := api.StateMinerProvingDeadline(ctx, addr, ts.Key())
		if err != nil {
			return xerrors.Errorf("getting miner info: %w", err)
		}

		fmt.Printf("Proving Period Start:\t%s\n", EpochTime(cd.CurrentEpoch, cd.PeriodStart))

		return nil
	},
}

func ParseTipSetString(ts string) ([]cid.Cid, error) {
	strs := strings.Split(ts, ",")

	var cids []cid.Cid
	for _, s := range strs {
		c, err := cid.Parse(strings.TrimSpace(s))
		if err != nil {
			return nil, err
		}
		cids = append(cids, c)
	}

	return cids, nil
}

func LoadTipSet(ctx context.Context, cctx *cli.Context, api api.FullNode) (*types.TipSet, error) {
	tss := cctx.String("tipset")
	if tss == "" {
		return nil, nil
	}

	return ParseTipSetRef(ctx, api, tss)
}

func ParseTipSetRef(ctx context.Context, api api.FullNode, tss string) (*types.TipSet, error) {
	if tss[0] == '@' {
		if tss == "@head" {
			return api.ChainHead(ctx)
		}

		var h uint64
		if _, err := fmt.Sscanf(tss, "@%d", &h); err != nil {
			return nil, xerrors.Errorf("parsing height tipset ref: %w", err)
		}

		return api.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(h), types.EmptyTSK)
	}

	cids, err := ParseTipSetString(tss)
	if err != nil {
		return nil, err
	}

	if len(cids) == 0 {
		return nil, nil
	}

	k := types.NewTipSetKey(cids...)
	ts, err := api.ChainGetTipSet(ctx, k)
	if err != nil {
		return nil, err
	}

	return ts, nil
}

var statePowerCmd = &cli.Command{
	Name:      "power",
	Usage:     "Query network or miner power",
	ArgsUsage: "[<minerAddress> (optional)]",
	Action: func(cctx *cli.Context) error {
		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		var maddr address.Address
		if cctx.Args().Present() {
			maddr, err = address.NewFromString(cctx.Args().First())
			if err != nil {
				return err
			}
		}

		ts, err := LoadTipSet(ctx, cctx, api)
		if err != nil {
			return err
		}

		power, err := api.StateMinerPower(ctx, maddr, ts.Key())
		if err != nil {
			return err
		}

		tp := power.TotalPower
		if cctx.Args().Present() {
			mp := power.MinerPower
			percI := types.BigDiv(types.BigMul(mp.QualityAdjPower, types.NewInt(1000000)), tp.QualityAdjPower)
			fmt.Printf("%s(%s) / %s(%s) ~= %0.4f%%\n", mp.QualityAdjPower.String(), types.SizeStr(mp.QualityAdjPower), tp.QualityAdjPower.String(), types.SizeStr(tp.QualityAdjPower), float64(percI.Int64())/10000)
		} else {
			fmt.Printf("%s(%s)\n", tp.QualityAdjPower.String(), types.SizeStr(tp.QualityAdjPower))
		}

		return nil
	},
}

var stateSectorsCmd = &cli.Command{
	Name:      "sectors",
	Usage:     "Query the sector set of a miner",
	ArgsUsage: "[minerAddress]",
	Action: func(cctx *cli.Context) error {
		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		if !cctx.Args().Present() {
			return fmt.Errorf("must specify miner to list sectors for")
		}

		maddr, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return err
		}

		ts, err := LoadTipSet(ctx, cctx, api)
		if err != nil {
			return err
		}

		sectors, err := api.StateMinerSectors(ctx, maddr, nil, ts.Key())
		if err != nil {
			return err
		}

		for _, s := range sectors {
			fmt.Printf("%d: %x\n", s.SectorNumber, s.SealedCID)
		}

		return nil
	},
}

var stateActiveSectorsCmd = &cli.Command{
	Name:      "active-sectors",
	Usage:     "Query the active sector set of a miner",
	ArgsUsage: "[minerAddress]",
	Action: func(cctx *cli.Context) error {
		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		if !cctx.Args().Present() {
			return fmt.Errorf("must specify miner to list sectors for")
		}

		maddr, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return err
		}

		ts, err := LoadTipSet(ctx, cctx, api)
		if err != nil {
			return err
		}

		sectors, err := api.StateMinerActiveSectors(ctx, maddr, ts.Key())
		if err != nil {
			return err
		}

		for _, s := range sectors {
			fmt.Printf("%d: %x\n", s.SectorNumber, s.SealedCID)
		}

		return nil
	},
}

var stateExecTraceCmd = &cli.Command{
	Name:      "exec-trace",
	Usage:     "Get the execution trace of a given message",
	ArgsUsage: "<messageCid>",
	Action: func(cctx *cli.Context) error {
		if !cctx.Args().Present() {
			return ShowHelp(cctx, fmt.Errorf("must pass message cid"))
		}

		mcid, err := cid.Decode(cctx.Args().First())
		if err != nil {
			return fmt.Errorf("message cid was invalid: %s", err)
		}

		capi, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		msg, err := capi.ChainGetMessage(ctx, mcid)
		if err != nil {
			return err
		}

		lookup, err := capi.StateSearchMsg(ctx, mcid)
		if err != nil {
			return err
		}

		ts, err := capi.ChainGetTipSet(ctx, lookup.TipSet)
		if err != nil {
			return err
		}

		pts, err := capi.ChainGetTipSet(ctx, ts.Parents())
		if err != nil {
			return err
		}

		cso, err := capi.StateCompute(ctx, pts.Height(), nil, pts.Key())
		if err != nil {
			return err
		}

		var trace *api.InvocResult
		for _, t := range cso.Trace {
			if t.Msg.From == msg.From && t.Msg.Nonce == msg.Nonce {
				trace = t
				break
			}
		}
		if trace == nil {
			return fmt.Errorf("failed to find message in tipset trace output")
		}

		out, err := json.MarshalIndent(trace, "", "  ")
		if err != nil {
			return err
		}

		fmt.Println(string(out))
		return nil
	},
}

var stateGetDealSetCmd = &cli.Command{
	Name:      "get-deal",
	Usage:     "View on-chain deal info",
	ArgsUsage: "[dealId]",
	Action: func(cctx *cli.Context) error {
		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		if !cctx.Args().Present() {
			return fmt.Errorf("must specify deal ID")
		}

		dealid, err := strconv.ParseUint(cctx.Args().First(), 10, 64)
		if err != nil {
			return xerrors.Errorf("parsing deal ID: %w", err)
		}

		ts, err := LoadTipSet(ctx, cctx, api)
		if err != nil {
			return err
		}

		deal, err := api.StateMarketStorageDeal(ctx, abi.DealID(dealid), ts.Key())
		if err != nil {
			return err
		}

		data, err := json.MarshalIndent(deal, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))

		return nil
	},
}

var stateListMinersCmd = &cli.Command{
	Name:  "list-miners",
	Usage: "list all miners in the network",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "sort-by",
			Usage: "criteria to sort miners by (none, num-deals)",
		},
	},
	Action: func(cctx *cli.Context) error {
		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		ts, err := LoadTipSet(ctx, cctx, api)
		if err != nil {
			return err
		}

		miners, err := api.StateListMiners(ctx, ts.Key())
		if err != nil {
			return err
		}

		switch cctx.String("sort-by") {
		case "num-deals":
			ndm, err := getDealsCounts(ctx, api)
			if err != nil {
				return err
			}

			sort.Slice(miners, func(i, j int) bool {
				return ndm[miners[i]] > ndm[miners[j]]
			})

			for i := 0; i < 50 && i < len(miners); i++ {
				fmt.Printf("%s %d\n", miners[i], ndm[miners[i]])
			}
			return nil
		default:
			return fmt.Errorf("unrecognized sorting order")
		case "", "none":
		}

		for _, m := range miners {
			fmt.Println(m.String())
		}

		return nil
	},
}

func getDealsCounts(ctx context.Context, lapi api.FullNode) (map[address.Address]int, error) {
	allDeals, err := lapi.StateMarketDeals(ctx, types.EmptyTSK)
	if err != nil {
		return nil, err
	}

	out := make(map[address.Address]int)
	for _, d := range allDeals {
		if d.State.SectorStartEpoch != -1 {
			out[d.Proposal.Provider]++
		}
	}

	return out, nil
}

var stateListActorsCmd = &cli.Command{
	Name:  "list-actors",
	Usage: "list all actors in the network",
	Action: func(cctx *cli.Context) error {
		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		ts, err := LoadTipSet(ctx, cctx, api)
		if err != nil {
			return err
		}

		actors, err := api.StateListActors(ctx, ts.Key())
		if err != nil {
			return err
		}

		for _, a := range actors {
			fmt.Println(a.String())
		}

		return nil
	},
}

var stateGetActorCmd = &cli.Command{
	Name:      "get-actor",
	Usage:     "Print actor information",
	ArgsUsage: "[actorrAddress]",
	Action: func(cctx *cli.Context) error {
		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		if !cctx.Args().Present() {
			return fmt.Errorf("must pass address of actor to get")
		}

		addr, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return err
		}

		ts, err := LoadTipSet(ctx, cctx, api)
		if err != nil {
			return err
		}

		a, err := api.StateGetActor(ctx, addr, ts.Key())
		if err != nil {
			return err
		}

		strtype := builtin.ActorNameByCode(a.Code)

		fmt.Printf("Address:\t%s\n", addr)
		fmt.Printf("Balance:\t%s\n", types.FIL(a.Balance))
		fmt.Printf("Nonce:\t\t%d\n", a.Nonce)
		fmt.Printf("Code:\t\t%s (%s)\n", a.Code, strtype)
		fmt.Printf("Head:\t\t%s\n", a.Head)

		return nil
	},
}

var stateLookupIDCmd = &cli.Command{
	Name:      "lookup",
	Usage:     "Find corresponding ID address",
	ArgsUsage: "[address]",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "reverse",
			Aliases: []string{"r"},
			Usage:   "Perform reverse lookup",
		},
	},
	Action: func(cctx *cli.Context) error {
		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		if !cctx.Args().Present() {
			return fmt.Errorf("must pass address of actor to get")
		}

		addr, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return err
		}

		ts, err := LoadTipSet(ctx, cctx, api)
		if err != nil {
			return err
		}

		var a address.Address
		if !cctx.Bool("reverse") {
			a, err = api.StateLookupID(ctx, addr, ts.Key())
		} else {
			a, err = api.StateAccountKey(ctx, addr, ts.Key())
		}

		if err != nil {
			return err
		}

		fmt.Printf("%s\n", a)

		return nil
	},
}

var stateSectorSizeCmd = &cli.Command{
	Name:      "sector-size",
	Usage:     "Look up miners sector size",
	ArgsUsage: "[minerAddress]",
	Action: func(cctx *cli.Context) error {
		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		if !cctx.Args().Present() {
			return fmt.Errorf("must pass miner's address")
		}

		addr, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return err
		}

		ts, err := LoadTipSet(ctx, cctx, api)
		if err != nil {
			return err
		}

		mi, err := api.StateMinerInfo(ctx, addr, ts.Key())
		if err != nil {
			return err
		}

		fmt.Printf("%s (%d)\n", types.SizeStr(types.NewInt(uint64(mi.SectorSize))), mi.SectorSize)
		return nil
	},
}

var stateReadStateCmd = &cli.Command{
	Name:      "read-state",
	Usage:     "View a json representation of an actors state",
	ArgsUsage: "[actorAddress]",
	Action: func(cctx *cli.Context) error {
		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		if !cctx.Args().Present() {
			return fmt.Errorf("must pass address of actor to get")
		}

		addr, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return err
		}

		ts, err := LoadTipSet(ctx, cctx, api)
		if err != nil {
			return err
		}

		as, err := api.StateReadState(ctx, addr, ts.Key())
		if err != nil {
			return err
		}

		data, err := json.MarshalIndent(as.State, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))

		return nil
	},
}

var stateListMessagesCmd = &cli.Command{
	Name:  "list-messages",
	Usage: "list messages on chain matching given criteria",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "to",
			Usage: "return messages to a given address",
		},
		&cli.StringFlag{
			Name:  "from",
			Usage: "return messages from a given address",
		},
		&cli.Uint64Flag{
			Name:  "toheight",
			Usage: "don't look before given block height",
		},
		&cli.BoolFlag{
			Name:  "cids",
			Usage: "print message CIDs instead of messages",
		},
	},
	Action: func(cctx *cli.Context) error {
		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		var toa, froma address.Address
		if tos := cctx.String("to"); tos != "" {
			a, err := address.NewFromString(tos)
			if err != nil {
				return fmt.Errorf("given 'to' address %q was invalid: %w", tos, err)
			}
			toa = a
		}

		if froms := cctx.String("from"); froms != "" {
			a, err := address.NewFromString(froms)
			if err != nil {
				return fmt.Errorf("given 'from' address %q was invalid: %w", froms, err)
			}
			froma = a
		}

		toh := abi.ChainEpoch(cctx.Uint64("toheight"))

		ts, err := LoadTipSet(ctx, cctx, api)
		if err != nil {
			return err
		}

		if ts == nil {
			head, err := api.ChainHead(ctx)
			if err != nil {
				return err
			}
			ts = head
		}

		windowSize := abi.ChainEpoch(100)

		cur := ts
		for cur.Height() > toh {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			end := toh
			if cur.Height()-windowSize > end {
				end = cur.Height() - windowSize
			}

			msgs, err := api.StateListMessages(ctx, &lapi.MessageMatch{To: toa, From: froma}, cur.Key(), end)
			if err != nil {
				return err
			}

			for _, c := range msgs {
				if cctx.Bool("cids") {
					fmt.Println(c.String())
					continue
				}

				m, err := api.ChainGetMessage(ctx, c)
				if err != nil {
					return err
				}
				b, err := json.MarshalIndent(m, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(b))
			}

			if end <= 0 {
				break
			}

			next, err := api.ChainGetTipSetByHeight(ctx, end-1, cur.Key())
			if err != nil {
				return err
			}

			cur = next
		}

		return nil
	},
}

type callMeta struct {
	types.ExecutionTrace
	Subcall bool
	Hash    string
}

func call(e types.ExecutionTrace, subcall bool, hash string) callMeta {
	return callMeta{
		ExecutionTrace: e,
		Subcall:        subcall,
		Hash:           hash,
	}
}

func codeStr(c cid.Cid) string {
	cmh, err := multihash.Decode(c.Hash())
	if err != nil {
		panic(err)
	}
	return string(cmh.Digest)
}

func toFil(f types.BigInt) types.FIL {
	return types.FIL(f)
}

func isSlow(t time.Duration) bool {
	return t > 10*time.Millisecond
}

func isVerySlow(t time.Duration) bool {
	return t > 50*time.Millisecond
}

func sumGas(changes []*types.GasTrace) types.GasTrace {
	var out types.GasTrace
	for _, gc := range changes {
		out.TotalGas += gc.TotalGas
		out.ComputeGas += gc.ComputeGas
		out.StorageGas += gc.StorageGas

		out.TotalVirtualGas += gc.TotalVirtualGas
		out.VirtualComputeGas += gc.VirtualComputeGas
		out.VirtualStorageGas += gc.VirtualStorageGas
	}

	return out
}

var stateSearchMsgCmd = &cli.Command{
	Name:      "search-msg",
	Usage:     "Search to see whether a message has appeared on chain",
	ArgsUsage: "[messageCid]",
	Action: func(cctx *cli.Context) error {
		if !cctx.Args().Present() {
			return fmt.Errorf("must specify message cid to search for")
		}

		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		msg, err := cid.Decode(cctx.Args().First())
		if err != nil {
			return err
		}

		mw, err := api.StateSearchMsg(ctx, msg)
		if err != nil {
			return err
		}

		if mw != nil {
			fmt.Printf("message was executed in tipset: %s", mw.TipSet.Cids())
			fmt.Printf("\nExit Code: %d", mw.Receipt.ExitCode)
			fmt.Printf("\nGas Used: %d", mw.Receipt.GasUsed)
			fmt.Printf("\nReturn: %x", mw.Receipt.Return)
		} else {
			fmt.Print("message was not found on chain")
		}
		return nil
	},
}

func formatOutput(t string, val []byte) (string, error) {
	switch t {
	case "raw", "hex":
		return fmt.Sprintf("%x", val), nil
	case "address", "addr", "a":
		a, err := address.NewFromBytes(val)
		if err != nil {
			return "", err
		}
		return a.String(), nil
	case "big", "int", "bigint":
		bi := types.BigFromBytes(val)
		return bi.String(), nil
	case "fil":
		bi := types.FIL(types.BigFromBytes(val))
		return bi.String(), nil
	case "pid", "peerid", "peer":
		pid, err := peer.IDFromBytes(val)
		if err != nil {
			return "", err
		}

		return pid.Pretty(), nil
	case "auto":
		if len(val) == 0 {
			return "", nil
		}

		a, err := address.NewFromBytes(val)
		if err == nil {
			return "address: " + a.String(), nil
		}

		pid, err := peer.IDFromBytes(val)
		if err == nil {
			return "peerID: " + pid.Pretty(), nil
		}

		bi := types.BigFromBytes(val)
		return "bigint: " + bi.String(), nil
	default:
		return "", fmt.Errorf("unrecognized output type: %q", t)
	}
}

var stateCircSupplyCmd = &cli.Command{
	Name:  "circulating-supply",
	Usage: "Get the exact current circulating supply of Filecoin",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "vm-supply",
			Usage: "calculates the approximation of the circulating supply used internally by the VM (instead of the exact amount)",
			Value: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		ts, err := LoadTipSet(ctx, cctx, api)
		if err != nil {
			return err
		}

		if cctx.IsSet("vm-supply") {
			circ, err := api.StateVMCirculatingSupplyInternal(ctx, ts.Key())
			if err != nil {
				return err
			}

			fmt.Println("Circulating supply: ", types.FIL(circ.FilCirculating))
			fmt.Println("Mined: ", types.FIL(circ.FilMined))
			fmt.Println("Vested: ", types.FIL(circ.FilVested))
			fmt.Println("Burnt: ", types.FIL(circ.FilBurnt))
			fmt.Println("Locked: ", types.FIL(circ.FilLocked))
		} else {
			circ, err := api.StateCirculatingSupply(ctx, ts.Key())
			if err != nil {
				return err
			}

			fmt.Println("Exact circulating supply: ", types.FIL(circ))
			return nil
		}

		return nil
	},
}

var stateSectorCmd = &cli.Command{
	Name:      "sector",
	Usage:     "Get miner sector info",
	ArgsUsage: "[miner address] [sector number]",
	Action: func(cctx *cli.Context) error {
		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		if cctx.Args().Len() != 2 {
			return xerrors.Errorf("expected 2 params")
		}

		ts, err := LoadTipSet(ctx, cctx, api)
		if err != nil {
			return err
		}

		if ts == nil {
			ts, err = api.ChainHead(ctx)
			if err != nil {
				return err
			}
		}

		maddr, err := address.NewFromString(cctx.Args().Get(0))
		if err != nil {
			return err
		}

		sid, err := strconv.ParseInt(cctx.Args().Get(1), 10, 64)
		if err != nil {
			return err
		}

		si, err := api.StateSectorGetInfo(ctx, maddr, abi.SectorNumber(sid), ts.Key())
		if err != nil {
			return err
		}
		if si == nil {
			return xerrors.Errorf("sector %d for miner %s not found", sid, maddr)
		}

		fmt.Println("SectorNumber: ", si.SectorNumber)
		fmt.Println("SealProof: ", si.SealProof)
		fmt.Println("SealedCID: ", si.SealedCID)
		fmt.Println("DealIDs: ", si.DealIDs)
		fmt.Println()
		fmt.Println("Activation: ", EpochTime(ts.Height(), si.Activation))
		fmt.Println("Expiration: ", EpochTime(ts.Height(), si.Expiration))
		fmt.Println()
		fmt.Println("DealWeight: ", si.DealWeight)
		fmt.Println("VerifiedDealWeight: ", si.VerifiedDealWeight)
		fmt.Println("InitialPledge: ", types.FIL(si.InitialPledge))
		fmt.Println("ExpectedDayReward: ", types.FIL(si.ExpectedDayReward))
		fmt.Println("ExpectedStoragePledge: ", types.FIL(si.ExpectedStoragePledge))
		fmt.Println()

		sp, err := api.StateSectorPartition(ctx, maddr, abi.SectorNumber(sid), ts.Key())
		if err != nil {
			return err
		}

		fmt.Println("Deadline: ", sp.Deadline)
		fmt.Println("Partition: ", sp.Partition)

		return nil
	},
}

var stateMarketCmd = &cli.Command{
	Name:  "market",
	Usage: "Inspect the storage market actor",
	Subcommands: []*cli.Command{
		stateMarketBalanceCmd,
	},
}

var stateMarketBalanceCmd = &cli.Command{
	Name:  "balance",
	Usage: "Get the market balance (locked and escrowed) for a given account",
	Action: func(cctx *cli.Context) error {
		if !cctx.Args().Present() {
			return ShowHelp(cctx, fmt.Errorf("must specify address to print market balance for"))
		}

		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		ts, err := LoadTipSet(ctx, cctx, api)
		if err != nil {
			return err
		}

		addr, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return err
		}

		balance, err := api.StateMarketBalance(ctx, addr, ts.Key())
		if err != nil {
			return err
		}

		fmt.Printf("Escrow: %s\n", types.FIL(balance.Escrow))
		fmt.Printf("Locked: %s\n", types.FIL(balance.Locked))

		return nil
	},
}

var stateNtwkVersionCmd = &cli.Command{
	Name:  "network-version",
	Usage: "Returns the network version",
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Present() {
			return ShowHelp(cctx, fmt.Errorf("doesn't expect any arguments"))
		}

		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		ts, err := LoadTipSet(ctx, cctx, api)
		if err != nil {
			return err
		}

		nv, err := api.StateNetworkVersion(ctx, ts.Key())
		if err != nil {
			return err
		}

		fmt.Printf("Network Version: %d\n", nv)

		return nil
	},
}
