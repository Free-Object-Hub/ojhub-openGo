package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	//
	_ "github.com/go-sql-driver/mysql"
)

func fullSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "form error "+err.Error(), http.StatusBadRequest)
		return
	}
	query := r.URL.Query()
	tagsStr := query["tags[]"]
	tags := make([]int, 0, len(tagsStr))
	for _, tag := range tagsStr {
		if num, err := strconv.Atoi(tag); err == nil {
			tags = append(tags, num)
		}
	}
	osStr := query["os[]"]
	os := make([]int, 0, len(osStr))
	for _, o := range osStr {
		if num, err := strconv.Atoi(o); err == nil {
			os = append(os, num)
		}
	}
	methodStr := query.Get("method")
	method := 0
	if methodStr != "" {
		if m, err := strconv.Atoi(methodStr); err == nil {
			method = m
		}
	}
	channelStr := query.Get("channel")
	channel := 0
	if channelStr != "" {
		if c, err := strconv.Atoi(channelStr); err == nil {
			channel = c
		}
	}
	token := GetUserToken(r)
	userId := 0
	if token != "" && channel == -5 {
		user, err := GetUserByToken(token)
		if err != nil {
			http.Error(w, "Error with user", http.StatusInternalServerError)
			fmt.Fprint(w, err)
			return
		}
		userId = user.UserId
	}
	pageStr := query.Get("page")
	page := 0
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil {
			page = p
		}
	}
	name := query.Get("name")
	lgbtVal := 1
	data, err := NewProjectsFinder(
		userId,
		method,
		channel,
		page,
		tags,
		os,
		name,
		lgbtVal,
	)
	if err != nil {
		http.Error(w, "find error "+err.Error(), http.StatusBadRequest)
		return
	}
	//result := make(map[string]interface{})
	var prefix string
	switch channel {
	case -5:
		prefix = "v"
	case -1:
		prefix = "w"
	case 0:
		prefix = "c"
	case 1:
		prefix = "s"
	case 2:
		prefix = "p"
	case 3:
		prefix = "t"
	default:
		prefix = "x"
	}
	var result interface{}
	switch channel {
	case -5:
		token := GetUserToken(r)
		userId := 0
		if token != "" {
			user, err := GetUserByToken(token)
			if err != nil {
				http.Error(w, "Error with user", http.StatusInternalServerError)
				fmt.Fprint(w, err)
				return
			}
			userId = user.UserId
		}
		if vacans, ok := data.([]Vacancy); ok {
			shortWikis := make([]Vacan, 0, len(vacans))
			for _, vac := range vacans {
				shortWikis = append(shortWikis, vac.ToShort(userId))
			}
			result = GenerateOrderedMap(shortWikis, func(vac Vacan) string {
				return "v" + strconv.Itoa(int(vac.ID))
			})
		}
	case -1:
		if wikis, ok := data.([]Wiki); ok {
			shortWikis := make([]WikiShort, 0, len(wikis))
			for _, wiki := range wikis {
				shortWikis = append(shortWikis, wiki.ToShort())
			}
			result = GenerateOrderedMap(shortWikis, func(wiki WikiShort) string {
				return "w" + strconv.Itoa(wiki.ID)
			})
		}
	case 0, 1, 2, 3:
		if projects, ok := data.([]GDPS); ok {
			shortProjects := make([]GDPSshort, 0, len(projects))
			for _, project := range projects {
				shortProjects = append(shortProjects, project.ToShort(false))
			}
			result = GenerateOrderedMap(shortProjects, func(project GDPSshort) string {
				return prefix + strconv.Itoa(project.ID)
			})
		}
	}
	jsonData, err := json.Marshal(result)
	if err != nil {
		http.Error(w, "User data error: "+err.Error(), http.StatusInternalServerError)
	}
	fmt.Fprint(w, string(jsonData))
}

func wikiSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	wikisPre, err := NewWikiFinder(0, -1, page, []int{}, []int{}, "", 0)
	if err != nil {
		http.Error(w, "Failed to fetch wikis: "+err.Error(), http.StatusInternalServerError)
		return
	}
	shortWikis := make([]WikiShort, 0, len(wikisPre))
	for _, wiki := range wikisPre {
		shortWikis = append(shortWikis, wiki.ToShort())
	}
	wikis := GenerateOrderedMap(shortWikis, func(wiki WikiShort) string {
		return "w" + strconv.Itoa(wiki.ID)
	})
	if err := json.NewEncoder(w).Encode(wikis); err != nil {
		log.Println("Failed to encode wikis:", err)
	}
}

func vacsSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	token := GetUserToken(r)
	userId := 0
	if token != "" {
		user, err := GetUserByToken(token)
		if err != nil {
			http.Error(w, "Error with user", http.StatusInternalServerError)
			fmt.Fprint(w, err)
			return
		}
		userId = user.UserId
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	vacsPre, err := NewVacsFinder(userId, 0, -5, page, []int{}, []int{}, "", 0)
	if err != nil {
		http.Error(w, "Failed to fetch wikis: "+err.Error(), http.StatusInternalServerError)
		return
	}
	shortVacs := make([]Vacan, 0, len(vacsPre))
	for _, vac := range vacsPre {
		shortVacs = append(shortVacs, vac.ToShort(userId))
	}
	wikis := GenerateOrderedMap(shortVacs, func(vac Vacan) string {
		return "v" + strconv.Itoa(int(vac.ID))
	})

	if err := json.NewEncoder(w).Encode(wikis); err != nil {
		log.Println("Failed to encode wikis:", err)
	}
}
