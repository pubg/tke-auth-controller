FROM golang:1.17-alpine3.13 as BUILD

WORKDIR /root

RUN apk add --no-cache git

COPY . .

RUN go mod download

RUN GOOS=linux GOARCH=amd64 go build -o tke-auth-controller

FROM ubuntu:focal

RUN useradd -ms /bin/bash controller
USER controller
WORKDIR /home/controller

COPY --from=BUILD /root/tke-auth-controller /home/controller/tke-auth-controller
