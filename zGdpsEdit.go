package main

import (
	"encoding/json"
	"fmt"
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

	// GET-режим: title/description отсутствуют - отдаём текущие данные для формы
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
	description = ExploitPatch(description)
	short := ExploitPatch(r.FormValue("short"))
	language := ExploitPatch(r.FormValue("language"))

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
	mask := createBitmask(mergeAndSortTags(stringsToInts(tagsList), stringsToInts(osList)))

	if err := EditGdpsItem(contentType, title, link, img, ban, description, short, string(tagsJSON), string(osJSON), mask, language, gdpsId); err != nil {
		w.Write([]byte("-1"))
		return
	}

	// editCount берётся ДО инкремента (как в php - $gdps->editCount, старое значение)
	if imgFile != nil {
		ext := GetFileExt(imgFile)
		imgPath := fmt.Sprintf("%scustomuser/i%d-%d.%s", getEnv("IMGS", "./imgs/"), gdpsId, gdps.EditCount, ext)
		if err := SaveFile(imgFile, imgPath); err != nil {
			ApiError(w, 500, "File error", "-3")
			return
		}
		webpPath := fmt.Sprintf("%scustomuser/i%d-%d.webp", getEnv("IMGS", "./imgs/"), gdpsId, gdps.EditCount)
		if err := ConvertToWebp(imgPath, webpPath, 256, 256); err != nil {
			fmt.Printf("webp conversion failed: %v\n", err)
			// если файл сломался то оригинал остаётся, img путь не меняется - деградация без падения
		} else {
			img = fmt.Sprintf("%simgs/customuser/i%d-%d.webp", HELPER_URL, gdpsId, gdps.EditCount)
		}
	}
	if banFile != nil {
		ext := GetFileExt(banFile)
		banPath := fmt.Sprintf("%scustomuser/b%d-%d.%s", getEnv("IMGS", "./imgs/"), gdpsId, gdps.EditCount, ext)
		if err := SaveFile(banFile, banPath); err != nil {
			ApiError(w, 500, "File error", "-3")
			return
		}
		webpPath := fmt.Sprintf("%scustomuser/b%d-%d.webp", getEnv("IMGS", "./imgs/"), gdpsId, gdps.EditCount)
		if err := ConvertToWebp(banPath, webpPath, 720, 300); err != nil {
			fmt.Printf("webp conversion failed: %v\n", err)
			// если файл сломался то оригинал остаётся, img путь не меняется - деградация без падения
		} else {
			ban = fmt.Sprintf("%simgs/customuser/b%d-%d.webp", HELPER_URL, gdpsId, gdps.EditCount)
		}
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
