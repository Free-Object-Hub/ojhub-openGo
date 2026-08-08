package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
)

func vacApply(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	vacID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || vacID <= 0 {
		http.Error(w, "Invalid vacancy ID", http.StatusBadRequest)
		return
	}
	applyID, err := ApplyToVacancy(vacID, user.UserId)
	if err != nil {
		if errors.Is(err, ErrAlreadyApplied) {
			fmt.Fprint(w, "-1")
			return
		}
		http.Error(w, "Failed to apply to vacancy", http.StatusInternalServerError)
		return
	}
	fmt.Fprint(w, applyID)
}

// FIXME: сделать "дешборд совместимым" АКА добавить логику для работы
// авторов проектов (чтобы они могли удалять отклики на своих вакансиях)
func vacUnapply(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	user, ok := RequireDevice(w, r)
	if !ok {
		return
	}
	applyID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || applyID <= 0 {
		http.Error(w, "Invalid apply ID", http.StatusBadRequest)
		return
	}
	// сначала пробуем удалить как свой собственный отклик
	err = RemoveVacancyApply(applyID, user.UserId)
	if err == nil {
		fmt.Fprint(w, "1")
		return
	}
	if !errors.Is(err, ErrApplyNotFound) {
		http.Error(w, "Failed to remove vacancy apply", http.StatusInternalServerError)
		return
	}
	// не свой отклик - проверяем права владельца через vacId->gdpsId
	realGdpsId, err := GetGdpsIdByApplyId(applyID)
	if err != nil {
		http.Error(w, "Apply not found", http.StatusNotFound)
		return
	}
	access, err := CheckGdpsAccess(user.UserId, realGdpsId)
	if err != nil || (access == 0 && user.Priority == 0) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}
	if err := RemoveVacancyApplyAsOwner(applyID); err != nil {
		http.Error(w, "Failed to remove vacancy apply", http.StatusInternalServerError)
		return
	}
	fmt.Fprint(w, "1")
}

type VacancyPageResponse struct {
	GDPS     map[string]Vacan `json:"gdps"`
	Comments []CommResp       `json:"comments"`
}

func GetVacancyPage(vacID, userID int) (*VacancyPageResponse, error) {
	vacancy, err := VACSfetchById(vacID, userID)
	if err != nil {
		return nil, err
	}
	comms, err := GetComms(5, int(vacID), 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get vacancy comments: %w", err)
	}
	result := &VacancyPageResponse{
		GDPS: map[string]Vacan{
			"v" + strconv.Itoa(vacancy.ID): vacancy.ToFull(userID, false),
		},
		Comments: comms,
	}
	return result, nil
}

func GetOneVac(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	vacID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, "invalid vacancy id", http.StatusBadRequest)
		return
	}
	userID := 0
	if token := GetUserToken(r); token != "" {
		user, err := GetUserByToken(token)
		if err != nil {
			http.Error(w, "Error with user", http.StatusInternalServerError)
			fmt.Fprint(w, err)
			return
		}
		if user != nil {
			userID = user.UserId
		}
	}
	result, err := GetVacancyPage(vacID, userID)
	if errors.Is(err, ErrVacancyNotFound) {
		fmt.Fprint(w, `["NONE"]`)
		return
	}
	if err != nil {
		http.Error(w, "failed to get vacancy: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Println("failed to encode vacancy:", err)
	}
}
