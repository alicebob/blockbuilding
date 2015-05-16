package main

import (
	"encoding/csv"
	"io"
	"log"
	"net/url"
	"os"
)

type Entry struct {
	Action string `json:"action"`
	Type   string `json:"type"`
	URL    string `json:"url"`
	Reason string `json:"reason"`
	TabID  int    `json:"tabId"`
	TabURL string `json:"tab"`
}

type logFilter func(Entry, *url.URL)

func readLog(filename string, cb logFilter) error {
	f, err := os.Open(logFile)
	defer f.Close()
	if err != nil {
		return err
	}

	fr := csv.NewReader(f)
	for {
		r, err := fr.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}

		reqURL := r[3]
		u, err := url.Parse(reqURL)
		if err != nil {
			log.Printf("url %q: %s", reqURL, err)
			continue
		}
		cb(Entry{
			// r[0] is timestamp
			Action: r[1],
			Type:   r[2],
			URL:    reqURL,
			TabURL: r[4],
		}, u)
	}
	return nil
}
