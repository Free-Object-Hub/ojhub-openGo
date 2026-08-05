package main

import (
	"encoding/base64"
	"fmt"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
)

type News struct {
	ID          int    `db:"ID"`
	GdpsId      int    `db:"gdpsId"`
	Title       string `db:"title"`
	Text        string `db:"text"`
	GdpsTitle   string `db:"gTitle"`
	GdpsImg     string `db:"gImg"`
	GdpsChannel int    `db:"gChannel"`
	UserId      int    `db:"userId"`
	UserName    string `db:"uUsername"`
	NickName    string `db:"uNickname"`
	Date        int    `db:"date"`
	Likes       int    `db:"likes"`
	Disls       int    `db:"disls"`
	Checked     int    `db:"checked"`
	CommsCount  int    `db:"commsCount"`
	HasFile     string `db:"hasFile"`
	// legacy поля которые мне лень удалять
	Lusername string `db:"username,omitempty"`
}

func NEWSfetchById(ID int) (*News, error) {
	var news News
	query := `SELECT n.*, u.username as uUsername, u.nickname as uNickname, g.title as gTitle, g.img as gImg, g.channel as gChannel 
		FROM news n
		LEFT JOIN users u ON n.userId = u.userId 
		LEFT JOIN gdpses g ON n.gdpsId = g.ID
		WHERE n.ID = ?`
	err := DB.Get(&news, query, ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get news by ID: %w", err)
	}
	return &news, nil
}

func GetLocalNews(gdpsId int, page int) ([]News, error) {
	var news []News
	query := `SELECT n.*, u.username as uUsername, u.nickname as uNickname, g.title as gTitle, g.img as gImg, g.channel as gChannel 
		FROM news n
		LEFT JOIN users u ON n.userId = u.userId 
		LEFT JOIN gdpses g ON n.gdpsId = g.ID 
		WHERE n.gdpsId = ?
		ORDER BY n.ID DESC LIMIT 11 OFFSET ?`
	var page2 int
	if page <= 0 {
		page = 0
		page2 = 0
	} else {
		page2 = page * 10
	}
	err := DB.Select(&news, query, gdpsId, page2)
	if err != nil {
		return nil, fmt.Errorf("failed to get news by ID: %w", err)
	}
	return news, nil
}

func GetGlobalNews(page int) ([]News, error) {
	var news []News
	query := `SELECT n.*, u.username as uUsername, u.nickname as uNickname, g.title as gTitle, g.img as gImg, g.channel as gChannel 
		FROM news n
		LEFT JOIN users u ON n.userId = u.userId 
		LEFT JOIN gdpses g ON n.gdpsId = g.ID 
		WHERE n.checked = 1
		ORDER BY n.ID DESC LIMIT 11 OFFSET ?`
	var page2 int
	if page <= 0 {
		page = 0
		page2 = 0
	} else {
		page2 = page * 10
	}
	err := DB.Select(&news, query, page2)
	if err != nil {
		return nil, fmt.Errorf("failed to get news by ID: %w", err)
	}
	return news, nil
}

type NewsResp struct {
	ID       int    `json:"ID"`
	Title    string `json:"title"`
	Text     string `json:"text"`
	UserId   int    `json:"author"`
	UserName string `json:"username"`
	// гдпсайди строка потому что там буква канала перед ней идёт (c123, s234, p345, t456...)
	GdpsId    string `json:"gdpsId"`
	GdpsTitle string `json:"gdpsTitle"`
	GdpsImg   string `json:"gdpsImg"`
	Date      int    `json:"date"`
	Likes     [3]int `json:"likes"`
	// там не 0/1 а формат файла (png,jpg,webp...)
	HasFile string `json:"hasFile"`
}

func (p News) NewsRender() NewsResp {
	gdpsId := ChannelIdsToString(p.GdpsChannel) + strconv.Itoa(p.GdpsId)
	userName := p.UserName
	if userName == "" {
		userName = p.NickName
	}
	decodedText, err := base64.StdEncoding.DecodeString(p.Text)
	text := p.Text
	if err == nil {
		text = string(decodedText)
	}
	return NewsResp{
		ID:        p.ID,
		Title:     p.Title,
		Text:      text,
		UserId:    p.UserId,
		UserName:  userName,
		Date:      p.Date,
		GdpsId:    gdpsId,
		GdpsTitle: p.GdpsTitle,
		GdpsImg:   p.GdpsImg,
		Likes: [3]int{
			p.Likes,
			p.Disls,
			p.CommsCount,
		},
		HasFile: p.HasFile,
	}
}

func RenderNewsMap(newsPre []News) map[string]NewsResp {
	news := make(map[string]NewsResp, len(newsPre))
	for _, n := range newsPre {
		news["n"+strconv.Itoa(n.ID)] = n.NewsRender()
	}
	return news
}

func NEWSpost(userId, ID int, text string, date int64, title string, checked int, hasFile string) (int64, error) {
	res, err := DB.Exec(
		"INSERT INTO news (userId, gdpsId, date, title, text, checked, hasFile) VALUES (?,?,?,?,?,?,?)",
		userId, ID, date, title, text, checked, hasFile,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func NEWSedit(ID int, text, title string, gdpsId int) (int64, error) {
	res, err := DB.Exec(
		"UPDATE `news` SET `text` = ?, `title` = ? WHERE `ID` = ? AND gdpsId = ?",
		text, title, ID, gdpsId,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func NEWSdelete(id int) error {
	tx, err := DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(
		`DELETE FROM likes
		 WHERE whereIz = ? AND channel = 6`,
		id,
	); err != nil {
		return err
	}
	if _, err = tx.Exec(
		`DELETE FROM news
		 WHERE ID = ?`,
		id,
	); err != nil {
		return err
	}
	return tx.Commit()
}
