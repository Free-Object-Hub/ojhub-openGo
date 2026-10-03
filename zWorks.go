package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"
)

func WorksGet(w http.ResponseWriter, r *http.Request) {
	userId, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		fmt.Fprint(w, "-1")
		return
	}
	user, err := GetUserById(userId)
	if err != nil {
		fmt.Fprint(w, "-2")
		return
	}
	works, err := WORKfetchByUserId(userId)
	if err != nil {
		fmt.Fprint(w, "-3")
		return
	}
	resp := make([]interface{}, 0, len(works)+1)
	resp = append(resp, map[string]interface{}{"resume": user.Resume, "username": user.GetNick()})
	for _, work := range works {
		resp = append(resp, work.RenderWork())
	}
	json.NewEncoder(w).Encode(resp)
}

const (
	workTitleMax    = 100  // TODO: сверь с длиной колонки title
	workTextMax     = 2000 // TODO: сверь с типом колонки text
	workLinkMax     = 255  // TODO: сверь с колонкой gdps
	worksPerUserMax = 30   // TODO: свой лимит
)

func workAddHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Write([]byte("Bad method"))
		return
	}
	user, ok := RequireDevice(w, r)
	if !ok {
		w.Write([]byte("Access denied"))
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		w.Write([]byte("Bad form"))
		return
	}
	title := strings.TrimSpace(r.FormValue("title"))
	text := strings.TrimSpace(r.FormValue("text"))
	link := strings.TrimSpace(r.FormValue("gdpsId"))
	linkType, err := strconv.Atoi(r.FormValue("linkType"))
	if err != nil || (linkType != 0 && linkType != 1) {
		w.Write([]byte("Bad linkType"))
		return
	}
	if title == "" || utf8.RuneCountInString(title) > workTitleMax {
		w.Write([]byte("Bad title"))
		return
	}
	if text == "" || utf8.RuneCountInString(text) > workTextMax {
		w.Write([]byte("Bad text"))
		return
	}
	if link == "" || utf8.RuneCountInString(link) > workLinkMax {
		w.Write([]byte("Bad link"))
		return
	}
	switch linkType {
	case 0:
		gid, err := strconv.ParseUint(link, 10, 32)
		if err != nil || gid == 0 {
			w.Write([]byte("Bad project id"))
			return
		}
		var n int
		if err := DB.Get(&n, `SELECT COUNT(*) FROM gdpses WHERE ID = ?`, gid); err != nil {
			log.Println("workAdd: gdps check:", err)
			w.Write([]byte("Internal error"))
			return
		}
		if n == 0 {
			w.Write([]byte("Project not found"))
			return
		}
		link = strconv.FormatUint(gid, 10) // без "007" и "+5"
	case 1:
		u, err := url.Parse(link)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			w.Write([]byte("Bad url"))
			return
		}
	}
	var cnt int
	if err := DB.Get(&cnt, `SELECT COUNT(*) FROM works WHERE userId = ?`, user.UserId); err != nil {
		log.Println("workAdd: count:", err)
		w.Write([]byte("Internal error"))
		return
	}
	if cnt >= worksPerUserMax {
		w.Write([]byte("Works limit reached"))
		return
	}
	id, err := AddWork(user.UserId, title, text, "[]", 0, linkType, link)
	if err != nil {
		log.Println("workAdd:", err)
		w.Write([]byte("Internal error"))
		return
	}
	newWork, err := WORKfetchById(id)
	if err != nil {
		w.Write([]byte("Dead work"))
		return
	}
	resp := newWork.RenderWork()
	//resp := map[string]int{"id": id}
	json.NewEncoder(w).Encode(resp)
}

func workEditHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	// ErrNotMultipart не ошибка: GET-запрос формы (helperRequest) может прийти urlencoded
	if err := r.ParseMultipartForm(1 << 20); err != nil && err != http.ErrNotMultipart {
		ApiError(w, 400, "Bad form", "-7")
		return
	}
	// FormValue смотрит и в тело, и в query, так что работает и для первого запроса, и для onsubmit с ?id=
	workId, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		ApiError(w, 400, "Bad id", "-7")
		return
	}
	work, err := WORKfetchById(workId)
	if err != nil || work.UserID != user.UserId {
		// не найдена и чужая отвечают одинаково, чтобы не светить существование
		w.Write([]byte("-2"))
		return
	}
	title := r.FormValue("title")
	text := r.FormValue("text")
	// GET-режим: title/text отсутствуют, отдаём текущие данные для формы
	if title == "" || text == "" {
		resp := map[string]any{
			"title":    work.Title,
			"text":     work.Text,
			"tags":     work.Tags, // сырая JSON-строка, клиент сам делает JSON.parse
			"linkType": work.LinkType,
			"gdps":     work.LinkOrGdpsId,
		}
		json.NewEncoder(w).Encode(resp)
		return
	}
	if user.Activated == 0 {
		w.Write([]byte("Access denied"))
		return
	}
	title = strings.TrimSpace(ExploitPatch(title))
	text = strings.TrimSpace(ExploitPatch(text))
	link := strings.TrimSpace(r.FormValue("gdpsId"))
	if title == "" || utf8.RuneCountInString(title) > workTitleMax {
		ApiError(w, 400, "Bad title", "-27")
		return
	}
	if text == "" || utf8.RuneCountInString(text) > workTextMax {
		ApiError(w, 400, "Bad text", "-27")
		return
	}
	linkType, err := strconv.Atoi(r.FormValue("linkType"))
	if err != nil || (linkType != 0 && linkType != 1) {
		ApiError(w, 400, "Bad linkType", "-28")
		return
	}
	if link == "" || utf8.RuneCountInString(link) > workLinkMax {
		ApiError(w, 400, "Bad link", "-28")
		return
	}
	switch linkType {
	case 0:
		gid, err := strconv.ParseUint(link, 10, 32)
		if err != nil || gid == 0 {
			ApiError(w, 400, "Bad project id", "-28")
			return
		}
		var n int
		if err := DB.Get(&n, `SELECT COUNT(*) FROM gdpses WHERE ID = ?`, gid); err != nil {
			log.Println("workEdit: gdps check:", err)
			w.Write([]byte("-1"))
			return
		}
		if n == 0 {
			ApiError(w, 404, "Project not found", "-6")
			return
		}
		link = strconv.FormatUint(gid, 10)
	case 1:
		u, err := url.Parse(link)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			ApiError(w, 400, "Bad url", "-28")
			return
		}
	}
	// теги пока не трогаем (задел на будущее), см. EditWork
	if _, err := EditWork(workId, user.UserId, title, text, linkType, link); err != nil {
		log.Println("workEdit:", err)
		w.Write([]byte("-1"))
		return
	}
	row, err := WORKfetchById(workId)
	if err != nil {
		log.Println("workEdit: fetch:", err)
		w.Write([]byte("-1"))
		return
	}
	json.NewEncoder(w).Encode(row.RenderWork())
}

func workDeleteHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	if user.Activated == 0 {
		w.Write([]byte("Access denied"))
		return
	}
	workId, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		ApiError(w, 400, "Bad id", "-7")
		return
	}
	rows, err := DeleteWork(workId, user.UserId)
	if err != nil {
		log.Println("workDelete:", err)
		w.Write([]byte("-1"))
		return
	}
	if rows == 0 {
		// не найдена или чужая, отвечаем одинаково
		w.Write([]byte("-1"))
		return
	}
	w.Write([]byte(strconv.Itoa(workId)))
}

func workVerifyHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	if user.Activated == 0 {
		w.Write([]byte("Access denied"))
		return
	}
	workId, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		ApiError(w, 400, "Bad id", "-7")
		return
	}
	rows, err := ToggleWorkVerification(workId, user.UserId)
	if err != nil {
		log.Println("workVerify:", err)
		w.Write([]byte("-1"))
		return
	}
	if rows == 0 {
		// нет работы, это внешняя ссылка или ты не автор проекта
		w.Write([]byte("-2"))
		return
	}
	work, err := WORKfetchById(workId)
	if err != nil {
		w.Write([]byte("-1"))
		return
	}
	w.Write([]byte(strconv.Itoa(work.OwnerChecked))) // новое состояние: 0 или 1
}
