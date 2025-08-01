#!/bin/bash
export HURL_BASE64_IMAGE=$(base64 -i ./images/qrcode.png)
hurl hurl/add-post.hurl
