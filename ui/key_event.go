package ui

func keyIsRelease(key Key) bool {
	return key.EventType == EventRelease
}
