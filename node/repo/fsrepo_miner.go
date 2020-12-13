// +build !clientretrieve

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
		return config.DefaultStorageMiner()
	case Worker:
		return &struct{}{}
	case Wallet:
		return &struct{}{}
	default:
		panic(fmt.Sprintf("unknown RepoType(%d)", int(t)))
	}
}
