package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"
)

func GetAlarms(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDeviceForInactiveAccs(w, r)
	if !ok {
		return
	}
	uId := user.UserId
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	alarmsPre, err := GetAlarmsList(uId, user.Priority > 0, page)
	if err != nil {
		w.Write([]byte("[]"))
		return
	}
	if len(alarmsPre) == 0 {
		w.Write([]byte("[]"))
		return
	}
	alarms := [][]interface{}{}
	for _, el := range alarmsPre {
		//alarms = append(alarms, el.RenderMini())
		alarms = append(alarms, []interface{}{el.ID, el.Title, el.Public})
	}
	json.NewEncoder(w).Encode(alarms)
}

func GetAlarm(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDeviceForInactiveAccs(w, r)
	if !ok {
		return
	}
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	alarm, err := GetFullAlarm(id)
	if err != nil {
		w.Write([]byte("{}"))
		return
	}
	if alarm.UserId == user.UserId || user.Priority > 0 {
		UpdateAlarm(id)
		json.NewEncoder(w).Encode(alarm.Render())
	} else {
		w.Write([]byte("{}"))
	}
}

func DeleteAlarm(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDeviceForInactiveAccs(w, r)
	if !ok {
		return
	}
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	alarm, err := GetFullAlarm(id)
	if err != nil {
		w.Write([]byte("{}"))
		return
	}
	if alarm.UserId == user.UserId || user.Priority > 0 {
		RemoveAlarm(id)
		json.NewEncoder(w).Encode(alarm.Render())
	} else {
		w.Write([]byte("{}"))
	}
}

func WriteAlarmHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDeviceForInactiveAccs(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		w.Write([]byte("0"))
		return
	}
	r.ParseMultipartForm(32 << 20)
	title := r.PostFormValue("title")
	text := r.PostFormValue("text")
	anonymus := r.PostFormValue("anonymus")
	log.Println(title, text)
	//
	var userId int
	var adminName string
	var adminId int
	if user.Priority != 0 && anonymus == "on" {
		adminName = "Object Hub"
		adminId = 0
	} else {
		adminName = user.Nickname
		if adminName == "" {
			adminName = user.Username
		}
		adminId = user.UserId
	}
	if user.Priority == 0 {
		userId = 0
	} else {
		userId, _ = strconv.Atoi(r.PostFormValue("user"))
	}
	date := time.Now().Unix()
	_, err := FullWrite(title, text, userId, date, adminId)
	if err != nil {
		w.Write([]byte("0"))
		return
	}
	w.Write([]byte("1"))
}
