#!/bin/bash
export HURL_ID=1
export HURL_NAME="Foo"
export HURL_BASE64_IMAGE=$(base64 -i ./images/qrcode.png)
hurl hurl/add-post.hurl
