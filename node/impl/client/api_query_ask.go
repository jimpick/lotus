package client

import (
	"go.uber.org/fx"

	"github.com/filecoin-project/go-fil-markets/storagemarket"

	"github.com/filecoin-project/lotus/node/modules/moduleapi"
)

type API struct {
	fx.In

	moduleapi.StateModuleAPI

	SMDealClient storagemarket.StorageClient
}
