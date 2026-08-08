package main

// XXX: этот модуль целиком и полностью был портирован chatgpt, я просто привёл его к моему стилю кода

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"
)

var (
	ErrVacancyNotFound = errors.New("vacancy not found")
	ErrApplyNotFound   = errors.New("apply not found")
	ErrAlreadyApplied  = errors.New("already applied")
)

// Vacancy — запись вакансии + связанные данные,
// необходимые публичному API.
type Vacancy struct {
	ID      int    `db:"ID"`
	Title   string `db:"title"`
	Tags    string `db:"tags"`
	Mask    int    `db:"mask"`
	Text    string `db:"text"`
	Short   string `db:"short"`
	VacID   int    `db:"vacId"`
	GdpsID  int    `db:"gdpsId"`
	UserID  int    `db:"userId"`
	Checked int    `db:"checked"`
	HasLgbt int    `db:"hasLgbt"`
	Date    int    `db:"date"`
	Status  int    `db:"status"`
	// Статистика.
	Likes      int `db:"likes"`
	Dislikes   int `db:"disls"`
	CommsCount int `db:"commsCount"`
	// Связанный GDPS.
	GTitle   *string `db:"gTitle"`
	GChannel *int    `db:"gChannel"`
	// ID отклика текущего пользователя.
	// Никаких likes/isLiked здесь больше нет.
	Applied *int `db:"applied"`
}

// VacancyApply — отклик пользователя на вакансию.
type VacancyApply struct {
	ID     int `db:"ID"`
	VacID  int `db:"vacId"`
	UserID int `db:"userId"`
	Date   int `db:"date"`
	Status int `db:"status"`
	// Данные пользователя, если отклики получаем через JOIN users.
	Username string `db:"uUsername"`
	Nickname string `db:"uNickname"`
	Resume   string `db:"uResume"`
}

type Vacan struct {
	ID        int     `json:"ID"`
	Title     string  `json:"title"`
	Tags      string  `json:"tags"`
	Text      string  `json:"text"`
	Short     string  `json:"short,omitempty"`
	Date      int     `json:"date"`
	IsApplied int     `json:"isApplied"`
	Likes     [3]int  `json:"likes"`
	GID       int     `json:"gId"`
	GTitle    *string `json:"gTitle"`
	GChannel  *int    `json:"gChannel"`
}

func (v Vacancy) ToFull(userID int, alsoAddShort bool) Vacan {
	applied := 0
	if userID != 0 && v.Applied != nil {
		applied = *v.Applied
	}
	short := ""
	if alsoAddShort == true {
		short = v.Short
	}
	return Vacan{
		ID:        v.ID,
		Title:     v.Title,
		Tags:      v.Tags,
		Text:      v.Text,
		Short:     short,
		Date:      v.Date,
		IsApplied: applied,
		Likes: [3]int{
			v.Likes,
			v.Dislikes,
			v.CommsCount,
		},
		GID:      v.GdpsID,
		GTitle:   v.GTitle,
		GChannel: v.GChannel,
	}
}

func (v Vacancy) ToShort(userID int) Vacan {
	text := v.Short
	if text == "" {
		text = v.Text
		runes := []rune(text)
		if len(runes) > 121 {
			text = string(runes[:121])
		}
	}
	applied := int(0)
	log.Println(userID, v.Applied)
	if userID != 0 && v.Applied != nil {
		applied = *v.Applied
	}
	return Vacan{
		ID:        v.ID,
		Title:     v.Title,
		Tags:      v.Tags,
		Text:      text,
		Date:      v.Date,
		IsApplied: applied,
		Likes: [3]int{
			v.Likes,
			v.Dislikes,
			v.CommsCount,
		},
		GID:      v.GdpsID,
		GChannel: v.GChannel,
		GTitle:   v.GTitle,
	}
}

func VACSfetchById(vacID int, userID int) (*Vacancy, error) {
	var vacancy Vacancy
	var err error
	if userID == 0 {
		err = DB.Get(
			&vacancy,
			`SELECT v.*, g.channel AS gChannel, g.title AS gTitle 
			FROM vacans v
			LEFT JOIN gdpses g ON v.gdpsId = g.ID
			WHERE v.ID = ? LIMIT 1`,
			vacID,
		)
	} else {
		err = DB.Get(
			&vacancy,
			`
			SELECT v.*, a.ID AS applied, g.channel AS gChannel, g.title AS gTitle
			FROM vacans v
			LEFT JOIN vacsApplies a ON v.ID = a.vacId AND a.userId = ?
			LEFT JOIN gdpses g ON v.gdpsId = g.ID
			WHERE v.ID = ? LIMIT 1`,
			userID,
			vacID,
		)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrVacancyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get vacancy %d: %w", vacID, err)
	}
	return &vacancy, nil
}

func GetPublicVacancies(page int) ([]Vacancy, error) {
	if page < 0 {
		page = 0
	}
	offset := page * 8
	vacancies := make([]Vacancy, 0)
	err := DB.Select(
		&vacancies,
		`SELECT v.*, g.channel AS gChannel, g.title AS gTitle
		FROM vacans v
		LEFT JOIN gdpses g ON v.gdpsId = g.ID
		WHERE v.checked = 1 ORDER BY v.ID DESC LIMIT 9 OFFSET ?`,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get public vacancies: %w", err)
	}
	return vacancies, nil
}

func GetVacanciesByGDPS(gdpsID, userID int, isAdmin bool, page int) ([]Vacancy, error) {
	if page < 0 {
		page = 0
	}
	offset := page * 8
	var vacancies []Vacancy
	if userID != 0 {
		query := `SELECT v.*, a.ID AS applied, g.channel AS gChannel, g.title AS gTitle
			FROM vacans v
			LEFT JOIN vacsApplies a ON v.ID = a.vacId AND a.userId = ?
			LEFT JOIN gdpses g ON v.gdpsId = g.ID
			WHERE v.gdpsId = ?`
		args := []interface{}{
			userID,
			gdpsID,
		}
		if !isAdmin {
			query += ` AND v.checked = 1`
		}
		query += ` ORDER BY v.ID DESC
			LIMIT 9 OFFSET ?`
		args = append(args, offset)
		if err := DB.Select(&vacancies, query, args...); err != nil {
			return nil, fmt.Errorf(
				"failed to get vacancies for gdps %d: %w",
				gdpsID,
				err,
			)
		}
		return vacancies, nil
	}
	query := `SELECT v.*, g.channel AS gChannel, g.title AS gTitle
		FROM vacans v
		LEFT JOIN gdpses g ON v.gdpsId = g.ID
		WHERE v.gdpsId = ?`
	args := []interface{}{gdpsID}
	if !isAdmin {
		query += ` AND v.checked = 1`
	}
	query += ` ORDER BY v.ID DESC
		LIMIT 9 OFFSET ?`
	args = append(args, offset)
	if err := DB.Select(&vacancies, query, args...); err != nil {
		return nil, fmt.Errorf(
			"failed to get vacancies for gdps %d: %w",
			gdpsID,
			err,
		)
	}
	return vacancies, nil
}

func AddVacancy(title, text, short, tags string, mask int, gdpsID int, checked, hasLgbt int, date int) (int, error) {
	result, err := DB.Exec(`INSERT INTO vacans (title, text, short, tags, mask, gdpsId, checked, hasLgbt, date) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		title, text, short, tags, mask, gdpsID, checked, hasLgbt, date)
	if err != nil {
		return 0, fmt.Errorf("failed to add vacancy: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get vacancy id: %w", err)
	}
	return int(id), nil
}

func EditVacancy(vacID int, title, text, short, tags string, mask int, gdpsID int, checked, hasLgbt int) error {
	result, err := DB.Exec(`UPDATE vacans SET title = ?, text = ?, short = ?, tags = ?, mask = ?, gdpsId = ?, checked = ?, hasLgbt = ? WHERE ID = ?`,
		title, text, short, tags, mask, gdpsID, checked, hasLgbt, vacID)
	if err != nil {
		return fmt.Errorf("failed to edit vacancy %d: %w", vacID, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check vacancy update: %w", err)
	}
	if affected == 0 {
		return ErrVacancyNotFound
	}
	return nil
}

func RemoveVacancy(vacID int) error {
	tx, err := DB.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin vacancy deletion: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if _, err = tx.Exec(`DELETE FROM vacsApplies WHERE vacId = ?`, vacID); err != nil {
		return fmt.Errorf("failed to remove vacancy applies: %w", err)
	}
	result, err := tx.Exec(`DELETE FROM vacans WHERE ID = ?`, vacID)
	if err != nil {
		return fmt.Errorf("failed to remove vacancy: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check vacancy deletion: %w", err)
	}
	if affected == 0 {
		return ErrVacancyNotFound
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit vacancy deletion: %w", err)
	}
	return nil
}

func CheckVacancyApply(vacID, userID int) (*VacancyApply, error) {
	var apply VacancyApply
	err := DB.Get(&apply, `SELECT ID, vacId, userId, date, status FROM vacsApplies WHERE vacId = ?  AND userId = ? LIMIT 1`, vacID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to check vacancy apply: %w", err)
	}
	return &apply, nil
}

func (a VacancyApply) RenderApply() map[string]interface{} {
	username := a.Nickname
	if username == "" {
		username = a.Username
	}
	if username == "" {
		username = "???"
	}
	return map[string]interface{}{
		"ID":       a.ID,
		"vacId":    a.VacID,
		"userId":   a.UserID,
		"username": username,
		"resume":   a.Resume,
		"date":     a.Date,
		"status":   a.Status,
	}
}

func ApplyToVacancy(vacID, userID int) (int, error) {
	existing, err := CheckVacancyApply(vacID, userID)
	if err != nil {
		return 0, err
	}
	if existing != nil {
		return 0, ErrAlreadyApplied
	}
	result, err := DB.Exec(`INSERT INTO vacsApplies (vacId, userId, date) VALUES (?, ?, ?) `, vacID, userID, time.Now().Unix())
	if err != nil {
		return 0, fmt.Errorf("failed to apply to vacancy: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get apply id: %w", err)
	}
	return int(id), nil
}

func RemoveVacancyApply(applyID, userID int) error {
	result, err := DB.Exec(`DELETE FROM vacsApplies WHERE ID = ? AND userId = ?`, applyID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove vacancy apply: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check apply deletion: %w", err)
	}
	if affected == 0 {
		return ErrApplyNotFound
	}
	return nil
}

func GetGdpsIdByApplyId(applyId int) (int, error) {
	var gdpsId int
	err := DB.Get(&gdpsId, `
		SELECT v.gdpsId
		FROM vacsApplies a
		JOIN vacans v ON a.vacId = v.ID
		WHERE a.ID = ?
	`, applyId)
	if err != nil {
		return 0, fmt.Errorf("failed to resolve gdps for apply %d: %w", applyId, err)
	}
	return gdpsId, nil
}

func RemoveVacancyApplyAsOwner(applyID int) error {
	result, err := DB.Exec(`DELETE FROM vacsApplies WHERE ID = ?`, applyID)
	if err != nil {
		return fmt.Errorf("failed to remove vacancy apply as owner: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrApplyNotFound
	}
	return nil
}

func GetVacancyApplies(vacID int, page int) ([]VacancyApply, error) {
	if page < 0 {
		page = 0
	}
	offset := page * 20
	applies := make([]VacancyApply, 0)
	err := DB.Select(
		&applies,
		`SELECT a.ID, a.vacId, a.userId, a.date, a.status, u.username AS uUsername, u.nickname AS uNickname, u.resume AS uResume
		FROM vacsApplies a
		JOIN users u ON a.userId = u.userId
		WHERE a.vacId = ? ORDER BY a.ID DESC LIMIT 21 OFFSET ?`,
		vacID, offset,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get applies for vacancy %d: %w",
			vacID,
			err,
		)
	}
	return applies, nil
}
