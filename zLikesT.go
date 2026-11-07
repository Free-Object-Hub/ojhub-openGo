package main

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
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

	result := map[string]interface{}{
		"p":     []int{},
		"c":     []int{},
		"n":     []int{},
		"g":     []int{},
		"w":     []int{},
		"f":     []int{},
		"v":     []int{},
		"subs":  []int{},
		"polls": map[string]int{},
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

	// Likes
	for _, l := range likes {
		ch, ok := LikeExportChannels[l.Channel]
		if !ok {
			continue
		}

		result[ch] = append(
			result[ch].([]int),
			l.WhereIz*l.Type,
		)
	}

	// Sort likes
	for _, ch := range []string{"p", "c", "n", "g", "w", "f", "v"} {
		sort.Slice(result[ch].([]int), func(i, j int) bool {
			return abs(result[ch].([]int)[i]) > abs(result[ch].([]int)[j])
		})
	}

	// Subscriptions
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

	result["subs"] = subRows

	// Polls
	polls := map[string]int{}
	var pv []struct {
		NewsId int `db:"newsId"`
		Mask   int `db:"mask"`
	}
	err = DB.Select(
		&pv,
		`SELECT newsId, mask FROM pollVotes WHERE userId = ?`,
		user.UserId,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, v := range pv {
		polls["n"+strconv.Itoa(v.NewsId)] = v.Mask
	}
	result["polls"] = polls

	json.NewEncoder(w).Encode(result)
}
