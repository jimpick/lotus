// +build js

package cli

// FIXME: duplicated in cli/cmd.go

import (
	"context"
	"fmt"
	"os"
	"syscall/js"

	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/apistruct"
	"github.com/filecoin-project/lotus/node/repo"
)

var log = logging.Logger("cli")

const (
	metadataTraceContext = "traceContext"
)

// custom CLI error

type ErrCmdFailed struct {
	msg string
}

func (e *ErrCmdFailed) Error() string {
	return e.msg
}

func NewCliError(s string) error {
	return &ErrCmdFailed{s}
}

// ApiConnector returns API instance
type ApiConnector func() api.FullNode

// The flag passed on the command line with the listen address of the API
// server (only used by the tests)
func flagForAPI(t repo.RepoType) string {
	switch t {
	case repo.FullNode:
		return "api-url"
	case repo.StorageMiner:
		return "miner-api-url"
	case repo.Worker:
		return "worker-api-url"
	case repo.RetrieveAPI:
		return "retrieve-api-url"
	default:
		panic(fmt.Sprintf("Unknown repo type: %v", t))
	}
}

func flagForRepo(t repo.RepoType) string {
	switch t {
	case repo.FullNode:
		return "repo"
	case repo.StorageMiner:
		return "miner-repo"
	case repo.Worker:
		return "worker-repo"
	case repo.RetrieveAPI:
		return "retrieve-api-repo"
	default:
		panic(fmt.Sprintf("Unknown repo type: %v", t))
	}
}

func envForRepo(t repo.RepoType) string {
	switch t {
	case repo.FullNode:
		return "FULLNODE_API_INFO"
	case repo.StorageMiner:
		return "MINER_API_INFO"
	case repo.Worker:
		return "WORKER_API_INFO"
	case repo.RetrieveAPI:
		return "RETRIEVE_API_INFO"
	default:
		panic(fmt.Sprintf("Unknown repo type: %v", t))
	}
}

func envForRepoDeprecation(t repo.RepoType) string {
	switch t {
	case repo.FullNode:
		return "FULLNODE_API_INFO"
	case repo.StorageMiner:
		return "STORAGE_API_INFO"
	case repo.Worker:
		return "WORKER_API_INFO"
	case repo.RetrieveAPI:
		return "RETRIEVE_API_INFO"
	default:
		panic(fmt.Sprintf("Unknown repo type: %v", t))
	}
}

/*
func GetAPIInfo(ctx *cli.Context, t repo.RepoType) (cliutil.APIInfo, error) {
	// Check if there was a flag passed with the listen address of the API
	// server (only used by the tests)
	apiFlag := flagForAPI(t)
	if ctx.IsSet(apiFlag) {
		strma := ctx.String(apiFlag)
		strma = strings.TrimSpace(strma)

		return cliutil.APIInfo{Addr: strma}, nil
	}

	envKey := envForRepo(t)
	env, ok := os.LookupEnv(envKey)
	if !ok {
		// TODO remove after deprecation period
		envKey = envForRepoDeprecation(t)
		env, ok = os.LookupEnv(envKey)
		if ok {
			log.Warnf("Use deprecation env(%s) value, please use env(%s) instead.", envKey, envForRepo(t))
		}
	}
	if ok {
		return cliutil.ParseApiInfo(env), nil
	}

	repoFlag := flagForRepo(t)

	p, err := homedir.Expand(ctx.String(repoFlag))
	if err != nil {
		return cliutil.APIInfo{}, xerrors.Errorf("could not expand home dir (%s): %w", repoFlag, err)
	}

	r, err := repo.NewFS(p)
	if err != nil {
		return cliutil.APIInfo{}, xerrors.Errorf("could not open repo at path: %s; %w", p, err)
	}

	ma, err := r.APIEndpoint()
	if err != nil {
		return cliutil.APIInfo{}, xerrors.Errorf("could not get api endpoint: %w", err)
	}

	token, err := r.APIToken()
	if err != nil {
		log.Warnf("Couldn't load CLI token, capabilities may be limited: %v", err)
	}

	return cliutil.APIInfo{
		Addr:  ma.String(),
		Token: token,
	}, nil
}


func GetRawAPI(ctx *cli.Context, t repo.RepoType) (string, http.Header, error) {
	ainfo, err := GetAPIInfo(ctx, t)
	if err != nil {
		return "", nil, xerrors.Errorf("could not get API info: %w", err)
	}

	addr, err := ainfo.DialArgs()
	if err != nil {
		return "", nil, xerrors.Errorf("could not get DialArgs: %w", err)
	}

	return addr, ainfo.AuthHeader(), nil
}
*/

func GetAPI(ctx *cli.Context) (api.Common, jsonrpc.ClientCloser, error) {
	/*
		ti, ok := ctx.App.Metadata["repoType"]
		if !ok {
			log.Errorf("unknown repo type, are you sure you want to use GetAPI?")
			ti = repo.FullNode
		}
		t, ok := ti.(repo.RepoType)
		if !ok {
			log.Errorf("repoType type does not match the type of repo.RepoType")
		}

		if tn, ok := ctx.App.Metadata["testnode-full"]; ok {
			return tn.(api.FullNode), func() {}, nil
		}

		addr, headers, err := GetRawAPI(ctx, t)
		if err != nil {
			return nil, nil, err
		}

		return client.NewCommonRPC(ctx.Context, addr, headers)
	*/
	requestsForLotusHandler := js.Global().Get("requestsForLotusHandler")
	var fullNodeApi apistruct.FullNodeStruct
	closer, err := jsonrpc.NewJSMergeClient(context.Background(), requestsForLotusHandler, "Filecoin",
		[]interface{}{
			&fullNodeApi.CommonStruct.Internal,
			&fullNodeApi.Internal,
		})
	return &fullNodeApi, closer, err
}

func GetFullNodeAPI(ctx *cli.Context) (api.FullNode, jsonrpc.ClientCloser, error) {
	requestsForLotusHandler := js.Global().Get("requestsForLotusHandler")
	var fullNodeApi apistruct.FullNodeStruct
	closer, err := jsonrpc.NewJSMergeClient(context.Background(), requestsForLotusHandler, "Filecoin",
		[]interface{}{
			&fullNodeApi.CommonStruct.Internal,
			&fullNodeApi.Internal,
		})
	/*
		if err != nil {
			fmt.Printf("connecting with lotus failed: %s\n", err)
			panic(err)
		}
	*/
	/*
		if tn, ok := ctx.App.Metadata["testnode-full"]; ok {
			return tn.(api.FullNode), func() {}, nil
		}

		addr, headers, err := GetRawAPI(ctx, repo.FullNode)
		if err != nil {
			return nil, nil, err
		}

		return client.NewFullNodeRPC(ctx.Context, addr, headers)
	*/
	return &fullNodeApi, closer, err
}

func DaemonContext(cctx *cli.Context) context.Context {
	if mtCtx, ok := cctx.App.Metadata[metadataTraceContext]; ok {
		return mtCtx.(context.Context)
	}

	return context.Background()
}

// ReqContext returns context for cli execution. Calling it for the first time
// installs SIGTERM handler that will close returned context.
// Not safe for concurrent execution.
func ReqContext(cctx *cli.Context) context.Context {
	tCtx := DaemonContext(cctx)

	ctx, done := context.WithCancel(tCtx)
	sigChan := make(chan os.Signal, 2)
	go func() {
		<-sigChan
		done()
	}()
	// signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

	return ctx
}

var CommonCommands = []*cli.Command{
	netCmd,
	// authCmd,
	logCmd,
	waitApiCmd,
	fetchParamCmd,
	// pprofCmd,
	VersionCmd,
}

var Commands = []*cli.Command{
	WithCategory("basic", sendCmd),
	WithCategory("basic", walletCmd),
	// WithCategory("basic", clientCmd),
	WithCategory("basic", multisigCmd),
	WithCategory("basic", paychCmd),
	// WithCategory("developer", authCmd),
	// WithCategory("developer", mpoolCmd),
	WithCategory("developer", stateCmd),
	WithCategory("developer", chainCmd),
	WithCategory("developer", logCmd),
	WithCategory("developer", waitApiCmd),
	WithCategory("developer", fetchParamCmd),
	WithCategory("network", netCmd),
	WithCategory("network", syncCmd),
	// pprofCmd,
	VersionCmd,
}

func WithCategory(cat string, cmd *cli.Command) *cli.Command {
	cmd.Category = cat
	return cmd
}
