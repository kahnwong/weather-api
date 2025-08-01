FROM golang:1.24-alpine AS build

WORKDIR /build

RUN apk add --no-cache build-base

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./

RUN CGO_ENABLED=1 go build -ldflags "-w -s" -o /qrcode-api

# hadolint ignore=DL3007
FROM alpine:latest AS deploy

# hadolint ignore=DL3045
COPY --from=build /qrcode-api /

EXPOSE 3000
CMD ["/qrcode-api"]
