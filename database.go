package main

import (
	"context"
	"fmt"
	"log"
	"net/netip"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/oschwald/geoip2-golang/v2"
)

var DB *sqlx.DB

func InitDB() {
	user := getEnv("SQL_USER", "root")
	password := getEnv("SQL_PASSWD", "gdpshelper")
	host := getEnv("SQL_HOST", "localhost")
	port := getEnv("SQL_PORT", "3306")
	database := getEnv("SQL_DB", "cfmotru_ojhub")
	//
	log.Printf("> Connecting to DB: user=%s, host=%s, port=%s, DB=%s",
		user, host, port, database)
	//
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=Local&timeout=30s&interpolateParams=true",
		user, password, host, port, database)
	//
	var err error
	//
	DB, err = sqlx.Open("mysql", dsn)
	if err != nil {
		DB.Close()
		log.Fatalf("Failed to open database: %v", err)
	}
	//
	DB.SetConnMaxLifetime(15 * time.Minute)
	DB.SetConnMaxIdleTime(5 * time.Minute)
	DB.SetMaxOpenConns(60)
	DB.SetMaxIdleConns(10)
	//
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	//
	if err := DB.PingContext(ctx); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	//
	log.Println("> Successfully connected to MySQL database")
}

var GeoDb *geoip2.Reader

func InitGeoDb() {
	var err error
	GeoDb, err = geoip2.Open("GeoLite2-City.mmdb")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
}

func GetCity(preIp string) ([2]string, error) {
	if preIp == "127.0.0.1" {
		return [2]string{"Localhost", "CurrentPC"}, nil
	}
	ip, err := netip.ParseAddr(preIp)
	if err != nil {
		log.Printf("Warning: failed to parse ip: %v", err)
		return [2]string{"Unknown", "Unknown"}, err
	}
	record, err := GeoDb.City(ip)
	if err != nil {
		log.Printf("Warning: failed to scan ip: %v", err)
		return [2]string{"Unknown", "Unknown"}, err
	}
	if !record.HasData() {
		fmt.Println("No data found for this IP")
		return [2]string{"Unknown", "Unknown"}, nil
	}
	return [2]string{record.Country.Names.English, record.City.Names.English}, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
