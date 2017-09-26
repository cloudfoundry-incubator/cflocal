package main

import (
	"encoding/json"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/sclevine/cflocal/service"
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

	http.HandleFunc("/services", func(w http.ResponseWriter, r *http.Request) {
		vcapServices := map[string][]service.Service{}
		if err := json.Unmarshal([]byte(os.Getenv("VCAP_SERVICES")), &vcapServices); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%s\n", err)
			return
		}
		for _, services := range vcapServices {
			for _, service := range services {
				uri := service.Credentials["uri"].(string)
				fmt.Fprintf(w, "Name: %s\nURI: %s\n", service.Name, uri)
				req, err := http.NewRequest("GET", uri, nil)
				if err != nil {
					fmt.Fprintf(w, "Error: %s\n\n", err)
					continue
				}
				req.Host = service.Credentials["host_header"].(string)
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					fmt.Fprintf(w, "Error: %s\n\n", err)
					continue
				}
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					fmt.Fprintf(w, "Error: %s\n\n", err)
					continue
				}
				fmt.Fprintf(w, "Response: %s\n\n", body)
			}
		}
	})

	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), nil))
}
