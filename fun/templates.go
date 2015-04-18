package main

import (
	"html/template"
)

var tmpl = template.Must(template.New("all").Parse(`
{{define "header"}}
	<head>
		<style>
			ul {
				overflow: auto;
				white-space: nowrap;
			}
			ul ul ul a {
				color: black;
			}
			.toggle .main {
				display: none;
			}
			.toggle.hide .main {
				display: block;
			}
			.toggle .open {
				display: block;
			}
			.toggle.hide .open {
				display: none;
			}
			.toggle a.toggler {
				text-decoration: none;
			}
		</style>
	</head>
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
	function toggle(el) {
		// div -> div -> a
		var cl = el.parentNode.parentNode.classList;
		if (cl.contains("hide")) {
			cl.remove("hide");
		} else {
			cl.add("hide");
		}
		return false
	}
	</script>
{{end}}

{{define "list"}}
	<ul>
	{{range .}}
		<li><a href="{{.String}}" target="_blank">{{.String}}</a> ({{.Count}})</li>
	{{end}}
	</ul>
{{end}}

{{/* tlist is a un-collapsable list, collapsed by default */}}
{{define "tlist"}}
	<ul>
		<div class="toggle">
			<div class="main">
				<a href="#" onclick="return toggle(this)" class="toggler">▼</a>
				{{range .}}
				<li><a href="{{.String}}" target="_blank">{{.String}}</a> ({{.Count}})</li>
				{{end}}
			</div>
			<div class="open">
				<a href="#" onclick="return toggle(this)" class="toggler">►</a>
			</div>
		</div>
	</ul>
{{end}}


{{define "page_index"}}
	Stats:<br />
	<a href="/unblocked">Unblocked domains by 3rdparty usage</a><br />
{{end}}


{{define "page_unblocked"}}
	{{template "header"}}
	<h1>3rd party domain usage</h1>
	{{ if .filters }}
	Effectively unblocked requests: <a href="?full=1">no block</a>
	{{ else }}
	All requests: <a href="?">with blocks</a>
	{{ end }}
	<br />
	Ordered by subdomain count.<br />
	<br />
	{{range .stats.Domains}}
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
			<li>{{.String}} ({{.Count}}) <a href="/srclog/{{.String}}">stats</a></li>
			{{end}}
			</ul>
		</li>

		<li>usage:
			<ul>
			{{if .XMLHTTPs}}
				<li>xmlhttps: {{len .XMLHTTPs}}<br />
					{{ template "tlist" .XMLHTTPs }}
				</li>
			{{end}}
			{{if .Images}}
				<li>images: {{len .Images}}<br />
					{{ template "tlist" .Images }}
				</li>
			{{end}}
			{{if .StyleSheets}}
				<li>stylesheets: {{len .StyleSheets}}<br />
					{{ template "tlist" .StyleSheets }}
				</li>
			{{end}}
			{{if .Scripts}}
				<li>scripts: {{len .Scripts}}<br />
					{{ template "tlist" .Scripts }}
				</li>
			{{end}}
			{{if .SubFrames}}
				<li>subframes: {{len .SubFrames}}<br />
					{{ template "tlist" .SubFrames }}
				</li>
			{{end}}
			{{if .Others}}
				<li>others: {{len .Others}}<br />
					{{ template "tlist" .Others }}
				</li>
			{{end}}
			</ul>
		</li>
		</ul>

		<br />
	{{end}}
{{end}}


{{define "page_log"}}
	{{template "header"}}
	<h1>Pages originating from {{.subject}}</h1>
	{{range .stats.Domains}}
		<b>{{.Domain}}</b> <a href="/unblocked/{{ .Domain }}">full</a>
		<ul>
		{{if .XMLHTTPs}}
			<li>xmlhttps: {{len .XMLHTTPs}}<br />
				{{ template "list" .XMLHTTPs }}
			</li>
		{{end}}
		{{if .Images}}
			<li>images: {{len .Images}}<br />
				{{ template "list" .Images }}
			</li>
		{{end}}
		{{if .StyleSheets}}
			<li>stylesheets: {{len .StyleSheets}}<br />
				{{ template "list" .StyleSheets }}
			</li>
		{{end}}
		{{if .Scripts}}
			<li>scripts: {{len .Scripts}}<br />
				{{ template "list" .Scripts }}
			</li>
		{{end}}
		{{if .SubFrames}}
			<li>subframes: {{len .SubFrames}}<br />
				{{ template "list" .SubFrames }}
			</li>
		{{end}}
		{{if .Others}}
			<li>others: {{len .Others}}<br />
				{{ template "list" .Others }}
			</li>
		{{end}}
		</ul>
	{{end}}
{{end}}
`))
