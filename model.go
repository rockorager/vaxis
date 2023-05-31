package rtk

// A Model represents the state of an application
type Model interface {
	// Update is called when a Msg is received. The Update method should
	// handle all Model mutations in order to maintain thread safety. The
	// Update method can post Msgs back into the queue by calling PostMsg.
	// If Paint() has not been called, the View method will be skipped. This
	// can allow Msgs to bubble up from a child Model to a Parent model by
	// not painting, and sending a Msg back up to the main model.
	//
	// Models may also PostCmds using PostCmd.
	Update(Msg)

	// Draw is called after Update. Draw draws the application state to
	// the Models' viewport.
	Draw(Surface)
}
