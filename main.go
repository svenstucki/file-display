package main

import ()

var files = []File{
	{"./test", "test"},
}

func main() {
	// setup and start server
	disp := NewDisplay(":8000")
	disp.Files = files
	disp.Run()
}
