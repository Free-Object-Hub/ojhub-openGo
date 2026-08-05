package main

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func gdpsSub(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDeviceForInactiveAccs(w, r)
	if !ok {
		return
	}
	gdpsID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || gdpsID <= 0 {
		w.Write([]byte("-7"))
		return
	}
	gdps, err := GDPSfetchById(gdpsID)
	if err != nil || gdps == nil {
		w.Write([]byte("-6"))
		return
	}
	already, err := IsSubscribed(user.UserId, gdpsID)
	if err != nil {
		w.Write([]byte("-8"))
		return
	}
	if already {
		w.Write([]byte("-1"))
		return
	}
	err = SubToGdps(user.UserId, gdpsID)
	if err != nil {
		w.Write([]byte("-8"))
		return
	}
	w.Write([]byte("1"))
}

func gdpsUnsub(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDeviceForInactiveAccs(w, r)
	if !ok {
		return
	}
	gdpsID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || gdpsID <= 0 {
		w.Write([]byte("-7"))
		return
	}
	subscribed, err := IsSubscribed(user.UserId, gdpsID)
	if err != nil {
		w.Write([]byte("-8"))
		return
	}
	if !subscribed {
		w.Write([]byte("-1"))
		return
	}
	err = UnsubFromGdps(user.UserId, gdpsID)
	if err != nil {
		w.Write([]byte("-8"))
		return
	}
	w.Write([]byte("1"))
}

func gdpsSubs(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDeviceForInactiveAccs(w, r)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	gdpslist, err := GetGdpsSubsList(user.UserId, page)
	if err != nil {
		w.Write([]byte("[]"))
		return
	}
	if len(gdpslist) == 0 {
		w.Write([]byte("[]"))
		return
	}
	result := []map[string]interface{}{}
	for _, gdps := range gdpslist {
		result = append(result, map[string]interface{}{
			"ID":       gdps.ID,
			"title":    gdps.Title,
			"text":     gdps.Description,
			"author":   gdps.Author,
			"username": gdps.Username,
			"img":      gdps.Img,
			"channel":  gdps.Channel,
		})
	}
	json.NewEncoder(w).Encode(result)
}

func gdpsSubs2(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDeviceForInactiveAccs(w, r)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	gdpslist, err := GetGdpsSubsList(user.UserId, page)
	if err != nil {
		w.Write([]byte("[]"))
		return
	}
	if len(gdpslist) == 0 {
		w.Write([]byte("[]"))
		return
	}
	json.NewEncoder(w).Encode(gdpslist)
}
