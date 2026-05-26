// Package closeme collects io.Closer and StopCloser resources behind a
// single Manager and tears them down on demand.
//
// # Why
//
// A long-lived program typically opens a handful of resources at start
// (database pools, file handles, listeners, background workers) and
// must release them all when it shuts down. Tracking those cleanups by
// hand — remembering every defer, propagating them through layers,
// joining their errors — is repetitive and easy to get wrong. closeme
// concentrates that work into one Manager.Close call.
//
// # Storage model
//
// A Manager keeps registrations as a flat slice of cleanup functions.
// Each AddClose / AddStop pushes one entry; Close pops them in LIFO
// order (last registered runs first) so cleanup naturally mirrors
// initialization, the way nested defers do. Close joins every
// individual error with errors.Join, and is idempotent — a second call
// is a no-op and does not re-fire registered cleanups. Registrations
// made after Close are silently dropped, which mirrors how a shutdown
// race would behave anyway.
//
// # Basic usage
//
//	mgr := closeme.NewManager()
//	defer mgr.Close()
//
//	db, _ := sql.Open("mysql", dsn)
//	closeme.AddClose(mgr, db)
//
//	worker := startWorker()
//	closeme.AddStop(mgr, worker) // worker exposes Stop() error
//
// When a value satisfies both io.Closer and StopCloser, Add dispatches
// to AddClose — Close wins, since it is the standard library contract
// and almost every resource that bothers to expose Stop also implements
// Close. Use AddStop explicitly when you want the Stop branch on such a
// value. Values that satisfy neither interface are dropped.
//
// # Global manager
//
// Default is a package-level Manager so the process can register and
// tear down resources without threading a *Manager through every
// constructor:
//
//	func main() {
//	    defer closeme.Default.Close()
//
//	    db := mustOpenDB()
//	    closeme.AddClose(closeme.Default, db)
//	    ...
//	}
//
// Default is an exported variable so tests can swap it for an isolated
// Manager. Always swap before the first Add* call in the test, and
// restore via t.Cleanup.
//
// # Context-based dependency injection
//
// Manager satisfies services.ContextInjector, so a Manager can register
// itself as a service for downstream code to discover:
//
//	mgr := closeme.NewManager()
//	ctx = mgr.ContextInject(ctx)
//
//	// elsewhere:
//	if m := services.Lookup[closeme.Manager](ctx); m != nil {
//	    closeme.AddClose(m, someResource)
//	}
//
// # Shutdown via services.LifeCycle
//
// Manager satisfies io.Closer, so services.CloseService(mgr) works:
//
//	services.CloseService(mgr)
package closeme
