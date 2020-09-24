package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/go-jsonrpc"
)

const listenAddr = "127.0.0.1:1238"

var daemonCmd = &cli.Command{
	Name:  "daemon",
	Usage: "run retrieve api daemon",
	Action: func(cctx *cli.Context) error {
		rpcServer := jsonrpc.NewServer()
		a := &api{}
		rpcServer.Register("Filecoin", a)

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
