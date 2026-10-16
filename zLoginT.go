package main

/*
InitOjhub fixes what's broken!

And now seriously. Previously in php/node I could afford
short initialization functions thanks to the dynamism of the languages.
In openGo I can no longer afford this!
That's why I had to split the old loginToken into
publicInit and privateInit, which deliberately cache the broken Json.
In turn, OjhubInit will glue these 2 broken Jsons
into the good old loginT.php array, which is read by any client
of Object hub starting from GDPS Helper 1.7

NOTE!
In this code there is VEEEEERY much manual json assembly. I do this to deliberately
avoid json parsing - go is noticeably faster than factcgi/v8, here it will be felt
So if you're going to add new fields to UserResponse, go through the checklist:
1. users.go => structure changed
2. GreatGuestState patched
Why do I torture myself like this and break all the typing here?
This endpoint under traffic spikes will be the hottest spot, of course such optimization
is unlikely to help me with the database choking, but spending extra CPU cycles on a bunch of json.Marshal
I definitely don't want to.
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
	ip := ExtractIP(r)
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
		gdpsShorts = append(gdpsShorts, p.ToShort(false, false))
	}
	GDPSes := GenerateOrderedMap(
		gdpsShorts,
		func(p GDPSshort) string {
			return fmt.Sprintf("%s%d", ChannelIdsToString(p.Channel), p.ID)
		},
	)
	newsShorts := make([][]interface{}, 0, len(cachedNews))
	for _, n := range cachedNews {
		newsShorts = append(newsShorts, n.NewsRenderLegacy())
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
		// In case the ID turned out to be the last field of the object.
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
					userGdpses = append(userGdpses, p.ToShort(true, true))
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

// port of loginT and loginToken from php/node, now it's "site initialization" and not some "login by token"
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
