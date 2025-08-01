add:
	./scripts/add.sh
get-title:
	hurl hurl/title-get.hurl
get-png:
	hurl hurl/png-get.hurl --output tmp/qrcode.png
