package main

import (
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
)

func GDPSaddNewItem(
	channel int, title, link, img, ban, description, short, tags, os string,
	mask int, author int, username, language string,
) (int64, error) {
	result, err := DB.Exec(
		`INSERT INTO gdpses (channel, title, link, img, ban, description, short, tags, os, mask, author, username, language) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		channel, title, link, img, ban, description, short, tags, os, mask, author, username, language,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert gdps: %w", err)
	}
	return result.LastInsertId()
}

func gdpsAddHandler(w http.ResponseWriter, r *http.Request, contentType int) {
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	if user.Activated == 0 {
		w.Write([]byte("Access denied"))
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		ApiError(w, 500, "Parse error", "-1")
		return
	}

	title := r.FormValue("title")
	if title == "" {
		w.Write([]byte("-27"))
		return
	}
	title = ExploitPatch(title)
	language := ExploitPatch(r.FormValue("language"))

	// --- description: новый multi-клиент (language пустой) либо legacy single ---
	var descriptionToStore string
	rawDescs := r.Form["description"]

	if language == "" && len(rawDescs) > 0 {
		multiMap, err := ParseMultiField(rawDescs, MAX_DESC_LEN)
		if err != nil || len(multiMap) == 0 {
			log.Println(err)
			ApiError(w, 400, "Bad description", "-28")
			return
		}
		descJSON, _ := json.Marshal(multiMap)
		descriptionToStore = string(descJSON)
	} else {
		description := ExploitPatch(r.FormValue("description"))
		if description == "" {
			w.Write([]byte("-27"))
			return
		}
		descriptionToStore = description
	}

	// --- short: та же multi/single развилка, отдельный лимит длины ---
	var shortToStore string
	rawShorts := r.Form["short"]

	if language == "" && len(rawShorts) > 0 {
		multiShortMap, err := ParseMultiField(rawShorts, MAX_SHORT_LEN)
		if err != nil {
			ApiError(w, 400, "Bad short", "-29")
			return
		}
		if len(multiShortMap) == 0 {
			shortToStore = ""
		} else {
			shortJSON, _ := json.Marshal(multiShortMap)
			shortToStore = string(shortJSON)
		}
	} else {
		shortToStore = ExploitPatch(r.FormValue("short"))
	}

	var imgFile, banFile *multipart.FileHeader
	var img, ban string

	if files := r.MultipartForm.File["img"]; len(files) > 0 {
		imgFile = files[0]
		img = "FILE"
	} else {
		img = ExploitPatch(r.FormValue("img"))
	}

	if files := r.MultipartForm.File["ban"]; len(files) > 0 {
		banFile = files[0]
		ban = "FILE"
	} else {
		ban = ExploitPatch(r.FormValue("ban"))
	}

	links := r.Form["links[]"]
	var linkBuilder strings.Builder
	linkBuilder.WriteString("{")
	for i := 0; i+1 < len(links); i += 2 {
		if i != 0 {
			linkBuilder.WriteString(",")
		}
		linkBuilder.WriteString(`\"`)
		linkBuilder.WriteString(ExploitPatch(links[i]))
		linkBuilder.WriteString(`\":\"`)
		linkBuilder.WriteString(ExploitPatch(links[i+1]))
		linkBuilder.WriteString(`\"`)
	}
	linkBuilder.WriteString("}")
	link := linkBuilder.String()

	tagsList := r.Form["tags[]"]
	osList := r.Form["os[]"]
	tagsJSON, _ := json.Marshal(tagsList)
	osJSON, _ := json.Marshal(osList)

	tagsInt := stringsToInts(tagsList)
	osInt := stringsToInts(osList)
	mask := createBitmask(mergeAndSortTags(tagsInt, osInt))

	username := user.Nickname
	if username == "" {
		username = user.Username
	}

	gdpsId, err := GDPSaddNewItem(
		contentType, title, link, img, ban, descriptionToStore, shortToStore,
		string(tagsJSON), string(osJSON), mask, user.UserId, username, language,
	)
	if err != nil {
		w.Write([]byte("-1"))
		return
	}

	if imgFile != nil {
		ext := GetFileExt(imgFile)
		imgPath := fmt.Sprintf("%scustomuser/i%d.%s", getEnv("IMGS", "./imgs/"), gdpsId, ext)
		if err := SaveFile(imgFile, imgPath); err != nil {
			log.Println(err)
			ApiError(w, 500, "File error", "-3")
			return
		}
		webpPath := fmt.Sprintf("%scustomuser/i%d.webp", getEnv("IMGS", "./imgs/"), gdpsId)
		if err := ConvertToWebp(imgPath, webpPath, 256, 256); err != nil {
			fmt.Printf("webp conversion failed: %v\n", err)
			log.Println(err)
		} else {
			img = fmt.Sprintf("%simgs/customuser/i%d.webp", HELPER_URL, gdpsId)
		}
	}
	if banFile != nil {
		ext := GetFileExt(banFile)
		banPath := fmt.Sprintf("%scustomuser/b%d.%s", getEnv("IMGS", "./imgs/"), gdpsId, ext)
		if err := SaveFile(banFile, banPath); err != nil {
			log.Println(err)
			ApiError(w, 500, "File error", "-3")
			return
		}
		webpPath := fmt.Sprintf("%scustomuser/b%d.webp", getEnv("IMGS", "./imgs/"), gdpsId)
		if err := ConvertToWebp(banPath, webpPath, 720, 300); err != nil {
			fmt.Printf("webp conversion failed: %v\n", err)
			log.Println(err)
		} else {
			ban = fmt.Sprintf("%simgs/customuser/b%d.webp", HELPER_URL, gdpsId)
		}
	}

	if err := EditGdpsPictures(gdpsId, img, ban); err != nil {
		w.Write([]byte("-1"))
		return
	}

	jsonData, err := InitOjhub(ExtractIP(r), user.Token, GetDeviceToken(r), false, false, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	TGWebhookLog(fmt.Sprintf("NEW GDPS, ID: %d", gdpsId))
	json.NewEncoder(w).Encode(jsonData)
}

func stringsToInts(s []string) []int {
	result := make([]int, 0, len(s))
	for _, v := range s {
		if n, err := strconv.Atoi(v); err == nil {
			result = append(result, n)
		}
	}
	return result
}

func CampAdd(w http.ResponseWriter, r *http.Request) { gdpsAddHandler(w, r, 0) }
func ShowAdd(w http.ResponseWriter, r *http.Request) { gdpsAddHandler(w, r, 1) }
func PereAdd(w http.ResponseWriter, r *http.Request) { gdpsAddHandler(w, r, 2) }
func TeleAdd(w http.ResponseWriter, r *http.Request) { gdpsAddHandler(w, r, 3) }
