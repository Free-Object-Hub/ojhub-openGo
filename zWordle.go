package main

/*
 * Wordle module
 * Author: Claude
 * Port made by a chat bot in 2 minutes
 * Yes, vibe-coded, but in just 2 minutes though!
 */

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var wordleRUWords = []string{
	"АБЕР", "АВЗИ", "БЗТЗ", "БУКА", "ЕБАР", "ИНМТ", "НСОШ", "ЭЗМГ", "ОЛЕГ", "ОШКФ", "ОШФБ", "ПИПП", "СЗНП", "СОКА", "ЁШКА",
	"АРТИВ", "САЛКИ", "ГОЙДА", "ФРИДЖ", "КЛЕВИ", "ОШХАБ", "КОСМО", "МЁРФИ", "НОВЛО", "КНАЙФ", "ПСЁВЯ", "ПШИКА",
	"ИКОТИК", "ФЕДКАР", "САЙДАМ", "ПИКЧЕР", "ХЕЛОИЗ", "ШПИНАТ", "НЕСКЬЮ", "ДЕЛЬТА", "СТЁРКА",
}

var wordleENWords = []string{
	"BFDI", "EXOL", "IDFK", "HOST", "LIAM", "TACO", "AIRY", "FOUR", "IDFB", "TDOS", "ITFT", "MOSS", "BOOK", "GATY",
	"CFMOT", "SACRI", "COINY", "DAVID", "FIREY", "CABBY", "BFDIA",
	"BURNER", "FLOWER", "HFJONE", "NEEDLE", "CHEESY", "THANOS", "PENCIL",
}

// in this version the function deliberately has a broken protocol because
// the PHP original loaded the CPU quite a bit
func getDailyIndex(words []string, day int64) int {
	x := uint64(day)
	x = (x ^ (x >> 33)) * 0xff51afd7ed558ccd
	x = (x ^ (x >> 33)) * 0xc4ceb9fe1a85ec53
	x ^= x >> 33
	return int(x % uint64(len(words)))
}

// wordleGuess — shared letter comparison logic, reused by RU and EN.
func wordleGuess(userWord, secretWord string) string {
	userLetters := []rune(userWord)
	secretLetters := []rune(secretWord)
	result := make([]int, len(secretLetters))
	// exact match
	for i := 0; i < len(secretLetters) && i < len(userLetters); i++ {
		if userLetters[i] == secretLetters[i] {
			result[i] = 2
			secretLetters[i] = 0
			userLetters[i] = 0
		}
	}
	// letter is present, but not in its place
	for i := 0; i < len(userLetters); i++ {
		if userLetters[i] != 0 {
			for j := 0; j < len(secretLetters); j++ {
				if secretLetters[j] != 0 && secretLetters[j] == userLetters[i] {
					result[i] = 1
					secretLetters[j] = 0
					break
				}
			}
		}
	}
	var sb strings.Builder
	for _, v := range result {
		sb.WriteString(strconv.Itoa(v))
	}
	return sb.String()
}

// wordleHandler — shared handler factory for RU/EN endpoints.
func wordleHandler(words []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		userWord := r.PostForm.Get("word")
		day := time.Now().Unix() / 86400
		secretWord := strings.ToLower(words[getDailyIndex(words, day)])
		if userWord == "" {
			fmt.Fprint(w, len([]rune(secretWord)))
			return
		}
		fmt.Fprint(w, wordleGuess(userWord, secretWord))
	}
}

var (
	WordleRU = wordleHandler(wordleRUWords)
	WordleEN = wordleHandler(wordleENWords)
)
