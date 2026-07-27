package main

/*
 * НЕТ, это не middleware!
 * в привычном вам смысле.
 *
 * Это просто функции которые выглядят как middleware, но в прямом смысле им не являются
 * Ну и ещё тут сложены прочие хелперы
 */

import (
	"html"
	"net/http"
	//
	_ "github.com/go-sql-driver/mysql"
)

// хелперы для костылей совместимости
func GetUserToken(r *http.Request) string {
	token := r.Header.Get("User-Token")
	if token != "" {
		return token
	}
	return r.FormValue("token")
}

func GetDeviceToken(r *http.Request) string {
	device := r.Header.Get("Device-Static")
	if device != "" {
		return device
	}
	return r.FormValue("device")
}

func ExploitPatch(s string) string {
	return html.EscapeString(s)
}

// логика
func RequireToken(w http.ResponseWriter, r *http.Request) (*User, bool) {
	token := GetUserToken(r)
	if token == "" {
		//http.Error(w, `{"error":"Authentication required","code":"-2"}`, http.StatusUnauthorized)
		w.Write([]byte("Access denied"))
		return nil, false
	}
	user, err := GetUserByToken(token)
	if err != nil || user == nil {
		//http.Error(w, `{"error":"No user data","code":"-3"}`, http.StatusUnauthorized)
		w.Write([]byte("Access denied"))
		return nil, false
	}
	return user, true
}

func RequireDevice(w http.ResponseWriter, r *http.Request) (*User, bool) {
	token := GetUserToken(r)
	device := GetDeviceToken(r)
	if token == "" || device == "" {
		//http.Error(w, `{"error":"Authentication required","code":"-2"}`, http.StatusUnauthorized)
		w.Write([]byte("Access denied"))
		return nil, false
	}
	user, err := GetUserByTokenAndDevice(token, device)
	if err != nil || user == nil {
		//http.Error(w, `{"error":"No user data","code":"-3"}`, http.StatusUnauthorized)
		w.Write([]byte("Access denied"))
		return nil, false
	}
	if user.Activated == 0 {
		//http.Error(w, `{"error":"Account not verified","code":"-4"}`, http.StatusForbidden)
		w.Write([]byte("Access denied"))
		return nil, false
	}
	return user, true
}

func RequireDeviceForInactiveAccs(w http.ResponseWriter, r *http.Request) (*User, bool) {
	token := GetUserToken(r)
	device := GetDeviceToken(r)
	if token == "" || device == "" {
		//http.Error(w, `{"error":"Authentication required","code":"-2"}`, http.StatusUnauthorized)
		w.Write([]byte("Access denied"))
		return nil, false
	}
	user, err := GetUserByTokenAndDevice(token, device)
	if err != nil || user == nil {
		//http.Error(w, `{"error":"No user data","code":"-3"}`, http.StatusUnauthorized)
		w.Write([]byte("Access denied"))
		return nil, false
	}
	return user, true
}

func RequirePerms(w http.ResponseWriter, r *http.Request) (*User, bool) {
	user, ok := RequireToken(w, r)
	if !ok {
		return nil, false
	}
	if user.Priority == 0 {
		return nil, false
	}
	return user, true
}
