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
	return s, func(e Entry, u *url.URL) {
		s.Count(e, u)
	}
}

// filter everything which was logged as blocked.
func filterBlocked(next logFilter) logFilter {
	return func(e Entry, u *url.URL) {
		if e.Action == "block" {
			return
		}
		next(e, u)
	}
}

func filterURL(urls []string, next logFilter) logFilter {
	return func(e Entry, u *url.URL) {
		for _, i := range urls {
			if matchesDomain(u, i) {
				return
			}
		}
		next(e, u)
	}
}

func filterDomain(dom string, next logFilter) logFilter {
	return func(e Entry, u *url.URL) {
		if u.Host != dom {
			return
		}
		next(e, u)
	}
}

func filterTabDomain(dom string, next logFilter) logFilter {
	return func(e Entry, u *url.URL) {
		r, err := url.Parse(e.TabURL)
		if err != nil {
			log.Printf("%q: %s", e.TabURL, err)
			return
		}
		if r.Host != dom {
			return
		}
		next(e, u)
	}
}

func (s DomainStats) Count(e Entry, u *url.URL) {
	if u.Scheme == "chrome-extension" {
		return
	}

	tURL, err := url.Parse(e.TabURL)
	if err != nil {
		panic(err)
	}
	if u.Host == tURL.Host {
		// same domain. Not interesting.
		return
	}

	d, ok := s[u.Host]
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

	d.Domain = u.Host
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
	s[u.Host] = d
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
func matchesDomain(u *url.URL, domain string) bool {
	// any subdomain
	if domain[0] == '.' {
		return strings.HasSuffix(u.Host, domain)
	}
	// exact match
	return u.Host == domain
}
