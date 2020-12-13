// +build clientretrieve

package apistruct

import (
	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/filecoin-project/lotus/api"
)

const (
	// When changing these, update docs/API.md too

	PermRead  auth.Permission = "read" // default
	PermWrite auth.Permission = "write"
	PermSign  auth.Permission = "sign"  // Use wallet keys for signing
	PermAdmin auth.Permission = "admin" // Manage permissions
)

var AllPermissions = []auth.Permission{PermRead, PermWrite, PermSign, PermAdmin}
var DefaultPerms = []auth.Permission{PermRead}

func PermissionedFullAPI(a api.FullNode) api.FullNode {
	var out FullNodeStruct
	auth.PermissionedProxy(AllPermissions, DefaultPerms, a, &out.Internal)
	auth.PermissionedProxy(AllPermissions, DefaultPerms, a, &out.CommonStruct.Internal)
	return &out
}

func PermissionedWalletAPI(a api.WalletAPI) api.WalletAPI {
	var out WalletStruct
	auth.PermissionedProxy(AllPermissions, DefaultPerms, a, &out.Internal)
	return &out
}
