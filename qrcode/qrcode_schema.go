package qrcode

var tableSchemas = map[string]string{
	"qrcode": `
	CREATE TABLE qrcode (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT,
    image BLOB
);`,
}

var allExpectedColumns = map[string]map[string]string{
	"qrcode": {
		"id":    "INTEGER",
		"name":  "TEXT",
		"image": "BLOB",
	},
}
