package term

type RedrawMsg struct {
	Term *Model
}

type ClosedMsg struct {
	Term  *Model
	Error error
}
