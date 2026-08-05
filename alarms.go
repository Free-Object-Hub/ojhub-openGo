package main

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type Alarm struct {
	ID        int    `db:"ID"`
	Title     string `db:"title"`
	Text      string `db:"text"`
	UserId    int    `db:"userId"`
	Date      int64  `db:"date"`
	AdminName string `db:"adminName"`
	AdminId   int    `db:"adminId"`
	Public    int    `db:"public"`
}

// renderMini аналог - минимальный набор полей
type AlarmMini struct {
	ID     int    `json:"ID"`
	Title  string `json:"title"`
	Public int    `json:"public"`
}

func (a *Alarm) RenderMini() AlarmMini {
	return AlarmMini{
		ID:     a.ID,
		Title:  a.Title,
		Public: a.Public,
	}
}

// render аналог - полный набор полей, с эскейпом переносов строк как в node
type AlarmFull struct {
	ID        int    `json:"ID"`
	Title     string `json:"title"`
	Text      string `json:"text"`
	Date      int64  `json:"date"`
	AdminName string `json:"adminName"`
	AdminId   int    `json:"adminId"`
}

func (a *Alarm) Render() AlarmFull {
	return AlarmFull{
		ID:        a.ID,
		Title:     a.Title,
		Text:      strings.ReplaceAll(a.Text, "\n", "\\n"),
		Date:      a.Date,
		AdminName: a.AdminName,
		AdminId:   a.AdminId,
	}
}

func CheckAlarms(userId int) (bool, error) {
	var id int
	err := DB.Get(&id, "SELECT ID FROM alarms WHERE userId = ? AND public = 1 LIMIT 1", userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check alarms: %w", err)
	}
	return true, nil
}

func UpdateAlarm(id int) (int64, error) {
	result, err := DB.Exec("UPDATE alarms SET public = 2 WHERE ID = ?", id)
	if err != nil {
		return 0, fmt.Errorf("failed to update alarm: %w", err)
	}
	return result.RowsAffected()
}

func GetFullAlarm(id int) (*Alarm, error) {
	var alarm Alarm
	err := DB.Get(&alarm, "SELECT * FROM alarms WHERE ID = ?", id)
	if err != nil {
		return nil, fmt.Errorf("failed to get alarm: %w", err)
	}
	return &alarm, nil
}

func GetAlarmsList(userId int, isAdmin bool, page int) ([]Alarm, error) {
	offset := page * 10
	admText := ""
	if isAdmin {
		admText = " OR userId = 0"
	}
	query := fmt.Sprintf(
		"SELECT * FROM alarms WHERE public != 0 AND (userId = ?%s) ORDER BY date DESC LIMIT 11 OFFSET ?",
		admText,
	)
	var alarms []Alarm
	err := DB.Select(&alarms, query, userId, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get alarms list: %w", err)
	}
	return alarms, nil
}

func RemoveAlarm(id int) (int64, error) {
	result, err := DB.Exec("UPDATE alarms SET public = 0 WHERE ID = ?", id)
	if err != nil {
		return 0, fmt.Errorf("failed to remove alarm: %w", err)
	}
	return result.RowsAffected()
}

func WriteAlarm(title, text string, userId int, date int64, adminName string, adminId int) (int64, error) {
	result, err := DB.Exec(
		"INSERT INTO alarms (title, text, userId, date, adminName, adminId) VALUES (?, ?, ?, ?, ?, ?)",
		title, text, userId, date, adminName, adminId,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to write alarm: %w", err)
	}
	return result.LastInsertId()
}

func FullWrite(title, text string, userId int, date int64, adminId int) (int, error) {
	uName := "Object Hub"
	if adminId != 0 {
		admin, err := GetUserById(adminId)
		if err == nil && admin != nil {
			if admin.Nickname != "" {
				uName = admin.Nickname
			} else {
				uName = admin.Username
			}
		}
	}

	alarmId, err := WriteAlarm(title, text, userId, date, uName, adminId)
	if err != nil {
		return 0, err
	}

	// пуш - fire and forget, как в node (.catch без await)
	go func() {
		body := text
		if len(body) > 100 {
			body = body[:100]
		}
		if err := SendPushToUser(userId, title, body); err != nil {
			fmt.Println("alarm push failed:", err)
		}
	}()

	return int(alarmId), nil
}
