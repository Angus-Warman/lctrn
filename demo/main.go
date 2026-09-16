package main

import (
	_ "embed"
	"fmt"
	"net/http"
	"time"

	"github.com/Angus-Warman/lctrn"
)

//go:embed demo.html
var home []byte

func main() {
	app := lctrn.New().
		ServeBytesAtRoot(home).
		HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "pong %v", time.Now())
		})

	app.HandleFunc("/api/popup", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		go app.Popup("Hello World", "Test Popup")
	})

	// app.StartMaximised = true
	// app.StartFullscreen = true

	app.Launch()
}
