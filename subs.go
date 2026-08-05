package main

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

func SubToGdps(userId, gdpsId int) error {
	_, err := DB.Exec("INSERT INTO gdpsSubs (userId, gdpsId, date) VALUES (?, ?, ?)", userId, gdpsId, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}
	return nil
}

func UnsubFromGdps(userId, gdpsId int) error {
	_, err := DB.Exec("DELETE FROM gdpsSubs WHERE userId = ? AND gdpsId = ?", userId, gdpsId)
	if err != nil {
		return fmt.Errorf("failed to unsubscribe: %w", err)
	}
	return nil
}

func IsSubscribed(userId, gdpsId int) (bool, error) {
	var id int
	err := DB.Get(&id, "SELECT ID FROM gdpsSubs WHERE userId = ? AND gdpsId = ? LIMIT 1", userId, gdpsId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check subscription: %w", err)
	}
	return true, nil
}

func GetGdpsSubsList(userId int, page int) ([]GDPS, error) {
	offset := page * 20
	var gdps []GDPS
	err := DB.Select(
		&gdps,
		`SELECT g.* FROM gdpses g
		 INNER JOIN gdpsSubs s ON g.ID = s.gdpsId
		 WHERE s.userId = ?
		 ORDER BY s.date DESC
		 LIMIT 21 OFFSET ?`,
		userId, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get subs list: %w", err)
	}
	return gdps, nil
}

func GetGdpsSubscribers(gdpsID int) ([]int, error) {
	var subscribers []int
	err := DB.Select(&subscribers, "SELECT userId FROM gdpsSubs WHERE gdpsId = ?", gdpsID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscribers: %w", err)
	}
	return subscribers, nil
}
