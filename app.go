package rtk

//
//
// type App struct {
// 	rtk *RTK
// }
//
// // Creates a new full-screen application. Upon creation, an App will clear the
// // screen and enter the alternate screen buffer.
// func NewApp(opts *Options) (*App, error) {
// 	app := &App{}
// 	rtk, err := New(opts)
// 	if err != nil {
// 		return nil, err
// 	}
// 	app.rtk = rtk
// 	app.rtk.HideCursor()
// 	app.rtk.EnterAltScreen()
// 	return app, nil
// }
//
// // Run runs an App. All Msgs will be delivered to the App's Update method. Run
// // is an immediate mode UI, meaning each Msg will trigger a render of the UI.
// func (app *App) Run(model Model) error {
// 	for msg := range app.rtk.Msgs() {
// 		if msg == nil {
// 			continue
// 		}
// 		switch msg := msg.(type) {
// 		case Quit:
// 			model.Update(msg)
// 			return nil
// 		case sendMsg:
// 			msg.model.Update(msg)
// 		default:
// 			model.Update(msg)
// 		}
// 		model.Draw(app.rtk.StdSurface())
// 		app.rtk.Render()
// 	}
// 	return nil
// }
//
// func (app *App) Close() {
// 	app.rtk.Close()
// 	app.rtk.ExitAltScreen()
// }
//
// func (app *App) PostMsg(msg Msg) {
// 	app.rtk.PostMsg(msg)
// }
//
// func (app *App) SendMsg(msg Msg, model Model) {
// 	app.rtk.PostMsg(sendMsg{
// 		msg:   msg,
// 		model: model,
// 	})
// }
//
// // Refresh instructs the application to force a full render on the next pass.
// func (app *App) Refresh() {
// 	app.rtk.refresh = true
// }
