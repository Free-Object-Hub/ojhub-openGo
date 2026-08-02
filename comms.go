package main

/*
КОММЕНТАРИИ - АД
Приична - whereIz и каналы.

Я сам не понимаю как и почему, но у базы данных есть небольшой рассинхрон
значений который мне приходится поддерживать

список каналов (вытащил из ojhub.js 0.97.7):
0 => GDPS
1 => *???
2 => Страницы вики (гайды)
3 => Новости
4 => Форум посты
5 => Вакансии
* - вообще это текстур паки ещё с гдпс хелпера, но оно по факту нигде не используется
    т.к. текстурпаки в gpds helper были удалены ещё в 1.8

почему это ад? потому что канал приходит с клиента, и никаких эндпоинтов
"комментарии проекта" или "комментарии вакансии" не существует!!!
терпите.

а ещё большая беда - каналы лайков:
1 => GDPS
4 => Новости (поверх текстур)
5 => Страницы вики (гайды)
10 => Форум посты
12 => Вакансии

где же тут рассинхрон?
А в половине случаев каналы гайдов и новостей перемешиваются и добавляются не туда!
*/

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

type Comm struct {
	ID       int    `db:"ID"`
	Text     string `db:"text"`
	WhereIz  int    `db:"whereIz"`
	UserID   int    `db:"userId"`
	Username string `db:"username"`
	Date     int    `db:"date"`
	Likes    int    `db:"likes"`
	Disls    int    `db:"disls"`
	Channel  int    `db:"channel"`

	UUsername string `db:"uUsername"`
	UNickname string `db:"uNickname"`
	UPriority int    `db:"uPriority"`
}

func COMMfetchById(ID int) (*Comm, error) {
	var comm Comm
	query := `SELECT c.*, u.username as uUsername, u.nickname as uNickname, u.priority as uPriority FROM comments c 
		LEFT JOIN users u ON c.userId = u.userId 
		WHERE c.ID = ?`
	err := DB.Get(&comm, query, ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get news by ID: %w", err)
	}
	return &comm, nil
}

func GetComms(channel int, whereIz int, page int) ([]CommResp, error) {
	cackeKey := fmt.Sprintf("comms:%d:%d", channel, whereIz)
	if page == 0 {
		var err error
		cached, err := RamGet(cackeKey)
		if err != nil {
			return nil, err
		}
		if cached != "" {
			var comments []CommResp
			if err := json.Unmarshal([]byte(cached), &comments); err != nil {
				log.Printf("Failed to decode cached comments %s: %v", cackeKey, err)
			} else {
				return comments, nil
			}
		}
	}
	var comments []Comm
	query := `SELECT c.*, u.username as uUsername, u.nickname as uNickname, u.priority as uPriority 
	FROM comments c
	LEFT JOIN users u ON c.userId = u.userId
	WHERE c.whereIz = ? AND c.channel = ? ORDER BY c.ID DESC LIMIT 11 OFFSET ?`
	if page < 0 {
		page = 0
	}
	offset := page * 10
	err := DB.Select(
		&comments,
		query,
		whereIz,
		channel,
		offset,
	)
	result := make([]CommResp, len(comments))
	for i, comm := range comments {
		result[i] = comm.CommRender()
	}
	if page == 0 {
		if commCache, err := json.Marshal(result); err != nil {
			log.Printf("Failed to decode cached comments %s: %v", cackeKey, err)
		} else {
			log.Println(commCache)
			RamSet(
				fmt.Sprintf("comms:%d:%d", channel, whereIz),
				string(commCache),
				10*time.Minute,
			)
		}
	}
	return result, err
}

type CommResp [8]interface{}

func (c Comm) CommRender() CommResp {
	text, err := base64.StdEncoding.DecodeString(c.Text)
	if err != nil {
		text = []byte("")
	}

	username := c.UUsername
	if username == "" {
		username = "???"
	}
	if c.UNickname != "" {
		username = c.UNickname
	}

	return CommResp{
		c.ID,
		username,
		string(text),
		c.UserID,
		c.UPriority,
		[2]int{
			c.Likes,
			c.Disls,
		},
		c.Date,
		0,
	}
}

func COMMadd(userID, whereIz int, text string, date int64, channel int) error {
	_, err := DB.Exec(
		"INSERT INTO comments (userId, whereIz, text, date, channel) VALUES (?, ?, ?, ?, ?)",
		userID,
		whereIz,
		text,
		date,
		channel,
	)
	if err != nil {
		return err
	}
	if err := RamDel(
		fmt.Sprintf("comms:%d:%d", channel, whereIz),
	); err != nil {
		log.Printf("failed to invalidate comments cache: %v", err)
	}
	return err
}

func COMMmodify(id int, text string) error {
	_, err := DB.Exec(
		"UPDATE comments SET text = ? WHERE ID = ?",
		text,
		id,
	)

	return err
}

func COMMdelete(id int, likeChannel int) error {
	tx, err := DB.Beginx()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	_, err = tx.Exec(
		"DELETE FROM comments WHERE ID = ?",
		id,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(
		"DELETE FROM likes WHERE whereIz = ? AND channel = ?",
		id,
		likeChannel,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}
