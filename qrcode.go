package main

import (
	"os"

	"github.com/jmoiron/sqlx"
)

var dbFileName string
var Qrcode *Application

type Application struct {
	DB *sqlx.DB
}

func init() {
	if os.Getenv("MODE") != "DEVELOPMENT" {
		dbFileName = "/data/qrcode.sqlite"
	} else {
		dbFileName = "./qrcode.sqlite"
	}

	// init app
	dbExists := isDBExists()
	Qrcode = &Application{
		DB: initDB(),
	}
	Qrcode.InitSchema(dbExists)
}
