# Dockerfile References: https://docs.docker.com/engine/reference/builder/

ARG GOLANG_VERSION='1.26.1'

FROM golang:${GOLANG_VERSION}-alpine as builder

RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh

WORKDIR /app

COPY ./go.mod ./go.sum ./

RUN go mod download

COPY ./ ./
COPY ./.env .env

RUN go build -o main ./cmd/main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder . .

EXPOSE 8080

CMD ["./app/main"]
