FROM golang:1.25-alpine AS build

WORKDIR /build


COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./

RUN CGO_ENABLED=0 go build -ldflags "-w -s" -o /weather-api

# hadolint ignore=DL3007
FROM alpine:latest AS deploy

# hadolint ignore=DL3045
COPY --from=build /weather-api /

EXPOSE 3000
CMD ["/weather-api"]
