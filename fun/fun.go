package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

const (
	listen = "localhost:1709"
)

func main() {

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/log", logHandler)

	fmt.Printf("listening on %s...\n", listen)
	log.Fatal(http.ListenAndServe(listen, nil))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Nothing to see here")
}

func logHandler(w http.ResponseWriter, r *http.Request) {
	b, _ := ioutil.ReadAll(r.Body)
	log.Printf("body: %q", b)
	fmt.Fprintf(w, "thanks!")
}
