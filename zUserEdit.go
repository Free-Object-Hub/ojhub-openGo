package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

func GetConfInfo(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDeviceForInactiveAccs(w, r)
	if !ok {
		return
	}
	password := r.FormValue("password")
	if PasswordVerify(password, user.Password) {
		json.NewEncoder(w).Encode([]string{
			user.Username,
			user.Mail,
		})
		return
	}
	w.Write([]byte("-1"))
}

func SetNicknameHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDeviceForInactiveAccs(w, r)
	if !ok {
		return
	}
	nickname := ExploitPatch(r.URL.Query().Get("name"))

	tx, err := DB.Beginx()
	if err != nil {
		w.Write([]byte("Access denied"))
		return
	}
	defer tx.Rollback()

	if _, err := tx.Exec("UPDATE gdpses SET username = ? WHERE author = ?", nickname, user.UserId); err != nil {
		w.Write([]byte("Access denied"))
		return
	}
	if _, err := tx.Exec("UPDATE forumPosts SET username = ? WHERE userId = ?", nickname, user.UserId); err != nil {
		w.Write([]byte("Access denied"))
		return
	}
	if _, err := tx.Exec("UPDATE users SET nickname = ? WHERE userId = ?", nickname, user.UserId); err != nil {
		w.Write([]byte("Access denied"))
		return
	}

	if err := tx.Commit(); err != nil {
		w.Write([]byte("Access denied"))
		return
	}

	w.Write([]byte(nickname))
}

func SetResumeHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDeviceForInactiveAccs(w, r)
	if !ok {
		return
	}
	resume := ExploitPatch(r.FormValue("name"))
	resume = strings.ReplaceAll(resume, "\\\\n", "\\n")

	_, err := DB.Exec("UPDATE users SET resume = ? WHERE userId = ?", resume, user.UserId)
	if err != nil {
		w.Write([]byte("Access denied"))
		return
	}
	w.Write([]byte(resume))
}

func SetSocialsHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDeviceForInactiveAccs(w, r)
	if !ok {
		return
	}
	socials := ExploitPatch(r.FormValue("name"))
	socials = strings.ReplaceAll(socials, "\\\\n", "\\n")

	_, err := DB.Exec("UPDATE users SET socials = ? WHERE userId = ?", socials, user.UserId)
	if err != nil {
		w.Write([]byte("Access denied"))
		return
	}
	w.Write([]byte(socials))
}
