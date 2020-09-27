package main

import (
	"context"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/node/impl"
	"github.com/filecoin-project/lotus/node/modules"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/lotus/node/repo"
	"go.uber.org/fx"
	"golang.org/x/xerrors"
)

// From node/builder.go

// special is a type used to give keys to modules which
//  can't really be identified by the returned type
type special struct{ id int }

type invoke int

// Invokes are called in the order they are defined.
//nolint:golint
const (
	// InitJournal at position 0 initializes the journal global var as soon as
	// the system starts, so that it's available for all other components.
	InitJournalKey = invoke(iota)

	// daemon
	ExtractApiKey

	_nInvokes // keep this last
)

type Settings struct {
	// modules is a map of constructors for DI
	//
	// In most cases the index will be a reflect. Type of element returned by
	// the constructor, but for some 'constructors' it's hard to specify what's
	// the return type should be (or the constructor returns fx group)
	modules map[interface{}]fx.Option

	// invokes are separate from modules as they can't be referenced by return
	// type, and must be applied in correct order
	invokes []fx.Option

	nodeType repo.RepoType

	Online bool // Online option applied
	Config bool // Config option applied

}

func Repo(r repo.Repo) Option {
	return func(settings *Settings) error {
		/*
			lr, err := r.Lock(settings.nodeType)
			if err != nil {
				return err
			}
			c, err := lr.Config()
			if err != nil {
				return err
			}
		*/

		return Options(
			// Override(new(repo.LockedRepo), modules.LockedRepo(lr)), // module handles closing

			// Override(new(dtypes.MetadataDS), modules.Datastore),
			// Override(new(dtypes.ChainBlockstore), modules.ChainBlockstore),

			// Override(new(dtypes.ClientImportMgr), modules.ClientImportMgr),
			// Override(new(dtypes.ClientMultiDstore), modules.ClientMultiDatastore),

			// Override(new(dtypes.ClientBlockstore), modules.ClientBlockstore),
			Override(new(dtypes.ClientRetrievalStoreManager), modules.ClientRetrievalStoreManager),
			// Override(new(ci.PrivKey), lp2p.PrivKey),
			// Override(new(ci.PubKey), ci.PrivKey.GetPublic),
			// Override(new(peer.ID), peer.IDFromPublicKey),

			// Override(new(types.KeyStore), modules.KeyStore),

			// Override(new(*dtypes.APIAlg), modules.APISecret),

			// ApplyIf(isType(repo.FullNode), ConfigFullNode(c)),
			// ApplyIf(isType(repo.StorageMiner), ConfigStorageMiner(c)),
		)(settings)
	}
}

// func FullAPI(out *api.Retrieve) Option {
func RetrieveAPI(out *api.Retrieve) Option {
	return Options(
		func(s *Settings) error {
			s.nodeType = repo.RetrieveAPI
			return nil
		},
		func(s *Settings) error {
			resAPI := &impl.RetrieveAPI{}
			s.invokes[ExtractApiKey] = fx.Populate(resAPI)
			*out = resAPI
			return nil
		},
	)
}

func defaults() []Option {
	return []Option{
		/*
			// global system journal.
			Override(new(journal.DisabledEvents), func() journal.DisabledEvents {
				if env, ok := os.LookupEnv(EnvJournalDisabledEvents); ok {
					if ret, err := journal.ParseDisabledEvents(env); err == nil {
						return ret
					}
				}
				// fallback if env variable is not set, or if it failed to parse.
				return journal.DefaultDisabledEvents
			}),
			Override(new(journal.Journal), modules.OpenFilesystemJournal),
			Override(InitJournalKey, func(j journal.Journal) {
				journal.J = j // eagerly sets the global journal through fx.Invoke.
			}),

			Override(new(helpers.MetricsCtx), context.Background),
			Override(new(record.Validator), modules.RecordValidator),
			Override(new(dtypes.Bootstrapper), dtypes.Bootstrapper(false)),
			Override(new(dtypes.ShutdownChan), make(chan struct{})),

			// Filecoin modules
		*/
	}
}

type StopFunc func(context.Context) error

// New builds and starts new Filecoin node
func New(ctx context.Context, opts ...Option) (StopFunc, error) {
	settings := Settings{
		modules: map[interface{}]fx.Option{},
		invokes: make([]fx.Option, _nInvokes),
	}

	// apply module options in the right order
	if err := Options(Options(defaults()...), Options(opts...))(&settings); err != nil {
		return nil, xerrors.Errorf("applying node options failed: %w", err)
	}

	// gather constructors for fx.Options
	ctors := make([]fx.Option, 0, len(settings.modules))
	for _, opt := range settings.modules {
		ctors = append(ctors, opt)
	}

	// fill holes in invokes for use in fx.Options
	for i, opt := range settings.invokes {
		if opt == nil {
			settings.invokes[i] = fx.Options()
		}
	}

	app := fx.New(
		fx.Options(ctors...),
		fx.Options(settings.invokes...),

		fx.NopLogger,
	)

	// TODO: we probably should have a 'firewall' for Closing signal
	//  on this context, and implement closing logic through lifecycles
	//  correctly
	if err := app.Start(ctx); err != nil {
		// comment fx.NopLogger few lines above for easier debugging
		return nil, xerrors.Errorf("starting node: %w", err)
	}

	return app.Stop, nil
}
