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

func firstQueryValue(q url.Values, keys ...string) string {
	for _, key := range keys {
		if value := q.Get(key); value != "" {
			return value
		}
	}
	return ""
}

func futureLinkFormat(r *http.Request) (string, int, bool) {
	// FIXME:
	// 1. сделать парсер для гайдов
	// 2. придумать что делать с ссылками формата:
	// "?wiki/{wikiId}/page/{guideId}"
	// да, это тот самый фикс ми который будет висеть ещё не один год
	for key := range r.URL.Query() {
		parts := strings.SplitN(key, "/", 2)
		log.Println(parts)
		if len(parts) != 2 {
			continue
		}
		id, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		return parts[0], id, true
	}
	return "", 0, false
}

func buildMetaTags(r *http.Request) string {
	metaTitle := "Object hub"
	metaDescription := "Удобный сервис для поиска и размещения своих обджект шоу и кемпов!"
	metaImage := "https://objecthub.xyz/imgs/hubbig.png"
	//
	route, id, ok := futureLinkFormat(r)
	if ok {
		switch route {
		case "VacsC":
			log.Println(id)
			if vac, err := VACSfetchById(id, 0); err == nil && vac != nil {
				metaTitle = vac.Title
				metaDescription = vac.Text
				if gdps, err := GDPSfetchById(vac.GdpsID); err == nil && gdps != nil {
					metaImage = gdps.ToShort(false).Img
				}
			}
		}
	} else {
		q := r.URL.Query()
		log.Println(q)
		if _, ok := q["Wikis"]; ok {
			metaTitle = "Object Hub Wiki"
			metaDescription = "Object Hub Wiki - не Mediawiki! добро пожаловать на наш вики движок!"
		}
		if gdpsID := firstQueryValue(q, "camp", "show", "pere", "tele"); gdpsID != "" {
			if id, err := strconv.Atoi(gdpsID); err == nil {
				if gdps, err := GDPSfetchById(id); err == nil && gdps != nil {
					gdpsFull := gdps.ToShort(false)
					metaTitle = gdpsFull.Title
					metaDescription = gdpsFull.Text
					metaImage = gdpsFull.Img
				}
			}
		}
		if wikiID := q.Get("wiki"); wikiID != "" {
			if id, err := strconv.Atoi(wikiID); err == nil {
				if wiki, err := WIKIfetchById(id); err == nil && wiki != nil {
					metaTitle = wiki.Title
					metaDescription = wiki.Text
				}
			}
		}
		if newsID := q.Get("news/comms"); newsID != "" {
			parts := strings.SplitN(newsID, "|", 2)
			if len(parts) > 0 {
				if id, err := strconv.Atoi(parts[0]); err == nil {
					if news, err := NEWSfetchById(id); err == nil && news != nil {
						metaTitle = news.Title
						if decoded, err := base64.StdEncoding.DecodeString(news.Text); err == nil {
							metaDescription = string(decoded)
						} else {
							metaDescription = ""
						}
						if gdps, err := GDPSfetchById(news.GdpsId); err == nil && gdps != nil {
							metaImage = gdps.ToShort(false).Img
						}
					}
				}
			}
		}
	}
	return fmt.Sprintf(
		`<meta property="og:title" content="%s">
		<meta property="og:description" content="%s">
		<meta property="og:image" content="%s">`,
		metaTitle,
		metaDescription,
		metaImage,
	)
}

func IndexParser(w http.ResponseWriter, r *http.Request) {
	verName := LoaderDefVer
	if c, err := r.Cookie("cli_ver"); err == nil {
		if _, ok := findVersion(c.Value); ok {
			verName = c.Value
		}
	}
	cv, _ := findVersion(verName) // defaultVer гарантированно есть в versions
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
