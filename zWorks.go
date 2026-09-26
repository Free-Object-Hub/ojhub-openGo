package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
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
