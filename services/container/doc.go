// Package container provides a hash-table-backed value store that can
// be attached to a context.Context.
//
// # Motivation
//
// Values placed on a context with context.WithValue form a linked
// list: each ctx.Value(k) call walks the chain from the innermost
// wrapper up to the root, comparing the requested key against every
// entry along the way. With N WithValue layers, a single lookup is
// O(N), and the cost is paid on every read. When a request scope
// accumulates many injected dependencies (config, logger, db, tracer,
// feature flags, ...), this overhead becomes both real and noisy in
// profiles.
//
// container collapses all of those entries into a single sync.Map
// living behind one context.Value slot, turning O(N) chain walks into
// O(1) map lookups regardless of how many values have been registered.
//
// # Who creates the Container
//
// Library code in this package never creates a Container on the
// caller's behalf. A Container is only ever attached to a context by
// an explicit Provider.New call from the application wiring. The
// package-level helpers (StoreContext, LoadContext, StorePtrContext,
// LoadPtrContext) act on whichever Container is already on ctx; when
// none is attached they fall through to one of two behaviors driven
// by the debug build tag:
//
//   - debug.Enabled == true: panic with a message pointing at the
//     missing Provider.New call. This surfaces wiring mistakes
//     loudly in development.
//   - debug.Enabled == false (production): silent no-op. Load returns
//     (zero, false); Store / StorePtr drop the value. This trades
//     visibility for resilience — a request that somehow reached the
//     handler without a Container still serves rather than crashes.
//
// # Wiring
//
// Construct a Provider once at process start and attach a Container
// to each request scope through it:
//
//	provider := container.NewDefaultProvider(nil) // nil → NewContainer factory
//	ctx = provider.ContextInject(ctx)             // register provider in services
//
//	// per request:
//	ctx = provider.New(ctx)
//	defer provider.Release(ctx)
//	container.StoreContext(ctx, "db", db)
//	db, _ := container.LoadContext[*sql.DB](ctx, "db")
//
// Pass a non-nil Factory to NewDefaultProvider to swap in a custom
// Container implementation (e.g. one that pre-seeds defaults, traces
// accesses, or enforces additional invariants).
//
// # Pooling
//
// DefaultProvider holds an internal sync.Pool of Containers.
// Provider.New pulls from the pool (or invokes the Factory on a
// miss); Provider.Release calls Container.Reset to drop every entry
// and returns the Container to the pool. This keeps per-request
// allocations down for hot paths. Reset releases held references so
// previously-stored values become eligible for GC immediately.
//
// Two consequences:
//
//   - After Release, the ctx must be treated as exhausted for further
//     container operations — the underlying Container may already
//     belong to another scope.
//   - Skipping Release does not leak: the Container is just dropped
//     and garbage-collected the usual way. Pooling is an
//     optimization, not a correctness requirement.
//
// # Keys
//
// Keys are plain strings. There is no built-in namespacing — callers
// should pick keys that are unlikely to collide across packages
// (a "<pkg>.<name>" convention works well).
//
// # Generic helpers
//
// Load / Store helpers are generic over the value type so call sites
// avoid the type assertion themselves. Load returns (zero, false)
// when the key is absent; if the key exists but the stored value
// cannot be asserted to the requested type, Load panics — a type
// mismatch on a known key indicates the producer and consumer
// disagree about what is stored, which is a bug, not a runtime
// condition to recover from.
//
// # Provider
//
// Provider has two orthogonal responsibilities:
//
//   - New / Release manage the Container itself — pull from the
//     pool, attach to ctx, reset on the way back.
//   - ContextInject satisfies the services.ContextInjector contract
//     and registers the Provider itself into ctx's service registry
//     under its interface type, so downstream code can resolve it
//     via services.Lookup[container.Provider].
//
// The two methods are deliberately separate. ContextInject is about
// the Provider being discoverable as a service; New is about doing
// the Provider's actual job. Typical wiring at process start:
//
//	p := container.NewDefaultProvider(nil)
//	ctx = p.ContextInject(ctx) // make Provider lookup-able
//
// Per request:
//
//	p := services.Lookup[container.Provider](ctx)
//	ctx = p.New(ctx)
//	defer p.Release(ctx)
//
// # Concurrency
//
// The default Container is backed by sync.Map and is safe for
// concurrent Load and Store. The Provider's sync.Pool is also safe
// for concurrent New / Release.
package container
