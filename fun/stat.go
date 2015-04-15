package main

import (
	"encoding/csv"
	"io"
	"log"
	"net/url"
	"os"
	"sort"
)

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

func readStats() (DomainStats, error) {
	f, err := os.Open(logFile)
	defer f.Close()
	if err != nil {
		return nil, err
	}

	s := DomainStats{}
	fr := csv.NewReader(f)
	for {
		r, err := fr.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		e := Entry{
			// r[0] is timestamp
			Action: r[1],
			Type:   r[2],
			URL:    r[3],
			// TabID
			TabURL: r[4],
		}

		switch e.Action {
		case "allow":
			s.Count(e)
		case "block":
			//
		default:
			log.Printf("unknown action: %s", e.Action)
		}
	}
	return s, nil
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
	sort.Sort(sort.Reverse(r))
	return r
}
