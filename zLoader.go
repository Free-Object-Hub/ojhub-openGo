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
 * ^^^ устарело
 *
 * Слияние улучшений openRust обратно в openGo:
 * - два независимых статуса вместо одного (жизненный цикл + техническое состояние сборки)
 * - две таблицы: живые версии сверху (включая incompile — они хоть и не собираются,
 *   но не являются "мёртвыми" в смысле потери медиа), lostmedia снизу
 * - полноценная HTML-страница вместо фрагмента (как всегда было в openGo)
 * - структура asset{} для CSS/JS сохранена как есть, никакого хардкода строкой
 * - порядок колонок таблицы (ver, status, date, desc) сохранён как в оригинале openGo
 * - экранирование через html.EscapeString оставлено везде, даже где избыточно
 * - CLI_VER теперь реально используется как дефолт версии, с фоллбеком на "stable"
 *   вместо панической остановки, если версии в CLI_VER не существует
 * - исправлен баг с очисткой куки (was: cli_ver='\'\'' — буквальная строка из
 *   экранированных кавычек вместо пустого значения)
 */

import (
	"fmt"
	"html"
	"net/http"
	"os"
)

// LifecycleStatus — где версия находится в жизненном цикле продукта
type LifecycleStatus string

const (
	LifecycleDev       LifecycleStatus = "dev"
	LifecycleStable    LifecycleStatus = "stable"
	LifecycleSupported LifecycleStatus = "supported"
	LifecycleLegacy    LifecycleStatus = "legacy"
	LifecycleArchived  LifecycleStatus = "archived"
)

// BuildState — техническое состояние сборки конкретной версии
type BuildState string

const (
	StateWorking   BuildState = "working"
	StateIncompile BuildState = "incompile"
	StateLostMedia BuildState = "lostmedia"
)

type asset struct {
	Href  string
	Query string
	Defer bool
}

type clientVersion struct {
	Ver, Date, Desc string
	Lifecycle       LifecycleStatus
	State           BuildState
	CSS             []asset
	JS              []asset
	Extra           []string
}

// isWorking — версия кликабельна в лоадере только если реально собирается и запускается.
// Incompile-версии видны (сверху, живые), но кнопки не имеют — как и раньше в Go.
func (v clientVersion) isWorking() bool {
	return v.State == StateWorking
}

var versions = []clientVersion{

	{
		Ver: "0.98.2", Date: "?? ??? 2026", Desc: "",
		Lifecycle: LifecycleDev, State: StateWorking,
		CSS: []asset{
			{Href: "main.css", Query: "?ver=21"},
			{Href: "window.css", Query: "?ver=21"},
		},
		JS: []asset{
			{Href: "newHelper.js", Query: "?ver=29", Defer: true},
			{Href: "nhConfig.js", Query: "?ver=29", Defer: true},
			{Href: "ojhub.js", Query: "?ver=28", Defer: true},
		},
	},

	{
		Ver: "0.98.1", Date: "15 Sep 2026", Desc: "GHE 2.2 and Jails init",
		Lifecycle: LifecycleStable, State: StateWorking,
		CSS: []asset{
			{Href: "main.css", Query: "?ver=21"},
			{Href: "window.css", Query: "?ver=21"},
		},
		JS: []asset{
			{Href: "newHelper.js", Query: "?ver=29", Defer: true},
			{Href: "nhConfig.js", Query: "?ver=29", Defer: true},
			{Href: "ojhub.js", Query: "?ver=28", Defer: true},
		},
	},

	{
		Ver: "0.98", Date: "28 Aug 2026", Desc: "",
		Lifecycle: LifecycleSupported, State: StateWorking,
		CSS: []asset{
			{Href: "main.css", Query: "?ver=21"},
			{Href: "window.css", Query: "?ver=21"},
		},
		JS: []asset{
			{Href: "newHelper.js", Query: "?ver=29", Defer: true},
			{Href: "nhConfig.js", Query: "?ver=29", Defer: true},
			{Href: "ojhub.js", Query: "?ver=29", Defer: true},
		},
	},

	{
		// п.10: Go-лоадер перестал сопровождаться с выходом openRust, и на тот момент
		// 0.97.7 всё ещё была сырой поделкой — значит не Stable/working, а Legacy+Incompile.
		Ver: "0.97.7", Date: "27 Jul 2026", Desc: "openGo init",
		Lifecycle: LifecycleLegacy, State: StateWorking,
		CSS: []asset{
			{Href: "main.css", Query: "?ver=20"},
			{Href: "window.css", Query: "?ver=20"},
		},
		JS: []asset{
			{Href: "newHelper.js", Query: "?ver=23", Defer: true},
			{Href: "nhConfig.js", Query: "?ver=24", Defer: true},
			{Href: "ojhub.js", Query: "?ver=24", Defer: true},
		},
	},

	{
		// п.2: nodejs-based мёртвые версии остаются в верхней таблице как Incompile,
		// а не проваливаются в lostmedia — медиа у них не потеряны, просто не собираются.
		Ver: "0.97.6", Date: "16 Jun 2026", Desc: "latest nodejs-based",
		Lifecycle: LifecycleArchived, State: StateIncompile,
		CSS: []asset{
			{Href: "main.css", Query: "?ver=20"},
			{Href: "window.css", Query: "?ver=20"},
		},
		JS: []asset{
			{Href: "newHelper.js", Query: "?ver=21", Defer: true},
			{Href: "nhConfig.js", Query: "?ver=21", Defer: true},
			{Href: "ojhub.js", Query: "?ver=22", Defer: true},
		},
	},

	{
		Ver: "0.97.5", Date: "10 Jun 2026", Desc: "nodejs-based",
		Lifecycle: LifecycleArchived, State: StateIncompile,
		CSS: []asset{
			{Href: "main.css", Query: "?ver=20"},
			{Href: "window.css", Query: "?ver=20"},
		},
		JS: []asset{
			{Href: "newHelper.js", Query: "?ver=21", Defer: true},
			{Href: "nhConfig.js", Query: "?ver=21", Defer: true},
			{Href: "ojhub.js", Query: "?ver=22", Defer: true},
		},
	},

	{
		Ver: "0.97.4", Date: "9 Jun 2026", Desc: "newHelper.js 2.1 release",
		Lifecycle: LifecycleArchived, State: StateLostMedia,
	},

	{
		Ver: "0.97.33", Date: "31 Jan 2026", Desc: "latest-php",
		Lifecycle: LifecycleArchived, State: StateWorking,
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
	},

	{Ver: "0.97.32", Date: "23 Jan 2026", Desc: "", Lifecycle: LifecycleArchived, State: StateLostMedia},

	{Ver: "0.97.31", Date: "2 Jan 2026", Desc: "", Lifecycle: LifecycleArchived, State: StateLostMedia},

	{Ver: "0.97.3", Date: "20 Dec 2025", Desc: "", Lifecycle: LifecycleArchived, State: StateLostMedia},

	{Ver: "0.97.2", Date: "6 Dec 2025", Desc: "", Lifecycle: LifecycleArchived, State: StateLostMedia},

	{Ver: "0.97.1", Date: "26 Nov 2025", Desc: "", Lifecycle: LifecycleArchived, State: StateLostMedia},

	{Ver: "0.97", Date: "17 Nov 2025", Desc: "newHelper.js 2.0 release", Lifecycle: LifecycleArchived, State: StateLostMedia},

	{
		Ver: "0.96.3", Date: "12 Sep 2025", Desc: "wiki isnt working",
		Lifecycle: LifecycleArchived, State: StateWorking,
		CSS: []asset{
			{Href: "main.css", Query: "?ver=18"},
			{Href: "window.css", Query: "?ver=18"},
		},
		JS: []asset{
			{Href: "ojhub.js", Query: "?ver=18&helper", Defer: true},
		},
	},

	{Ver: "0.96.2", Date: "10 Sep 2025", Desc: "", Lifecycle: LifecycleArchived, State: StateLostMedia},

	{Ver: "0.96.1", Date: "3 Sep 2025", Desc: "", Lifecycle: LifecycleArchived, State: StateLostMedia},

	{Ver: "0.96", Date: "20 Aug 2025", Desc: "", Lifecycle: LifecycleArchived, State: StateLostMedia},

	{Ver: "0.95.4", Date: "14 Aug 2025", Desc: "", Lifecycle: LifecycleArchived, State: StateLostMedia},

	{Ver: "0.95.3", Date: "12 Aug 2025", Desc: "", Lifecycle: LifecycleArchived, State: StateLostMedia},

	{Ver: "0.95.2", Date: "10 Aug 2025", Desc: "", Lifecycle: LifecycleArchived, State: StateLostMedia},

	{Ver: "0.95.1", Date: "20 Jul 2025", Desc: "", Lifecycle: LifecycleArchived, State: StateLostMedia},

	{Ver: "0.95", Date: "19 Jul 2025", Desc: "", Lifecycle: LifecycleArchived, State: StateLostMedia},

	{
		Ver: "0.942", Date: "1 Jul 2025", Desc: "accounts and profiles isnt working",
		Lifecycle: LifecycleArchived, State: StateWorking,
		CSS: []asset{{Href: "main.css", Query: "?ver=18"}},
		JS:  []asset{{Href: "ojhub.js", Query: "?ver=18&helper", Defer: true}},
		Extra: []string{
			`<link rel="preconnect" href="https://fonts.googleapis.com">`,
			`<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>`,
			`<link href="https://fonts.googleapis.com/css2?family=Archivo+Narrow:ital,wght@0,400..700;1,400..700&family=Huninn&family=Unbounded:wght@200..900&display=swap" rel="stylesheet">`,
		},
	},

	{Ver: "0.941", Date: "8 Jun 2025", Desc: "", Lifecycle: LifecycleArchived, State: StateLostMedia},

	{Ver: "0.94", Date: "1 Jun 2025", Desc: "", Lifecycle: LifecycleArchived, State: StateLostMedia},

	{Ver: "0.932", Date: "23 May 2025", Desc: "", Lifecycle: LifecycleArchived, State: StateLostMedia},

	{Ver: "0.931", Date: "21 May 2025", Desc: "", Lifecycle: LifecycleArchived, State: StateLostMedia},

	{Ver: "0.93", Date: "21 May 2025", Desc: "", Lifecycle: LifecycleArchived, State: StateLostMedia},

	{Ver: "0.922", Date: "17 May 2025", Desc: "", Lifecycle: LifecycleArchived, State: StateLostMedia},

	{Ver: "0.913", Date: "2 Apr 2025", Desc: "", Lifecycle: LifecycleArchived, State: StateLostMedia},

	{Ver: "0.91", Date: "22 Mar 2025", Desc: "", Lifecycle: LifecycleArchived, State: StateLostMedia},

	{Ver: "0.9", Date: "4 Sep 2024", Desc: "", Lifecycle: LifecycleArchived, State: StateLostMedia},

	{
		Ver: "GHE1.9", Date: "24 Nov 2024", Desc: "GDPS Helper 1.901, not object hub",
		Lifecycle: LifecycleArchived, State: StateWorking,
		CSS: []asset{{Href: "main.css"}},
		JS:  []asset{{Href: "newHelper.js", Defer: true}},
		Extra: []string{
			`<style id="stule">:root{--color-main:rgb(157,97,42);--color-light:rgb(255,134,0);--color-weekly:rgb(189,99,0);--color-weekly-alpha:rgba(189,99,0,.6);--color-black:rgb(29,28,22);--color-black-alpha:rgba(29,28,22,.6);--color-profile:rgb(32,31,24);--color-profile-alpha:rgb(32,31,24,.6);}</style>`,
			`<script defer>setTimeout(()=>document.body.style="background-color:rgb(12,12,3)",100)</script>`,
		},
	},

	{Ver: "0.8", Date: "28 Aug 2024", Desc: "Initial release", Lifecycle: LifecycleArchived, State: StateLostMedia},
}

func findVersion(v string) (clientVersion, bool) {
	for _, x := range versions {
		if x.Ver == v {
			return x, true
		}
	}
	return clientVersion{}, false
}

var LoaderDefVer = os.Getenv("CLI_VER")

// п.9: CLI_VER теперь реально читается (как в openRust default_ver/default_version),
// но без паники если версии там нет — просто фоллбек на "stable", сервер не должен
// падать из-за опечатки в env var.
func defaultVer() string {
	if v := LoaderDefVer; v != "" {
		if _, ok := findVersion(v); ok {
			return v
		}
	}
	return "stable"
}

func lifecycleLabel(l LifecycleStatus) string {
	switch l {
	case LifecycleDev:
		return "dev"
	case LifecycleStable:
		return "stable"
	case LifecycleSupported:
		return "supported"
	case LifecycleLegacy:
		return "legacy"
	case LifecycleArchived:
		return "archived"
	}
	return string(l)
}

func stateLabel(s BuildState) string {
	switch s {
	case StateWorking:
		return "working"
	case StateIncompile:
		return "incompile"
	case StateLostMedia:
		return "lostmedia"
	}
	return string(s)
}

// renderRow — одна строка таблицы, порядок колонок как в оригинальном openGo:
// ver | status | date | desc. Раз статусов теперь два — оба идут в одной ячейке
// "status", через слэш, чтобы не ломать исходный порядок колонок.
func renderRow(v clientVersion) string {
	status := fmt.Sprintf("%s / %s", lifecycleLabel(v.Lifecycle), stateLabel(v.State))

	verCell := html.EscapeString(v.Ver)
	if v.isWorking() {
		verCell = fmt.Sprintf(
			`<button onclick="(document.cookie='cli_ver=%s;path=/;max-age=%d');location.pathname=''">%s</button>`,
			html.EscapeString(v.Ver), 60*60*24*365, html.EscapeString(v.Ver),
		)
	}

	return fmt.Sprintf(
		`<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>`,
		verCell, html.EscapeString(status), html.EscapeString(v.Date), html.EscapeString(v.Desc),
	)
}

// CliLoader — п.4: полноценная самостоятельная страница, как и было в openGo
// (в отличие от Rust-варианта тут менять поведение не нужно — оно уже такое).
func CliLoader(w http.ResponseWriter, r *http.Request) {
	current := defaultVer()
	if c, err := r.Cookie("cli_ver"); err == nil {
		if _, ok := findVersion(c.Value); ok {
			current = c.Value
		}
	}

	var alive, lost string

	for _, v := range versions {
		if v.State == StateLostMedia {
			lost += renderRow(v)
		} else {
			alive += renderRow(v)
		}
	}

	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width,initial-scale=1.0">
	<title>OJHUB LOADER v1.30</title>
</head>
<body style="display:flex;justify-content:center;align-items:center;min-height:100vh;flex-direction:column">
	<h1>OJHUB LOADER v1.30</h1>
	<p>selected: %s</p>

	<h2>Avaliable</h2>
	<table border="1">
		<tr><th>ver</th><th>status</th><th>date</th><th>desc</th></tr>
		<tr>
			<td><button onclick="(document.cookie='cli_ver=;path=/;max-age=0');location.pathname=''">stable</button></td>
			<td></td><td></td><td></td>
		</tr>
		%s
	</table>

	<h2>Lost media</h2>
	<table border="1">
		<tr><th>ver</th><th>status</th><th>date</th><th>desc</th></tr>
		%s
	</table>
</body>
</html>`, html.EscapeString(current), alive, lost)
}
