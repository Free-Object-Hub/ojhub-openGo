package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
	"unicode"
	"unicode/utf8"
)

func HandleNewWiki(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	title := r.FormValue("title")
	if title == "" {
		// Claude:
		// PHP-эталон: isset($_POST['title']) как единственный гейт,
		// без title хендлер вообще ничего не делает и не пишет ответ
		// MIOBOMB:
		// да поебать пусть -2 возвращает лол
		w.Write([]byte("-2"))
		return
	}
	text := ExploitPatch(r.FormValue("text"))
	language := ExploitPatch(r.FormValue("language"))
	img := ExploitPatch(r.FormValue("img"))
	titleClean := ExploitPatch(title)
	wiki, err := WIKIcreate(user.UserId, titleClean, text, language, img, time.Now().Unix())
	if err != nil {
		w.Write([]byte(err.Error()))
		//w.Write([]byte("-1"))
		return
	}
	TGWebhookLog(
		fmt.Sprintf("New WIKI %d", wiki.ID),
	)
	json.NewEncoder(w).Encode(wiki.ToLT2())
}

func HandleEditWiki(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	wikiIdStr := r.FormValue("wikiId")
	wikiId, err := strconv.Atoi(wikiIdStr)
	if err != nil {
		w.Write([]byte("Access denied"))
		return
	}
	access, err := CheckWikiAccess(user.UserId, wikiId)
	if err != nil || access == 0 {
		w.Write([]byte("-2"))
		return
	}
	title := r.FormValue("title")
	if title == "" {
		return
	}
	text := ExploitPatch(r.FormValue("text"))
	language := ExploitPatch(r.FormValue("language"))
	img := ExploitPatch(r.FormValue("img"))
	titleClean := ExploitPatch(title)
	wiki, err := WIKIedit(wikiId, titleClean, text, language, img)
	if err != nil {
		w.Write([]byte("-1"))
		return
	}
	TGWebhookLog(
		fmt.Sprintf("New WIKI %d", wiki.ID),
	)
	json.NewEncoder(w).Encode(wiki.ToLT2())
}

func HandleWikiColors(w http.ResponseWriter, r *http.Request) {
	// Claude:
	// унифицировано с остальными, было fetchByToken без device
	// MIOBOMB:
	// раньше можно было просто украв токен изменить цвета
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	wikiId, err := strconv.Atoi(r.FormValue("wiki"))
	if err != nil {
		w.Write([]byte("-1"))
		return
	}
	access, err := CheckWikiAccess(user.UserId, wikiId)
	if err != nil || access == 0 {
		w.Write([]byte("-2"))
		return
	}
	color := ExploitPatch(r.FormValue("color"))
	if err := WIKIsetColor(wikiId, color); err != nil {
		w.Write([]byte("-1"))
		return
	}
	w.Write([]byte("true"))
}

func HandleSetMainWiki(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	wikiId, err := strconv.Atoi(r.FormValue("wiki"))
	if err != nil {
		w.Write([]byte("-1"))
		return
	}
	access, err := CheckWikiAccess(user.UserId, wikiId)
	if err != nil || access == 0 {
		w.Write([]byte("-2"))
		return
	}
	guide, err := GuidesFetchByTag(r.FormValue("guide"), wikiId)
	if err != nil || guide == nil {
		w.Write([]byte("-1"))
		return
	}
	guideId, err := WIKIsetMainWiki(wikiId, guide.ID)
	if err != nil {
		w.Write([]byte("-1"))
		return
	}
	w.Write([]byte(strconv.Itoa(guideId)))
}

func GetGuidesAdmin(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireToken(w, r)
	if !ok {
		return
	}
	wikiId, err := strconv.Atoi(r.URL.Query().Get("wiki"))
	if err != nil {
		w.Write([]byte("-2"))
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	access, err := CheckWikiAccess(user.UserId, wikiId)
	if err != nil || access == 0 {
		w.Write([]byte("-2"))
		return
	}
	guides, err := WIKIfetchGuidesAdmin(wikiId, page)
	if err != nil {
		w.Write([]byte("-2"))
		return
	}
	Guides := make([]interface{}, 0, len(guides))
	for _, g := range guides {
		Guides = append(Guides, renderGuideMini(g))
	}
	json.NewEncoder(w).Encode(Guides)
}

func NewGuide(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB буфер в памяти
		// на случай если клиент всё же шлёт urlencoded, а не multipart
		r.ParseForm()
	}
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	wikiId, err := strconv.Atoi(r.FormValue("wikiId"))
	if err != nil {
		w.Write([]byte("-2"))
		return
	}
	access, err := CheckWikiAccess(user.UserId, wikiId)
	if err != nil || access == 0 {
		w.Write([]byte("-2"))
		return
	}
	if r.FormValue("title") == "" {
		return
	}
	title := ExploitPatch(r.FormValue("title"))
	language := ExploitPatch(r.FormValue("language"))
	aftertext := ExploitPatch(r.FormValue("aftertext"))
	subtitles := r.Form["subtitle[]"]
	subtexts := r.Form["subtext[]"]
	img := r.FormValue("img")
	guideinfo, templatesString := buildGuideInfo(subtitles, subtexts)
	guidId, err := UploadGuide(user.UserId, title, aftertext, guideinfo, language, img, templatesString, time.Now().Unix(), wikiId)
	if err != nil {
		w.Write([]byte("-2"))
		return
	}
	TGWebhookLog(fmt.Sprintf("NEW GUIDE %d UNDER %d", guidId, wikiId))
	fmt.Fprint(w, guidId)
}

func EditGuideHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB буфер в памяти
		// на случай если клиент всё же шлёт urlencoded, а не multipart
		r.ParseForm()
	}
	guidId, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		w.Write([]byte("-2"))
		return
	}
	guide, err := GuidesFetchById(guidId)
	if err != nil {
		w.Write([]byte("-2"))
		return
	}
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	if guide == nil {
		w.Write([]byte(`["NONE"]`))
		return
	}
	access, err := CheckWikiAccess(user.UserId, guide.WikiChannel)
	if err != nil || access == 0 {
		w.Write([]byte("-2"))
		return
	}
	if r.FormValue("title") != "" {
		title := ExploitPatch(r.FormValue("title"))
		language := ExploitPatch(r.FormValue("language"))
		aftertext := ExploitPatch(r.FormValue("aftertext"))
		subtitles := r.Form["subtitle[]"]
		subtexts := r.Form["subtext[]"]
		img := r.FormValue("img")
		guideinfo, templatesString := buildGuideInfo(subtitles, subtexts)
		affected, err := EditGuide(title, aftertext, guideinfo, language, img, templatesString, guidId)
		if err != nil {
			w.Write([]byte("-2"))
			return
		}
		TGWebhookLog(fmt.Sprintf("EDITED GUIDE %d UNDER %d", guidId, guide.WikiChannel))
		fmt.Fprint(w, affected)
		return
	}
	rendered, err := renderGuide(*guide)
	if err != nil {
		w.Write([]byte("-2"))
		return
	}
	json.NewEncoder(w).Encode(rendered)
}

func SetWikiTagHandler(w http.ResponseWriter, r *http.Request) {
	guidId, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		w.Write([]byte("-2"))
		return
	}
	guide, err := GuidesFetchById(guidId)
	if err != nil {
		w.Write([]byte("-2"))
		return
	}
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	if guide == nil {
		w.Write([]byte("-2"))
		return
	}
	access, err := CheckWikiAccess(user.UserId, guide.WikiChannel)
	if err != nil || access == 0 {
		w.Write([]byte("-2"))
		return
	}
	tag := ExploitPatch(r.URL.Query().Get("tag"))
	wikiId, err := strconv.Atoi(r.URL.Query().Get("wiki"))
	if err != nil {
		w.Write([]byte("-2"))
		return
	}
	firstRune, _ := utf8.DecodeRuneInString(tag)
	if unicode.IsDigit(firstRune) {
		w.Write([]byte("-1"))
		return
	}
	if !tagPattern.MatchString(tag) {
		w.Write([]byte("-2"))
		return
	}
	result, err := SetWikiTag(guidId, wikiId, tag)
	if err != nil {
		w.Write([]byte("-2"))
		return
	}
	fmt.Fprint(w, result)
}
