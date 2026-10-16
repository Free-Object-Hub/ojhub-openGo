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

func EditGdpsItem(
	channel int, title, link, img, ban, description, short, tags, os string,
	mask int, language string, gdpsId int,
) error {
	_, err := DB.Exec(
		`UPDATE gdpses SET channel=?, title=?, link=?, img=?, ban=?, description=?, short=?, tags=?, os=?, mask=?, language=?, editCount=editCount+1 WHERE ID=?`,
		channel, title, link, img, ban, description, short, tags, os, mask, language, gdpsId,
	)
	if err != nil {
		return fmt.Errorf("failed to edit gdps: %w", err)
	}
	return nil
}

func firstFile(f *multipart.Form, key string) *multipart.FileHeader {
	if f == nil {
		return nil
	}
	if fs := f.File[key]; len(fs) > 0 {
		return fs[0]
	}
	return nil
}

func gdpsEditHandler(w http.ResponseWriter, r *http.Request, contentType int) {
	gdpsIdStr := r.URL.Query().Get("id")
	gdpsId, err := strconv.Atoi(gdpsIdStr)
	if err != nil {
		ApiError(w, 400, "Bad id", "-7")
		return
	}

	gdps, err := GDPSfetchById(gdpsId)
	if err != nil {
		ApiError(w, 404, "Data not found", "-6")
		return
	}

	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}

	access, err := CheckGdpsAccess(user.UserId, gdpsId)
	if err != nil || access == 0 {
		w.Write([]byte("-2"))
		return
	}

	title := ""
	description := ""

	if err := r.ParseMultipartForm(32 << 20); err == nil {
		title = r.FormValue("title")
		description = r.FormValue("description")
	}

	// GET mode: title/description are missing - return current data for the form
	// description/short are returned as is (raw string) - the frontend decides itself whether it's JSON or legacy plain text
	if title == "" || description == "" {
		links := strings.ReplaceAll(gdps.Link, `\"`, `"`)
		var linksOut any = links
		if strings.HasPrefix(links, "{") {
			var parsed map[string]string
			if err := json.Unmarshal([]byte(links), &parsed); err == nil {
				linksOut = parsed
			}
		}
		resp := []any{
			gdps.Title, gdps.Description, gdps.Short, linksOut,
			gdps.Img, gdps.Ban, gdps.Tags, gdps.Os, gdps.Language,
		}
		json.NewEncoder(w).Encode(resp)
		return
	}

	if user.Activated == 0 {
		w.Write([]byte("Access denied"))
		return
	}

	title = ExploitPatch(title)
	language := ExploitPatch(r.FormValue("language"))

	// --- description: new multi-client (empty language) or legacy single ---
	var descriptionToStore string
	rawDescs := r.Form["description"]

	if language == "" && len(rawDescs) > 0 {
		multiMap, err := ParseMultiField(rawDescs, MAX_DESC_LEN)
		if err != nil || len(multiMap) == 0 {
			ApiError(w, 400, "Bad description", "-28")
			return
		}
		descJSON, _ := json.Marshal(multiMap)
		descriptionToStore = string(descJSON)
	} else {
		descriptionSingle := ExploitPatch(r.FormValue("description"))
		if descriptionSingle == "" {
			w.Write([]byte("-27"))
			return
		}
		descriptionToStore = descriptionSingle
	}

	// --- short: same multi/single fork, empty string if the map is empty ---
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

	imgFile := firstFile(r.MultipartForm, "img")
	banFile := firstFile(r.MultipartForm, "ban")

	// by default what's already in the DB stays
	img, ban := gdps.Img, gdps.Ban
	if imgFile == nil {
		if v := ExploitPatch(r.FormValue("img")); v != "" {
			img = v
		}
	}
	if banFile == nil {
		if v := ExploitPatch(r.FormValue("ban")); v != "" {
			ban = v
		}
	}

	if imgFile != nil {
		if err := CheckImage(imgFile); err != nil {
			ApiError(w, 400, "Bad image", "-3")
			return
		}
	}
	if banFile != nil {
		if err := CheckImage(banFile); err != nil {
			ApiError(w, 400, "Bad image", "-3")
			return
		}
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
	mask := createBitmask(mergeAndSortTags(stringsToInts(tagsList), stringsToInts(osList)))

	if err := EditGdpsItem(contentType, title, link, img, ban, descriptionToStore, shortToStore, string(tagsJSON), string(osJSON), mask, language, gdpsId); err != nil {
		w.Write([]byte("-1"))
		return
	}

	imgsDir := getEnv("IMGS", "./imgs/")
	// editCount is taken BEFORE the increment (as in php)
	if imgFile != nil {
		p := fmt.Sprintf("%scustomuser/i%d-%d.webp", imgsDir, gdpsId, gdps.EditCount)
		if err := SaveAsWebp(imgFile, p, 256, 256); err != nil {
			log.Println("img:", err)
			ApiError(w, 500, "File error", "-3")
			return
		}
		img = fmt.Sprintf("%simgs/customuser/i%d-%d.webp", HELPER_URL, gdpsId, gdps.EditCount)
	}
	if banFile != nil {
		p := fmt.Sprintf("%scustomuser/b%d-%d.webp", imgsDir, gdpsId, gdps.EditCount)
		if err := SaveAsWebp(banFile, p, 720, 300); err != nil {
			log.Println("ban:", err)
			ApiError(w, 500, "File error", "-3")
			return
		}
		ban = fmt.Sprintf("%simgs/customuser/b%d-%d.webp", HELPER_URL, gdpsId, gdps.EditCount)
	}

	if err := EditGdpsPictures(int64(gdpsId), img, ban); err != nil {
		w.Write([]byte("-1"))
		return
	}

	TGWebhookLog(fmt.Sprintf("EDIT GDPS, ID: %d", gdpsId))

	jsonData, err := InitOjhub(ExtractIP(r), user.Token, GetDeviceToken(r), false, false, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(jsonData)
}

func CampEdit(w http.ResponseWriter, r *http.Request) { gdpsEditHandler(w, r, 0) }
func ShowEdit(w http.ResponseWriter, r *http.Request) { gdpsEditHandler(w, r, 1) }
func PereEdit(w http.ResponseWriter, r *http.Request) { gdpsEditHandler(w, r, 2) }
func TeleEdit(w http.ResponseWriter, r *http.Request) { gdpsEditHandler(w, r, 3) }
