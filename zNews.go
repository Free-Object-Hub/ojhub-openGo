package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

// FIXME: заменить это на более топорные выдатчики ошибок
type APIError struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func ApiError(w http.ResponseWriter, status int, message string, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	json.NewEncoder(w).Encode(APIError{
		Error: message,
		Code:  code,
	})
}

func AddNews(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	form, file, err := ParseMultipart(r)
	if err != nil {
		ApiError(w, 400, "Form error", "-7")
		return
	}
	gdpsID, err := strconv.Atoi(form.Value["gdps"][0])
	if err != nil || gdpsID <= 0 {
		ApiError(w, 400, "Bad gdps", "-7")
		return
	}
	if user.Activated == 0 {
		return
	}
	// FIXME: что тут делать с ошибкой
	check, err := CheckGdpsAccess(user.UserId, gdpsID)
	if err != nil {
		ApiError(w, 404, "check not found", "-6")
		return
	}
	if check == 0 {
		w.Write([]byte("-2"))
		return
	}
	gdps, err := GDPSfetchById(gdpsID)
	if err != nil {
		ApiError(w, 404, "Data not found", "-6")
		return
	}
	titleRaw, ok1 := form.Value["title"]
	textRaw, ok2 := form.Value["text"]
	if !ok1 || !ok2 {
		return
	}
	title := ExploitPatch(titleRaw[0])
	textPre := ExploitPatch(textRaw[0])
	text := base64.StdEncoding.EncodeToString(
		[]byte(textPre),
	)
	newsID, err := NEWSpost(
		user.UserId,
		gdpsID,
		text,
		time.Now().Unix(),
		title,
		gdps.Checked,
		GetFileExt(file),
	)
	if err != nil {
		ApiError(w, 500, "Database error", "-8")
		return
	}
	if file != nil {
		path := fmt.Sprintf(
			// надеюсь что IMGS имеет слеш на конце
			"%scustomnews/%d.%s",
			getEnv("IMGS", "./imgs/"),
			newsID,
			GetFileExt(file),
		)
		err = SaveFile(file, path)
		if err != nil {
			ApiError(w, 500, "File error", "-9")
		}
	}

	go func() {
		subscribers, err := GetGdpsSubscribers(gdpsID)
		if err != nil {
			log.Println("failed to get subscribers:", err)
			return
		}
		alarmTitle := title
		if len(alarmTitle) > 100 {
			alarmTitle = alarmTitle[:100]
		}
		for _, subId := range subscribers {
			log.Println(subId)
			alarmID, err := FullWrite(
				alarmTitle,
				textPre,
				subId,
				time.Now().Unix(),
				0,
			)
			log.Println("SUBS:", alarmID)
			if err != nil {
				log.Printf("alarm creation failed for user %d: %v\n", subId, err)
				continue
			}
			log.Printf("created alarm %d for user %d\n", alarmID, subId)
		}
	}()

	fmt.Fprintf(w, "%d", newsID)
}

func NewsEdit(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error parsing form: "+err.Error(), http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		http.Error(w, "Invalid news id", http.StatusBadRequest)
		return
	}
	gdpsID, err := strconv.Atoi(r.FormValue("gdps"))
	if err != nil {
		http.Error(w, "Invalid gdps id", http.StatusBadRequest)
		return
	}
	if user.Priority == 0 {
		check, err := CheckGdpsAccess(user.UserId, gdpsID)
		if err != nil {
			http.Error(w, "Permission error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if check == 0 {
			w.Write([]byte("-2"))
			return
		}
	}
	title := ExploitPatch(r.FormValue("title"))
	textPre := ExploitPatch(r.FormValue("text"))
	if r.FormValue("title") == "" || r.FormValue("text") == "" {
		return
	}
	text := base64.StdEncoding.EncodeToString(
		[]byte(textPre),
	)
	newsID, err := NEWSedit(id, text, title, gdpsID)
	if err != nil {
		http.Error(w, "News edit error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if newsID > 0 {
		log.Printf("EDIT NEWS %d with name %s:\n%s", id, title, text)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(
		`["` + title + `","` + textPre + `"]`,
	))
}

func NewsDelete(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	id, err := strconv.Atoi(r.URL.Query().Get("ide"))
	if err != nil {
		http.Error(w, "Invalid news id", http.StatusBadRequest)
		return
	}
	if user.Priority == 0 {
		news, err := NEWSfetchById(id)
		if err != nil {
			http.Error(w, "News fetch error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		check, err := CheckGdpsAccess(user.UserId, news.GdpsId)
		if err != nil {
			http.Error(w, "Permission error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if check == 0 {
			w.Write([]byte("-1"))
			return
		}
	}
	err = NEWSdelete(id)
	if err != nil {
		http.Error(w, "Delete error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(strconv.Itoa(id)))
}
