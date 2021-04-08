#! /bin/bash

export GOFLAGS=-tags=clientretrieve,nodaemon

if [ -z "$1" ]; then
  cd cmd/lotus; GOOS=js GOARCH=wasm go list $GOFLAGS -e -json -compiled=true -test=true -deps=true . | jq -C .
  exit
fi

# why ffi
#cd cmd/lotus; GOOS=js GOARCH=wasm go list $GOFLAGS -e -json -compiled=true -test=true -deps=true . | jq -C '. | select(.Imports) | select(.Imports[] | in({"github.com/filecoin-project/filecoin-ffi": ""})) | .ImportPath'

# why $1
cd cmd/lotus; GOOS=js GOARCH=wasm go list $GOFLAGS -e -json -compiled=true -test=true -deps=true . | jq -C ". | select(.Imports) | select(.Imports[] | select( . == \"$1\" )) | .ImportPath"
