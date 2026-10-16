package main

/*
 * GDPS Helper Finder (extended)
 * Author: DenisC
 * Contributor: MIOBOMB (extension of search to channel-splitting logic and openGo port
 *              (yes, the bitmask logic is also by DenisC)
 *
 * The good old crutchy but very reliable search almost straight from GDPS Helper 1.7
 * Its fate in openGo is not the most enviable - for the sake of convenience and flexibility I had to violate DRY
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

// createBitmask creates a bitmask from tag IDs
// if you, like me, don't know why it works -
// I understand you
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
	// Filter by name
	if name != "" {
		queryBuilder.WriteString(" AND LOWER(g.title) LIKE LOWER(?)")
		args = append(args, "%"+name+"%")
	}
	// Filter by tags VIA BITMASK
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
	// Filter by name
	if name != "" {
		queryBuilder.WriteString(" AND LOWER(g.title) LIKE LOWER(?)")
		args = append(args, "%"+name+"%")
	}
	// FIXME: either add methods to wiki search, or remove them from here
	switch method {
	case 0:
		queryBuilder.WriteString(" ORDER BY g.ID DESC")
	case 1:
		queryBuilder.WriteString(" ORDER BY g.likes DESC")
	case 2:
		queryBuilder.WriteString(" ORDER BY g.disls")
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

func NewGuidFinder(wikiId, method, typeFilter, page int, tags, oss []int, name string, lgbtBan int) ([]Guide, error) {
	var queryBuilder strings.Builder
	var args []interface{}
	queryBuilder.WriteString(`
		SELECT g.*
		FROM guides g
		WHERE g.checked = 1 AND wikiChannel = ?
	`)
	args = append(args, wikiId)
	// Filter by name
	if name != "" {
		queryBuilder.WriteString(" AND LOWER(g.title) LIKE LOWER(?)")
		args = append(args, "%"+name+"%")
	}
	// Filter by tags via bitmask
	if len(tags) > 0 || len(oss) > 0 {
		mergedTags := mergeAndSortTags(tags, oss)
		bitmask := createBitmask(mergedTags)
		if bitmask != 0 {
			queryBuilder.WriteString(" AND (g.mask & ?) = ?")
			args = append(args, bitmask, bitmask)
		}
	}
	// Sorting
	switch method {
	case 0:
		queryBuilder.WriteString(" ORDER BY g.ID DESC")
	case 1:
		queryBuilder.WriteString(" ORDER BY g.likes DESC")
	case 2:
		queryBuilder.WriteString(" ORDER BY g.disls")
	}
	// Pagination: 9 elements, 8 new ones on the next page.
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
	var guides []Guide
	err := DB.Select(
		&guides,
		queryBuilder.String(),
		args...,
	)
	return guides, err
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
	// LGBT filter
	if lgbtBan != 0 {
		queryBuilder.WriteString(" AND g.hasLgbt = 0")
	}
	// Filter by name
	if name != "" {
		queryBuilder.WriteString(" AND LOWER(g.title) LIKE LOWER(?)")
		args = append(args, "%"+name+"%")
	}
	// Filter by tags via bitmask
	if len(tags) > 0 || len(oss) > 0 {
		mergedTags := mergeAndSortTags(tags, oss)
		bitmask := createBitmask(mergedTags)
		if bitmask != 0 {
			queryBuilder.WriteString(" AND (g.mask & ?) = ?")
			args = append(args, bitmask, bitmask)
		}
	}
	// Sorting
	switch method {
	case 0:
		queryBuilder.WriteString(" ORDER BY g.ID DESC")
	case 1:
		queryBuilder.WriteString(" ORDER BY g.likes DESC")
	case 2:
		queryBuilder.WriteString(" ORDER BY g.disls")
	}
	// Pagination: 9 elements, 8 new ones on the next page.
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

func NewProjectsFinder(userId, method, channel, page int, tags, oss []int, name string, lgbtBan int) (interface{}, error) {
	switch channel {
	case -5:
		// Vacancies
		return NewVacsFinder(
			userId,
			method,
			channel,
			page,
			tags,
			oss,
			name,
			lgbtBan,
		)
	case -2:
		// Guides
		return NewGuidFinder(
			userId,
			method,
			channel,
			page,
			tags,
			oss,
			name,
			lgbtBan,
		)
	case -1:
		// Wiki
		return NewWikiFinder(
			method,
			channel,
			page,
			tags,
			oss,
			name,
			lgbtBan,
		)
	case 0, 1, 2, 3:
		// Projects of a specific channel
		return NewGdpsFinder(
			method,
			channel,
			page,
			tags,
			oss,
			name,
			lgbtBan,
		)
	// change the error to search of all channels because GDPS Helper + old clients didn't send channel
	default:
		// All projects
		return NewGdpsFinder(
			method,
			-1,
			page,
			tags,
			oss,
			name,
			lgbtBan,
		)
	}
}
