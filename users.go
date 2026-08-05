package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"net/mail"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/mileusna/useragent"
	"golang.org/x/crypto/bcrypt"
)

// #region users
type User struct {
	UserId    int       `json:"ID" db:"userId"`
	Username  string    `json:"username" db:"username"`
	Nickname  string    `json:"nickname,omitempty" db:"nickname"`
	CommTime  int       `json:"-" db:"commTime"`
	Password  string    `json:"-" db:"password"`
	Mail      string    `json:"-" db:"mail"`
	Activated int       `json:"isActive" db:"activated"`
	Code      string    `json:"-" db:"code"`
	Priority  int       `json:"role" db:"priority"`
	Token     string    `json:"token,omitempty" db:"token"`
	Resume    string    `json:"resume" db:"resume"`
	Socials   string    `json:"socials" db:"socials"`
	CityData  [2]string `json:"cityData" db:"cityData"`
}

type UserResponse struct {
	ID       int       `json:"ID"`
	Username string    `json:"username"`
	IsActive int       `json:"isActive"`
	Role     int       `json:"role"`
	Token    string    `json:"token,omitempty"`
	Resume   string    `json:"resume"`
	Socials  string    `json:"socials"`
	CityData [2]string `json:"cityData"`
}

type UserPublic struct {
	ID       int    `json:"ID"`
	Username string `json:"username"`
	IsActive int    `json:"isActive"`
	Role     int    `json:"role"`
	Resume   string `json:"resume"`
	Socials  string `json:"socials"`
}

func (u User) PublicProfile() UserPublic {
	displayName := u.Username
	if u.Nickname != "" {
		displayName = u.Nickname
	}
	return UserPublic{
		ID:       u.UserId,
		Username: displayName,
		IsActive: u.Activated,
		Role:     u.Priority,
		Resume:   u.Resume,
		Socials:  u.Socials,
	}
}

func (u User) PrivateProfile(renderToken bool) UserResponse {
	displayName := u.Username
	if u.Nickname != "" {
		displayName = u.Nickname
	}
	//
	token := ""
	if renderToken {
		token = u.Token
	}
	//
	return UserResponse{
		ID:       u.UserId,
		Username: displayName,
		IsActive: u.Activated,
		Role:     u.Priority,
		Resume:   u.Resume,
		Token:    token,
		Socials:  u.Socials,
		CityData: u.CityData,
	}
}

func GetUserByToken(token string) (*User, error) {
	var user User
	query := `SELECT * FROM users WHERE token = ?`
	err := DB.Get(&user, query, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by token: %w", err)
	}
	return &user, nil
}

// функция чисто для удобства, чтобы не слать 2 запроса к бд
func GetUserByTokenAndDevice(token string, staticFp string) (*User, error) {
	var user User
	query := `SELECT u.* FROM users u INNER JOIN devices d on d.userId = u.userId WHERE u.token = ? AND d.staticFp = ?`
	err := DB.Get(&user, query, token, staticFp)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by token: %w", err)
	}
	return &user, nil
}

func GetUserByEmail(email string) (*User, error) {
	var user User
	query := `SELECT * FROM users WHERE mail = ?`
	err := DB.Get(&user, query, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by mail: %w", err)
	}
	return &user, nil
}

func GetUserByUsername(username string) (*User, error) {
	var user User
	query := `SELECT * FROM users WHERE username = ?`
	err := DB.Get(&user, query, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}
	return &user, nil
}

func GetUserById(userId int) (*User, error) {
	var user User
	query := `SELECT * FROM users WHERE userId = ?`
	err := DB.Get(&user, query, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by userId: %w", err)
	}
	return &user, nil
}

func UserHasUsed(email, username string) (bool, error) {
	var check bool
	query := `SELECT EXISTS (SELECT 1 FROM users WHERE mail = ? OR username = ?)`
	err := DB.Get(&check, query, email, username)
	if err != nil {
		return false, fmt.Errorf("failed to check user email: %w", err)
	}
	return check, nil
}

func NewUserToken(username string, password string, email string, activated string, token string, priority int) (*User, error) {
	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	query := `
		INSERT INTO users
			(username, password, mail, code, token, priority)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := DB.Exec(
		query,
		username,
		string(passwordHash),
		email,
		activated,
		token,
		priority,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get created user id: %w", err)
	}

	return &User{
		UserId:   int(userID),
		Username: username,
		Password: string(passwordHash),
		Mail:     email,
		Code:     activated,
		Token:    token,
		Priority: priority,
	}, nil
}

func ValidateEmail(email string) bool {
	email = strings.TrimSpace(email)
	if email == "" || strings.ContainsAny(email, "\r\n") {
		return false
	}
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}
	return addr.Address == email
}

// #endregion
// #region devices
type Device struct {
	ID        int    `db:"ID"`
	UserID    int    `db:"userId"`
	UserAgent string `db:"userAgent"`
	IP        string `db:"ip"`
	Country   string `db:"country"`
	City      string `db:"city"`
	Platform  string `db:"platform"`
	Browser   string `db:"browser"`
	StaticFP  string `db:"staticFp"`
	DynamicFP string `db:"dynamicFp"`
	AddDate   int    `db:"addDate"`
	LastJoin  int    `db:"lastJoin"`
}

func (d Device) Render() map[string]any {
	return map[string]any{
		"userAgent": d.UserAgent,
		"country":   d.Country,
		"city":      d.City,
		"platform":  d.Platform,
		"browser":   d.Browser,
	}
}

func GetDeviceByToken(token string) (*Device, error) {
	var devices Device
	query := "SELECT * FROM devices WHERE staticFp = ?"
	err := DB.Get(&devices, query, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by token: %w", err)
	}
	return &devices, err
}

func (u *User) GetDevices() ([]Device, error) {
	var devices []Device
	query := "SELECT * FROM `devices` WHERE userId = ?"
	err := DB.Select(&devices, query, u.UserId)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by token: %w", err)
	}
	return devices, err
}

func (u *User) CheckDevice(UAgent, country, city, staticFp string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM devices WHERE userId = ? AND userAgent = ? AND country = ? AND city = ? AND staticFp = ?)`
	//
	err := DB.Get(&exists, query, u.UserId, UAgent, country, city, staticFp)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func EasyCheckDevice(userId int, staticFp string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM devices WHERE userId = ? AND staticFp = ?)`
	//
	err := DB.Get(&exists, query, userId, staticFp)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (u *User) AddDevice(UAgent, IP, country, city, platform, browser, staticFp, dynamicFp string) error {
	query := `INSERT INTO devices (userId, userAgent, ip, country, city, platform, browser, staticFp, dynamicFp) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := DB.Exec(
		query,
		u.UserId,
		UAgent,
		IP,
		country,
		city,
		platform,
		browser,
		staticFp,
		dynamicFp,
	)
	return err
}

func (u *User) RemoveDevice(UAgent, IP, country, city string) error {
	query := `DELETE FROM devices WHERE userId = ? AND userAgent = ? AND ip = ? AND country = ? AND city = ?`
	result, err := DB.Exec(
		query,
		u.UserId,
		UAgent,
		IP,
		country,
		city,
	)
	if err != nil {
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("device not found")
	}
	return nil
}

func (u *User) RemoveDeviceById(deviceId int) (bool, error) {
	query := `DELETE FROM devices WHERE userId = ? AND ID = ?`
	result, err := DB.Exec(
		query,
		u.UserId,
		deviceId,
	)
	if err != nil {
		return false, err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return false, fmt.Errorf("device not found")
	}
	return rowsAffected > 0, nil
}

func InsertDevice(
	user *User,
	ip string,
	country string,
	city string,
	userAgent string,
	staticFp string,
	dynamicFp string,
) error {
	exists, err := user.CheckDevice(
		userAgent,
		country,
		city,
		staticFp,
	)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	ua := useragent.Parse(userAgent)
	return user.AddDevice(
		userAgent,
		ip,
		country,
		city,
		ua.OS+" "+ua.OSVersion,
		ua.Name+" "+ua.Version,
		staticFp,
		dynamicFp,
	)
}

// #endregion
// #region security
func PasswordHash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func PasswordVerify(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func reCaptcha(token string) (bool, error) {
	return true, nil
	resp, err := http.PostForm("https://www.google.com/recaptcha/api/siteverify",
		map[string][]string{
			"secret":   {os.Getenv("RECAPTCHA")},
			"response": {token},
		})
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	//
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	//
	var result struct {
		Success bool `json:"success"`
	}
	//
	if err := json.Unmarshal(body, &result); err != nil {
		return false, err
	}
	//
	return result.Success, nil
}

const randomCharset = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandomString(length int) (string, error) {
	if length <= 0 {
		return "", nil
	}

	result := make([]byte, length)

	for i := range result {
		n, err := rand.Int(
			rand.Reader,
			big.NewInt(int64(len(randomCharset))),
		)
		if err != nil {
			return "", fmt.Errorf("failed to generate random string: %w", err)
		}

		result[i] = randomCharset[n.Int64()]
	}

	return string(result), nil
}

func GenerateUserToken(username string) (string, error) {
	randomString, err := RandomString(16)
	if err != nil {
		return "", fmt.Errorf("failed to generate token entropy: %w", err)
	}

	hash := sha256.Sum256([]byte(randomString + username))
	return hex.EncodeToString(hash[:]), nil
}

func GenerateUserVerifyCode() (string, error) {
	const max = uint32(1_000_000)

	limit := math.MaxUint32 - (math.MaxUint32 % max)

	var buf [4]byte

	for {
		if _, err := rand.Read(buf[:]); err != nil {
			return "", err
		}

		n := binary.BigEndian.Uint32(buf[:])

		if n < limit {
			return fmt.Sprintf("%06d", n%max), nil
		}
	}
}

// #endregion
