package vaxis

// async is an asynchronous queue, provided as a helper for applications
var async = newQueue[Msg]()

func PostAsync(msg Msg) {
	async.push(msg)
}

func PollAsync() Msg {
	var m Msg
	for msg := range async.ch {
		if msg == nil {
			continue
		}
		m = msg
		break
	}
	return m
}

func ChanAsync() chan Msg {
	return async.Chan()
}
