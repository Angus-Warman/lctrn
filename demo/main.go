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
	app := lctrn.New().
		ServeBytesAtRoot(home).
		HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "pong %v", time.Now())
		})

	if err := app.Run(); err != nil {
		log.Fatalln(err)
	}
}
