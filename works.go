package main

import (
	"database/sql"
	"fmt"
	"time"
)

type Works struct {
	ID           int    `db:"ID"`
	UserID       int    `db:"userId"`
	Title        string `db:"title"`
	Text         string `db:"text"`
	Tags         string `db:"tags"`
	Mask         int    `db:"mask"`
	LinkType     int    `db:"linkType"`
	LinkOrGdpsId string `db:"gdps"`
	OwnerChecked int    `db:"gdpsChecked"`
	Date         int    `db:"date"`
}

type WorksRow struct {
	ID           int            `db:"ID"`
	Title        string         `db:"title"`
	Text         string         `db:"text"`
	Tags         string         `db:"tags"`
	LinkType     int            `db:"linkType"`
	LinkOrGdpsId string         `db:"gdps"`
	OwnerChecked int            `db:"gdpsChecked"`
	Date         int            `db:"date"`
	GdpsTitle    sql.NullString `db:"gTitle"`
	GdpsChannel  sql.NullInt64  `db:"gChannel"`
}

func (r WorksRow) RenderWork() []interface{} {
	var gTitle interface{}
	if r.LinkType == 1 || !r.GdpsTitle.Valid {
		gTitle = 0
	} else {
		gTitle = r.GdpsTitle.String
	}

	return []interface{}{
		r.ID,
		gTitle,
		int(r.GdpsChannel.Int64),
		r.Title,
		r.Text,
		r.Tags,
		r.LinkOrGdpsId,
		r.OwnerChecked,
		r.Date,
	}
}

func WORKfetchById(ID int) (*Works, error) {
	var work Works
	query := `SELECT * FROM works WHERE ID = ?`
	err := DB.Get(&work, query, ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get work by ID: %w", err)
	}
	return &work, nil
}

func WORKfetchByUserId(userId int) ([]WorksRow, error) {
	var rows []WorksRow
	query := `SELECT w.ID, w.title, w.text, w.tags, w.linkType, w.gdps, w.gdpsChecked, w.date,
			g.title AS gTitle, g.channel AS gChannel
		FROM works w
		LEFT JOIN gdpses g ON w.linkType = 0 AND CAST(w.gdps AS UNSIGNED) = g.ID
		WHERE w.userId = ?`
	if err := DB.Select(&rows, query, userId); err != nil {
		return nil, fmt.Errorf("failed to get works by userId: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows, nil
}

func AddWork(userId int, title, text, tags string, mask, linkType int, gdps string) (int, error) {
	// пока что создам работы без тегов, я не хочу мучать клиент ещё большими костылями
	tags = "[]"
	mask = 0
	result, err := DB.Exec(
		`INSERT INTO works (userId, title, text, tags, mask, linkType, gdps, gdpsChecked, date) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		userId, title, text, tags, mask, linkType, gdps, 0, time.Now().Unix(),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert work: %w", err)
	}
	id, err := result.LastInsertId()
	return int(id), err
}

func EditWork(workId, userId int, title, text string, linkType int, gdps string) (int, error) {
	result, err := DB.Exec(
		`UPDATE works
		 SET title = ?, text = ?, linkType = ?, gdps = ?, gdpsChecked = IF(gdps <> ?, 0, gdpsChecked)
		 WHERE ID = ? AND userId = ?`,
		title, text, linkType, gdps, gdps, workId, userId,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to edit work: %w", err)
	}
	rows, err := result.RowsAffected()
	return int(rows), err
}

func ToggleWorkVerification(workId, verifierUserId int) (int, error) {
	result, err := DB.Exec(
		`UPDATE works w
		 JOIN gdpses g ON CAST(w.gdps AS UNSIGNED) = g.ID
		 SET w.gdpsChecked = 1 - w.gdpsChecked
		 WHERE w.ID = ? AND w.linkType = 0 AND g.author = ?`,
		workId, verifierUserId,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to toggle work verification: %w", err)
	}
	rows, err := result.RowsAffected()
	return int(rows), err
}
