package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

func TemplateGet(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireToken(w, r)
	if !ok {
		return
	}
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || id == 0 {
		return // тихо, как в оригинале
	}
	access, err := CheckWikiAccess(user.UserId, id)
	if err != nil || access == 0 {
		return // тихо, как в оригинале
	}
	template, err := WikiTemplateGetOne(id, r.URL.Query().Get("name"))
	if err != nil || template == nil {
		return
	}
	json.NewEncoder(w).Encode(template.RenderTemplate())
}

func TemplatesGet(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireToken(w, r)
	if !ok {
		return
	}
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || id == 0 {
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	access, err := CheckWikiAccess(user.UserId, id)
	if err != nil || access == 0 {
		return
	}
	templates, err := WikiTemplateGetAll(id, page)
	if err != nil {
		return
	}
	result := make(map[string]interface{}, len(templates))
	for _, t := range templates {
		result[t.Name] = t.RenderTemplate()
	}
	json.NewEncoder(w).Encode(result)
}

func allowedMethods(method string) string {
	switch method {
	case "Markdown":
		return "Markdown"
	case "wikiText":
		return "wikiText"
	default:
		return "Markdown"
	}
}

var safeNameRe = regexp.MustCompile(`[^a-zA-Z0-9_\-]`)

func doSafeName(name string) string {
	safe := safeNameRe.ReplaceAllString(name, "")
	if len(safe) == 0 { // фикс: guard до обращения к [0], в PHP тут был undefined-index warning без краша
		return "falseName"
	}
	if unicode.IsDigit(rune(safe[0])) {
		return "falseName"
	}
	return safe
}

func doSafeCode(mdCode string, args []string) string {
	mdCode = strings.ReplaceAll(mdCode, "`", "")
	for _, m := range templateArgRe.FindAllStringSubmatch(mdCode, -1) {
		param := m[1]
		found := false
		for _, a := range args {
			if strings.TrimSpace(param) == a {
				found = true
				break
			}
		}
		if !found {
			mdCode = strings.ReplaceAll(mdCode, "${"+param+"}", "")
		}
	}
	mdCode = templateCallRe.ReplaceAllString(mdCode, "")
	mdCode = strings.NewReplacer(
		`\u`, `u`,
		`\x`, `x`,
		`\`, `\\`,
	).Replace(mdCode)
	return mdCode
}

var (
	templateArgRe  = regexp.MustCompile(`(?s)\$\{([^}]+)\}`)
	templateCallRe = regexp.MustCompile(`(?s)\{\{[^}]+\}\}`)
)

func TemplateSave(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		w.Write([]byte("-1"))
		return
	}
	access, err := CheckWikiAccess(user.UserId, id)
	if err != nil || access == 0 {
		w.Write([]byte("-1"))
		return
	}
	if id == 0 {
		return // тихо, как в оригинале
	}
	name := doSafeName(r.FormValue("name"))
	method := allowedMethods(r.FormValue("method"))
	rawArgs := r.Form["arg[]"] // как и с subtitle[]/subtext[] — реальное имя поля с квадратными скобками
	var argsJSON string
	if len(rawArgs) > 0 {
		safeArgs := make([]string, len(rawArgs))
		for i, a := range rawArgs {
			safeArgs[i] = doSafeName(a)
		}
		b, _ := json.Marshal(safeArgs)
		argsJSON = string(b)
	} else {
		argsJSON = ""
	}
	var argsForCode []string
	json.Unmarshal([]byte(argsJSON), &argsForCode) // если argsJSON == "", просто останется nil — эквивалент json_decode('') в PHP
	content := doSafeCode(r.FormValue("content"), argsForCode)
	template, err := WikiTemplateSave(id, name, argsJSON, method, content)
	if err != nil || template == nil {
		return
	}
	json.NewEncoder(w).Encode(template.RenderTemplate())
}

func TemplateDelete(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || id == 0 {
		return
	}
	access, err := CheckWikiAccess(user.UserId, id)
	if err != nil || access == 0 {
		return
	}
	if err := WikiTemplateDelete(id, r.URL.Query().Get("name")); err != nil {
		return
	}
	fmt.Fprint(w, "1") // PHP echo bool от PDOStatement::execute() — true печатается как "1"
}
