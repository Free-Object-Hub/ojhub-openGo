package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

func userLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form: "+err.Error(), http.StatusBadRequest)
		return
	}
	if r.FormValue("g-recaptcha-response") == "" {
		w.Write([]byte("-3"))
		return
	}
	success, err := reCaptcha(r.FormValue("g-recaptcha-response"))
	if err != nil {
		w.Write([]byte("-3"))
		return
	}
	if !success {
		w.Write([]byte("-3"))
		return
	}

	username := ExploitPatch(r.FormValue("username"))
	password := r.FormValue("password")
	var user *User
	if strings.Contains(username, "@") {
		user, err = GetUserByEmail(username)
	} else {
		user, err = GetUserByUsername(username)
	}
	if err != nil {
		http.Error(w, "Error with user", http.StatusInternalServerError)
		log.Println(err)
		return
	}
	if user == nil {
		w.Write([]byte("-2"))
		return
	}

	if !PasswordVerify(password, user.Password) {
		w.Write([]byte("-1"))
		return
	}

	ip := r.Header.Get("X-Real-Ip")
	city, err := GetCity(ip)
	if err != nil {
		http.Error(w, "Error with city", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	device := r.FormValue("device")
	err = InsertDevice(
		user,
		ip,
		city[0],
		city[1],
		r.Header.Get("User-Agent"),
		device,
		r.FormValue("deviceDynamic"),
	)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	user.CityData = city

	jsonData, err := InitOjhub(ip, user.Token, device, true, true, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(w).Encode(jsonData); err != nil {
		http.Error(w, "User data error: "+err.Error(), http.StatusInternalServerError)
	}
}

func userRegister(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(
			w,
			"Error parsing form: "+err.Error(),
			http.StatusBadRequest,
		)
		return
	}

	recaptcha := r.FormValue("g-recaptcha-response")
	if recaptcha == "" {
		w.Write([]byte("-2"))
		return
	}
	success, err := reCaptcha(recaptcha)
	if err != nil {
		log.Println("reCAPTCHA error:", err)
		w.Write([]byte("-2"))
		return
	}
	if !success {
		w.Write([]byte("-2"))
		return
	}

	username := ExploitPatch(r.FormValue("username"))
	password := r.FormValue("password")
	email := ExploitPatch(r.FormValue("email"))
	if username == "" || password == "" {
		w.Write([]byte("-4"))
		return
	}

	used, err := UserHasUsed(email, username)
	if err != nil {
		log.Println("Failed to check user uniqueness:", err)
		http.Error(
			w,
			"Error checking user",
			http.StatusInternalServerError,
		)
		return
	}
	if used {
		w.Write([]byte("-4"))
		return
	}

	if !ValidateEmail(email) {
		w.Write([]byte("-4"))
		return
	}

	activated, err := GenerateUserVerifyCode()
	if err != nil {
		log.Println("cant get 'activated' value:", err)
		w.Write([]byte("-4"))
		return
	}
	token, err := GenerateUserToken(username)
	if err != nil {
		log.Println("cant generate token:", err)
		w.Write([]byte("-4"))
		return
	}

	user, err := NewUserToken(
		username,
		password,
		email,
		activated,
		token,
		0,
	)
	if err != nil {
		log.Println("Failed to create user:", err)
		http.Error(
			w,
			"Failed to create user",
			http.StatusInternalServerError,
		)
		return
	}

	/*TODO: почта
	if err := SendUserVerifyMail(
		user,
		email,
		activated,
	); err != nil {
		log.Println("Failed to send verification email:", err)
	}
	*/

	ip := r.Header.Get("X-Real-Ip")

	city, err := GetCity(ip)
	if err != nil {
		log.Println("Failed to get city:", err)
		city = [2]string{
			"Unknown",
			"Unknown",
		}
	}

	device := r.FormValue("device")
	err = InsertDevice(
		user,
		ip,
		city[0],
		city[1],
		r.Header.Get("User-Agent"),
		device,
		r.FormValue("deviceDynamic"),
	)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	jsonData, err := InitOjhub(
		ip,
		user.Token,
		device,
		true,
		false,
		false,
	)
	if err != nil {
		http.Error(
			w,
			"Login error: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	if err := json.NewEncoder(w).Encode(jsonData); err != nil {
		http.Error(
			w,
			"User data error: "+err.Error(),
			http.StatusInternalServerError,
		)
	}
}
