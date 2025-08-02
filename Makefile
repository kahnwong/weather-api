add-post:
	./scripts/add.sh
title-get:
	hurl hurl/title-get.hurl
image-get:
	hurl hurl/image-get.hurl --output tmp/qrcode.png
