package build

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/filecoin-project/lotus/lib/addrutil"
	"golang.org/x/xerrors"

	rice "github.com/GeertJohan/go.rice"
	"github.com/libp2p/go-libp2p-core/peer"
)

func BuiltinBootstrap() ([]peer.AddrInfo, error) {
	fmt.Println("Jim BuiltinBootstrap")
	if DisableBuiltinAssets {
		return nil, nil
	}

	var out []peer.AddrInfo

	b := rice.MustFindBox("bootstrap")
	err := b.Walk("", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return xerrors.Errorf("failed to walk box: %w", err)
		}

		if !strings.HasSuffix(path, ".pi") {
			return nil
		}
		spi := b.MustString(path)
		if spi == "" {
			return nil
		}
		pi, err := addrutil.ParseAddresses(context.TODO(), strings.Split(strings.TrimSpace(spi), "\n"))
		out = append(out, pi...)
		extra := []string{
			"/ip4/192.168.240.129/tcp/34205/p2p/12D3KooWHsom4LPf85FaWUVxfuMaeSpaj5ikzLGb1T4Y3QydRBcG",
			"/ip4/192.168.240.129/tcp/44299/p2p/12D3KooWAJ1Hd3Mu1W5N35AyPeti1oQt5QYWnk584nKdUbk9wedq",
		}
		extraPeers, err := addrutil.ParseAddresses(context.TODO(), extra)
		out = append(out, extraPeers...)
		return err
	})
	return out, err
}
