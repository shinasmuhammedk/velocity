package app

// Bootstrap creates and initializes the application.
//
// It serves as the composition root of Velocity.
// All application dependencies are wired together here.
func Bootstrap() (*Container, error) {

	container, err := Startup()
	if err != nil {
		return nil, err
	}

	// --------------------------------------------------
	// Future Wiring
	// --------------------------------------------------
	//
	// Register repositories
	//
	// Register services
	//
	// Register HTTP handlers
	//
	// Register WebSocket hub
	//
	// Register background workers
	//
	// Register matching engine
	//

	return container, nil
}