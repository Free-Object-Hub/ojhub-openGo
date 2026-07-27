package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type WikiTemplate struct {
	ID      int    `db:"ID"`
	WikiID  int    `db:"wikiId"`
	Name    string `db:"name"`
	Args    string `db:"args"`
	Method  string `db:"method"`
	Content string `db:"content"`
}

func (t WikiTemplate) RenderTemplate() []interface{} {
	var args interface{}

	if err := json.Unmarshal([]byte(t.Args), &args); err != nil {
		args = ""
	}

	return []interface{}{
		args,
		strings.ReplaceAll(t.Content, `\n`, "\n"),
		t.Method,
	}
}

func WikiTemplateGetOne(wikiID int, name string) (*WikiTemplate, error) {
	var template WikiTemplate
	query := `SELECT * FROM wikiTemplates WHERE wikiId = ? AND name = ?`
	err := DB.Get(&template, query, wikiID, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &template, nil
}

func WikiTemplateGetMany(wikiID int, names []string) ([]WikiTemplate, error) {
	if len(names) == 0 {
		return []WikiTemplate{}, nil
	}
	placeholders := make([]string, 0, len(names))
	args := make([]interface{}, 0, len(names)+1)
	args = append(args, wikiID)
	for _, name := range names {
		placeholders = append(placeholders, "?")
		args = append(args, name)
	}
	query := fmt.Sprintf(`SELECT * FROM wikiTemplates WHERE wikiId = ? AND name IN (%s)`, strings.Join(placeholders, ","))
	var templates []WikiTemplate
	err := DB.Select(&templates, query, args...)
	if err != nil {
		return nil, err
	}
	return templates, nil
}

func WikiTemplateGetAll(wikiID, page int) ([]WikiTemplate, error) {
	offset := page * 10
	var templates []WikiTemplate
	err := DB.Select(&templates, `SELECT * FROM wikiTemplates WHERE wikiId = ? ORDER BY ID DESC LIMIT 11 OFFSET ?`, wikiID, offset)
	if err != nil {
		return nil, err
	}
	return templates, nil
}

func WikiTemplateSave(wikiID int, name, args, method, content string) (*WikiTemplate, error) {
	var exists WikiTemplate
	err := DB.Get(&exists, `SELECT * FROM wikiTemplates WHERE wikiId = ? AND name = ?`, wikiID, name)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if err == nil {
		_, err = DB.Exec(`UPDATE wikiTemplates SET args = ?, method = ?, content = ? WHERE wikiId = ? AND name = ?`, args, method, content, wikiID, name)
	} else {
		_, err = DB.Exec(`INSERT INTO wikiTemplates (args, method, content, wikiId, name) VALUES (?, ?, ?, ?, ?)`, args, method, content, wikiID, name)
	}
	if err != nil {
		return nil, err
	}
	return WikiTemplateGetOne(wikiID, name)
}

func WikiTemplateDelete(wikiID int, name string) error {
	_, err := DB.Exec(`DELETE FROM wikiTemplates WHERE wikiId = ? AND name = ?`, wikiID, name)
	return err
}

func ParseTemplateNames(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}
