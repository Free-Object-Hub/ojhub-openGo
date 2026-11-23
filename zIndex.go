package main

import (
	"encoding/base64"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type meta struct{ title, desc, image string }

func parseRoute(r *http.Request) (segs []string, value string) {
	raw := r.URL.RawQuery
	if i := strings.IndexByte(raw, '&'); i >= 0 {
		raw = raw[:i] // дальше только команды
	}
	if i := strings.IndexByte(raw, '='); i >= 0 {
		value, _ = url.QueryUnescape(raw[i+1:])
		raw = raw[:i]
	}
	if p, err := url.PathUnescape(raw); err == nil {
		raw = p
	}
	return strings.Split(raw, "/"), value
}

func (m *meta) fromGDPS(idStr string) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return
	}
	if gdps, err := GDPSfetchById(id); err == nil && gdps != nil {
		f := gdps.ToShort(false, false)
		m.title, m.desc, m.image = f.Title, f.Text, f.Img
	}
}

func (m *meta) fromWiki(idStr string) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return
	}
	if wiki, err := WIKIfetchById(id); err == nil && wiki != nil {
		m.title, m.desc = wiki.Title, wiki.Text
	}
}

func (m *meta) fromNews(idStr string) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return
	}
	news, err := NEWSfetchById(id)
	if err != nil || news == nil {
		return
	}
	m.title = news.Title
	m.desc = ""
	if decoded, err := base64.StdEncoding.DecodeString(news.Text); err == nil {
		m.desc = string(decoded)
	}
	if gdps, err := GDPSfetchById(news.GdpsId); err == nil && gdps != nil {
		m.image = gdps.ToShort(false, false).Img
	}
}

func (m *meta) fromVac(idStr string) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return
	}
	if vac, err := VACSfetchById(id, 0); err == nil && vac != nil {
		m.title, m.desc = vac.Title, vac.Text
		if gdps, err := GDPSfetchById(vac.GdpsID); err == nil && gdps != nil {
			m.image = gdps.ToShort(false, false).Img
		}
	}
}

func buildMetaTags(r *http.Request) string {
	m := meta{
		title: "Object hub",
		desc:  "A convenient service for finding and hosting your object shows and camps!",
		image: "https://objecthub.xyz/imgs/hubbig.png",
	}
	segs, value := parseRoute(r)
	switch segs[0] {
	case "gdps": // ?gdps/ch/gId[/news[/nId]]
		if len(segs) >= 3 {
			if len(segs) == 5 && segs[3] == "news" {
				m.fromNews(segs[4])
			} else {
				m.fromGDPS(segs[2]) // и сам GDPS, и список его новостей
			}
		}
	case "wiki": // ?wiki=12 и будущий ?wiki/12/...
		if len(segs) >= 2 {
			m.fromWiki(segs[1])
		} else {
			m.fromWiki(value)
		}
	case "Wikis", "wikis":
		m.title = "Object Hub Wiki"
		m.desc = "Object Hub Wiki - not Mediawiki! welcome to our wiki engine!"
	case "VacsC": // ?VacsC/5
		if len(segs) >= 2 {
			m.fromVac(segs[1])
		}
	// deprecated, пока живут
	case "camp", "show", "pere", "tele":
		m.fromGDPS(value)
	case "news": // ?news/comms=5|...
		if len(segs) == 2 && segs[1] == "comms" {
			m.fromNews(strings.SplitN(value, "|", 2)[0])
		}
	}
	return fmt.Sprintf(
		`<meta property="og:title" content="%s">
		<meta property="og:description" content="%s">
		<meta property="og:image" content="%s">`,
		html.EscapeString(m.title),
		html.EscapeString(m.desc),
		html.EscapeString(m.image),
	)
}

func IndexParser(w http.ResponseWriter, r *http.Request) {
	verName := LoaderDefVer
	if c, err := r.Cookie("cli_ver"); err == nil {
		if _, ok := findVersion(c.Value); ok {
			verName = c.Value
		}
	}
	cv, _ := findVersion(verName) // defaultVer is guaranteed to be in versions
	//
	v := html.EscapeString(cv.Ver)
	//
	var cssTags, extra, jsTags strings.Builder
	for _, a := range cv.CSS {
		cssTags.WriteString(fmt.Sprintf(`<link href="./cli/%s/%s%s" rel=stylesheet>`+"\n\t\t",
			v, html.EscapeString(a.Href), html.EscapeString(a.Query)))
	}
	for _, tag := range cv.Extra {
		extra.WriteString(tag + "\n\t\t")
	}
	for _, a := range cv.JS {
		deferAttr := ""
		if a.Defer {
			deferAttr = "defer "
		}
		jsTags.WriteString(fmt.Sprintf(`<script %ssrc="./cli/%s/%s%s"></script>`+"\n\t\t",
			deferAttr, v, html.EscapeString(a.Href), html.EscapeString(a.Query)))
	}

	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
	<head>
		<meta name=viewport content="width=device-width,initial-scale=1.0">
		<meta charset=UTF-8>
		%s
		<title>Object Hub</title>
		<link rel=icon>
		%s%s%s<style id=wikiStyle></style>
	</head>
	<body style="background-color:var(--color-bg)">
		<div id=1st></div>
		<div id=windowsXP>
			<div id=Professional class=hider></div>
		</div>
		<div id=alerts class=alerts></div>
	</body>
</html>`, buildMetaTags(r), cssTags.String(), extra.String(), jsTags.String())
}
