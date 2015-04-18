package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/publicsuffix"
)

const (
	listen        = "localhost:1709"
	logFile       = "log.txt"
	blocklistFile = "blocklist.txt"
	ignoreFile    = "ignore.txt"
)

func main() {

	r := httprouter.New()
	r.GET("/", indexHandler)
	r.POST("/log", logHandler)
	r.POST("/block/:domain", blockHandler)
	r.POST("/ignore/:domain", ignoreHandler)
	r.GET("/list", listHandler)
	r.GET("/unblocked", statsHandler)
	r.GET("/unblocked/:domain", statsHandler)
	r.GET("/srclog/:domain", srclogHandler)

	fmt.Printf("listening on %s...\n", listen)
	log.Fatal(http.ListenAndServe(listen, r))
}

func indexHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Set("Content-type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "page_index", nil); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

type StatsDomain struct {
	Domain       string
	PublicSuffix string
	SrcDomains   stringCounts
	XMLHTTPs     stringCounts
	Images       stringCounts
	StyleSheets  stringCounts
	Scripts      stringCounts
	SubFrames    stringCounts
	Others       stringCounts
}
type StatsPage struct {
	Domains []StatsDomain
}

func logHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	b, _ := ioutil.ReadAll(r.Body)
	var e Entry
	if err := json.Unmarshal(b, &e); err != nil {
		log.Printf("json: %q: %v", b, err)
		return
	}

	f, err := os.OpenFile(logFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	c := csv.NewWriter(f)

	// log.Printf("body: %q", b)
	c.Write([]string{
		time.Now().UTC().Format(time.RFC3339),
		e.Action,
		e.Type,
		e.URL,
		e.TabURL,
	})
	c.Flush()
	fmt.Fprintf(w, "thanks!")
}

// blockHandler adds the domain to the block list
func blockHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	defer r.Body.Close()
	addDomainFile(blocklistFile, ps.ByName("domain"))
	fmt.Fprintf(w, "done")
}

// ignoreHandler adds the domain to the ignore list
func ignoreHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	defer r.Body.Close()
	addDomainFile(ignoreFile, ps.ByName("domain"))
	fmt.Fprintf(w, "done")
}

// listHandler returns the blocklist.
func listHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer r.Body.Close()

	bl, err := readDomainFile(blocklistFile)
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

// statsHandler has a page with all unblocked domains
func statsHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	dom := ps.ByName("domain")

	stat, f := filterCount()

	noFilters := r.URL.Query().Get("full") != ""

	// Maybe limit to single domain.
	if dom != "" {
		f = filterDomain(dom, f)
	}

	if !noFilters {
		// Blocklist
		if block, err := readDomainFile(blocklistFile); err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		} else {
			f = filterURL(block, f)
		}

		// Ignorelist
		if ign, err := readDomainFile(ignoreFile); err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		} else {
			f = filterURL(ign, f)
		}
	}

	if err := readLog(logFile, f); err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	page := toPageStats(stat)
	w.Header().Set("Content-type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "page_unblocked", map[string]interface{}{
		"filters": !noFilters,
		"stats":   page,
	}); err != nil {
		log.Print(err)
	}
}

// srclogHandler has a page with all requests from a source domain.
func srclogHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	dom := ps.ByName("domain")
	stat, f := filterCount()
	f = filterTabDomain(dom, f)

	if err := readLog(logFile, f); err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "page_log", map[string]interface{}{
		"subject": dom,
		"stats":   toPageStats(stat),
	}); err != nil {
		log.Print(err)
	}
}

func readDomainFile(filename string) ([]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
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

func addDomainFile(filename, d string) error {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	fmt.Fprintf(f, "%s\n", d)
	return nil
}

func toPageStats(stat DomainStats) StatsPage {
	st := make([]DomainStat, 0, len(stat))
	for _, s := range stat {
		st = append(st, s)
	}
	sort.Sort(sort.Reverse(BySrcCount(st)))

	page := StatsPage{}
	for _, s := range st {
		pubsuf, _ := publicsuffix.EffectiveTLDPlusOne(s.Domain)
		if pubsuf != "" {
			pubsuf = "." + pubsuf
		}
		page.Domains = append(page.Domains, StatsDomain{
			Domain:       s.Domain,
			PublicSuffix: pubsuf,
			SrcDomains:   orderMap(s.SrcDomains),
			XMLHTTPs:     orderMap(s.XMLHTTPs),
			Images:       orderMap(s.Images),
			StyleSheets:  orderMap(s.StyleSheets),
			Scripts:      orderMap(s.Scripts),
			SubFrames:    orderMap(s.SubFrames),
			Others:       orderMap(s.Others),
		})
	}
	return page
}
