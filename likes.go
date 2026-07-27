package main

import (
	"database/sql"
)

type LikeData struct {
	ID   int `db:"ID"`
	Type int `db:"type"`
}

type LikeCnt struct {
	Likes int `db:"likes"`
	Disls int `db:"disls"`
}

func CheckLike(id int, userID int, channel int) (*LikeData, error) {
	var like LikeData
	err := DB.Get(
		&like,
		"SELECT ID, type FROM likes WHERE whereIz = ? AND userId = ? AND channel = ? LIMIT 1",
		id,
		userID,
		channel,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &like, nil
}

func LikeSet(id int, lt LikeTypeData, userID int, dislike bool) ([2]int, error) {
	var result [2]int
	tx, err := DB.Beginx()
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	column := "likes"
	operator := "+"
	likeType := -1
	if dislike {
		column = "disls"
		operator = "-"
		likeType = 1
	}
	query := "UPDATE " + lt.Table + " SET " + column + " = " + column + " " + operator + " 1 WHERE ID = ?"
	args := []any{id}
	if lt.Table == "comments" {
		query += " AND channel = ?"
		args = append(args, lt.CommChannel)
	}
	_, err = tx.Exec(query, args...)
	if err != nil {
		return result, err
	}
	_, err = tx.Exec(
		"INSERT INTO likes (whereIz, userId, type, channel) VALUES (?, ?, ?, ?)",
		id,
		userID,
		likeType,
		lt.LikeChannel,
	)
	if err != nil {
		return result, err
	}
	var cnt LikeCnt
	err = tx.Get(
		&cnt,
		"SELECT likes, disls FROM "+lt.Table+" WHERE ID = ?",
		id,
	)
	if err != nil {
		return result, err
	}
	result[0] = cnt.Likes
	result[1] = cnt.Disls
	err = tx.Commit()
	if err != nil {
		return [2]int{}, err
	}
	return result, err
}

func RemoveLike(data *LikeData, id int, lt LikeTypeData) ([2]int, error) {
	var result [2]int
	tx, err := DB.Beginx()
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	column := "likes"
	operator := "-"
	if data.Type == 1 {
		column = "disls"
		operator = "+"
	}
	query := "UPDATE " + lt.Table + " SET " + column + " = " + column + " " + operator + " 1 WHERE ID = ?"
	args := []any{id}
	if lt.Table == "comments" {
		query += " AND channel = ?"
		args = append(args, lt.CommChannel)
	}
	_, err = tx.Exec(query, args...)
	if err != nil {
		return result, err
	}
	_, err = tx.Exec(
		"DELETE FROM likes WHERE ID = ?",
		data.ID,
	)
	if err != nil {
		return result, err
	}
	var cnt LikeCnt
	err = tx.Get(
		&cnt,
		"SELECT likes, disls FROM "+lt.Table+" WHERE ID = ?",
		id,
	)
	if err != nil {
		return result, err
	}
	result[0] = cnt.Likes
	result[1] = cnt.Disls
	err = tx.Commit()
	if err != nil {
		return [2]int{}, err
	}
	return result, err
}
