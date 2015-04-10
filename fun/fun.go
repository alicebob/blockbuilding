package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	listen  = "localhost:1709"
	logFile = "log.txt"
)

type Entry struct {
	Action string `json:"action"`
	Type   string `json:"type"`
	URL    string `json:"url"`
	Reason string `json:"reason"`
	TabID  int    `json:"tabId"`
	TabURL string `json:"tab"`
}

func main() {

	f, err := os.OpenFile(logFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	c := csv.NewWriter(f)
	defer c.Flush()

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/log", makeLogHandler(c))

	fmt.Printf("listening on %s...\n", listen)
	log.Fatal(http.ListenAndServe(listen, nil))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Nothing to see here")
}

func makeLogHandler(f *csv.Writer) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		b, _ := ioutil.ReadAll(r.Body)
		var e Entry
		if err := json.Unmarshal(b, &e); err != nil {
			log.Printf("json: %q: %v", b, err)
			return
		}
		// log.Printf("body: %q", b)
		f.Write([]string{
			time.Now().UTC().Format(time.RFC3339),
			e.Action,
			e.Type,
			e.URL,
			e.TabURL,
		})
		f.Flush()
		fmt.Fprintf(w, "thanks!")
	}
}
