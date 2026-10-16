package main

// MODULE VIBE-CODED BY chatGpt!
// I didn't verify it at all
// 11:10:2026 No, Claude patched it

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// scrubErr removes the URL (and therefore the bot token) from http.Client / url.Parse errors.
func scrubErr(err error) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		return fmt.Errorf("%s: %w", ue.Op, ue.Err)
	}
	return err
}

const (
	GDPSban    = "Забанен ПРОЕКТ - "
	GDPSunban  = "Разбанен ПРОЕКТ - "
	GDPSdelete = "Удалён ПРОЕКТ - "
	GDPSlgbt1  = "это ЛГБТ ПРОЕКТ - "
	GDPSlgbt2  = "это НЕ ЛГБТ ПРОЕКТ - "

	GUIDban    = "Забанен ГАЙД - "
	GUIDunban  = "Разбанен ГАЙД - "
	GUIDdelete = "Удалён ГАЙД - "
	GUIDlgbt1  = "это ЛГБТ ГАЙД - "
	GUIDlgbt2  = "это НЕ ЛГБТ ГАЙД - "

	WIKIban    = "Забанен ВИКИ - "
	WIKIunban  = "Разбанен ВИКИ - "
	WIKIdelete = "Удалён ВИКИ - "
	WIKIlgbt1  = "это ЛГБТ ВИКИ - "
	WIKIlgbt2  = "это НЕ ЛГБТ ВИКИ - "

	VACban    = "Забанена ВАКАНСИЯ - "
	VACunban  = "Разбанена ВАКАНСИЯ - "
	VACdelete = "Удалена ВАКАНСИЯ - "
	VAClgbt1  = "это ЛГБТ ВАКАНСИЯ - "
	VAClgbt2  = "это НЕ ЛГБТ ВАКАНСИЯ - "
)

var GlobalHTTPClient = &http.Client{
	Timeout: 5 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
	},
}

// TGWebhookLog sends a technical log to Telegram.
// A log sending error must not break the main request.
func TGWebhookLog(msg string) error {
	log.Println(msg)
	botToken := os.Getenv("TG_BOT_TOKEN")
	chatID := os.Getenv("TG_NEWS_RESENDER")

	if botToken == "" || chatID == "" {
		log.Printf("[TGWebhookLog] ENV MISSING: token_set=%v chat_set=%v", botToken != "", chatID != "")
		return fmt.Errorf("telegram configuration is missing")
	}

	message := msg + "\n\n" + HELPER_VER

	data := url.Values{}
	data.Set("chat_id", chatID)
	data.Set("text", message)

	endpoint := "https://api.telegram.org/" +
		botToken +
		"/sendMessage"

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		endpoint,
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		err = scrubErr(err)
		log.Println("failed to create telegram request: %w", err)
		return fmt.Errorf("failed to create telegram request: %w", err)
	}

	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	resp, err := GlobalHTTPClient.Do(req)
	if err != nil {
		err = scrubErr(err)
		log.Printf("[TGWebhookLog] http request failed: %v", err)
		return fmt.Errorf("failed to send telegram log: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println("telegram returned status: ", resp.StatusCode)
		return fmt.Errorf(
			"telegram returned status %d",
			resp.StatusCode,
		)
	}

	return nil
}
