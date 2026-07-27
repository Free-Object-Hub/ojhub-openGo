package main

/*
 * Loader 1.10
 *
 * Author: Claude code, с промптами и патчами MIOBOMB (2026)
 *
 * Ojhub Loader, штука которая подставляет в index роут нужные html теги для загрузки
 * разных версий сайта. Перед вами же реализация загрузчика сайта на go.
 * В отличии от nodejs загрузчика он умеет подставлять кастомные теги для каждой версии,
 * отсюда и версия 1.10 у openGo.
 * Для понимания в nodejs "/" маршруте все теги были захардкожены и в них просто подставлялась
 * кука с версией, это ломало 0.97.33 в Chromium браузерах, здесь такого уже нет.
 */

import (
	"fmt"
	"html"
	"net/http"
	"os"
)

const (
	LoaderStatusDev       = "dev-snapshot"
	LoaderStatusStable    = "stable"
	LoaderStatusLegacy    = "legacy"
	LoaderStatusIncompile = "incompile"
	LoaderStatusLostMedia = "lostmedia"
)

type asset struct {
	Href  string // относительный путь внутри ./cli/<ver>/, например "main.css"
	Query string // "?ver=20"
	Defer bool   // только для JS
}

type clientVersion struct {
	Ver, Date, Desc, Status string
	CSS                     []asset
	JS                      []asset
	Extra                   []string
	IsWorking               bool
}

var versions = []clientVersion{
	{
		Ver: "0.97.7", Date: "?? ??? 2026", Desc: "openGo, not working in prod", Status: LoaderStatusDev,
		CSS: []asset{
			{Href: "main.css", Query: "?ver=20"},
			{Href: "window.css", Query: "?ver=20"},
		},
		JS: []asset{
			{Href: "newHelper.js", Query: "?ver=23", Defer: true},
			{Href: "nhConfig.js", Query: "?ver=24", Defer: true},
			{Href: "ojhub.js", Query: "?ver=23", Defer: true},
		},
		IsWorking: true,
	},

	{
		Ver: "0.97.6", Date: "16 Jun 2026", Desc: "latest nodejs-based", Status: LoaderStatusIncompile,
		CSS: []asset{
			{Href: "main.css", Query: "?ver=20"},
			{Href: "window.css", Query: "?ver=20"},
		},
		JS: []asset{
			{Href: "newHelper.js", Query: "?ver=21", Defer: true},
			{Href: "nhConfig.js", Query: "?ver=21", Defer: true},
			{Href: "ojhub.js", Query: "?ver=22", Defer: true},
		},
		IsWorking: false,
	},

	{
		Ver: "0.97.5", Date: "10 Jun 2026", Desc: "nodejs-based", Status: LoaderStatusIncompile,
		CSS: []asset{
			{Href: "main.css", Query: "?ver=20"},
			{Href: "window.css", Query: "?ver=20"},
		},
		JS: []asset{
			{Href: "newHelper.js", Query: "?ver=21", Defer: true},
			{Href: "nhConfig.js", Query: "?ver=21", Defer: true},
			{Href: "ojhub.js", Query: "?ver=22", Defer: true},
		},
		IsWorking: false,
	},

	{
		Ver: "0.97.4", Date: "9 Jun 2026", Desc: "newHelper.js 2.1 release", Status: LoaderStatusLostMedia,
		IsWorking: false,
	},

	{
		Ver: "0.97.33", Date: "31 Jan 2026", Desc: "latest-php", Status: LoaderStatusLegacy,
		CSS: []asset{
			{Href: "main.css", Query: "?ver=20"},
			{Href: "window.css", Query: "?ver=20"},
		},
		JS: []asset{
			{Href: "newHelper.js", Query: "?ver=21", Defer: true},
			{Href: "nhConfig.js", Query: "?ver=21", Defer: true},
		},
		Extra: []string{
			`<link rel="preconnect" href="https://fonts.googleapis.com">`,
			`<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>`,
			`<link href="https://fonts.googleapis.com/css2?family=Comfortaa:wght@300..700&family=Unbounded:wght@200..900&display=swap" rel="stylesheet">`,
			`<link href="https://fonts.googleapis.com/css2?family=Comfortaa:wght@300..700&family=Huninn&family=Manrope:wght@200..800&family=News+Cycle:wght@400;700&family=Unbounded:wght@200..900&display=swap" rel="stylesheet">`,
		},
		IsWorking: true,
	},

	{
		Ver: "0.97.32", Date: "23 Jan 2026", Desc: "", Status: LoaderStatusLostMedia,
		IsWorking: false,
	},

	{
		Ver: "0.97.31", Date: "2 Jan 2026", Desc: "", Status: LoaderStatusLostMedia,
		IsWorking: false,
	},

	{
		Ver: "0.97.3", Date: "20 Dec 2025", Desc: "", Status: LoaderStatusLostMedia,
		IsWorking: false,
	},

	{
		Ver: "0.97.2", Date: "6 Dec 2025", Desc: "", Status: LoaderStatusLostMedia,
		IsWorking: false,
	},

	{
		Ver: "0.97.1", Date: "26 Nov 2025", Desc: "", Status: LoaderStatusLostMedia,
		IsWorking: false,
	},

	{
		Ver: "0.97", Date: "17 Nov 2025", Desc: "newHelper.js 2.0 release", Status: LoaderStatusLostMedia,
		IsWorking: false,
	},

	{
		Ver: "0.96.3", Date: "12 Sep 2025", Desc: "wiki isnt working", Status: LoaderStatusLegacy,
		CSS: []asset{
			{Href: "main.css", Query: "?ver=18"},
			{Href: "window.css", Query: "?ver=`18"},
		},
		JS: []asset{
			{Href: "ojhub.js", Query: "?ver=18&helper", Defer: true},
		},
		IsWorking: true,
	},

	{
		Ver: "0.96.2", Date: "10 Sep 2025", Desc: "", Status: LoaderStatusLostMedia,
		IsWorking: false,
	},

	{
		Ver: "0.96.1", Date: "3 Sep 2025", Desc: "", Status: LoaderStatusLostMedia,
		IsWorking: false,
	},

	{
		Ver: "0.96", Date: "20 Aug 2025", Desc: "", Status: LoaderStatusLostMedia,
		IsWorking: false,
	},

	{
		Ver: "0.95.4", Date: "14 Aug 2025", Desc: "", Status: LoaderStatusLostMedia,
		IsWorking: false,
	},

	{
		Ver: "0.95.3", Date: "12 Aug 2025", Desc: "", Status: LoaderStatusLostMedia,
		IsWorking: false,
	},

	{
		Ver: "0.95.2", Date: "10 Aug 2025", Desc: "", Status: LoaderStatusLostMedia,
		IsWorking: false,
	},

	{
		Ver: "0.95.1", Date: "20 Jul 2025", Desc: "", Status: LoaderStatusLostMedia,
		IsWorking: false,
	},

	{
		Ver: "0.95", Date: "19 Jul 2025", Desc: "", Status: LoaderStatusLostMedia,
		IsWorking: false,
	},

	{
		Ver: "0.942", Date: "1 Jul 2025", Desc: "", Status: LoaderStatusLostMedia,
		IsWorking: false,
	},

	{
		Ver: "0.941", Date: "8 Jun 2025", Desc: "", Status: LoaderStatusLostMedia,
		IsWorking: false,
	},

	{
		Ver: "0.94", Date: "1 Jun 2025", Desc: "", Status: LoaderStatusLostMedia,
		IsWorking: false,
	},

	{
		Ver: "0.932", Date: "23 May 2025", Desc: "", Status: LoaderStatusLostMedia,
		IsWorking: false,
	},

	{
		Ver: "0.931", Date: "21 May 2025", Desc: "", Status: LoaderStatusLostMedia,
		IsWorking: false,
	},

	{
		Ver: "0.93", Date: "21 May 2025", Desc: "", Status: LoaderStatusLostMedia,
		IsWorking: false,
	},

	{
		Ver: "0.922", Date: "17 May 2025", Desc: "", Status: LoaderStatusLostMedia,
		IsWorking: false,
	},

	{
		Ver: "0.913", Date: "2 Apr 2025", Desc: "", Status: LoaderStatusLostMedia,
		IsWorking: false,
	},

	{
		Ver: "0.91", Date: "22 Mar 2025", Desc: "", Status: LoaderStatusLostMedia,
		IsWorking: false,
	},

	{
		Ver: "0.9", Date: "4 Sep 2024", Desc: "", Status: LoaderStatusLostMedia,
		IsWorking: false,
	},

	{
		Ver: "0.8", Date: "28 Aug 2024", Desc: "Initial release", Status: LoaderStatusLostMedia,
		IsWorking: false,
	},

	//
}

var LoaderDefVer = os.Getenv("CLI_VER")

func findVersion(v string) (clientVersion, bool) {
	for _, x := range versions {
		if x.Ver == v {
			return x, true
		}
	}
	return clientVersion{}, false
}

func CliLoader(w http.ResponseWriter, r *http.Request) {
	verName := "stable"
	if c, err := r.Cookie("cli_ver"); err == nil {
		if _, ok := findVersion(c.Value); ok {
			verName = c.Value
		}
	}
	fmt.Fprintf(w, `<div style=display:flex;justify-content:center;align-items:center;min-height:100vh;flex-direction:column>
	<h1>OJHUB LOADER v1.10</h1>
	<p>selected: %s</p>
	<table border=1>
		<tr><th>ver</th><th>status</th><th>date</th><th>desc</th></tr>
		<tr>
			<td><button onclick=(document.cookie='cli_ver=\'\';path=/;max-age=0');location.pathname=''>stable</button></td>
			<td></td>
			<td></td>
			<td></td>
		</tr>`, verName)

	for _, v := range versions {
		if v.IsWorking {
			fmt.Fprintf(w, `
			<tr>
				<td><button onclick=(document.cookie='cli_ver=%s;path=/;max-age=%d');location.pathname=''>%s</button></td>
				<td>%s</td>
				<td>%s</td>
				<td>%s</td>
			</tr>`,
				html.EscapeString(v.Ver), 60*60*24*365, html.EscapeString(v.Ver),
				html.EscapeString(v.Status),
				html.EscapeString(v.Date), html.EscapeString(v.Desc))
		} else {
			fmt.Fprintf(w, `
			<tr>
				<td>%s</td>
				<td>%s</td>
				<td>%s</td>
				<td>%s</td>
			</tr>`,
				html.EscapeString(v.Ver),
				html.EscapeString(v.Status),
				html.EscapeString(v.Date), html.EscapeString(v.Desc))
		}
	}

	fmt.Fprint(w, `
	</table>
</div>`)
}
