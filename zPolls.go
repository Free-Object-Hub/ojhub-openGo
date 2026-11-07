package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

var PollsEnabled = true // false => во всех новостях poll = 1

// news.poll = [ {"m":0,"h":0,"e":123,"r":[0,0]}, "опция0", "опция1", ... ]
type PollParams struct {
	M int   `json:"m"`
	H int   `json:"h"`
	E int64 `json:"e"`
	F int   `json:"f,omitempty"` // только в БД: окончательно закрыт, голоса удалены, r финальный
	C int   `json:"c,omitempty"` // только в ответе: закрыт
	R []int `json:"r,omitempty"` // живые счётчики, после закрытия финальные
}

func (p *PollParams) closed() bool { return p.F == 1 || time.Now().Unix() >= p.E }

func pollParse(raw string) (*PollParams, []json.RawMessage) {
	var arr []json.RawMessage
	if json.Unmarshal([]byte(raw), &arr) != nil || len(arr) < 3 {
		return nil, nil
	}
	var p PollParams
	if json.Unmarshal(arr[0], &p) != nil {
		return nil, nil
	}
	return &p, arr[1:]
}

func pollEncode(p *PollParams, opts []json.RawMessage) string {
	out := make([]any, 0, len(opts)+1)
	out = append(out, p)
	for _, o := range opts {
		out = append(out, o)
	}
	b, _ := json.Marshal(out)
	return string(b)
}

// 13-й элемент массива новости. Чистая функция, БД не трогает.
func (p News) PollElem() any {
	if p.Poll == "" {
		return 0
	}
	if !PollsEnabled {
		return 1
	}
	pp, opts := pollParse(p.Poll)
	if pp == nil {
		return 0
	}
	closed := pp.closed()
	if closed {
		pp.C = 1
	}
	if !(pp.H == 0 || (pp.H == 1 && closed)) {
		pp.R = nil // публично не показываем
	}
	pp.F = 0
	out := make([]any, 0, len(opts)+1)
	out = append(out, pp)
	for _, o := range opts {
		out = append(out, o)
	}
	return out
}

func pollBuild(opts []string, m, h, days string) (string, bool) {
	if len(opts) < 2 || len(opts) > 10 {
		return "", false
	}
	mi, e1 := strconv.Atoi(m)
	hi, e2 := strconv.Atoi(h)
	di, e3 := strconv.Atoi(days)
	if e1 != nil || e2 != nil || e3 != nil || mi < 0 || mi > 1 || hi < 0 || hi > 2 || di < 1 || di > 365 {
		return "", false
	}
	out := []any{PollParams{M: mi, H: hi, E: time.Now().Unix() + int64(di)*86400, R: make([]int, len(opts))}}
	for _, o := range opts {
		o = strings.TrimSpace(o)
		if o == "" || utf8.RuneCountInString(o) > 100 {
			return "", false
		}
		out = append(out, ExploitPatch(o))
	}
	b, err := json.Marshal(out)
	return string(b), err == nil
}

// точные счётчики из таблицы голосов: группируем по маске, раскладываем биты
func pollCountsOne(newsId, n int) ([]int, error) {
	var rows []struct {
		Mask int `db:"mask"`
		C    int `db:"c"`
	}
	if err := DB.Select(&rows, "SELECT mask, COUNT(*) AS c FROM pollVotes WHERE newsId = ? GROUP BY mask", newsId); err != nil {
		return nil, err
	}
	res := make([]int, n)
	for _, r := range rows {
		for i := range res {
			if (r.Mask>>uint(i))&1 == 1 {
				res[i] += r.C
			}
		}
	}
	return res, nil
}

// пересчитать r и записать в строку (compare-and-swap, 3 попытки)
func pollRefresh(newsId int) []int {
	for try := 0; try < 3; try++ {
		var raw string
		if err := DB.Get(&raw, "SELECT poll FROM news WHERE ID = ?", newsId); err != nil {
			return nil
		}
		p, opts := pollParse(raw)
		if p == nil || p.F == 1 {
			return nil
		}
		counts, err := pollCountsOne(newsId, len(opts))
		if err != nil {
			return nil
		}
		p.R = counts
		b := pollEncode(p, opts)
		if b == raw {
			return counts
		}
		res, err := DB.Exec("UPDATE news SET poll = ? WHERE ID = ? AND poll = ?", b, newsId, raw)
		if err != nil {
			return nil
		}
		if n, _ := res.RowsAffected(); n == 1 {
			return counts
		}
	}
	return nil
}

// закрыть окончательно: точный r, f=1, удалить голоса. early=true ставит e=now. Идемпотентно.
func PollStop(newsId int, early bool) ([]int, error) {
	var raw string
	if err := DB.Get(&raw, "SELECT poll FROM news WHERE ID = ?", newsId); err != nil {
		return nil, err
	}
	p, opts := pollParse(raw)
	if p == nil {
		return nil, errors.New("no poll")
	}
	if p.F == 1 {
		return p.R, nil
	}
	counts, err := pollCountsOne(newsId, len(opts))
	if err != nil {
		return nil, err
	}
	p.R, p.F = counts, 1
	if now := time.Now().Unix(); early && p.E > now {
		p.E = now
	}
	tx, err := DB.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	res, err := tx.Exec("UPDATE news SET poll = ? WHERE ID = ? AND poll = ?", pollEncode(p, opts), newsId, raw)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 1 {
		if _, err := tx.Exec("DELETE FROM pollVotes WHERE newsId = ?", newsId); err != nil {
			return nil, err
		}
	}
	return counts, tx.Commit()
}

// уборка просроченных; запускать из тикера в main()
func PollSweep() {
	var rows []struct {
		ID   int    `db:"ID"`
		Poll string `db:"poll"`
	}
	if err := DB.Select(&rows, "SELECT ID, poll FROM news WHERE poll <> ''"); err != nil {
		log.Println("poll sweep:", err)
		return
	}
	for _, r := range rows {
		if p, _ := pollParse(r.Poll); p != nil && p.F == 0 && p.closed() {
			if _, err := PollStop(r.ID, false); err != nil {
				log.Println("poll sweep stop:", r.ID, err)
			}
		}
	}
}

func pollManage(userId, priority, gdpsId int) bool {
	if priority > 0 {
		return true
	}
	c, err := CheckGdpsAccess(userId, gdpsId)
	return err == nil && c != 0
}

// POST polls/vote.php: newsId, mask
// ответ: [counts] только при h=0; при успехе с h!=0 ПУСТОЕ тело; ошибки числами
func PollVote(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	if r.ParseForm() != nil {
		w.Write([]byte("-7"))
		return
	}
	newsId, e1 := strconv.Atoi(r.FormValue("newsId"))
	mask, e2 := strconv.Atoi(r.FormValue("mask"))
	if e1 != nil || e2 != nil {
		w.Write([]byte("-7"))
		return
	}
	news, err := NEWSfetchById(newsId)
	if err != nil || news.Poll == "" {
		w.Write([]byte("-6"))
		return
	}
	p, opts := pollParse(news.Poll)
	if p == nil {
		w.Write([]byte("-6"))
		return
	}
	if p.closed() {
		w.Write([]byte("-3"))
		return
	}
	if mask < 0 || mask >= 1<<uint(len(opts)) || (p.M == 0 && mask&(mask-1) != 0) {
		w.Write([]byte("-7"))
		return
	}
	if mask == 0 {
		_, err = DB.Exec("DELETE FROM pollVotes WHERE newsId = ? AND userId = ?", newsId, user.UserId)
	} else {
		_, err = DB.Exec(
			"INSERT INTO pollVotes (newsId, userId, mask) VALUES (?,?,?) ON DUPLICATE KEY UPDATE mask = VALUES(mask)",
			newsId, user.UserId, mask)
	}
	if err != nil {
		log.Println("poll vote:", err)
		w.Write([]byte("-8"))
		return
	}
	counts := pollRefresh(newsId)
	if p.H != 0 || counts == nil {
		return // пустой ответ = успех
	}
	json.NewEncoder(w).Encode(counts)
}

// POST polls/results.php: newsId — владелец/совладелец/priority>0
func PollResults(w http.ResponseWriter, r *http.Request) {
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
	news, err := NEWSfetchById(newsId)
	if err != nil || news.Poll == "" {
		w.Write([]byte("-6"))
		return
	}
	if !pollManage(user.UserId, user.Priority, news.GdpsId) {
		w.Write([]byte("-2"))
		return
	}
	p, opts := pollParse(news.Poll)
	if p == nil {
		w.Write([]byte("-6"))
		return
	}
	counts := p.R
	if p.F == 0 { // редкий путь владельца: считаем точно
		if counts, err = pollCountsOne(newsId, len(opts)); err != nil {
			w.Write([]byte("-8"))
			return
		}
	}
	json.NewEncoder(w).Encode(counts)
}

// POST polls/stop.php: newsId
func PollStopHandler(w http.ResponseWriter, r *http.Request) {
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
	news, err := NEWSfetchById(newsId)
	if err != nil || news.Poll == "" {
		w.Write([]byte("-6"))
		return
	}
	if !pollManage(user.UserId, user.Priority, news.GdpsId) {
		w.Write([]byte("-2"))
		return
	}
	if p, _ := pollParse(news.Poll); p != nil && p.F == 1 {
		w.Write([]byte("-3")) // уже закрыт окончательно
		return
	}
	counts, err := PollStop(newsId, true)
	if err != nil {
		log.Println("poll stop:", err)
		w.Write([]byte("-8"))
		return
	}
	json.NewEncoder(w).Encode(counts)
}
