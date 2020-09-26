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
)

const listenAddr = "127.0.0.1:1238"

var daemonCmd = &cli.Command{
	Name:  "daemon",
	Usage: "run retrieve api daemon",
	Action: func(cctx *cli.Context) error {
		var api api.Retrieve

		ctx := context.Background()

		// from lotus/daemon.go where it called node.New()
		// stop, err := New(ctx,
		_, err := New(ctx,
			RetrieveAPI(&api),

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
		Commands: []*cli.Command{
			daemonCmd,
		},
	}
	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}
