// +build clientretrieve
// +build !js

package repo

import (
	"fmt"

	"github.com/filecoin-project/lotus/node/config"
)

func defConfForType(t RepoType) interface{} {
	switch t {
	case FullNode:
		return config.DefaultFullNode()
	case StorageMiner:
		return &struct{}{}
	case Worker:
		return &struct{}{}
	case Wallet:
		return &struct{}{}
	case RetrieveAPI:
		return &struct{}{}
	default:
		panic(fmt.Sprintf("unknown RepoType(%d)", int(t)))
	}
}
