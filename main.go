package main

import (
	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
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

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func server() {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[HTTP] %s %s\n", r.Method, html.EscapeString(r.URL.Path))

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("WebSocket upgrade error:")
			log.Println(err)
			return
		}

		go func() {
			for {
				messageType, p, err := conn.ReadMessage()
				if err != nil {
					log.Println("WebSocket read error:")
					log.Println(err)
					return
				}

				if err = conn.WriteMessage(messageType, p); err != nil {
					log.Println("WebSocket write error:")
					log.Println(err)
					return
				}
			}
		}()
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[HTTP] %s %s\n", r.Method, html.EscapeString(r.URL.Path))

		path := r.URL.Path[1:]
		if path == "" {
			path = "index.html"
		}

		// serve file with this name if it exists
		fn := "./html/" + path
		if _, err := os.Stat(fn); err == nil {
			// serve static content
			log.Printf("[OK] Serving static file '%s'\n", fn)
			http.ServeFile(w, r, fn)
			return
		}

		// return 404 error
		httpErrorHandler(w, r, http.StatusNotFound)
	})

	go func() {
		bind := ":8000"
		log.Printf("Starting server on %s...", bind)
		http.ListenAndServe(bind, nil)
	}()
}

func httpErrorHandler(w http.ResponseWriter, r *http.Request, status int) {
	w.WriteHeader(status)
	log.Printf("Error serving '%s': %d", r.URL.Path, status)
}
