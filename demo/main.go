package main

import (
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Angus-Warman/lctrn"
)

//go:embed demo.html
var home []byte

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write(home)
	})
	mux.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "pong %v", time.Now())
	})

	app := lctrn.FromMux(mux)

	if err := app.Run(); err != nil {
		log.Fatalln(err)
	}
}
