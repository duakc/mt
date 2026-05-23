// Package closeme provides a resource cleanup manager.
//
// A [Manager] collects [io.Closer] or [StopCloser] resources and
// closes them all at once — useful for graceful shutdown of global
// resources like database connections, file handles, or network
// listeners.
//
// Each registered pointer is used as its own key in the underlying
// map. This means multiple instances of the same type coexist
// without colliding, and the pointer reference keeps the value
// alive until [Manager.Close] is called.
//
// A resource must be registered only once. Registering the same
// pointer via both AddClose and AddStop will overwrite the previous
// entry, and only the last-registered cleanup method will run at
// shutdown.
//
// # Basic usage
//
//	mgr := closeme.NewManager()
//	defer mgr.Close()
//
//	db, _ := sql.Open("mysql", dsn1)
//	closeme.AddClose(mgr, db)
//
//	db2, _ := sql.Open("mysql", dsn2)
//	closeme.AddClose(mgr, db2) // same type, different pointer — both tracked
//
// # Concrete pointer types (AddClosePtr)
//
// Use [AddClosePtr] when only the pointer receiver implements
// [io.Closer]:
//
//	type session struct{ ... }
//	func (*session) Close() error { return nil }
//
//	s := &session{}
//	closeme.AddClosePtr(mgr, s)
//
// # Context-based dependency injection
//
// Store the Manager in the services DI container:
//
//	ctx := context.Background()
//	ctx = services.NewRegistry(ctx, services.NewDefaultRegistry())
//	services.Store(ctx, closeme.NewManager())
//
//	// ... elsewhere ...
//	mgr := services.Lookup[closeme.Manager](ctx)
//	if mgr != nil {
//	    closeme.AddClose(mgr, someResource)
//	}
//
// # Shutdown via services.LifeCycle
//
// Manager implements [io.Closer], so it integrates naturally:
//
//	services.CloseService(mgr)
package closeme
