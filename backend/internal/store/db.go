package store

import (
	"fmt"
	"os"

	"github.com/andreka2000000/Smart_dispatching_of_field_brigades/backend/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}
	user := os.Getenv("DB_USER")
	if user == "" {
		user = "user"
	}
	pass := os.Getenv("DB_PASSWORD")
	if pass == "" {
		pass = "password"
	}
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "rotelcom"
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=5432 sslmode=disable",
		host, user, pass, dbname)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("Ошибка подключения к БД: " + err.Error())
	}

	err = DB.AutoMigrate(
		&model.ServiceCenter{},
		&model.AccessNode{},
		&model.WorkType{},
		&model.Staff{},
		&model.Order{},
		&model.FaultGroup{},
		&model.Plan{},
		&model.Route{},
		&model.Visit{},
	)
	if err != nil {
		panic("Ошибка миграции: " + err.Error())
	}

	fmt.Println("✅ БД подключена и мигрирована.")
}
