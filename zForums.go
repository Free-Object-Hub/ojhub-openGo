package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"time"
)

func ForumCreate(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	if user.Activated == 0 {
		return
	}
	wikiId, err := strconv.ParseInt(r.PostForm.Get("id"), 10, 64)
	if err != nil {
		return
	}
	access, err := CheckWikiAccess(int(user.UserId), int(wikiId))
	if err != nil || access <= 0 {
		return
	}
	forumId, err := CreateForum(wikiId, time.Now().Unix())
	if err != nil {
		http.Error(w, "internal error", 500)
		return
	}
	TGWebhookLog(fmt.Sprintf("Новый форум создан: %d", forumId))
	fmt.Fprint(w, forumId)
}

func ForumCreatePost(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		w.Write([]byte("Access denied"))
		return
	}
	if user.Activated == 0 {
		w.Write([]byte("Access denied"))
		return
	}
	text := r.FormValue("text")
	title := r.FormValue("title")
	if !r.PostForm.Has("text") || !r.PostForm.Has("title") {
		w.Write([]byte("-1"))
		return
	}
	forumId, err := strconv.ParseInt(r.PostForm.Get("forumId"), 10, 64)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	escapedText := html.EscapeString(text)
	encodedText := base64.StdEncoding.EncodeToString([]byte(escapedText))
	escapedTitle := html.EscapeString(title)
	date := time.Now().Unix()
	postId, err := UploadForumPost(forumId, user.PublicProfile().Username, user.UserId, escapedTitle, encodedText, date)
	if err != nil {
		http.Error(w, "internal error", 500)
		return
	}
	TGWebhookLog(fmt.Sprintf("Пост в форум %d: %s", forumId, text))
	fmt.Fprintf(w, "[%d,%d]", forumId, postId)
}

func ForumGetPost(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	postId, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		fmt.Fprint(w, `["NONE"]`)
		return
	}
	post, err := FetchForumPostById(postId)
	if err != nil || post == nil {
		fmt.Fprint(w, `["NONE"]`)
		return
	}
	comms, err := GetComms(4, int(postId), 0)
	if err != nil {
		http.Error(w, "internal error", 500)
		return
	}
	resp := map[string]interface{}{
		"post":     post.RenderPost(),
		"comments": comms,
	}
	json.NewEncoder(w).Encode(resp)
}

func ForumGetPosts(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	forumId, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		http.Error(w, "bad request", 400)
		return
	}
	posts, err := GetForumPosts(forumId)
	if err != nil {
		http.Error(w, "internal error", 500)
		return
	}
	forum, err := FetchForumById(forumId)
	if err != nil || forum == nil {
		http.Error(w, "not found", 404)
		return
	}
	wiki, err := WIKIfetchById(forum.WikiId)
	if err != nil || wiki == nil {
		http.Error(w, "not found", 404)
		return
	}
	result := []interface{}{wiki.Title}
	for _, p := range posts {
		result = append(result, p.RenderPost())
	}
	json.NewEncoder(w).Encode(result)
}
