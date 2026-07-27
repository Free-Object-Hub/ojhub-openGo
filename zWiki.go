package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

type WikiMini struct {
	ID    int    `json:"ID"`
	Title string `json:"title"`
	Color string `json:"color"`
}

func GuidesHandler(w http.ResponseWriter, r *http.Request) {
	page := 0
	if pagePre := r.URL.Query().Get("page"); pagePre != "" {
		if parsed, err := strconv.Atoi(pagePre); err == nil {
			page = parsed
		}
	}
	wikiID, err := strconv.Atoi(r.URL.Query().Get("wiki"))
	if err != nil {
		http.Error(w, "Invalid wiki parameter", http.StatusBadRequest)
		return
	}
	wiki, err := WIKIfetchById(wikiID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if wiki == nil {
		http.Error(w, "Wiki not found", http.StatusNotFound)
		return
	}
	guides, err := WIKIfetchGuides(wikiID, page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonData := make([]interface{}, 0, len(guides)+1)
	jsonData = append(jsonData, WikiMini{
		ID:    wiki.ForumID,
		Title: wiki.Title,
		Color: wiki.Colors,
	})
	for _, guide := range guides {
		jsonData = append(
			jsonData,
			renderGuideMini(guide),
		)
	}
	if err := json.NewEncoder(w).Encode(jsonData); err != nil {
		http.Error(
			w,
			"Failed to encode response",
			http.StatusInternalServerError,
		)
	}
}

func GuideHandler(w http.ResponseWriter, r *http.Request) {
	wikiID, err := strconv.Atoi(r.URL.Query().Get("wiki"))
	if err != nil {
		w.Write([]byte(`["NONE"]`))
		return
	}
	guideIDOrTag := r.URL.Query().Get("id")
	guide, err := GuidesFetchByIdOrTag(guideIDOrTag, wikiID)
	if err != nil || guide == nil {
		w.Write([]byte(`["NONE"]`))
		return
	}
	jsonData, err := renderGuide(*guide)
	if err != nil {
		log.Println(err)
		w.Write([]byte(`["NONE"]`))
		return
	}
	comms, err := GetComms(2, guide.ID, 0)
	if err != nil {
		log.Println(err)
		w.Write([]byte(`["NONE"]`))
		return
	}
	for _, comm := range comms {
		jsonData.Comments = append(
			jsonData.Comments,
			comm.CommRender(),
		)
	}

	templateNames := ParseTemplateNames(guide.Templates)
	log.Println(templateNames)
	if len(templateNames) > 0 {
		templates, err := WikiTemplateGetMany(
			guide.WikiChannel,
			templateNames,
		)
		if err != nil {
			log.Println(err)
			w.Write([]byte(`["NONE"]`))
			return
		}
		for _, template := range templates {
			jsonData.Templates[template.Name] = template.RenderTemplate()
			log.Println(template.Name)
		}
	}

	if err := json.NewEncoder(w).Encode(jsonData); err != nil {
		log.Println(err)
	}
}
