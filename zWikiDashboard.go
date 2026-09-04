package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
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
