package main

import (
	"log"
	"net/url"
	"sort"
	"strings"
)

type DomainStat struct {
	Domain      string
	URL         int
	SrcDomains  map[string]int
	XMLHTTPs    map[string]int
	Images      map[string]int
	StyleSheets map[string]int
	Scripts     map[string]int
	SubFrames   map[string]int
	Others      map[string]int
}
type DomainStats map[string]DomainStat

// number of distinct urls.
func (d DomainStat) URLcount() int {
	return len(d.XMLHTTPs) +
		len(d.Images) +
		len(d.StyleSheets) +
		len(d.Scripts) +
		len(d.SubFrames) +
		len(d.Others)
}

// sink filter.
func filterCount() (DomainStats, logFilter) {
	s := DomainStats{}
	return s, func(e Entry) {
		s.Count(e)
	}
}

// filter everything which was logged as blocked.
func filterBlocked(next logFilter) logFilter {
	return func(e Entry) {
		if e.Action == "block" {
			return
		}
		next(e)
	}
}

func filterURL(urls []string, next logFilter) logFilter {
	return func(e Entry) {
		for _, i := range urls {
			if matchesDomain(e.URL, i) {
				return
			}
		}
		next(e)
	}
}

func filterDomain(dom string, next logFilter) logFilter {
	return func(e Entry) {
		r, err := url.Parse(e.URL)
		if err != nil {
			log.Printf("%q: %s", e.URL, err)
			return
		}
		if r.Host != dom {
			return
		}
		next(e)
	}
}

func filterTabDomain(dom string, next logFilter) logFilter {
	return func(e Entry) {
		r, err := url.Parse(e.TabURL)
		if err != nil {
			log.Printf("%q: %s", e.TabURL, err)
			return
		}
		if r.Host != dom {
			return
		}
		next(e)
	}
}

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

	d, ok := s[rURL.Host]
	if !ok {
		d = DomainStat{
			SrcDomains:  map[string]int{},
			XMLHTTPs:    map[string]int{},
			Images:      map[string]int{},
			StyleSheets: map[string]int{},
			Scripts:     map[string]int{},
			SubFrames:   map[string]int{},
			Others:      map[string]int{},
		}
	}

	d.Domain = rURL.Host
	switch e.Type {
	case "xmlhttprequest":
		d.XMLHTTPs[e.URL]++
	case "image":
		d.Images[e.URL]++
	case "stylesheet":
		d.StyleSheets[e.URL]++
	case "script":
		d.Scripts[e.URL]++
	case "sub_frame":
		d.SubFrames[e.URL]++
	case "other":
		d.Others[e.URL]++
	default:
		panic(e.Type)
	}
	d.SrcDomains[tURL.Host]++
	d.URL++

	s[rURL.Host] = d
}

type BySrcCount []DomainStat

func (s BySrcCount) Len() int      { return len(s) }
func (s BySrcCount) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s BySrcCount) Less(i, j int) bool {
	if len(s[i].SrcDomains) != len(s[j].SrcDomains) {
		return len(s[i].SrcDomains) < len(s[j].SrcDomains)
	}
	if ic, jc := s[i].URLcount(), s[j].URLcount(); ic != jc {
		return ic < jc
	}
	return s[i].Domain < s[j].Domain
}

type stringCount struct {
	String string
	Count  int
}
type stringCounts []stringCount

func (s stringCounts) Len() int      { return len(s) }
func (s stringCounts) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s stringCounts) Less(i, j int) bool {
	if s[i].Count != s[j].Count {
		return s[i].Count < s[j].Count
	}
	return s[i].String < s[j].String
}

// convert a string count map to a string list, order by count desc.
func orderMap(m map[string]int) stringCounts {
	r := make(stringCounts, 0, len(m))
	for k, c := range m {
		r = append(r, stringCount{k, c})
	}
	sort.Sort(sort.Reverse(r))
	return r
}

// is url the domain/prefix of domain?
func matchesDomain(fullurl, domain string) bool {
	r, err := url.Parse(fullurl)
	if err != nil {
		log.Printf("%q: %s", fullurl, err)
		return false
	}
	// any subdomain
	if domain[0] == '.' {
		return strings.HasSuffix(r.Host, domain)
	}
	// exact match
	return r.Host == domain
}
