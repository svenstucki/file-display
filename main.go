package main

import (
	"golang.org/x/exp/inotify"
	"log"
)

const FN = "test"

func main() {
	watcher, err := inotify.NewWatcher()
	if err != nil {
		log.Panic(err)
	}

	err = watcher.Watch(FN)
	if err != nil {
		log.Panic(err)
	}

	for {
		select {
		case evt := <-watcher.Event:
			log.Println("Event: " + evt.String())
		case err := <-watcher.Error:
			log.Print("Error: ")
			log.Println(err)
		}
	}
}
