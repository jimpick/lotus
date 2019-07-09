package cli

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"gopkg.in/urfave/cli.v2"
)

// All keys used in App.Metadata should be defined here
const (
	metadataContext = "context"
)

// reqContext returns context for cli execution. Calling it for the first time
// installs SIGTERM handler that will close returned context.
// Not safe for concurrent execution.
func reqContext(cctx *cli.Context) context.Context {
	if uctx, ok := cctx.App.Metadata[metadataContext]; ok {
		// unchecked cast as if somethign else is in there
		// it is crash worthy either way
		return uctx.(context.Context)
	}
	ctx, done := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 2)
	go func() {
		<-sigChan
		done()
	}()
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	return ctx
}

// Commands is the root group of CLI commands
var Commands = []*cli.Command{
	versionCmd,
}
