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

type App struct {
	rtk *RTK
}

// Creates a new full-screen application. Upon creation, an App will clear the
// screen and enter the alternate screen buffer.
func NewApp() (*App, error) {
	app := &App{}
	rtk, err := New()
	if err != nil {
		return nil, err
	}
	app.rtk = rtk
	app.rtk.HideCursor()
	app.rtk.EnterAltScreen()
	return app, nil
}

// Run runs an App. All Msgs will be delivered to the App's Update method. Run
// is an immediate mode UI, meaning each Msg will trigger a render of the UI.
func (app *App) Run(model Model) error {
	for msg := range app.rtk.Msgs() {
		if msg == nil {
			continue
		}
		switch msg := msg.(type) {
		case Quit:
			model.Update(msg)
			return nil
		default:
			model.Update(msg)
		}
		model.Draw(app.rtk.StdSurface())
		app.rtk.Render()
	}
	return nil
}

func (app *App) Close() {
	app.rtk.ExitAltScreen()
	app.rtk.Close()
}

func (app *App) PostMsg(msg Msg) {
	app.rtk.PostMsg(msg)
}

// Refresh instructs the application to force a full render on the next pass.
func (app *App) Refresh() {
	app.rtk.refresh = true
}
