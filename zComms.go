package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

func LoadMoreComms(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	query := r.URL.Query()
	commentType, err := strconv.Atoi(r.FormValue("type"))
	if err != nil {
		commentType = 0
	}
	commentID, err := strconv.Atoi(query.Get("id"))
	if err != nil {
		commentID = 0
	}
	page, err := strconv.Atoi(query.Get("page"))
	if err != nil {
		page = 0
	}
	comms, err := GetComms(
		commentType,
		commentID,
		page,
	)
	if err != nil {
		http.Error(
			w,
			"Failed to get comments: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}
	if err := json.NewEncoder(w).Encode(comms); err != nil {
		http.Error(
			w,
			"Failed to encode comments: "+err.Error(),
			http.StatusInternalServerError,
		)
	}
}

func CommentSend(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	cached, err := RamGet(fmt.Sprintf("userCommRate:%d", user.UserId))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	channel, err := strconv.Atoi(r.FormValue("type"))
	if err != nil {
		channel = 0
	}
	if channel == 1 {
		channel = 0
	}
	id, err := strconv.Atoi(r.FormValue("ide"))
	if err != nil || id == 0 {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}
	textPre := ExploitPatch(r.FormValue("text"))
	text := base64.StdEncoding.EncodeToString([]byte(textPre))
	//
	if len(textPre) >= 10 {
		if cached != "yes" {
			err = COMMadd(
				user.UserId,
				id,
				text,
				time.Now().Unix(),
				channel,
			)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			RamSet(
				fmt.Sprintf("userCommRate:%d", user.UserId),
				"yes",
				5*time.Minute,
			)
		}
		comms, err := GetComms(channel, id, 0)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// TODO: TGwebhookLog(...)
		json.NewEncoder(w).Encode(comms)
		return
	} else {
		w.Write([]byte("-4"))
		return
	}
}

// FIXME: не делать рейт лимит на базе редиса
func CommentModify(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}
	textPre := r.FormValue("text")
	//
	if user.Activated == 0 {
		return
	}
	//
	if len(textPre) < 10 {
		w.Write([]byte("-4"))
		return
	}
	comm, err := COMMfetchById(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if comm.UserID != user.UserId && user.Priority == 0 {
		w.Write([]byte("-3"))
		return
	}
	text := base64.StdEncoding.EncodeToString(
		[]byte(ExploitPatch(textPre)),
	)
	err = COMMmodify(
		id,
		text,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// TODO TG webhook
	w.Write([]byte(textPre))
}

func CommentDelete(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	id, err := strconv.Atoi(r.URL.Query().Get("ide"))
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}
	comm, err := COMMfetchById(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if comm.UserID != user.UserId && user.Priority == 0 {
		w.Write([]byte("-1"))
		return
	}
	lt := CommLikeTypes[comm.Channel]
	err = COMMdelete(
		id,
		lt.LikeChannel,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// TODO TG webhook
	w.Write([]byte(strconv.Itoa(id)))
}
