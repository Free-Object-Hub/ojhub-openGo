package main

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func GetGdpsesByAuthorAndChannel(userID int, channel int) ([]GDPS, error) {
	var result []GDPS
	err := DB.Select(
		&result,
		"SELECT * FROM gdpses WHERE author = ? AND channel = ?",
		userID, channel,
	)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func GetWikisByAuthor(userID int) ([]Wiki, error) {
	var result []Wiki
	err := DB.Select(
		&result,
		"SELECT * FROM wikis WHERE userId = ?",
		userID,
	)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// #region хандлеры

func GetUserHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.Write([]byte("Access denied"))
		return
	}
	user, err := GetUserById(id)
	if err != nil || user == nil {
		w.Write([]byte("[\"NONE\"]"))
		return
	}
	resp, err := json.Marshal(user.PublicProfile())
	if err != nil {
		http.Error(w, "Fehler beim JSON-Parsing", http.StatusInternalServerError)
		return
	}
	w.Write(resp)
}

func GetAddedGdpsesHandler(w http.ResponseWriter, r *http.Request, channel int) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.Write([]byte("Access denied"))
		return
	}
	timeUser, err := GetUserById(id)
	if err != nil || timeUser == nil {
		w.Write([]byte("[\"NONE\"]"))
		return
	}
	gdpses, err := GetGdpsesByAuthorAndChannel(id, channel)
	if err != nil {
		w.Write([]byte("[\"NONE\"]"))
		return
	}
	items := make(map[string]GDPSshort, len(gdpses))
	for _, g := range gdpses {
		items["g"+strconv.Itoa(g.ID)] = g.ToShort(false, false)
	}
	displayName := timeUser.Username
	if timeUser.Nickname != "" {
		displayName = timeUser.Nickname
	}
	resp, err := json.Marshal([]interface{}{displayName, items})
	if err != nil {
		http.Error(w, "Fehler beim JSON-Parsing", http.StatusInternalServerError)
		return
	}
	w.Write(resp)
}

func GetAddedCampsHandler(w http.ResponseWriter, r *http.Request) {
	GetAddedGdpsesHandler(w, r, 0)
}

func GetAddedShowsHandler(w http.ResponseWriter, r *http.Request) {
	GetAddedGdpsesHandler(w, r, 1)
}

func GetAddedPeresHandler(w http.ResponseWriter, r *http.Request) {
	GetAddedGdpsesHandler(w, r, 2)
}

func GetAddedTelesHandler(w http.ResponseWriter, r *http.Request) {
	GetAddedGdpsesHandler(w, r, 3)
}

func GetUserGuidesHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.Write([]byte("Access denied"))
		return
	}
	timeUser, err := GetUserById(id)
	if err != nil || timeUser == nil {
		w.Write([]byte("[\"NONE\"]"))
		return
	}
	wikis, err := GetWikisByAuthor(id)
	if err != nil {
		w.Write([]byte("[\"NONE\"]"))
		return
	}
	items := make(map[string]WikiShort, len(wikis))
	for _, wk := range wikis {
		items["w"+strconv.Itoa(wk.ID)] = wk.ToShort()
	}
	displayName := timeUser.Username
	if timeUser.Nickname != "" {
		displayName = timeUser.Nickname
	}
	resp, err := json.Marshal([]interface{}{displayName, items})
	if err != nil {
		http.Error(w, "Fehler beim JSON-Parsing", http.StatusInternalServerError)
		return
	}
	w.Write(resp)
}

// #endregion
