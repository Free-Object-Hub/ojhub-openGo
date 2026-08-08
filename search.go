package main

/*
 * GDPS Helper Finder (расширенный)
 * Author: DenisC
 * Contributor: MIOBOMB (расширение поиска до логики деления на каналы и openGo порт
 *              (да, логика битмаски тоже за авторством DenisC)
 *
 * Старый добрый костыльный, но очень надёжный поиск почти прямиком из GDPS Helper 1.7
 * Его судьба в openGo не самая завидная - ради удобства и гибкости мне пришлось нарушить DRY
 */

import (
	"fmt"
	"sort"
	"strings"
	//
	_ "github.com/go-sql-driver/mysql"
)

func mergeAndSortTags(tags, oss []int) []int {
	result := append([]int{}, tags...)
	result = append(result, oss...)
	//
	seen := make(map[int]bool)
	unique := []int{}
	for _, v := range result {
		if !seen[v] {
			seen[v] = true
			unique = append(unique, v)
		}
	}
	sort.Ints(unique)
	return unique
}

// createBitmask создает битовую маску из ID тегов
// если вы как и я не знаете почему оно работает -
// я вас понимаю
func createBitmask(tagIDs []int) int {
	mask := 0
	for _, tagID := range tagIDs {
		if tagID > 0 {
			mask |= (1 << (tagID - 1))
		}
	}
	return mask
}

func NewGdpsFinder(method, typeFilter, page int, tags, oss []int, name string, lgbtBan int) ([]GDPS, error) {
	var queryBuilder strings.Builder
	var args []interface{}
	queryBuilder.WriteString(`
		SELECT g.ID,g.title,g.channel,g.description,g.short,g.tags,g.os,g.mask,g.likes,g.disls,g.commsCount,g.author,g.username,g.img,g.ban,g.connectedWiki 
		FROM gdpses g 
		WHERE g.checked = 1
	`)
	if typeFilter > -1 {
		queryBuilder.WriteString(` AND g.channel = ?`)
		args = append(args, typeFilter)
	}
	if lgbtBan != 0 {
		queryBuilder.WriteString(" AND g.hasLgbt = 0")
	}
	// Фильтр по названию
	if name != "" {
		queryBuilder.WriteString(" AND LOWER(g.title) LIKE LOWER(?)")
		args = append(args, "%"+name+"%")
	}
	// Фильтр по тегам ЧЕРЕЗ БИТМАСКУ
	if len(tags) > 0 || len(oss) > 0 {
		mergedTags := mergeAndSortTags(tags, oss)
		bitmask := createBitmask(mergedTags)
		//
		if bitmask != 0 {
			queryBuilder.WriteString(" AND (g.mask & ?) = ?")
			args = append(args, bitmask, bitmask)
		}
	}
	switch method {
	case 0:
		queryBuilder.WriteString(" ORDER BY g.ID DESC")
	case 1:
		queryBuilder.WriteString(" ORDER BY g.likes DESC")
	case 2:
		queryBuilder.WriteString(" ORDER BY g.disls")
	case 3:
		queryBuilder.WriteString(" ORDER BY g.points DESC")
	}
	limit := 9
	offset := 0
	if page > 0 {
		offset = page * 8
		limit = 9
	}
	queryBuilder.WriteString(fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset))
	query := queryBuilder.String()
	//
	var projects []GDPS
	err := DB.Select(&projects, query, args...)
	return projects, err
}

func NewWikiFinder(method, typeFilter, page int, tags, oss []int, name string, lgbtBan int) ([]Wiki, error) {
	var queryBuilder strings.Builder
	var args []interface{}
	queryBuilder.WriteString(`
		SELECT g.* 
		FROM wikis g 
		WHERE g.checked = 1
	`)
	if lgbtBan != 0 {
		queryBuilder.WriteString(" AND g.hasLgbt = 0")
	}
	// Фильтр по названию
	if name != "" {
		queryBuilder.WriteString(" AND LOWER(g.title) LIKE LOWER(?)")
		args = append(args, "%"+name+"%")
	}
	// FIXME: либо добавить методы в вики поиск, либо удалить их отсюда
	switch method {
	case 0:
		queryBuilder.WriteString(" ORDER BY g.ID DESC")
	case 1:
		queryBuilder.WriteString(" ORDER BY g.likes DESC")
	case 2:
		queryBuilder.WriteString(" ORDER BY g.disls")
	case 3:
		queryBuilder.WriteString(" ORDER BY g.points DESC")
	}
	limit := 9
	offset := 0
	if page > 0 {
		offset = page * 8
		limit = 9
	}
	queryBuilder.WriteString(fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset))
	query := queryBuilder.String()
	//
	var wikis []Wiki
	err := DB.Select(&wikis, query, args...)
	return wikis, err
}

func NewVacsFinder(userID, method, typeFilter, page int, tags, oss []int, name string, lgbtBan int) ([]Vacancy, error) {
	var queryBuilder strings.Builder
	var args []interface{}
	queryBuilder.WriteString(`
		SELECT g.*, p.channel as gChannel, p.title as gTitle, a.ID as applied
		FROM vacans g
		LEFT JOIN gdpses p ON g.gdpsId = p.ID
		LEFT JOIN vacsApplies a ON g.ID = a.vacId AND a.userId = ?
		WHERE g.checked = 1
	`)
	args = append(args, userID)
	// Фильтр LGBT
	if lgbtBan != 0 {
		queryBuilder.WriteString(" AND g.hasLgbt = 0")
	}
	// Фильтр по названию
	if name != "" {
		queryBuilder.WriteString(" AND LOWER(g.title) LIKE LOWER(?)")
		args = append(args, "%"+name+"%")
	}
	// Фильтр по тегам через битовую маску
	if len(tags) > 0 || len(oss) > 0 {
		mergedTags := mergeAndSortTags(tags, oss)
		bitmask := createBitmask(mergedTags)
		if bitmask != 0 {
			queryBuilder.WriteString(" AND (g.mask & ?) = ?")
			args = append(args, bitmask, bitmask)
		}
	}
	// Сортировка
	switch method {
	case 0:
		queryBuilder.WriteString(" ORDER BY g.ID DESC")
	case 1:
		queryBuilder.WriteString(" ORDER BY g.likes DESC")
	case 2:
		queryBuilder.WriteString(" ORDER BY g.disls")
	case 3:
		queryBuilder.WriteString(" ORDER BY g.points DESC")
	}
	// Пагинация: 9 элементов, 8 новых при следующей странице.
	limit := 9
	offset := 0
	if page > 0 {
		offset = page * 8
	}
	queryBuilder.WriteString(fmt.Sprintf(
		" LIMIT %d OFFSET %d",
		limit,
		offset,
	))
	var vacs []Vacancy
	err := DB.Select(
		&vacs,
		queryBuilder.String(),
		args...,
	)
	return vacs, err
}

func NewProjectsFinder(userId, method, typeFilter, page int, tags, oss []int, name string, lgbtBan int) (interface{}, error) {
	switch typeFilter {
	case -5:
		// Вакансии
		return NewVacsFinder(
			userId,
			method,
			typeFilter,
			page,
			tags,
			oss,
			name,
			lgbtBan,
		)
	// FIXME: выделить канал под поиск гайдов
	case -2:
		// Все проекты
		return NewGdpsFinder(
			method,
			-1,
			page,
			tags,
			oss,
			name,
			lgbtBan,
		)
	case -1:
		// Вики
		return NewWikiFinder(
			method,
			typeFilter,
			page,
			tags,
			oss,
			name,
			lgbtBan,
		)
	case 0, 1, 2, 3:
		// Проекты конкретного канала
		return NewGdpsFinder(
			method,
			typeFilter,
			page,
			tags,
			oss,
			name,
			lgbtBan,
		)
	default:
		return nil, fmt.Errorf(
			"неверный тип фильтра: %d",
			typeFilter,
		)
	}
}
