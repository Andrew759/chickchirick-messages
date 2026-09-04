ARG GOLANG_VERSION='1.26.1'

FROM golang:${GOLANG_VERSION}-alpine

RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh build-base && \
    go install github.com/go-delve/delve/cmd/dlv@master

WORKDIR /app

COPY ./go.mod ./go.sum ./

RUN go mod download

COPY ./ ./

COPY main /main
RUN chmod +x /main

#RUN go build -gcflags="all=-N -l" -o /main ./cmd/main.go

EXPOSE 8080 40000

CMD ["dlv", "--listen=:40000", "--headless=true", "--api-version=2", "--accept-multiclient", "exec", "/main", "--continue"]