package main

/*
InitOjhub чинит сломанное!

А теперь серьёзно. Раньше в php/node я мог позволить себе
короткие функции инициализации за счёт динамичности языков.
В openGo я такого позволить себе уже не могу!
По этому мне пришлось расколоть старый loginToken на
publicInit и privateInit, которые намернно кешируют битый Json.
В свою очередь OjhubInit будет склеивать эти 2 битых Json'а
в старый добрый массив loginT.php, который читается любым клиентом
Object hub начиная с GDPS Helper 1.7

ОБРАТИТЕ ВНИМАНИЕ!
В этом коде ОООООЧЕНЬ много ручной сборки json. Делаю я это чтобы намеренно
избегать парсинга json - го заметно быстрее factcgi/v8, тут это будет ощущаться
Так что если вы собираетесь добавлять новые поля в UserResponse, пройдитесь по чеклисту:
1. users.go => изменена структура
2. пропатчен GreatGuestState
Зачем я себя так мучаю и ломаю всю типизацию тут?
Этот эндпоинт при скачках трафика будет самым жарким местом, конечно от захлёбывания базы данных
мне подобная оптимизация наврядли поможет, но тратить лишние такты процессора на кучу json.Marshal
я точно не хочу.
*/

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	_ "github.com/go-sql-driver/mysql"
)

func FreeBSDcompile(w http.ResponseWriter, r *http.Request) {
	ip := r.Header.Get("X-Real-Ip")
	token := GetUserToken(r)
	device := GetDeviceToken(r)
	jsonData, err := InitOjhub(ip, token, device, false, false, false)
	if err != nil {
		http.Error(w, "Login error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(w).Encode(jsonData); err != nil {
		http.Error(w, "User data error: "+err.Error(), http.StatusInternalServerError)
	}
}

func GreatGuestState(cityData [2]string, needsToken bool) json.RawMessage {
	token := ""
	if needsToken {
		token = "false"
	}
	great := fmt.Sprintf(
		`{"ID":0,"username":"Object Hub","isActive":0,"role":0,"token":"%s","resume":"","socials":"","cityData":["%s","%s"]},[{},{}]`,
		token,
		cityData[0],
		cityData[1],
	)
	return json.RawMessage(great)
}

func publicInit() (json.RawMessage, error) {
	cached, err := RamGet("loginTcache")
	if err != nil {
		log.Printf("Redis GET loginTcache failed: %v", err)
	} else if cached != "" {
		return json.RawMessage(cached), nil
	}
	cachedGdpses, err := NewGdpsFinder(
		0,
		-1,
		0,
		[]int{},
		[]int{},
		"",
		0,
	)
	if err != nil {
		return nil, err
	}
	cachedNews, err := GetGlobalNews(0)
	if err != nil {
		return nil, err
	}
	gdpsShorts := make([]GDPSshort, 0, len(cachedGdpses))
	for _, p := range cachedGdpses {
		gdpsShorts = append(gdpsShorts, p.ToShort(false))
	}
	GDPSes := GenerateOrderedMap(
		gdpsShorts,
		func(p GDPSshort) string {
			return fmt.Sprintf("%s%d", ChannelIdsToString(p.Channel), p.ID)
		},
	)
	newsShorts := make([]NewsResp, 0, len(cachedNews))
	for _, p := range cachedNews {
		newsShorts = append(newsShorts, p.NewsRender())
	}
	gdpsJSON, err := json.Marshal(GDPSes)
	if err != nil {
		return nil, err
	}
	newsJSON, err := json.Marshal(newsShorts)
	if err != nil {
		return nil, err
	}
	redisData := fmt.Sprintf(
		"%s,%s",
		gdpsJSON,
		newsJSON,
	)
	if err := RamSet("loginTcache", redisData, 10*time.Minute); err != nil {
		log.Printf("Redis SET loginTcache failed: %v", err)
	}
	return json.RawMessage(redisData), nil
}

func extractUserID(cached string) (int, error) {
	const prefix = `"ID":`
	start := strings.Index(cached, prefix)
	if start == -1 {
		return 0, errors.New("user ID not found in cache")
	}
	start += len(prefix)
	end := strings.IndexByte(cached[start:], ',')
	if end == -1 {
		// На случай, если ID оказался последним полем объекта.
		end = strings.IndexByte(cached[start:], '}')
		if end == -1 {
			return 0, errors.New("invalid user cache format")
		}
	}
	end += start
	return strconv.Atoi(cached[start:end])
}

func injectToken(cached string, token string) (json.RawMessage, error) {
	if token == "" {
		return json.RawMessage(cached), nil
	}
	result := `{"token":"` + string(token) + `",` + cached[1:]
	return json.RawMessage(result), nil
}

func injectToken2(cached string, token string) (string, error) {
	if token == "" {
		return cached, nil
	}
	result := `{"token":"` + string(token) + `",` + cached[1:]
	return result, nil
}

func privateInit(ip string, token string, device string, showToken bool, ignoreDevice bool, bypassCache bool) (json.RawMessage, error) {
	if !bypassCache {
		cached, err := RamGet("userTc:" + token)
		if err != nil {
			log.Printf("Redis GET userTc failed: %v", err)
		} else if cached != "" {
			userID, err := extractUserID(cached)
			if err != nil {
				log.Printf("Failed to extract user ID from cache: %v", err)
			} else {
				deviceChk := true
				if !ignoreDevice {
					deviceChk, err = EasyCheckDevice(userID, device)
					if err != nil {
						return nil, err
					}
				}
				if deviceChk {
					if showToken {
						return injectToken(cached, token)
					}
					return json.RawMessage(cached), nil
				}
			}
		}
	}
	city, err := GetCity(ip)
	if err != nil {
		log.Println(err)
		city = [2]string{"Unknown", "Unknown"}
	}
	userResp := GreatGuestState(city, token != "")
	if token != "" {
		var user *User
		user, err = GetUserByToken(token)
		if err != nil {
			log.Println(err)
			user = nil
		}
		if user != nil {
			deviceChk := true
			if ignoreDevice == false {
				deviceChk, err = EasyCheckDevice(user.UserId, device)
				if err != nil {
					log.Println(err)
					return nil, err
				}
			}
			if deviceChk {
				userProf := user.PrivateProfile(false)
				var (
					userGdpsesPre []GDPS
					userWikiPre   []Wiki
				)
				g, _ := errgroup.WithContext(context.Background())
				g.Go(func() error {
					var err error
					userGdpsesPre, err = GetUserGdpsContent(user.UserId)
					return err
				})
				g.Go(func() error {
					var err error
					userWikiPre, err = GetUserWikiContent(user.UserId)
					return err
				})
				if err := g.Wait(); err != nil {
					return nil, err
				}

				userGdpses := make([]GDPSshort, 0, len(userGdpsesPre))
				for _, p := range userGdpsesPre {
					userGdpses = append(userGdpses, p.ToShort(true))
				}
				userGDPSes := GenerateOrderedMap(
					userGdpses,
					func(p GDPSshort) string {
						return fmt.Sprintf("%s%d", ChannelIdsToString(p.Channel), p.ID)
					},
				)
				userWikis := make([]WikiProfile, 0, len(userWikiPre))
				for _, p := range userWikiPre {
					userWikis = append(userWikis, p.ToFull())
				}
				userWIKI := GenerateOrderedMap(
					userWikis,
					func(p WikiProfile) string {
						return fmt.Sprintf("w%d", p.ID)
					},
				)
				userJSON, err := json.Marshal(userProf)
				if err != nil {
					return nil, err
				}
				gdpsJSON, err := json.Marshal(userGDPSes)
				if err != nil {
					return nil, err
				}
				wikiJSON, err := json.Marshal(userWIKI)
				if err != nil {
					return nil, err
				}
				respData := fmt.Sprintf(
					"%s,[%s,%s]",
					userJSON,
					gdpsJSON,
					wikiJSON,
				)
				redisData := respData
				if err := RamSet("userTc:"+token, redisData, 10*time.Minute); err != nil {
					log.Printf("Redis SET loginTcache failed: %v", err)
				}
				if showToken {
					respData, err = injectToken2(
						respData,
						token,
					)
				}
				return json.RawMessage(respData), nil
			}
		}
	}
	return userResp, nil
}

// порт loginT и loginToken из php/node, теперь это "инициализация сайта" а не какой то "вход по токену"
func InitOjhub(ip string, token string, device string, showToken bool, ignoreDevice bool, bypassCache bool) (json.RawMessage, error) {
	var (
		public  json.RawMessage
		private json.RawMessage
	)
	g, _ := errgroup.WithContext(context.Background())
	g.Go(func() error {
		var err error
		public, err = publicInit()
		return err
	})
	g.Go(func() error {
		var err error
		private, err = privateInit(ip, token, device, showToken, ignoreDevice, bypassCache)
		return err
	})
	if err := g.Wait(); err != nil {
		return nil, err
	}
	resp := fmt.Sprintf(
		"[%s,%s]",
		private,
		public,
	)
	return json.RawMessage(resp), nil
}
