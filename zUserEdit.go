package main

import (
	"encoding/json"
	"net/http"
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
