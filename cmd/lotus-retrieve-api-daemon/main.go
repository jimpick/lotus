package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/node/repo"
)

const listenAddr = "127.0.0.1:1238"

var daemonCmd = &cli.Command{
	Name:  "daemon",
	Usage: "run retrieve api daemon",
	Action: func(cctx *cli.Context) error {
		var api api.Retrieve

		ctx := context.Background()

		r, err := repo.NewFS(cctx.String("repo"))
		if err != nil {
			return xerrors.Errorf("opening fs repo: %w", err)
		}

		if err := r.Init(repo.RetrieveAPI); err != nil && err != repo.ErrRepoExists {
			return xerrors.Errorf("repo init error: %w", err)
		}

		// from lotus/daemon.go where it called node.New()
		// stop, err := New(ctx,
		_, err = New(ctx,
			RetrieveAPI(&api),
			Repo(r),

			/*
				node.Override(new(dtypes.Bootstrapper), isBootstrapper),
				node.Override(new(dtypes.ShutdownChan), shutdownChan),
				node.Online(),
				node.Repo(r),
				genesis,

				node.ApplyIf(func(s *node.Settings) bool { return cctx.IsSet("api") },
					node.Override(node.SetApiEndpointKey, func(lr repo.LockedRepo) error {
						apima, err := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/" +
							cctx.String("api"))
						if err != nil {
							return err
						}
						return lr.SetAPIEndpoint(apima)
					})),
				node.ApplyIf(func(s *node.Settings) bool { return !cctx.Bool("bootstrap") },
					node.Unset(node.RunPeerMgrKey),
					node.Unset(new(*peermgr.PeerMgr)),
				),
			*/
		)
		if err != nil {
			return xerrors.Errorf("initializing node: %w", err)
		}
		rpcServer := jsonrpc.NewServer()
		rpcServer.Register("Filecoin", api)

		http.Handle("/rpc/v0", rpcServer)

		fmt.Printf("Listening on http://%s\n", listenAddr)
		return http.ListenAndServe(listenAddr, nil)
	},
}

func main() {
	app := &cli.App{
		Name: "lotus-retrieve-api-daemon",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "repo",
				EnvVars: []string{"LOTUS_RETRIEVAL_PATH"},
				Hidden:  true,
				Value:   "~/.lotusretrieval", // TODO: Consider XDG_DATA_HOME
			},
		},
		Commands: []*cli.Command{
			daemonCmd,
		},
	}
	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}
