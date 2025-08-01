package main

var tableSchemas = map[string]string{
	"qrcode": `
	CREATE TABLE qrcode (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT,
    image BLOB
);`,
}

var allExpectedColumns = map[string]map[string]string{
	"qrcode": {
		"id":    "INTEGER",
		"title": "TEXT",
		"image": "BLOB",
	},
}
