package repo

const (
	fsAPI           = "api"
	fsAPIToken      = "token"
	fsConfig        = "config.toml"
	fsStorageConfig = "storage.json"
	fsDatastore     = "datastore"
	fsLock          = "repo.lock"
	fsKeystore      = "keystore"
)

type RepoType int

const (
	_                 = iota // Default is invalid
	FullNode RepoType = iota
	StorageMiner
	Worker
	Wallet
	RetrieveAPI
)
