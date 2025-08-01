# qrcode-api

TODO

## Usage

```bash
docker build -t qrcode-api .
docker run \
  -p 3000:3000 \
  -v $(pwd)/data:/opt/data \
  -e LISTEN_ADDR=:3000 \
  qrcode-api
```
