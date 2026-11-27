package main

import (
	"encoding/base64"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type meta struct{ title, desc, image string }

const (
	defTitle = "Object hub"
	defDesc  = "A convenient service for finding and hosting your object shows and camps!"
	defImg   = "https://objecthub.xyz/imgs/hubbig.png"
)

// set перезаписывает только непустые поля, дефолты не затираются
func (m *meta) set(title, desc, img string) {
	if title != "" {
		m.title = title
	}
	if desc != "" {
		m.desc = desc
	}
	if img != "" {
		m.image = img
	}
}

// clip схлопывает пробелы и переносы строк и режет по рунам
func clip(s string, n int) string {
	r := []rune(strings.Join(strings.Fields(s), " "))
	if len(r) > n {
		return string(r[:n]) + "…"
	}
	return string(r)
}

// cutCmd отрезает командную часть запроса (всё после первого &)
func cutCmd(raw string) string {
	if i := strings.IndexByte(raw, '&'); i >= 0 {
		return raw[:i]
	}
	return raw
}

func parseRoute(r *http.Request) (segs []string, value string) {
	raw := cutCmd(r.URL.RawQuery)
	if i := strings.IndexByte(raw, '='); i >= 0 {
		value, _ = url.QueryUnescape(raw[i+1:])
		raw = raw[:i]
	}
	if p, err := url.PathUnescape(raw); err == nil {
		raw = p
	}
	return strings.Split(raw, "/"), value
}

func (m *meta) fromUser(idStr string, portfolio bool) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return
	}
	u, err := GetUserById(id)
	if err != nil || u == nil {
		return
	}
	title := u.GetNick()
	if portfolio {
		title += " - portfolio"
	}
	m.set(title, u.Resume, "")
}

func (m *meta) fromGDPS(idStr string) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return
	}
	if gdps, err := GDPSfetchById(id); err == nil && gdps != nil {
		f := gdps.ToShort(false, false)
		m.set(f.Title, f.Text, f.Img)
	}
}

func (m *meta) fromWiki(idStr string) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return
	}
	if wiki, err := WIKIfetchById(id); err == nil && wiki != nil {
		m.set(wiki.Title, wiki.Text, "")
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
	desc := ""
	if decoded, err := base64.StdEncoding.DecodeString(news.Text); err == nil {
		desc = string(decoded)
	}
	img := ""
	if gdps, err := GDPSfetchById(news.GdpsId); err == nil && gdps != nil {
		img = gdps.ToShort(false, false).Img
	}
	m.set(news.Title, desc, img)
}

func (m *meta) fromVac(idStr string) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return
	}
	vac, err := VACSfetchById(id, 0)
	if err != nil || vac == nil {
		return
	}
	img := ""
	if gdps, err := GDPSfetchById(vac.GdpsID); err == nil && gdps != nil {
		img = gdps.ToShort(false, false).Img
	}
	m.set(vac.Title, vac.Text, img)
}

func buildMetaTags(r *http.Request) string {
	m := meta{title: defTitle, desc: defDesc, image: defImg}
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
	case "user": // ?user/12[/portfolio]
		if len(segs) >= 2 {
			m.fromUser(segs[1], len(segs) >= 3 && segs[2] == "portfolio")
		}
	case "profiles": // deprecated: ?profiles=12, ?profiles/portfolio=12
		m.fromUser(value, len(segs) >= 2 && segs[1] == "portfolio")
	case "wiki": // ?wiki=12 и будущий ?wiki/12/...
		if len(segs) >= 2 {
			m.fromWiki(segs[1])
		} else {
			m.fromWiki(value)
		}
	case "Wikis", "wikis":
		m.set("Object Hub Wiki", "Object Hub Wiki - not Mediawiki! welcome to our wiki engine!", "")
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

	log.Printf("meta: q=%q title=%q img=%q", r.URL.RawQuery, m.title, m.image) // ВРЕМЕННО, потом удалить

	return fmt.Sprintf(
		`<meta property="og:type" content="website">
		<meta property="og:site_name" content="Object Hub">
		<meta property="og:url" content="https://objecthub.xyz/?%s">
		<meta property="og:title" content="%s">
		<meta property="og:description" content="%s">
		<meta property="og:image" content="%s">
		<meta name="twitter:card" content="summary_large_image">`,
		html.EscapeString(cutCmd(r.URL.RawQuery)),
		html.EscapeString(clip(m.title, 90)),
		html.EscapeString(clip(m.desc, 200)),
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
