package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

func VacsGet(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	gdpsId, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || gdpsId == 0 {
		w.Write([]byte("-2"))
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	gdps, err := GDPSfetchById(gdpsId)
	if err != nil {
		w.Write([]byte("-2"))
		return
	}
	access, err := CheckGdpsAccess(user.UserId, gdpsId)
	if err != nil {
		access = 0
	}
	isAdmin := access != 0 || user.Priority > 0
	vacancies, err := GetVacanciesByGDPS(gdpsId, user.UserId, isAdmin, page)
	if err != nil {
		w.Write([]byte("-1"))
		return
	}
	vacsMap := make(map[string]Vacan)
	for _, v := range vacancies {
		vacsMap[fmt.Sprintf("v%d", v.ID)] = v.ToFull(user.UserId, false)
	}
	resp := map[string]any{
		"gdpsdata": []string{gdps.Title},
		"vacs":     vacsMap,
	}
	json.NewEncoder(w).Encode(resp)
}

func VacsAdd(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	gdpsId, err := strconv.Atoi(r.FormValue("id"))
	if err != nil || gdpsId == 0 {
		w.Write([]byte("-2"))
		return
	}
	gdps, err := GDPSfetchById(gdpsId)
	if err != nil {
		w.Write([]byte("-2"))
		return
	}
	access, err := CheckGdpsAccess(user.UserId, gdpsId)
	if err != nil || (access == 0 && user.Priority == 0) {
		w.Write([]byte("-2"))
		return
	}
	title := ExploitPatch(r.FormValue("title"))
	text := ExploitPatch(r.FormValue("text"))
	short := ExploitPatch(r.FormValue("short"))
	tagsList := r.Form["tags[]"]
	tagsJSON, _ := json.Marshal(tagsList)
	mask := createBitmask(mergeAndSortTags(stringsToInts(tagsList), nil))
	vacId, err := AddVacancy(title, text, short, string(tagsJSON), mask, gdpsId, gdps.Checked, gdps.HasLgbt, int(time.Now().Unix()))
	if err != nil {
		w.Write([]byte("-1"))
		return
	}
	fmt.Fprint(w, vacId)
}

func VacsEditPre(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	vacId, err := strconv.Atoi(r.FormValue("id"))
	if err != nil || vacId == 0 {
		w.Write([]byte("-2"))
		return
	}
	vacancy, err := VACSfetchById(vacId, user.UserId)
	if err != nil {
		w.Write([]byte("-2"))
		return
	}
	access, err := CheckGdpsAccess(user.UserId, vacancy.GdpsID)
	if err != nil || (access == 0 && user.Priority == 0) {
		w.Write([]byte("-2"))
		return
	}
	json.NewEncoder(w).Encode(vacancy.ToFull(user.UserId, true))
}

func VacsEdit(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		w.Write([]byte("-5"))
		return
	}
	vacId, err := strconv.Atoi(r.PostFormValue("id"))
	if err != nil || vacId == 0 {
		w.Write([]byte("-5"))
		return
	}
	vacancy, err := VACSfetchById(vacId, user.UserId)
	if err != nil {
		w.Write([]byte("-4"))
		return
	}
	gdpsId := vacancy.GdpsID
	gdps, err := GDPSfetchById(gdpsId)
	if err != nil {
		w.Write([]byte("-3"))
		return
	}
	access, err := CheckGdpsAccess(user.UserId, gdpsId)
	if err != nil || (access == 0 && user.Priority == 0) {
		w.Write([]byte("-2"))
		return
	}
	title := ExploitPatch(r.FormValue("title"))
	text := ExploitPatch(r.FormValue("text"))
	short := ExploitPatch(r.FormValue("short"))
	tagsList := r.Form["tags[]"]
	tagsJSON, _ := json.Marshal(tagsList)
	mask := createBitmask(mergeAndSortTags(stringsToInts(tagsList), nil))
	if err := EditVacancy(vacId, title, text, short, string(tagsJSON), mask, gdpsId, gdps.Checked, gdps.HasLgbt); err != nil {
		log.Println(vacId, err)
		w.Write([]byte("-1"))
		return
	}
	fmt.Fprint(w, "1")
}

func VacsRemove(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	vacId, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || vacId == 0 {
		w.Write([]byte("-2"))
		return
	}
	vacancy, err := VACSfetchById(vacId, user.UserId)
	if err != nil {
		w.Write([]byte("-2"))
		return
	}
	gdpsId := vacancy.GdpsID // <-- то же самое, gdpsId из вакансии
	access, err := CheckGdpsAccess(user.UserId, gdpsId)
	if err != nil || (access == 0 && user.Priority == 0) {
		w.Write([]byte("-2"))
		return
	}
	if err := RemoveVacancy(vacId); err != nil {
		w.Write([]byte("-1"))
		return
	}
	fmt.Fprint(w, "1")
}

func VacsApplies(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	vacId, err := strconv.Atoi(r.URL.Query().Get("vacid"))
	if err != nil || vacId == 0 {
		w.Write([]byte("-2"))
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	vacancy, err := VACSfetchById(vacId, user.UserId)
	if err != nil {
		w.Write([]byte("-2"))
		return
	}
	access, err := CheckGdpsAccess(user.UserId, vacancy.GdpsID)
	if err != nil || (access == 0 && user.Priority == 0) {
		w.Write([]byte("-2"))
		return
	}
	applies, err := GetVacancyApplies(vacId, page)
	if err != nil {
		w.Write([]byte("-1"))
		return
	}
	appliesMap := make(map[string]any)
	for _, a := range applies {
		appliesMap["a"+strconv.Itoa(a.ID)] = a.RenderApply()
	}
	resp := map[string]any{
		"gdpsdata": []string{vacancy.Title},
		"applies":  appliesMap,
	}
	json.NewEncoder(w).Encode(resp)
}
