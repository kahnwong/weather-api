# qrcode-api

So I don't have to fish out my phone to scan event entry qrcode.


## Usage

```bash
docker build -t qrcode-api .
docker run \
  -p 3000:3000 \
  --env-file .env \
  -v $(pwd)/data:/data \
  -e LISTEN_ADDR=:3000 \
  qrcode-api
```
