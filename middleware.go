package main

/*
 * NO, this is not middleware!
 * in the usual sense you're used to.
 *
 * These are just functions that look like middleware, but in the literal sense they are not
 * And also various other helpers are collected here
 */

import (
	"html"
	"net/http"
	//
	_ "github.com/go-sql-driver/mysql"
)

/*
Object Hub 0.97.3 and older are PHP generation versions,
due to the limitations of helperRequest they sent token and device tokens
in the body, all versions of the old generation worked roughly like this.
The nodejs migration made me think "maybe let the token be in the header?",
which is what I did. Hence such a mess of crutches for compatibility
*/
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

// logic
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

func truncateRunes(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}
