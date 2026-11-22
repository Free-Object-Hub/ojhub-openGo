package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"
)

const PollCustomMax = 200 // символов в одном ответе
const PollCustomPage = 50 // записей на страницу для владельца

// POST polls/custom.php: newsId, text (пустой text = снять свой ответ)
// успех: пустое тело
func PollCustomVote(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	if r.ParseForm() != nil {
		w.Write([]byte("-7"))
		return
	}
	newsId, err := strconv.Atoi(r.FormValue("newsId"))
	if err != nil {
		w.Write([]byte("-7"))
		return
	}
	text := strings.TrimSpace(r.FormValue("text"))
	news, err := NEWSfetchById(newsId)
	if err != nil || news.Poll == "" {
		w.Write([]byte("-6"))
		return
	}
	p, _ := pollParse(news.Poll)
	if p == nil {
		w.Write([]byte("-6"))
		return
	}
	if p.X != 1 {
		w.Write([]byte("-4")) // community choice выключен или поля нет
		return
	}
	if p.closed() {
		w.Write([]byte("-3"))
		return
	}
	if utf8.RuneCountInString(text) > PollCustomMax {
		w.Write([]byte("-9"))
		return
	}
	if text == "" {
		_, err = DB.Exec("DELETE FROM pollCustomAnswers WHERE newsId = ? AND userId = ?", newsId, user.UserId)
	} else {
		_, err = DB.Exec(
			"INSERT INTO pollCustomAnswers (newsId, userId, txt) VALUES (?,?,?) ON DUPLICATE KEY UPDATE txt = VALUES(txt)",
			newsId, user.UserId, text)
	}
	if err != nil {
		log.Println("poll custom vote:", err)
		w.Write([]byte("-8"))
	}
}

// POST polls/customs.php: newsId, after (курсор по ID, 0 = с начала) — только владелец/совладелец
// ответ: {"top":[{txt,c}...], "items":[{id,txt}...], "next": id или 0}; top только на первой странице
func PollCustomsList(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	if r.ParseForm() != nil {
		w.Write([]byte("-7"))
		return
	}
	newsId, err := strconv.Atoi(r.FormValue("newsId"))
	if err != nil {
		w.Write([]byte("-7"))
		return
	}
	after := 0
	if s := r.FormValue("after"); s != "" {
		if after, err = strconv.Atoi(s); err != nil {
			w.Write([]byte("-7"))
			return
		}
	}
	news, err := NEWSfetchById(newsId)
	if err != nil || news.Poll == "" {
		w.Write([]byte("-6"))
		return
	}
	if !pollManage(user.UserId, user.Priority, news.GdpsId) {
		w.Write([]byte("-2"))
		return
	}
	p, _ := pollParse(news.Poll)
	if p == nil {
		w.Write([]byte("-6"))
		return
	}
	if p.X != 1 {
		w.Write([]byte("-4"))
		return
	}

	var top []struct {
		Txt string `db:"txt" json:"txt"`
		C   int    `db:"c" json:"c"`
	}
	if after == 0 {
		if err := DB.Select(&top,
			"SELECT MIN(txt) AS txt, COUNT(*) AS c FROM pollCustomAnswers WHERE newsId = ? "+
				"GROUP BY LOWER(TRIM(txt)) ORDER BY c DESC LIMIT 10", newsId); err != nil {
			w.Write([]byte("-8"))
			return
		}
	}

	var items []struct {
		ID  int    `db:"ID" json:"id"`
		Txt string `db:"txt" json:"txt"`
	}
	if err := DB.Select(&items,
		"SELECT ID, txt FROM pollCustomAnswers WHERE newsId = ? AND ID > ? ORDER BY ID LIMIT ?",
		newsId, after, PollCustomPage+1); err != nil {
		w.Write([]byte("-8"))
		return
	}
	next := 0
	if len(items) > PollCustomPage {
		items = items[:PollCustomPage]
		next = items[PollCustomPage-1].ID
	}
	json.NewEncoder(w).Encode(map[string]any{"top": top, "items": items, "next": next})
}

// POST polls/customDelete.php: newsId, id — владелец/совладелец
func PollCustomDelete(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	if r.ParseForm() != nil {
		w.Write([]byte("-7"))
		return
	}
	newsId, e1 := strconv.Atoi(r.FormValue("newsId"))
	cid, e2 := strconv.Atoi(r.FormValue("id"))
	if e1 != nil || e2 != nil {
		w.Write([]byte("-7"))
		return
	}
	news, err := NEWSfetchById(newsId)
	if err != nil || news.Poll == "" {
		w.Write([]byte("-6"))
		return
	}
	if !pollManage(user.UserId, user.Priority, news.GdpsId) {
		w.Write([]byte("-2"))
		return
	}
	if _, err := DB.Exec("DELETE FROM pollCustomAnswers WHERE ID = ? AND newsId = ?", cid, newsId); err != nil {
		w.Write([]byte("-8"))
	}
}
