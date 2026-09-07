package main

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func AdminTakeAll(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		w.Write([]byte("Access denied"))
		return
	}
	if user.Priority == 0 {
		w.Write([]byte("Access denied"))
		return
	}
	if wikiParam := r.URL.Query().Get("wiki"); wikiParam != "" {
		wikiID, err := strconv.Atoi(wikiParam)
		if err != nil {
			http.Error(w, "bad wiki id", 400)
			return
		}
		guides, err := fetchGuidesByWiki(wikiID)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		rendered := make([][]interface{}, 0, len(guides))
		for _, g := range guides {
			rendered = append(rendered, renderGuideMini(g))
		}
		body, err := json.Marshal(rendered)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write(body)
		return
	}
	gdpses, err := fetchGdpses()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	wikis, err := fetchWikis()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	vacs, err := fetchVacs()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	resp := []interface{}{gdpses, wikis, vacs}
	body, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write(body)
}

type AdminGdps struct {
	ID      int    `db:"ID" json:"ID"`
	Title   string `db:"title" json:"title"`
	Img     string `db:"img" json:"img"`
	Channel int    `db:"channel" json:"channel"`
	Checked int    `db:"checked" json:"checked"`
}

type AdminWiki struct {
	ID      int    `db:"ID" json:"ID"`
	Title   string `db:"title" json:"title"`
	Img     string `db:"img" json:"img"`
	Checked int    `db:"checked" json:"checked"`
}

type AdminVac struct {
	ID      int    `db:"ID" json:"ID"`
	Title   string `db:"title" json:"title"`
	GdpsId  int    `db:"gdpsId" json:"gdpsId"`
	Checked int    `db:"checked" json:"checked"`
}

type AdminGuide struct {
	ID       int    `db:"ID" json:"ID"`
	Title    string `db:"title" json:"title"`
	Language string `db:"language" json:"language"`
	Date     int64  `db:"date" json:"date"`
	Img      string `db:"img" json:"img"`
	Checked  int    `db:"checked" json:"checked"`
	WikiTag  string `db:"wikiTag" json:"wikiTag"`
}

func fetchGdpses() ([]AdminGdps, error) {
	var out []AdminGdps
	err := DB.Select(&out, `
		SELECT ID, title, channel, img, checked
		FROM gdpses
		ORDER BY checked = 0 DESC
	`)
	return out, err
}

func fetchWikis() ([]AdminWiki, error) {
	var out []AdminWiki
	err := DB.Select(&out, `
		SELECT ID, title, img, checked
		FROM wikis
		ORDER BY checked = 0 DESC
	`)
	return out, err
}

func fetchVacs() ([]AdminVac, error) {
	var out []AdminVac
	err := DB.Select(&out, `
		SELECT ID, title, gdpsId, checked
		FROM vacans
		ORDER BY checked = 0 DESC
	`)
	return out, err
}

func fetchGuidesByWiki(wikiID int) ([]Guide, error) {
	var out []Guide
	err := DB.Select(&out, `
		SELECT ID, title, language, date, likes, img, checked
		FROM guides
		WHERE wikiChannel = ?
		ORDER BY checked = 0 DESC
	`, wikiID)
	return out, err
}

type entityType struct {
	table     string
	unbanLog  string
	banLog    string
	deleteLog string
	lgbt1Log  string
	lgbt2Log  string
}

var entityTypes = map[int]entityType{
	0:  {table: "gdpses", unbanLog: GDPSunban, banLog: GDPSban, deleteLog: GDPSdelete, lgbt1Log: GDPSlgbt1, lgbt2Log: GDPSlgbt2},
	-1: {table: "wikis", unbanLog: WIKIunban, banLog: WIKIban, deleteLog: WIKIdelete, lgbt1Log: WIKIlgbt1, lgbt2Log: WIKIlgbt2},
	-2: {table: "guides", unbanLog: GUIDunban, banLog: GUIDban, deleteLog: GUIDdelete, lgbt1Log: GUIDlgbt1, lgbt2Log: GUIDlgbt2},
	-5: {table: "vacans", unbanLog: VACunban, banLog: VACban, deleteLog: VACdelete, lgbt1Log: VAClgbt1, lgbt2Log: VAClgbt2},
}

func resolveEntityType(rawType int) (entityType, bool) {
	if rawType >= 0 {
		return entityTypes[0], true
	}
	et, ok := entityTypes[rawType]
	return et, ok
}

func Aaction(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		w.Write([]byte("Access denied"))
		return
	}
	if user.Priority < 1 {
		w.Write([]byte("Access denied"))
		return
	}

	rawType, err := strconv.Atoi(r.URL.Query().Get("type"))
	if err != nil {
		w.Write([]byte("Access denied"))
		return
	}
	et, ok := resolveEntityType(rawType)
	if !ok {
		w.Write([]byte("Access denied"))
		return
	}

	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		w.Write([]byte("Access denied"))
		return
	}

	uText := "; user- " + strconv.Itoa(user.UserId)
	action := r.URL.Query().Get("action")

	switch action {
	case "activate":
		if _, err := DB.Exec("UPDATE `"+et.table+"` SET `checked` = 1 WHERE `ID` = ?", id); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write([]byte("1"))
		go TGWebhookLog(et.unbanLog + strconv.Itoa(id) + uText)

	case "ban":
		if _, err := DB.Exec("UPDATE `"+et.table+"` SET `checked` = -1 WHERE `ID` = ?", id); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write([]byte("-1"))
		go TGWebhookLog(et.banLog + strconv.Itoa(id) + uText)

	case "delete":
		if user.Priority < 2 {
			w.Write([]byte("Access denied"))
			return
		}
		if _, err := DB.Exec("DELETE FROM `"+et.table+"` WHERE `ID` = ?", id); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write([]byte("-2"))
		go TGWebhookLog(et.deleteLog + strconv.Itoa(id) + uText)

	case "setlgbt":
		if _, err := DB.Exec("UPDATE `"+et.table+"` SET `hasLgbt` = 1 WHERE `ID` = ?", id); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write([]byte("0"))
		go TGWebhookLog(et.lgbt1Log + strconv.Itoa(id) + uText)

	case "dellgbt":
		if _, err := DB.Exec("UPDATE `"+et.table+"` SET `hasLgbt` = 0 WHERE `ID` = ?", id); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write([]byte("0"))
		go TGWebhookLog(et.lgbt2Log + strconv.Itoa(id) + uText)

	default:
		w.Write([]byte("Access denied"))
	}
}
