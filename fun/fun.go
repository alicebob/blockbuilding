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
	"net/url"
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

type DomainStat struct {
	Domain     string
	URLs       map[string]int
	SrcDomains map[string]int
	XMLHTTP    int
	Image      int
	StyleSheet int
	Script     int
	SubFrame   int
	Other      int
}
type DomainStats map[string]DomainStat

func main() {
	var (
		allowed = DomainStats{}
		blocked = DomainStats{}
	)

	f, err := os.OpenFile(logFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	c := csv.NewWriter(f)
	defer c.Flush()

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/log", makeLogHandler(allowed, blocked, c))
	http.HandleFunc("/list", listHandler)
	http.HandleFunc("/stats/allowed", makeStatsHandler(allowed))
	http.HandleFunc("/stats/blocked", makeStatsHandler(blocked))

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
<a href="/stats/blocked">Blocked</a><br />
<a href="/stats/allowed">Allowed</a><br />
`))

func makeLogHandler(allowed, blocked DomainStats, f *csv.Writer) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		b, _ := ioutil.ReadAll(r.Body)
		var e Entry
		if err := json.Unmarshal(b, &e); err != nil {
			log.Printf("json: %q: %v", b, err)
			return
		}
		switch e.Action {
		case "allow":
			allowed.Count(e)
		case "block":
			blocked.Count(e)
		default:
			log.Printf("unknown action: %s", e.Action)
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

func makeStatsHandler(stat DomainStats) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-type", "text/plain")
		fmt.Fprintf(w, "Ordered by subdomain count:\n")
		st := make([]DomainStat, 0, len(stat))
		for _, s := range stat {
			st = append(st, s)
		}
		sort.Sort(sort.Reverse(BySrcCount(st)))
		for _, s := range st {
			fmt.Fprintf(w, "%s\n", s.Domain)
			eff, err := publicsuffix.EffectiveTLDPlusOne(s.Domain)
			if err != nil {
				panic(err)
			}
			fmt.Fprintf(w, "  - eff: %s\n", eff)
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
}

//TODO: lock
func (s DomainStats) Count(e Entry) {
	rURL, err := url.Parse(e.URL)
	if err != nil {
		panic(err)
	}
	if rURL.Scheme == "chrome-extension" {
		return
	}

	tURL, err := url.Parse(e.TabURL)
	if err != nil {
		panic(err)
	}
	if rURL.Host == tURL.Host {
		// same domain. Not interesting.
		return
	}

	d := s[rURL.Host]

	d.Domain = rURL.Host
	switch e.Type {
	case "xmlhttprequest":
		d.XMLHTTP++
	case "image":
		d.Image++
	case "stylesheet":
		d.StyleSheet++
	case "script":
		d.Script++
	case "sub_frame":
		d.SubFrame++
	case "other":
		d.Other++
	default:
		panic(e.Type)
	}

	if d.SrcDomains == nil {
		d.SrcDomains = map[string]int{}
	}
	d.SrcDomains[tURL.Host]++

	if d.URLs == nil {
		d.URLs = map[string]int{}
	}
	d.URLs[e.URL]++

	s[rURL.Host] = d
}

type BySrcCount []DomainStat

func (s BySrcCount) Len() int      { return len(s) }
func (s BySrcCount) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s BySrcCount) Less(i, j int) bool {
	if len(s[i].SrcDomains) != len(s[j].SrcDomains) {
		return len(s[i].SrcDomains) < len(s[j].SrcDomains)
	}
	return s[i].Image < s[j].Image
}

// convert a string count map to a string lost, order by count desc.
func orderMap(m map[string]int) stringCounts {
	r := make(stringCounts, 0, len(m))
	for k, c := range m {
		r = append(r, stringCount{k, c})
	}
	sort.Sort(r)
	return r
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
