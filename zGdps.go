package main

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	_ "github.com/go-sql-driver/mysql"
)

// TODO: сделать lgbtBan
func GDPSopener(w http.ResponseWriter, r *http.Request) {
	idPre := r.URL.Query().Get("id")

	id, err := strconv.Atoi(idPre)
	if err != nil {
		http.Error(w, "Invalid id parameter", http.StatusBadRequest)
		return
	}

	var (
		gdpsJSON  json.RawMessage
		newsJSON  json.RawMessage
		commsJSON json.RawMessage
	)

	g, _ := errgroup.WithContext(r.Context())

	g.Go(func() error {
		cached, err := RamGet(fmt.Sprintf("gdps:%d", id))
		if err != nil {
			return err
		}
		if cached != "" {
			gdpsJSON = json.RawMessage(cached)
			return nil
		}
		gdps, err := GDPSfetchById(id)
		if err != nil {
			return err
		}
		gdpsJSON, err = json.Marshal(gdps.ToFull())
		if err != nil {
			return err
		}
		return RamSet(
			fmt.Sprintf("gdps:%d", id),
			string(gdpsJSON),
			10*time.Minute,
		)
	})

	g.Go(func() error {
		cached, err := RamGet(fmt.Sprintf("news:%d", id))
		if err != nil {
			return err
		}
		if cached != "" {
			newsJSON = json.RawMessage(cached)
			return nil
		}
		newsPre, err := GetLocalNews(id, 0)
		if err != nil {
			return err
		}
		news := make([]NewsResp, 0, len(newsPre))
		for _, n := range newsPre {
			news = append(news, n.NewsRender())
		}
		newsJSON, err = json.Marshal(news)
		if err != nil {
			return err
		}
		return RamSet(
			fmt.Sprintf(":news:%d", id),
			string(newsJSON),
			10*time.Minute,
		)
	})

	g.Go(func() error {
		cached, err := RamGet(fmt.Sprintf("comms:0:%d", id))
		if err != nil {
			return err
		}
		if cached != "" {
			commsJSON = json.RawMessage(cached)
			return nil
		}
		commPre, err := GetComms(0, id, 0)
		if err != nil {
			return err
		}
		comms := make([]CommResp, 0, len(commPre))
		for _, c := range commPre {
			comms = append(comms, c.CommRender())
		}
		commsJSON, err = json.Marshal(comms)
		if err != nil {
			return err
		}
		return RamSet(
			fmt.Sprintf("comms:0:%d", id),
			string(commsJSON),
			10*time.Minute,
		)
	})

	if err := g.Wait(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := fmt.Sprintf(
		`{"gdps":%s,"comments":%s,"news":%s}`,
		gdpsJSON,
		commsJSON,
		newsJSON,
	)
	w.Write([]byte(resp))
}

/* php:
public static function logJoin(int $gdpsId, int $userId, string $findKey) {
	global $conn;
	$conn->prepare('INSERT INTO `joinlog`(`gdpsId`, `userId`, `joinData`, `joinDate`) VALUES (?, ?, ?, ?)')
		->execute([$gdpsId, $userId, $findKey, time()]);
}
*/
// openGo:
func LogJoin(gdpsId, userId int, findKey string) error {
	_, err := DB.Exec(`INSERT INTO joinlog (gdpsId, userId, joinData, joinDate) VALUES (?,?,?,?)`, gdpsId, userId, findKey, time.Now().Unix())
	if err != nil {
		return err
	}
	return nil
}

func GDPSjoin(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, "Invalid GDPS ID", http.StatusBadRequest)
		return
	}
	gdps, err := GDPSfetchById(id)
	if err != nil {
		http.Error(w, "GDPS not found", http.StatusNotFound)
		return
	}
	q := r.URL.Query()
	findKey := "id=" + strconv.Itoa(id)
	if q.Get("m") == "1" {
		findKey = "mainPage"
	}
	var links map[string]string
	rawLink := strings.ReplaceAll(gdps.Link, `\"`, `"`)
	if strings.HasPrefix(rawLink, "{") {
		if err := json.Unmarshal([]byte(rawLink), &links); err != nil {
			http.Error(w, "Invalid GDPS links", http.StatusInternalServerError)
			return
		}
	}
	//
	var targetURL string
	if links != nil {
		count := len(links)
		if linkType := q.Get("type"); linkType != "" {
			if link, ok := links[linkType]; ok {
				targetURL = link
				findKey += "&" + linkType
			} else {
				http.Error(w, "Unknown GDPS link type", http.StatusNotFound)
				return
			}
		} else if count == 1 {
			for _, link := range links {
				targetURL = link
			}
		} else {
			writeGDPSJoinPage(w, links)
			return
		}
	} else {
		targetURL = gdps.Link
	}
	if _, err := url.ParseRequestURI(targetURL); err != nil {
		http.Error(w, "Invalid GDPS link", http.StatusInternalServerError)
		return
	}
	if gdps.Freejoin == 0 {
		userID := 0

		// Если есть авторизация через token — здесь нужно получить
		// пользователя твоим существующим способом.
		if token := r.URL.Query().Get("token"); token != "" {
			if user, err := GetUserByToken(token); err == nil && user != nil {
				userID = user.UserId
			}
		}
		if err := LogJoin(id, userID, findKey); err != nil {
			log.Printf("failed to log GDPS join: %v", err)
		}
		http.Redirect(w, r, targetURL, http.StatusFound)
		return
	}
	username := r.FormValue("username")
	password := r.FormValue("password")
	if username != "" && password != "" {
		user, err := GetUserByUsername(username)
		if err != nil {
			http.Error(w, "Invalid user", http.StatusInternalServerError)
		}
		if user != nil && PasswordVerify(password, user.Password) {
			if err := LogJoin(id, user.UserId, findKey); err != nil {
				log.Printf("failed to log GDPS join: %v", err)
			}
			http.Redirect(w, r, targetURL, http.StatusFound)
			return
		}
	}
	// Если token передан, пробуем авторизацию через него.
	if token := r.FormValue("token"); token != "" {
		user, err := GetUserByToken(token)
		if err == nil && user != nil {
			if err := LogJoin(id, user.UserId, findKey); err != nil {
				log.Printf("failed to log GDPS join: %v", err)
			}
			http.Redirect(w, r, targetURL, http.StatusFound)
			return
		}
	}
	// Здесь нужно показать форму логина.
	writeGDPSLoginPage(w, targetURL)
}

func writeGDPSJoinPage(w http.ResponseWriter, links map[string]string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	for linkType, link := range links {
		fmt.Fprintf(
			w,
			`<a href="%s">%s</a><br>`,
			html.EscapeString(link),
			html.EscapeString(linkType),
		)
	}
}

func writeGDPSLoginPage(w http.ResponseWriter, targetURL string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	fmt.Fprintf(w, `<h1>Login required</h1>
		<form method="POST">
			<input name="username" placeholder="Username">
			<input name="password" type="password" placeholder="Password">
			<button type="submit">Login</button>
		</form>`)
}
