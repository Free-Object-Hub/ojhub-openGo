// zAltcha.go
package main

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os"

	altcha "github.com/altcha-org/altcha-lib-go"
)

// AltchaSecret — глобальный секрет для ALTCHA, как и DB/GeoDb/RamDB
var AltchaSecret string

// InitAltcha — вызывается из main() рядом с InitDB()/InitGeoDb()/InitRedis()
func InitAltcha() {
	AltchaSecret = os.Getenv("ALTCHA_SECRET")
	if AltchaSecret == "" {
		log.Fatal("ALTCHA_SECRET is not set")
	}
}

// GET /server/133/challenge.php
func AltchaChallenge(w http.ResponseWriter, r *http.Request) {
	ch, err := altcha.CreateChallenge(altcha.ChallengeOptions{
		HMACKey:   AltchaSecret,
		MaxNumber: 100_000,
	})
	if err != nil {
		http.Error(w, "challenge error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ch)
}

func verifyAltcha(payloadRaw string) bool {
	decoded, err := base64.StdEncoding.DecodeString(payloadRaw)
	if err != nil || len(decoded) == 0 {
		return false
	}
	var payload altcha.Payload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return false
	}
	ok, err := altcha.VerifySolution(payload, AltchaSecret, true)
	return err == nil && ok
}
