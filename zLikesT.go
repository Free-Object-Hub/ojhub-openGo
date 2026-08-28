package main

import (
	"encoding/json"
	"net/http"
	"sort"
)

type LikeRow struct {
	WhereIz int `db:"whereIz"`
	Type    int `db:"type"`
	Channel int `db:"channel"`
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

var LikeExportChannels = map[int]string{
	0:  "p",
	1:  "c",
	2:  "n",
	3:  "c",
	4:  "c",
	5:  "c",
	6:  "n",
	7:  "g",
	8:  "w",
	9:  "f",
	10: "c",
	11: "v",
	12: "c",
}

func LikesT(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDeviceForInactiveAccs(w, r)
	if !ok {
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")

	result := map[string][]int{
		"p":    {},
		"c":    {},
		"n":    {},
		"g":    {},
		"w":    {},
		"f":    {},
		"v":    {},
		"subs": {},
	}

	if user.Activated == 0 {
		json.NewEncoder(w).Encode(result)
		return
	}

	var likes []LikeRow

	err := DB.Select(
		&likes,
		`SELECT whereIz, type, channel
		FROM likes
		WHERE userId = ?`,
		user.UserId,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, l := range likes {
		ch, ok := LikeExportChannels[l.Channel]
		if !ok {
			continue
		}

		result[ch] = append(
			result[ch],
			l.WhereIz*l.Type,
		)
	}

	for k := range result {
		sort.Slice(result[k], func(i, j int) bool {
			return abs(result[k][i]) > abs(result[k][j])
		})
	}

	var subRows []int
	err = DB.Select(
		&subRows,
		`SELECT gdpsId FROM gdpsSubs WHERE userId = ?`,
		user.UserId,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for k := range result {
		sort.Slice(result[k], func(i, j int) bool {
			return abs(result[k][i]) > abs(result[k][j])
		})
	}

	result["subs"] = subRows

	json.NewEncoder(w).Encode(result)
}
