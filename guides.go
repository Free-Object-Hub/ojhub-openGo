package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

func WIKIfetchGuidesAdmin(wikiID, page int) ([]Guide, error) {
	offset := page * 20
	var guides []Guide
	err := DB.Select(&guides, `
		SELECT * FROM guides WHERE wikiChannel = ?
		ORDER BY ID DESC LIMIT 20 OFFSET ?
	`, wikiID, offset)
	if err != nil {
		return nil, err
	}
	return guides, nil
}

func UploadGuide(userId int, title, aftertext, guideinfo, language, img, templates string, date int64, wikiId int) (int64, error) {
	res, err := DB.Exec(`
		INSERT INTO guides (userId, title, aftertext, guidetext, language, img, templates, date, wikiChannel, checked)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0)
	`, userId, title, aftertext, guideinfo, language, img, templates, date, wikiId)
	if err != nil {
		return 0, fmt.Errorf("failed to upload guide: %w", err)
	}
	return res.LastInsertId()
}

func EditGuide(title, aftertext, guideinfo, language, img, templates string, guidId int) (int, error) {
	_, err := DB.Exec(`
		UPDATE guides SET title = ?, aftertext = ?, guidetext = ?, language = ?, img = ?, templates = ?
		WHERE ID = ?
	`, title, aftertext, guideinfo, language, img, templates, guidId)
	if err != nil {
		return 0, fmt.Errorf("failed to edit guide: %w", err)
	}
	return guidId, nil
}

func SetWikiTag(id, wiki int, tag string) (string, error) {
	existing, err := GuidesFetchByTag(tag, wiki)
	if err != nil {
		return "", err
	}
	if existing != nil {
		return "-3", nil
	}
	_, err = DB.Exec(`UPDATE guides SET wikiTag = ? WHERE ID = ?`, tag, id)
	if err != nil {
		return "", err
	}
	return "", nil
}

var (
	tagPattern = regexp.MustCompile(`^[a-zA-Z0-9-]+$`)
	templateRe = regexp.MustCompile(`{{([^}|]+)`)
)

func buildGuideInfo(subtitles, subtexts []string) (string, string) {
	n := len(subtitles)
	if len(subtexts) < n {
		n = len(subtexts) // защита от рассинхрона длин — иначе паника на subtexts[i]
	}

	var sb strings.Builder
	sb.WriteByte('[')
	templates := []string{}
	seen := map[string]bool{}

	for i := 0; i < n; i++ {
		if i > 0 {
			sb.WriteByte(',')
		}
		subtitle := ExploitPatch(subtitles[i])
		subtext := ExploitPatch(subtexts[i])
		b, _ := json.Marshal([]string{subtitle, subtext})
		sb.Write(b)

		for _, m := range templateRe.FindAllStringSubmatch(subtexts[i], -1) {
			t := strings.TrimSpace(m[1])
			if !seen[t] {
				seen[t] = true
				templates = append(templates, t)
			}
		}
	}
	sb.WriteByte(']')
	return sb.String(), strings.Join(templates, ",")
}
