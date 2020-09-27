package impl

import (
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/node/impl/client-retrieve"
)

type RetrieveAPI struct {
	client.API
}

var _ api.Retrieve = &RetrieveAPI{}
