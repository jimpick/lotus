#! /bin/bash

set -e

export GOFLAGS=-tags=clientretrieve,nodaemon
#cd cmd/lotus; GOOS=js GOARCH=wasm go build $GOFLAGS -o lotus.wasm
pushd cmd/lotus
GOOS=js GOARCH=wasm go build -o lotus.wasm
popd
cp -f cmd/lotus/lotus.wasm ../../wasmer-js-browser/static/lotus.wasm
