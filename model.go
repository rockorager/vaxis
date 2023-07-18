package vaxis

// A Model represents the state of an application
type Model interface {
	// Update is called when a Msg is received. The Update method should
	// handle all Model mutations in order to maintain thread safety. The
	// Update method can post Msgs back into the queue by calling PostMsg.
	//
	// Models may also PostCmds using PostCmd.
	Update(Msg)

	// Draw is called after Update. Draw draws the application state to
	// the provided Window.
	Draw(Window)
}
