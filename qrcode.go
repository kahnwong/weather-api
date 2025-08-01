package main

import (
	"fmt"
	"os"

	"github.com/jmoiron/sqlx"
)

var dbFileName string
var Qrcode *Application

type Application struct {
	DB *sqlx.DB
}

type QrcodeItem struct {
	ID    int    `db:"id"`
	Name  string `db:"name"` // 0 for false, 1 for true
	Image []byte `db:"image"`
}

func (Qrcode *Application) Add(qrcode QrcodeItem) error {
	query := `INSERT OR REPLACE INTO qrcode (id, name, image) VALUES (?, ?, ?)`
	_, err := Qrcode.DB.Exec(query, qrcode.ID, qrcode.Name, qrcode.Image)
	if err != nil {
		return fmt.Errorf("error inserting activity for qrcode: '%s' - %w", qrcode.Name, err)
	}

	return nil
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

	// test
	filePath := "./assets/qrcode.png"
	imageData, _ := os.ReadFile(filePath)

	err := Qrcode.Add(QrcodeItem{
		ID:    1,
		Name:  "Foo",
		Image: imageData,
	})
	if err != nil {
		return
	}
}
