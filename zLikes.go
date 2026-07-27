package main

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func GetParam(r *http.Request, key string) string {
	value := r.URL.Query().Get(key)
	if value != "" {
		return value
	}

	r.ParseForm()
	return r.FormValue(key)
}

func LikeSend(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	ide := GetParam(r, "ide")
	typePre := GetParam(r, "type")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	id, err := strconv.Atoi(ide)
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}
	typeID, err := strconv.Atoi(typePre)
	if err != nil {
		http.Error(w, "Invalid type", http.StatusBadRequest)
		return
	}
	lt, ok := LikeTypes[typeID]
	if !ok {
		http.Error(w, "Unknown like type", http.StatusBadRequest)
		return
	}
	like, err := CheckLike(
		id,
		user.UserId,
		lt.LikeChannel,
	)
	if err != nil {
		http.Error(w, "1 "+err.Error(), http.StatusInternalServerError)
		return
	}
	var result [2]int
	if like == nil {
		result, err = LikeSet(
			id,
			lt,
			user.UserId,
			false,
		)
	} else {
		result, err = RemoveLike(
			like,
			id,
			lt,
		)
	}
	if err != nil {
		http.Error(w, "2 "+err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(result)
}

func DislSend(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	ide := GetParam(r, "ide")
	typePre := GetParam(r, "type")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	id, err := strconv.Atoi(ide)
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}
	typeID, err := strconv.Atoi(typePre)
	if err != nil {
		http.Error(w, "Invalid type", http.StatusBadRequest)
		return
	}
	lt, ok := LikeTypes[typeID]
	if !ok {
		http.Error(w, "Unknown like type", http.StatusBadRequest)
		return
	}
	like, err := CheckLike(
		id,
		user.UserId,
		lt.LikeChannel,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var result [2]int
	if like == nil {
		result, err = LikeSet(
			id,
			lt,
			user.UserId,
			true,
		)
	} else {
		result, err = RemoveLike(
			like,
			id,
			lt,
		)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(result)
}
