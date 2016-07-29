package main

import (
	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
	"html"
	"log"
	"net/http"
	"os"
	"time"
)

type File struct {
	FsPath  string
	UrlPath string
}

type Display struct {
	Files []File

	server *http.Server
	mux    *http.ServeMux

	upgrader *websocket.Upgrader

	watcher *fsnotify.Watcher
}

var files = []File{
	{"./test", "test"},
}

func main() {
	done := make(chan bool)
	/*
		// wait for file changes in background
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
	*/

	// setup and start server
	disp := Display{}
	disp.Start()

	<-done
}

func (disp *Display) Start() {
	// watcher setup
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Panic(err)
	}
	disp.watcher = watcher

	// http server and mux instance
	mux := http.NewServeMux()
	disp.mux = mux
	disp.server = &http.Server{
		Addr:         ":8000",
		Handler:      mux,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
	}

	// websocket upgrader
	disp.upgrader = &websocket.Upgrader{
		ReadBufferSize:  2048,
		WriteBufferSize: 2048,
	}

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[HTTP] %s %s\n", r.Method, html.EscapeString(r.URL.Path))

		conn, err := disp.upgrader.Upgrade(w, r, nil)
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

	// serve static files from html/ folder
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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
		log.Printf("[404] File not found '%s'\n", path)
		w.WriteHeader(http.StatusNotFound)
	})

	go func() {
		log.Printf("Starting server on %s...", disp.server.Addr)
		disp.server.ListenAndServe()
	}()
}
