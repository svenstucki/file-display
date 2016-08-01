package main

import (
	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type File struct {
	FsPath  string
	UrlPath string
}

type client struct {
	conn   *websocket.Conn
	writer chan interface{}
}

type Display struct {
	Files []File

	server *http.Server
	mux    *http.ServeMux

	upgrader *websocket.Upgrader

	watcher *fsnotify.Watcher

	clients []*client
	// TODO: mutex for clients
}

func NewDisplay(bind string) *Display {
	disp := &Display{}
	disp.clients = make([]*client, 0)

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
		Addr:         bind,
		Handler:      mux,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
	}

	// serve static files from html/ folder
	disp.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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

	// websocket upgrader
	disp.upgrader = &websocket.Upgrader{
		ReadBufferSize:  2048,
		WriteBufferSize: 2048,
	}

	// handle websocket connections
	disp.mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[HTTP] %s %s\n", r.Method, html.EscapeString(r.URL.Path))

		conn, err := disp.upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("WebSocket upgrade error:")
			log.Println(err)
			return
		}

		go disp.handleWebsocket(conn)
	})

	return disp
}

func (disp *Display) Run() {
	disp.setupFileWatchers()

	log.Printf("Starting server on %s...", disp.server.Addr)
	disp.server.ListenAndServe()
}

func (disp *Display) handleWebsocket(conn *websocket.Conn) {
	// Store connection handle
	c := &client{}
	c.conn = conn
	c.writer = make(chan interface{})
	// TODO: Use mutex for disp.clients
	disp.clients = append(disp.clients, c)

	// Handle concurrent writes to channel
	go func() {
		for v := range c.writer {
			log.Println("WriteJSON to ws:")
			log.Println(v)
			if err := conn.WriteJSON(v); err != nil {
				log.Println("WebSocket write error (retiring socket):")
				log.Println(err)
				disp.removeConnection(conn)
				return
			}
		}
	}()

	// Wait for and handle messages
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Println("WebSocket read error (retiring socket):")
			log.Println(err)
			disp.removeConnection(conn)
			return
		}

		log.Printf("Got websocket msg (type: %d): %s\n", messageType, p)
		c.writer <- string(p)
	}
}

func (disp *Display) removeConnection(conn *websocket.Conn) {
	// TODO: Use mutex for disp.clients
	found := false
	for i, v := range disp.clients {
		if conn == v.conn {
			disp.clients = append(disp.clients[:i], disp.clients[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		log.Println("Couldn't find connection handle in list")
		return
	}
}

func (disp *Display) setupFileWatchers() {
	// add files to watch
	for _, f := range disp.Files {
		err := disp.watcher.Add(f.FsPath)
		if err != nil {
			log.Panic(err)
		}

		go func() {
			for {
				select {
				case event := <-disp.watcher.Events:
					log.Println("Event in file " + event.String())

					// event.Op:
					// - WRITE	append or write to file
					// - REMOVE	file got deleted
					// - CHMOD	file got touched, chmod'ed, etc.
					// - RENAME	file got renamed
					//
					// Typical text editor save sequence:
					// RENAME, CHMOD, REMOVE (i.e. rename to '~file', stat, write to 'file')
					// -> Try to readd file after RENAME or REMOVE
					switch event.Op {
					case fsnotify.Write:
						// send update
						go disp.handleFileUpdate(event.Name)
					case fsnotify.Remove:
						fallthrough
					case fsnotify.Rename:
						// remove file from watcher
						if err := disp.watcher.Remove(event.Name); err != nil {
							log.Printf("Error removing '%s' from watcher\n", event.Name)
							break
						}

						go func() {
							// wait a short amount of time, then try to readd file to watcher
							time.Sleep(time.Millisecond * 100)

							if err := disp.watcher.Add(event.Name); err != nil {
								log.Printf("Error readding '%s' to watcher\n", event.Name)
								return
							}

							// send update for readded file
							disp.handleFileUpdate(event.Name)
						}()
					case fsnotify.Chmod:
						// ignore
					case fsnotify.Create:
						// ignore, shouldn't happen for files
					}

				case err := <-disp.watcher.Errors:
					log.Print("Watcher Error: ")
					log.Println(err)
				}
			}
		}()
	}
}

func (disp *Display) handleFileUpdate(fn string) {
	fn = filepath.Clean(fn)

	// find corresponding File
	var f *File
	for _, cf := range disp.Files {
		if filepath.Clean(cf.FsPath) == fn {
			f = &cf
			break
		}
	}
	if f == nil {
		log.Printf("Can't find file '%s' in list.\n", fn)
		return
	}

	// prepare update
	u := update{}
	u.File = f.UrlPath

	// read file
	fh, err := os.Open(fn)
	if err != nil {
		log.Printf("Error opening file '%s':\n", fn)
		log.Print(err)
		return
	}

	arr, err := ioutil.ReadAll(fh)
	if err != nil {
		log.Printf("Error reading file '%s':\n", fn)
		log.Print(err)
		return
	}
	u.Content = string(arr)

	// send update to all clients
	for _, c := range disp.clients {
		c.writer <- u
	}
}
