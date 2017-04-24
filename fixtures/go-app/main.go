package main

import (
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Path: %s", html.EscapeString(r.URL.Path))
	})

	http.HandleFunc("/env", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, strings.Join(os.Environ(), "\n"))
	})

	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), nil))
}
