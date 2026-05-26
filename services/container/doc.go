// Package container provides a hash-table-backed value store that can be
// attached to a context.Context.
//
// # Motivation
//
// Values placed on a context with context.WithValue form a linked list:
// each ctx.Value(k) call walks the chain from the innermost wrapper up to
// the root, comparing the requested key against every entry along the way.
// With N WithValue layers, a single lookup is O(N), and the cost is paid
// on every read. When a request scope accumulates many injected
// dependencies (config, logger, db, tracer, feature flags, ...), this
// overhead becomes both real and noisy in profiles.
//
// container collapses all of those entries into a single sync.Map living
// behind one context.Value slot, turning O(N) chain walks into O(1) map
// lookups regardless of how many values have been registered.
//
// # Usage
//
// Attach a Container once near the edge of a request and read/write
// through the package helpers:
//
//	ctx = container.WithContext(ctx)
//	ctx = container.StoreContext(ctx, "db", db)
//	ctx = container.StoreContext(ctx, "logger", logger)
//
//	db, _    := container.LoadContext[*sql.DB](ctx, "db")
//	logger, _ := container.LoadContext[*slog.Logger](ctx, "logger")
//
// WithContext is idempotent: if a Container is already attached to ctx,
// the same ctx is returned and no new map is allocated. Store*Context
// transparently calls WithContext on first use, so callers do not have to
// remember to seed the context up-front.
//
// # Keys
//
// Keys are plain strings. There is no built-in namespacing — callers
// should pick keys that are unlikely to collide across packages
// (a "<pkg>.<name>" convention works well). Strings were chosen over a
// typed key abstraction to keep the API trivially serializable, debuggable,
// and free of generic plumbing at every call site.
//
// # Generic helpers
//
// The Load/Store helpers are generic over the value type to remove the
// type assertion at call sites. Load returns (zero, false) when the key
// is absent. If the key exists but the stored value cannot be asserted
// to the requested type, Load panics: a type mismatch on a known key
// indicates the producer and consumer disagree about what is stored,
// which is a bug, not a runtime condition to recover from.
//
// # Provider
//
// Provider is the small interface that describes "something that can
// build a Container and that also knows how to register itself as a
// service into a context". DefaultProvider is the stock implementation;
// callers depending on a Provider can substitute their own (test
// doubles, decorating providers, etc.) without touching consumer code.
//
// Provider has two orthogonal responsibilities, exposed as two methods:
//
//   - New(ctx) ctx attaches a Container to ctx. The default
//     implementation delegates to WithContext, which is idempotent:
//     calling New twice on the same ctx returns the same Container.
//   - ContextInject(ctx) ctx satisfies the services.ContextInjector
//     contract and registers the Provider itself into ctx's service
//     registry. After this call, downstream code can retrieve the
//     Provider via services.Lookup and use it to build a Container.
//
// The two methods are deliberately separate. ContextInject is about the
// Provider being discoverable as a service; New is about the Provider
// doing its job (handing out a Container). A typical wiring at process
// start looks like:
//
//	p := &container.DefaultProvider{}
//	ctx = p.ContextInject(ctx)   // make the Provider lookup-able
//	ctx = p.New(ctx)             // attach a Container for this scope
//
// services.ContextInjector itself is the general "this service can
// inject itself into a context" contract — callers invoke
// ContextInject(ctx) to let the service write whatever bindings it
// needs into ctx. The services.InjectMe helper is just a convenient
// default body for ContextInject: it stores the receiver into the
// process-wide registry under the receiver's static type. Providers
// that need different placement semantics are free to write their own
// ContextInject body.
//
// # Concurrency
//
// The default Container is backed by sync.Map and is safe for concurrent
// Load and Store. Stores done after a context has been forked into
// children are visible to all holders of the same Container — the map is
// shared by reference, not copied. If isolation between scopes is needed,
// build a fresh Container with NewContainer and attach it explicitly.
package container
