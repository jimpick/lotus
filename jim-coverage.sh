#! /bin/bash

gotestsum -- -coverprofile=coverage.txt -coverpkg=github.com/filecoin-project/lotus/... ./cmd/lotus-retrieval-coverage/...
go tool cover -html=coverage.txt -o coverage.html
