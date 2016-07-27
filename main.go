package main

import (
	"github.com/fsnotify/fsnotify"
	"html"
	"log"
	"net/http"
	"os"
)

const FN = "test"

func main() {
	// watcher setup
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Panic(err)
	}
	defer watcher.Close()

	// wait for file changes in background
	done := make(chan bool)
	go func() {
		for {
			select {
			case evt := <-watcher.Events:
				log.Println("Event: " + evt.String())
				log.Printf("Evt in file '%s', command: 0x%x\n", evt.Name, evt.Op)

			case err := <-watcher.Errors:
				log.Print("Error: ")
				log.Println(err)
			}
		}
	}()

	// add files to watch
	err = watcher.Add(FN)
	if err != nil {
		log.Panic(err)
	}

	// setup and start server
	server()

	<-done
}

func server() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got http request for: " + html.EscapeString(r.URL.Path))

		path := r.URL.Path[1:]
		if path == "" {
			path = "index.html"
		}

		// serve file with this name if it exists
		fn := "./html/" + path
		if _, err := os.Stat(fn); err == nil {
			// serve static content
			http.ServeFile(w, r, fn)
			return
		}

		// return 404 error
		httpErrorHandler(w, r, http.StatusNotFound)
	})

	go func() {
		log.Println("Starting server...")
		http.ListenAndServe(":8080", nil)
	}()
}

func httpErrorHandler(w http.ResponseWriter, r *http.Request, status int) {
	w.WriteHeader(status)
	log.Printf("Error serving '%s': %d", r.URL.Path, status)
}
