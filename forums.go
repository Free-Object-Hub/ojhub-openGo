package main

/*
 * Форум-модуль
 * Порт с PHP forum/create, forum/createPost, forum/getPost, forum/getPosts, send/forumPost
 */

import (
	"database/sql"
	"encoding/base64"
)

type Forum struct {
	ID      int64
	WikiId  int
	Date    int64
	Checked int
}

type ForumPost struct {
	ID         int64
	ForumId    int64
	Username   string
	UserId     int64
	Title      string
	Text       string // base64, как и в PHP-оригинале
	Date       int64
	Likes      int
	Disls      int
	CommsCount int
}

func FetchForumById(id int64) (*Forum, error) {
	row := DB.QueryRow("SELECT `ID`, `wikiId`, `date`, `checked` FROM `forums` WHERE `ID` = ?", id)
	var f Forum
	err := row.Scan(&f.ID, &f.WikiId, &f.Date, &f.Checked)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func FetchForumPostById(id int64) (*ForumPost, error) {
	row := DB.QueryRow("SELECT `ID`,`forumId`,`username`,`userId`,`title`,`text`,`date`,`likes`,`disls`,`commsCount` FROM `forumPosts` WHERE `ID` = ?", id)
	var p ForumPost
	err := row.Scan(&p.ID, &p.ForumId, &p.Username, &p.UserId, &p.Title, &p.Text, &p.Date, &p.Likes, &p.Disls, &p.CommsCount)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func GetForumPosts(forumId int64) ([]ForumPost, error) {
	rows, err := DB.Query("SELECT `ID`,`forumId`,`username`,`userId`,`title`,`text`,`date`,`likes`,`disls`,`commsCount` FROM `forumPosts` WHERE `forumId` = ? ORDER BY `ID` DESC", forumId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var posts []ForumPost
	for rows.Next() {
		var p ForumPost
		if err := rows.Scan(&p.ID, &p.ForumId, &p.Username, &p.UserId, &p.Title, &p.Text, &p.Date, &p.Likes, &p.Disls, &p.CommsCount); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, nil
}

func CreateForum(wikiId int64, date int64) (int64, error) {
	tx, err := DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	res, err := tx.Exec("INSERT INTO `forums` (`wikiId`, `date`, `checked`) VALUES (?, ?, 0)", wikiId, date)
	if err != nil {
		return 0, err
	}
	forumId, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if _, err := tx.Exec("UPDATE `wikis` SET `forumId` = ? WHERE `ID` = ?", forumId, wikiId); err != nil {
		return 0, err
	}
	return forumId, tx.Commit()
}

func UploadForumPost(forumId int64, username string, userId int, title, textBase64 string, date int64) (int64, error) {
	res, err := DB.Exec("INSERT INTO `forumPosts` (`forumId`,`username`,`userId`,`title`,`text`,`date`) VALUES (?,?,?,?,?,?)",
		forumId, username, userId, title, textBase64, date)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (p ForumPost) RenderPost() []interface{} {
	decoded, _ := base64.StdEncoding.DecodeString(p.Text)
	text := truncateRunes(string(decoded), 121)
	return []interface{}{
		p.ID,
		p.ForumId,
		p.Username,
		p.UserId,
		p.Title,
		text,
		p.Date,
		[]int{p.Likes, p.Disls, p.CommsCount},
	}
}
