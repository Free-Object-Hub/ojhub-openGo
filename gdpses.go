package main

import (
	"encoding/json"
	"fmt"
	"strings"
	//
	_ "github.com/go-sql-driver/mysql"
)

type GDPS struct {
	ID            int    `db:"ID"`
	Title         string `db:"title"`
	Description   string `db:"description"`
	Tags          string `db:"tags"`
	Os            string `db:"os"`
	Mask          int    `db:"mask"`
	Likes         int    `db:"likes"`
	Disls         int    `db:"disls"`
	CommsCount    int    `db:"commsCount"`
	LikeRender    [3]int `db:"-"`
	Author        int    `db:"author"`
	Username      string `db:"username"`
	Img           string `db:"img"`
	Ban           string `db:"ban"`
	Channel       int    `db:"channel"`
	ConnectedWiki int    `db:"connectedWiki"`
	//
	Link      string `db:"link"`
	Short     string `db:"short"`
	Database  string `db:"database"`
	Checked   int    `db:"checked"`
	Status    int    `db:"status"`
	HasLgbt   int    `db:"hasLgbt"`
	Points    int    `db:"points"`
	Freejoin  int    `db:"freejoin"`
	Language  string `db:"language"`
	EditCount int    `db:"editCount"`
}

type UserGDPSContent struct {
	Owned   []GDPS
	SOOwned []GDPS
}

func GetAllUserGdpsContent(userID int) (UserGDPSContent, error) {
	var allGDPS []struct {
		GDPS
		Role string `db:"role"`
	}
	err := DB.Select(
		&allGDPS,
		`
		SELECT 
			g.*,
			CASE
				WHEN g.author = ? THEN 'owner'
				WHEN so.userId IS NOT NULL THEN 'soowner'
			END AS role
		FROM gdpses g
		LEFT JOIN soowners so
			ON g.ID = so.gdpsId
			AND so.userId = ?
		WHERE g.author = ?
			OR so.userId IS NOT NULL
		`,
		userID,
		userID,
		userID,
	)
	if err != nil {
		return UserGDPSContent{}, err
	}
	result := UserGDPSContent{
		Owned:   make([]GDPS, 0),
		SOOwned: make([]GDPS, 0),
	}
	for _, item := range allGDPS {
		switch item.Role {
		case "owner":
			result.Owned = append(result.Owned, item.GDPS)
		case "soowner":
			result.SOOwned = append(result.SOOwned, item.GDPS)
		}
	}
	return result, nil
}

func GetUserGdpsContent(userID int) ([]GDPS, error) {
	var allGDPS []GDPS
	err := DB.Select(
		&allGDPS,
		`
		SELECT DISTINCT g.*
		FROM gdpses g
		LEFT JOIN soowners so
			ON g.ID = so.gdpsId
			AND so.userId = ?
		WHERE g.author = ?
			OR so.userId IS NOT NULL
		`,
		userID,
		userID,
	)
	if err != nil {
		return nil, err
	}
	return allGDPS, nil
}

func GDPSfetchById(ID int) (*GDPS, error) {
	var gdps GDPS
	query := `SELECT * FROM gdpses WHERE ID = ?`
	err := DB.Get(&gdps, query, ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get gdps by ID: %w", err)
	}
	return &gdps, nil
}

func CheckGdpsAccess(userID, gdpsID int) (int, error) {
	var access int

	err := DB.Get(&access, `
		SELECT CASE
			WHEN EXISTS(
				SELECT 1 
				FROM gdpses 
				WHERE ID = ? AND author = ?
			) THEN 2

			WHEN EXISTS(
				SELECT 1 
				FROM soowners 
				WHERE gdpsId = ? AND userId = ?
			) THEN 1

			ELSE 0
		END AS access_level
		LIMIT 1
	`, gdpsID, userID, gdpsID, userID)
	if err != nil {
		return 0, err
	}

	return access, nil
}

func EditGdpsPictures(gdpsId int64, img, ban string) error {
	_, err := DB.Exec(`UPDATE gdpses SET img = ?, ban = ? WHERE ID = ?`, img, ban, gdpsId)
	if err != nil {
		return fmt.Errorf("failed to edit gdps pictures: %w", err)
	}
	return nil
}

// ParseMultiField парсит r.Form[fieldName] (записи вида "lang\ttext") в JSON-словарь.
// maxLen — лимит длины текста для конкретного поля (разный для description/short).
func ParseMultiField(raw []string, maxLen int) (map[string]string, error) {
	result := make(map[string]string)
	for _, entry := range raw {
		idx := strings.IndexByte(entry, '\t')
		if idx == -1 {
			return nil, fmt.Errorf("malformed field entry: no tab separator")
		}
		lang := entry[:idx]
		text := entry[idx+1:]
		lang = ExploitPatch(lang)
		text = ExploitPatch(text)
		if !IsValidLang(lang) {
			return nil, fmt.Errorf("unknown language: %s", lang)
		}
		if _, dup := result[lang]; dup {
			return nil, fmt.Errorf("duplicate language: %s", lang)
		}
		if len(text) > maxLen {
			return nil, fmt.Errorf("field too long for lang: %s", lang)
		}
		if text == "" {
			continue
		}
		result[lang] = text
	}
	return result, nil
}

type GDPSshort struct {
	ID       int    `json:"ID"`
	Title    string `json:"title"`
	Text     string `json:"text"`
	Tags     string `json:"tags"`
	Os       string `json:"os"`
	Likes    [3]int `json:"likes"`
	Author   int    `json:"author"`
	Username string `json:"username"`
	Img      string `json:"img"`
	Ban      string `json:"ban"`
	Channel  int    `json:"channel"`
	Wiki     int    `json:"wiki"`
	Checked  *int   `json:"checked,omitempty"`
	Points   *int   `json:"points,omitempty"`
}

func truncateDescriptionForPreview(description string, maxLen int) string {
	var multiMap map[string]string
	if err := json.Unmarshal([]byte(description), &multiMap); err == nil && len(multiMap) > 0 {
		truncated := make(map[string]string, len(multiMap))
		for lang, text := range multiMap {
			truncated[lang] = truncateRunes(text, maxLen)
		}
		result, _ := json.Marshal(truncated)
		return string(result)
	}
	// legacy plain text
	return truncateRunes(description, maxLen)
}

func (p GDPS) ToShort(fullText, renderChecked bool) GDPSshort {
	text := p.Description
	if !fullText {
		if p.Short != "" {
			text = p.Short
		} else if len(text) > 121 {
			text = truncateDescriptionForPreview(p.Description, 121)
		}
	}

	var checked *int
	var points *int
	if renderChecked {
		c := p.Checked
		checked = &c
		po := p.Points
		points = &po
	}

	return GDPSshort{
		ID:       p.ID,
		Title:    p.Title,
		Text:     text,
		Tags:     p.Tags,
		Os:       p.Os,
		Likes:    [3]int{p.Likes, p.Disls, p.CommsCount},
		Author:   p.Author,
		Username: p.Username,
		Img:      p.Img,
		Ban:      p.Ban,
		Channel:  p.Channel,
		Checked:  checked,
		Points:   points,
		Wiki:     p.ConnectedWiki,
	}
}

type GDPSfull struct {
	ID       int         `json:"ID"`
	Title    string      `json:"title"`
	Text     string      `json:"text"`
	Tags     string      `json:"tags"`
	Os       string      `json:"os"`
	Links    interface{} `json:"links"`
	Likes    [3]int      `json:"likes"`
	Author   int         `json:"author"`
	Username string      `json:"username"`
	Img      string      `json:"img"`
	Ban      string      `json:"ban"`
	Freejoin int         `json:"freejoin"`
	Language string      `json:"language"`
	Channel  int         `json:"channel"`
	Wiki     int         `json:"wiki"`
}

// пишем такие костыли ради совместимости с 50+ проектами у которых ссылка ещё осталась в формате простой строки
func ParseGdpsLinks(raw string) interface{} {
	raw = strings.ReplaceAll(raw, `\"`, `"`)
	if strings.HasPrefix(raw, "{") {
		var parsed interface{}
		if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
			return parsed
		}
	}
	return raw
}

func (p GDPS) ToFull() GDPSfull {
	return GDPSfull{
		ID:    p.ID,
		Title: p.Title,
		Text:  p.Description,
		Tags:  p.Tags,
		Os:    p.Os,
		Links: ParseGdpsLinks(p.Link),
		Likes: [3]int{
			p.Likes,
			p.Disls,
			p.CommsCount,
		},
		Author:   p.Author,
		Username: p.Username,
		Img:      p.Img,
		Ban:      p.Ban,
		Freejoin: p.Freejoin,
		Language: p.Language,
		Channel:  p.Channel,
		Wiki:     p.ConnectedWiki,
	}
}
