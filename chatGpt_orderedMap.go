package main

import (
	"encoding/json"
	"strings"
)

/*
 * OrderedMap
 * Author: ChatGPT 5 mini (по запросу MIOBOMB)
 *
 * Костыль для openGo, который реализует старое поведение
 * у объектов в json-ответе
 *
 * FIXME: убедиться что чатгпт не дал мне слишком много "удобных функций на будущее"
 *        или "полезного поведения"
 * FIXME: когда openGo будет готова к продакшну удалить лишние "удобные методы"
 */

type OrderedEntry[T any] struct {
	Key string
	Val T
}

type OrderedMap[T any] struct {
	entries []OrderedEntry[T]
	index   map[string]int
}

func NewOrderedMap[T any]() *OrderedMap[T] {
	return &OrderedMap[T]{
		entries: make([]OrderedEntry[T], 0),
		index:   make(map[string]int),
	}
}

// Set добавляет элемент или заменяет существующий.
// При замене порядок сохраняется.
func (o *OrderedMap[T]) Set(key string, value T) {
	if o.index == nil {
		o.index = make(map[string]int)
	}

	if idx, ok := o.index[key]; ok {
		o.entries[idx].Val = value
		return
	}

	o.index[key] = len(o.entries)

	o.entries = append(o.entries, OrderedEntry[T]{
		Key: key,
		Val: value,
	})
}

// Add добавляет элемент без проверки.
// Полезно, если гарантируется уникальность ключей.
func (o *OrderedMap[T]) Add(key string, value T) {
	if o.index == nil {
		o.index = make(map[string]int)
	}

	o.index[key] = len(o.entries)

	o.entries = append(o.entries, OrderedEntry[T]{
		Key: key,
		Val: value,
	})
}

func (o OrderedMap[T]) Get(key string) (T, bool) {
	var zero T

	if idx, ok := o.index[key]; ok {
		return o.entries[idx].Val, true
	}

	return zero, false
}

func (o OrderedMap[T]) Has(key string) bool {
	_, ok := o.index[key]
	return ok
}

func (o OrderedMap[T]) Delete(key string) {
	idx, ok := o.index[key]
	if !ok {
		return
	}

	delete(o.index, key)

	o.entries = append(
		o.entries[:idx],
		o.entries[idx+1:]...,
	)

	// перестраиваем индексы
	for i := idx; i < len(o.entries); i++ {
		o.index[o.entries[i].Key] = i
	}
}

func (o OrderedMap[T]) Len() int {
	return len(o.entries)
}

func (o OrderedMap[T]) Empty() bool {
	return len(o.entries) == 0
}

// Range идёт в порядке добавления.
func (o OrderedMap[T]) Range(fn func(key string, value T)) {
	for _, item := range o.entries {
		fn(item.Key, item.Val)
	}
}

// Keys возвращает ключи в порядке вставки.
func (o OrderedMap[T]) Keys() []string {
	keys := make([]string, 0, len(o.entries))

	for _, item := range o.entries {
		keys = append(keys, item.Key)
	}

	return keys
}

// Values возвращает значения в порядке вставки.
func (o OrderedMap[T]) Values() []T {
	values := make([]T, 0, len(o.entries))

	for _, item := range o.entries {
		values = append(values, item.Val)
	}

	return values
}

/*
 * OrderedMap generator
 *
 * Делает из обычного среза PHP-подобный объект:
 *
 * []T
 *   |
 *   v
 * {
 *   "prefix+id": T,
 *   "prefix+id": T
 * }
 *
 * Используется для совместимости с API Object Hub,
 * где ключи объектов имеют формат:
 *
 * c123
 * s456
 * p789
 * n101
 *
 */

func GenerateOrderedMap[T any](
	items []T,
	keyFunc func(T) string,
) *OrderedMap[T] {
	result := NewOrderedMap[T]()

	for _, item := range items {
		result.Set(
			keyFunc(item),
			item,
		)
	}

	return result
}

// MarshalJSON делает настоящий JSON object:
//
//	{
//	  "c123": {...},
//	  "s456": {...}
//	}
//
// но в порядке добавления.
func (o OrderedMap[T]) MarshalJSON() ([]byte, error) {
	var b strings.Builder

	b.WriteByte('{')

	for i, item := range o.entries {
		if i > 0 {
			b.WriteByte(',')
		}

		key, err := json.Marshal(item.Key)
		if err != nil {
			return nil, err
		}

		val, err := json.Marshal(item.Val)
		if err != nil {
			return nil, err
		}

		b.Write(key)
		b.WriteByte(':')
		b.Write(val)
	}

	b.WriteByte('}')

	return []byte(b.String()), nil
}
