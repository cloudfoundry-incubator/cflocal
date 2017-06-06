package main

import (
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	contents, err := ioutil.ReadFile("file")
	if err != nil {
		os.Exit(1)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Path: %s", html.EscapeString(r.URL.Path))
	})

	http.HandleFunc("/file", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s", contents)
	})

	http.HandleFunc("/env", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, strings.Join(os.Environ(), "\n"))
	})

	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), nil))
}
