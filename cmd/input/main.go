package main

import (
	"git.sr.ht/~rockorager/rtk"
	"git.sr.ht/~rockorager/rtk/log"
)

func main() {
	log.SetLevel(log.LevelTrace)
	some, err := rtk.New()
	if err != nil {
		panic(err)
	}
	defer some.Close()
	for msg := range some.Msgs() {
		switch msg := msg.(type) {
		case rtk.Key:
			log.Infof("Key %s\r", msg)
			if msg.String() == "<c-c>" {
				return
			}
		}
	}
}
