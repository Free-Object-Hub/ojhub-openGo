package main

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func GetOwners(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	channelType, err := strconv.Atoi(r.URL.Query().Get("type"))
	if err != nil {
		http.Error(w, "invalid type", http.StatusBadRequest)
		return
	}
	var title string
	var usernames [][2]any
	if channelType == -1 {
		wiki, err := WIKIfetchById(id) // предполагаю что такая функция уже есть
		if err != nil {
			http.Error(w, "wiki not found", http.StatusNotFound)
			return
		}
		title = wiki.Title
		owners, err := FetchOwnedWiki(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for _, o := range owners {
			username := "???"
			if o.UserId != 0 {
				u, err := GetUserById(o.UserId)
				if err == nil && u != nil {
					username = u.Nickname
					if username == "" {
						username = u.Username
					}
				}
			}
			usernames = append(usernames, [2]any{username, o.UserId})
		}
	} else if channelType == -4 {
		gdps, err := GDPSfetchById(id)
		if err != nil {
			http.Error(w, "gdps not found", http.StatusNotFound)
			return
		}
		title = gdps.Title
		entries, err := FetchJoinLog(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for _, e := range entries {
			username := "???"
			if e.UserId != 0 {
				u, err := GetUserById(e.UserId)
				if err == nil && u != nil {
					username = u.Nickname
					if username == "" {
						username = u.Username
					}
				}
			}
			usernames = append(usernames, [2]any{username, e.UserId})
		}
	} else {
		gdps, err := GDPSfetchById(id)
		if err != nil {
			http.Error(w, "gdps not found", http.StatusNotFound)
			return
		}
		title = gdps.Title
		owners, err := FetchOwnedGdps(id, channelType)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for _, o := range owners {
			username := "???"
			if o.UserId != 0 {
				u, err := GetUserById(o.UserId)
				if err == nil && u != nil {
					username = u.Nickname
					if username == "" {
						username = u.Username
					}
				}
			}
			usernames = append(usernames, [2]any{username, o.UserId})
		}
	}
	if usernames == nil {
		usernames = [][2]any{}
	}
	resp := []any{[]string{title}, usernames}
	json.NewEncoder(w).Encode(resp)
}

func PermAdd(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	gdpsId, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		w.Write([]byte("-2"))
		return
	}
	channelType, err := strconv.Atoi(r.URL.Query().Get("type"))
	if err != nil {
		w.Write([]byte("-2"))
		return
	}
	targetUserId, err := strconv.Atoi(r.URL.Query().Get("user"))
	if err != nil {
		w.Write([]byte("-2"))
		return
	}
	var access int
	if channelType == -1 {
		access, err = CheckWikiAccess(user.UserId, gdpsId)
	} else {
		access, err = CheckGdpsAccess(user.UserId, gdpsId)
	}
	if err != nil {
		w.Write([]byte("-2"))
		return
	}
	if access != 2 {
		w.Write([]byte("-2"))
		return
	}
	targetUser, err := GetUserById(targetUserId)
	if err != nil {
		w.Write([]byte("-2"))
		return
	}
	if channelType == -1 {
		err = AddOwnerWiki(gdpsId, targetUser.UserId)
	} else {
		err = AddOwnerGdps(gdpsId, targetUser.UserId, channelType)
	}
	if err != nil {
		w.Write([]byte("-2"))
		return
	}
	username := targetUser.Nickname
	if username == "" {
		username = targetUser.Username
	}
	resp := []any{username, targetUser.UserId}
	json.NewEncoder(w).Encode(resp)
}

func PermRemove(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	gdpsId, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		w.Write([]byte("-2"))
		return
	}
	channelType, err := strconv.Atoi(r.URL.Query().Get("type"))
	if err != nil {
		w.Write([]byte("-2"))
		return
	}
	targetUserId, err := strconv.Atoi(r.URL.Query().Get("user"))
	if err != nil {
		w.Write([]byte("-2"))
		return
	}
	var access int
	if channelType == -1 {
		access, err = CheckWikiAccess(user.UserId, gdpsId)
	} else {
		access, err = CheckGdpsAccess(user.UserId, gdpsId)
	}
	if err != nil || access != 2 {
		w.Write([]byte("-2"))
		return
	}
	if channelType == -1 {
		err = DeleteOwnerWiki(gdpsId, targetUserId)
	} else {
		err = DeleteOwnerGdps(gdpsId, targetUserId, channelType)
	}
	if err != nil {
		w.Write([]byte("-2"))
		return
	}
	resp := []any{access}
	json.NewEncoder(w).Encode(resp)
}
