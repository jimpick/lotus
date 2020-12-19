// +build js

package fsutil

import (
	"golang.org/x/xerrors"
)

func Statfs(path string) (FsStat, error) {
	return FsStat{}, xerrors.Errorf("Not available in WASM")
}
