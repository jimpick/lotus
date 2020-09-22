#! /bin/bash

export GOFLAGS+=-tags=2k

(
	echo Waiting for API...
	lotus wait-api
	./jim-connect.sh
	./jim-cli-test-retrieval.sh
) &

gotestsum -- -coverprofile=coverage.txt -coverpkg=github.com/filecoin-project/lotus/... ./cmd/lotus-retrieval-coverage/...
go tool cover -html=coverage.txt -o coverage.html
