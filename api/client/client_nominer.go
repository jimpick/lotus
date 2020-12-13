// +build clientretrieve

package client

import (
	"context"
	"net/http"

	"github.com/filecoin-project/go-jsonrpc"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/apistruct"
)

// NewCommonRPC creates a new http jsonrpc client.
func NewCommonRPC(ctx context.Context, addr string, requestHeader http.Header) (api.Common, jsonrpc.ClientCloser, error) {
	var res apistruct.CommonStruct
	closer, err := jsonrpc.NewMergeClient(ctx, addr, "Filecoin",
		[]interface{}{
			&res.Internal,
		},
		requestHeader,
	)

	return &res, closer, err
}

// NewFullNodeRPC creates a new http jsonrpc client.
func NewFullNodeRPC(ctx context.Context, addr string, requestHeader http.Header) (api.FullNode, jsonrpc.ClientCloser, error) {
	var res apistruct.FullNodeStruct
	closer, err := jsonrpc.NewMergeClient(ctx, addr, "Filecoin",
		[]interface{}{
			&res.CommonStruct.Internal,
			&res.Internal,
		}, requestHeader)

	return &res, closer, err
}

// NewGatewayRPC creates a new http jsonrpc client for a gateway node.
func NewGatewayRPC(ctx context.Context, addr string, requestHeader http.Header, opts ...jsonrpc.Option) (api.GatewayAPI, jsonrpc.ClientCloser, error) {
	var res apistruct.GatewayStruct
	closer, err := jsonrpc.NewMergeClient(ctx, addr, "Filecoin",
		[]interface{}{
			&res.Internal,
		},
		requestHeader,
		opts...,
	)

	return &res, closer, err
}

func NewWalletRPC(ctx context.Context, addr string, requestHeader http.Header) (api.WalletAPI, jsonrpc.ClientCloser, error) {
	var res apistruct.WalletStruct
	closer, err := jsonrpc.NewMergeClient(ctx, addr, "Filecoin",
		[]interface{}{
			&res.Internal,
		},
		requestHeader,
	)

	return &res, closer, err
}
