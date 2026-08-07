package main

/*
 * Обратите внимание! во всех рендерах вики поле Date закомментировано, хотя прописано.
 * Сделано это потому что legacy-php возвращал поле date, но клиент его не использовал
 * Всего лишь сохранение контракта с примесью микро-оптимизаций
 */

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Wiki struct {
	ID            int    `db:"ID"`
	Checked       int    `db:"checked"`
	UserID        int    `db:"userId"`
	Title         string `db:"title"`
	Text          string `db:"text"`
	Img           string `db:"img"`
	Language      string `db:"language"`
	Date          int    `db:"date"`
	Likes         int    `db:"likes"`
	Disls         int    `db:"disls"`
	HasLgbt       int    `db:"hasLgbt"`
	ConnectedGDPS int    `db:"connectedGdps"`
	ForumID       int    `db:"forumId"`
	MainWiki      int    `db:"mainWiki"`
	Files         string `db:"files"`
	FilesSize     int    `db:"filesSize"`
	Colors        string `db:"colors"`
}

type Guide struct {
	ID          int    `db:"ID"`
	UserID      int    `db:"userId"`
	WikiTag     string `db:"wikiTag"`
	Title       string `db:"title"`
	Aftertext   string `db:"aftertext"`
	Img         string `db:"img"`
	GuideText   string `db:"guidetext"`
	Language    string `db:"language"`
	Templates   string `db:"templates"` // "tempName,tempName,tempName"
	Checked     int    `db:"checked"`
	Date        int64  `db:"date"`
	Likes       int    `db:"likes"`
	Disls       int    `db:"disls"`
	CommsCount  int    `db:"commsCount"`
	WikiChannel int    `db:"wikiChannel"`
}

func WIKIfetchById(ID int) (*Wiki, error) {
	var wiki Wiki
	query := `SELECT * FROM wikis WHERE ID = ?`
	err := DB.Get(&wiki, query, ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get gdps by ID: %w", err)
	}
	return &wiki, nil
}

func WIKIfetchGuides(ID, page int) ([]Guide, error) {
	offset := page * 4
	var guides []Guide
	err := DB.Select(&guides, `SELECT * FROM guides WHERE checked = 1 AND wikiChannel = ? ORDER BY ID DESC LIMIT 5 OFFSET ?`, ID, offset)
	if err != nil {
		return nil, err
	}
	return guides, nil
}

func CheckWikiAccess(userID, wikiID int) (int, error) {
	var access int
	err := DB.Get(&access, `
		SELECT CASE
			WHEN EXISTS(
				SELECT 1
				FROM wikis
				WHERE ID = ? AND userId = ?
			) THEN 2
			WHEN EXISTS(
				SELECT 1
				FROM wikisoowners
				WHERE wikiId = ? AND userId = ?
			) THEN 1
			ELSE 0
		END AS access_level
		LIMIT 1
	`, wikiID, userID, wikiID, userID)
	if err != nil {
		return 0, err
	}
	return access, nil
}

type UserWikiContent struct {
	Owned   []Wiki
	SOOwned []Wiki
}

func GetAllUserWikiContent(userID int) (UserWikiContent, error) {
	var allWiki []struct {
		Wiki
		Role string `db:"role"`
	}
	err := DB.Select(
		&allWiki,
		`
		SELECT 
			w.*,
			CASE
				WHEN w.userId = ? THEN 'owner'
				WHEN so.userId IS NOT NULL THEN 'soowner'
			END AS role
		FROM wikis w
		LEFT JOIN wikisoowners so
			ON w.ID = so.wikiId
			AND so.userId = ?
		WHERE w.userId= ?
			OR so.userId IS NOT NULL
		`,
		userID,
		userID,
		userID,
	)
	if err != nil {
		return UserWikiContent{}, err
	}
	result := UserWikiContent{
		Owned:   make([]Wiki, 0),
		SOOwned: make([]Wiki, 0),
	}
	for _, item := range allWiki {
		switch item.Role {
		case "owner":
			result.Owned = append(result.Owned, item.Wiki)
		case "soowner":
			result.SOOwned = append(result.SOOwned, item.Wiki)
		}
	}
	return result, nil
}

func GetUserWikiContent(userID int) ([]Wiki, error) {
	var allWiki []Wiki
	err := DB.Select(
		&allWiki,
		`
		SELECT DISTINCT w.*
		FROM wikis w
		LEFT JOIN wikisoowners so
			ON w.ID = so.wikiId
			AND so.userId = ?
		WHERE w.userId = ?
			OR so.userId IS NOT NULL
		`,
		userID,
		userID,
	)
	if err != nil {
		return nil, err
	}
	return allWiki, nil
}

type WikiShort struct {
	ID       int    `json:"ID"`
	Title    string `json:"title"`
	Text     string `json:"text"`
	Img      string `json:"ban"`
	Language string `json:"language"`
	Date     int    `json:"date"`
	Likes    [2]int `json:"likes"`
	ForumID  int    `json:"forumId"`
	MainWiki int    `json:"mainWiki"`
}

func (p Wiki) ToShort() WikiShort {
	return WikiShort{
		ID:       p.ID,
		Title:    p.Title,
		Text:     p.Text,
		Img:      p.Img,
		Language: p.Language,
		// Date:     p.Date,
		Likes: [2]int{
			p.Likes,
			p.Disls,
		},
		ForumID:  p.ForumID,
		MainWiki: p.MainWiki,
	}
}

type WikiProfile struct {
	ID            int    `json:"ID"`
	Title         string `json:"title"`
	Text          string `json:"text"`
	Img           string `json:"ban"`
	Language      string `json:"language"`
	Date          int    `json:"date"`
	UserID        int    `json:"userId"`
	ConnectedGDPS int    `json:"connGdps"`
	ForumID       int    `json:"forumId"`
	MainWiki      int    `json:"mainWiki"`
	Colors        string `json:"color"`
}

func (p Wiki) ToFull() WikiProfile {
	return WikiProfile{
		ID:       p.ID,
		Title:    p.Title,
		Text:     p.Text,
		Img:      p.Img,
		Language: p.Language,
		// Date:  p.Date,
		UserID:        p.UserID,
		ConnectedGDPS: p.ConnectedGDPS,
		ForumID:       p.ForumID,
		MainWiki:      p.MainWiki,
		Colors:        p.Colors,
	}
}

func GuidesFetchByIdOrTag(idOrTag string, wikiID int) (*Guide, error) {
	if id, err := strconv.Atoi(idOrTag); err == nil && id != 0 {
		return GuidesFetchById(id)
	}
	return GuidesFetchByTag(idOrTag, wikiID)
}

func GuidesFetchById(id int) (*Guide, error) {
	var guide Guide
	err := DB.Get(&guide, `SELECT * FROM guides WHERE ID = ?`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &guide, nil
}

func GuidesFetchByTag(tag string, wikiID int) (*Guide, error) {
	var guide Guide
	err := DB.Get(&guide, `SELECT * FROM guides WHERE wikiTag = ? AND wikiChannel = ?`, tag, wikiID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &guide, nil
}

func renderGuideMini(g Guide) []interface{} {
	return []interface{}{
		g.ID,
		g.Title,
		g.Language,
		g.Date,
		[]int{
			g.Likes,
			g.Disls,
			g.CommsCount,
		},
		g.Img,
		g.WikiTag,
		0, // liked — старая система лайков больше не используется
	}
}

type GuideRender struct {
	GuideInfo []interface{}          `json:"guideinfo"`
	GuideData interface{}            `json:"guidedata"`
	Comments  []CommResp             `json:"comments"`
	Templates map[string]interface{} `json:"templates"`
}

func renderGuide(g Guide) (GuideRender, error) {
	guidetext := strings.NewReplacer(
		"\r\n", `\n`,
		"\r", `\n`,
		"\n", `\n`,
		"\t", `\t`,
	).Replace(g.GuideText)
	var guideData interface{}
	if err := json.Unmarshal([]byte(guidetext), &guideData); err != nil {
		return GuideRender{}, fmt.Errorf("failed to parse guide text: %w", err)
	}
	wiki, err := WIKIfetchById(g.WikiChannel)
	if err != nil {
		return GuideRender{}, fmt.Errorf("failed to fetch wiki: %w", err)
	}

	return GuideRender{
		GuideInfo: []interface{}{
			g.ID,
			g.Title,
			g.Aftertext,
			g.WikiTag,
			wiki.Colors,
		},
		GuideData: guideData,
		Comments:  make([]CommResp, 0),
		Templates: make(map[string]interface{}),
	}, nil
}
