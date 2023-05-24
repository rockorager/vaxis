package rtk

// Msg is a passable message conveying some event, data, or just a friendly
// hello
type Msg interface{}

// Init will always be the first Msg delivered
type Init struct{}

// Quit is delivered whenever the application is about to close
type Quit struct{}
