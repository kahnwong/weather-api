# qrcode-api

So I don't have to fish out my phone to scan event entry qrcode.

## Notes

- PNG should be around 4KB, otherwise garmin sdk might bork.

## Usage

```bash
docker build -t qrcode-api .
docker run \
  -p 3000:3000 \
  -v $(pwd)/data:/opt/data \
  -e LISTEN_ADDR=:3000 \
  qrcode-api
```
