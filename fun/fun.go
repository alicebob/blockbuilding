package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"
)

const (
	listen        = "localhost:1709"
	logFile       = "log.txt"
	blocklistFile = "blocklist.txt"
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
	http.HandleFunc("/list", listHandler)
	http.HandleFunc("/domains", statsHandler)

	fmt.Printf("listening on %s...\n", listen)
	log.Fatal(http.ListenAndServe(listen, nil))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "text/html")
	if err := tmplIndex.Execute(w, nil); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

var tmplIndex = template.Must(template.New("index").Parse(`Stats:<br />
<a href="/domains">Domains by 3rdparty usage</a><br />
`))

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

// listHandler returns the blocklist.
func listHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	bl, err := readBlocklist()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-type", "application/json")
	js, err := json.Marshal(bl)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(js)
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "text/plain")
	stat, err := readStats()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Ordered by subdomain count:\n")
	st := make([]DomainStat, 0, len(stat))
	for _, s := range stat {
		st = append(st, s)
	}
	sort.Sort(sort.Reverse(BySrcCount(st)))
	for _, s := range st {
		fmt.Fprintf(w, "%s\n", s.Domain)
		pubsuf, err := publicsuffix.EffectiveTLDPlusOne(s.Domain)
		if err != nil {
			log.Printf("publicsuffix for %s: %s", s.Domain, err)
			pubsuf = "unknown"
		}
		fmt.Fprintf(w, "  - public suffix: %s\n", pubsuf)
		fmt.Fprintf(w, "  - domains:\n")
		for _, u := range orderMap(s.SrcDomains) {
			fmt.Fprintf(w, "    - %s (%d)\n", u.string, u.int)
		}
		fmt.Fprintf(w, "  - usage: ")
		if s.XMLHTTP > 0 {
			fmt.Fprintf(w, " xmlhttp: %d", s.XMLHTTP)
		}
		if s.Image > 0 {
			fmt.Fprintf(w, " image: %d", s.Image)
		}
		if s.StyleSheet > 0 {
			fmt.Fprintf(w, " stylesheet: %d", s.StyleSheet)
		}
		if s.Script > 0 {
			fmt.Fprintf(w, " script: %d", s.Script)
		}
		if s.SubFrame > 0 {
			fmt.Fprintf(w, " subFrame: %d", s.SubFrame)
		}
		if s.Other > 0 {
			fmt.Fprintf(w, " other: %d", s.Other)
		}
		fmt.Fprintf(w, "\n")
		fmt.Fprintf(w, "  - urls:\n")
		for _, u := range orderMap(s.URLs) {
			fmt.Fprintf(w, "    - %s (%d)\n", u.string, u.int)
		}
		fmt.Fprintf(w, "\n")
	}
}

type stringCount struct {
	string
	int
}
type stringCounts []stringCount

func (s stringCounts) Len() int      { return len(s) }
func (s stringCounts) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s stringCounts) Less(i, j int) bool {
	return s[i].int < s[j].int
}

func readBlocklist() ([]string, error) {
	f, err := os.Open(blocklistFile)
	if err != nil {
		return nil, err
	}
	var bl []string
	b := bufio.NewReader(f)
	for {
		l, err := b.ReadString('\n')
		if err == io.EOF {
			return bl, nil
		}
		if err != nil {
			return nil, err
		}
		l = strings.TrimSpace(strings.SplitN(l, "#", 2)[0])
		if len(l) > 0 {
			bl = append(bl, l)
		}
	}
}
