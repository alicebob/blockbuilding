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

	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/publicsuffix"
)

const (
	listen        = "localhost:1709"
	logFile       = "log.txt"
	blocklistFile = "blocklist.txt"
	ignoreFile    = "ignore.txt"
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

	r := httprouter.New()
	r.GET("/", indexHandler)
	r.POST("/log", logHandler)
	r.POST("/block/:domain", blockHandler)
	r.POST("/ignore/:domain", ignoreHandler)
	r.GET("/list", listHandler)
	r.GET("/domains", statsHandler)

	fmt.Printf("listening on %s...\n", listen)
	log.Fatal(http.ListenAndServe(listen, r))
}

func indexHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Set("Content-type", "text/html")
	if err := tmplIndex.Execute(w, nil); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

var tmplIndex = template.Must(template.New("index").Parse(`Stats:<br />
<a href="/domains">Domains by 3rdparty usage</a><br />
`))

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

var tmplStats = template.Must(template.New("stat").Parse(`
<script>
function block(domain) {
    var req = new XMLHttpRequest();
    req.open('POST', "/block/" + domain);
    req.send(null);
}
function ignore(domain) {
	if (confirm("Add " + domain + " to ignore list? (undo by editing ./ignore.txt)")) {
		var req = new XMLHttpRequest();
		req.open('POST', "/ignore/" + domain);
		req.send(null);
	}
}
</script>
Ordered by subdomain count:<br />
<br />
{{range .Domains}}
	<b>{{.Domain}}</b>
			<a href="#" onclick="block({{.Domain}}); return false">block</a>
			<a href="#" onclick="ignore({{.Domain}}); return false">ignore</a>
		<br />
	<ul>
	{{if .PublicSuffix}}
		<li>suffix: {{.PublicSuffix}}
				<a href="#" onclick="block({{.PublicSuffix}}); return false">block</a>
				<a href="#" onclick="ignore({{.PublicSuffix}}); return false">ignore</a>
		</li>
	{{end}}

	<li>used on domains:
		<ul>
		{{range .SrcDomains}}
		<li>{{.String}} ({{.Count}})</li>
		{{end}}
		</ul>
	</li>

	<li>usage:
		<ul>
		{{if .XMLHTTPs}}<li>xmlhttps: {{len .XMLHTTPs}}<br />
			<ul>
				{{range .XMLHTTPs}}
				<li>{{.String}} ({{.Count}})</li>
				{{end}}
			</ul>
			</li>
		{{end}}
		{{if .Images}}<li>images: {{len .Images}}<br />
			<ul>
				{{range .Images}}
				<li><a href="{{.String}}" target="_blank">{{.String}}</a> ({{.Count}})</li>
				{{end}}
			</ul>
			</li>
		{{end}}
		{{if .StyleSheets}}<li>stylesheets: {{len .StyleSheets}}<br />
			<ul>
				{{range .StyleSheets}}
				<li>{{.String}} ({{.Count}})</li>
				{{end}}
			</ul>
			</li>
		{{end}}
		{{if .Scripts}}<li>scripts: {{len .Scripts}}<br />
			<ul>
				{{range .Scripts}}
				<li>{{.String}} ({{.Count}})</li>
				{{end}}
			</ul>
			</li>
		{{end}}
		{{if .SubFrames}}<li>subframes: {{len .SubFrames}}<br />
			<ul>
				{{range .SubFrames}}
				<li>{{.String}} ({{.Count}})</li>
				{{end}}
			</ul>
			</li>
		{{end}}
		{{if .Others}}<li>others: {{len .Others}}<br />
			<ul>
				{{range .Others}}
				<li>{{.String}} ({{.Count}})</li>
				{{end}}
			</ul>
			</li>
		{{end}}
		</ul>
	</li>
	</ul>

	<br />
{{end}}
`))

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
func statsHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	block, err := readDomainFile(blocklistFile)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ign, err := readDomainFile(ignoreFile)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	stat, err := readStats(block, ign)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	page := StatsPage{}

	st := make([]DomainStat, 0, len(stat))
	for _, s := range stat {
		st = append(st, s)
	}
	sort.Sort(sort.Reverse(BySrcCount(st)))
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
	w.Header().Set("Content-type", "text/html")
	if err := tmplStats.Execute(w, page); err != nil {
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
