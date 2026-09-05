package main

import (
	"net/http"
	"strconv"
)

func masterConnect(wikiId, gdpsId int) error {
	tx, err := DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE gdpses SET connectedWiki = ? WHERE ID = ?`, wikiId, gdpsId); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE wikis SET connectedGdps = ? WHERE ID = ?`, gdpsId, wikiId); err != nil {
		return err
	}
	return tx.Commit()
}

func dropConnect(wikiId, gdpsId int) error {
	tx, err := DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE gdpses SET connectedWiki = 0 WHERE ID = ?`, gdpsId); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE wikis SET connectedGdps = 0 WHERE ID = ?`, wikiId); err != nil {
		return err
	}
	return tx.Commit()
}

func ConnectContent(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	gdpsId, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		w.Write([]byte("Access denied"))
		return
	}
	wikiId, err := strconv.Atoi(r.FormValue("connectTo"))
	if err != nil {
		w.Write([]byte("Access denied"))
		return
	}
	if gdpsId == 0 {
		wAccess, err := CheckWikiAccess(user.UserId, wikiId)
		if err != nil || wAccess == 0 {
			w.Write([]byte("Access denied"))
			return
		}
		var currentGdps int
		if err := DB.Get(&currentGdps, `SELECT connectedGdps FROM wikis WHERE ID = ?`, wikiId); err != nil {
			w.Write([]byte("Access denied"))
			return
		}
		dropConnect(wikiId, currentGdps)
		return
	}
	gAccess, err1 := CheckGdpsAccess(user.UserId, gdpsId)
	wAccess, err2 := CheckWikiAccess(user.UserId, wikiId)
	if err1 != nil || err2 != nil || gAccess == 0 || wAccess == 0 {
		w.Write([]byte("Access denied"))
		return
	}
	if err := masterConnect(wikiId, gdpsId); err != nil {
		w.Write([]byte("Access denied"))
		return
	}
	var val string
	DB.Get(&val, `SELECT connectedWiki FROM gdpses WHERE ID = ?`, gdpsId)
	w.Write([]byte(val))
}

func ConnectWiki(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	wikiId, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		w.Write([]byte("Access denied"))
		return
	}
	gdpsId, err := strconv.Atoi(r.FormValue("connectTo"))
	if err != nil {
		w.Write([]byte("Access denied"))
		return
	}
	if wikiId == 0 {
		gAccess, err := CheckGdpsAccess(user.UserId, gdpsId)
		if err != nil || gAccess == 0 {
			w.Write([]byte("Access denied"))
			return
		}
		var currentWiki int
		if err := DB.Get(&currentWiki, `SELECT connectedWiki FROM gdpses WHERE ID = ?`, gdpsId); err != nil {
			w.Write([]byte("Access denied"))
			return
		}
		dropConnect(currentWiki, gdpsId)
		return
	}
	gAccess, err1 := CheckGdpsAccess(user.UserId, gdpsId)
	wAccess, err2 := CheckWikiAccess(user.UserId, wikiId)
	if err1 != nil || err2 != nil || gAccess == 0 || wAccess == 0 {
		w.Write([]byte("Access denied"))
		return
	}
	if err := masterConnect(wikiId, gdpsId); err != nil {
		w.Write([]byte("Access denied"))
		return
	}
	var val string
	DB.Get(&val, `SELECT connectedGdps FROM wikis WHERE ID = ?`, wikiId)
	w.Write([]byte(val))
}
