# precipitation-api

Because I want to check the weather real quick before heading out. To be used with a garmin watch.

## Usage

```bash
docker build -t precipitation-api .
docker run \
  -p 3000:3000 \
  --env-file .env \
  -e LISTEN_ADDR=:3000 \
  precipitationgs-api
```
