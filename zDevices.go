package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

type DeviceResp struct {
	UserAgent string `json:"userAgent"`
	Country   string `json:"country"`
	City      string `json:"city"`
	Platform  string `json:"platform"`
	Browser   string `json:"browser"`
}

func OpenDeviceTab(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDeviceForInactiveAccs(w, r)
	if !ok {
		return
	}
	devices, err := user.GetDevices()
	if err != nil {
		http.Error(
			w,
			"7 "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}
	result := map[string]DeviceResp{}
	for _, d := range devices {
		result[strconv.Itoa(d.ID)] = DeviceResp{
			UserAgent: d.UserAgent,
			Country:   d.Country,
			City:      d.City,
			Platform:  d.Platform,
			Browser:   d.Browser,
		}
	}
	json.NewEncoder(w).Encode(result)
}

func DeviceAddTab(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireToken(w, r)
	if !ok {
		return
	}
	password := r.FormValue("password")
	if !PasswordVerify(password, user.Password) {
		w.Write([]byte("-1"))
		return
	}
	ip := r.Header.Get("X-Real-Ip")
	city, err := GetCity(ip)
	if err != nil {
		city = [2]string{"Unknown", "Unknown"}
	}
	deviceToken := GetDeviceToken(r)
	err = InsertDevice(
		user,
		ip,
		city[0],
		city[1],
		r.Header.Get("User-Agent"),
		r.FormValue("device"),
		r.FormValue("deviceDynamic"),
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data, err := InitOjhub(
		r.Header.Get("X-Real-Ip"),
		user.Token,
		deviceToken,
		false,
		false,
		false,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(data)
}

func RemoveDevice(w http.ResponseWriter, r *http.Request) {
	log.Println("method:", r.Method)
	log.Println("content-type:", r.Header.Get("Content-Type"))

	if err := r.ParseForm(); err != nil {
		log.Println("ParseForm:", err)
	}

	log.Println("form:", r.Form)
	log.Println("post form:", r.PostForm)
	log.Println("query:", r.URL.Query())

	user, ok := RequireDeviceForInactiveAccs(w, r)
	if !ok {
		return
	}
	var err error
	deviceID, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		log.Println(err)
		http.Error(w, "Invalid id parameter", http.StatusBadRequest)
		return
	}
	user.RemoveDeviceById(deviceID)
}
