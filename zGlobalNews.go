package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"golang.org/x/sync/errgroup"
)

func GlobalNews(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		page = 0
	}
	news, err := GetGlobalNews(page)
	if err != nil {
		http.Error(w, "Error fetching news", http.StatusInternalServerError)
		log.Println(err)
		return
	}
	rendered := make([]NewsResp, len(news))
	for i, n := range news {
		rendered[i] = n.NewsRender()
	}
	if err := json.NewEncoder(w).Encode(rendered); err != nil {
		http.Error(w, "News data error", http.StatusInternalServerError)
	}
}

func LocalNews(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		page = 0
	}
	gdpsId, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, "Error converting gdpsId", http.StatusInternalServerError)
		return
	}
	news, err := GetLocalNews(gdpsId, page)
	if err != nil {
		http.Error(w, "Error fetching news", http.StatusInternalServerError)
		log.Println(err)
		return
	}
	rendered := make([]NewsResp, len(news))
	for i, n := range news {
		rendered[i] = n.NewsRender()
	}
	if err := json.NewEncoder(w).Encode(rendered); err != nil {
		http.Error(w, "News data error", http.StatusInternalServerError)
	}
}

type NewsOneResp struct {
	Gdps     map[string]NewsResp `json:"gdps"`
	Comments []CommResp          `json:"comments"`
}

func NewsGetOne(w http.ResponseWriter, r *http.Request) {
	idPre := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idPre)
	if err != nil {
		http.Error(w, "Invalid id parameter", http.StatusBadRequest)
		return
	}
	//
	var (
		news    *News
		commPre []Comm
	)
	g, _ := errgroup.WithContext(r.Context())
	g.Go(func() error {
		var err error
		news, err = NEWSfetchById(id)
		return err
	})
	g.Go(func() error {
		var err error
		commPre, err = GetComms(3, id, 0)
		return err
	})
	if err := g.Wait(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//
	comms := make([]CommResp, 0, len(commPre))
	for _, c := range commPre {
		comms = append(comms, c.CommRender())
	}
	jsonData := NewsOneResp{
		Gdps: map[string]NewsResp{
			"n" + strconv.Itoa(news.ID): news.NewsRender(),
		},
		Comments: comms,
	}
	if err := json.NewEncoder(w).Encode(jsonData); err != nil {
		http.Error(w, "News data error", http.StatusInternalServerError)
	}
}
