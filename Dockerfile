FROM golang:1.7rc6-alpine as BUILD

WORKDIR /root

RUN apk add --no-cache git

COPY . .

RUN go get

RUN GOOS=linux GOARCH=386 go build -o tke-auth-controller

FROM ubuntu:focal

RUN useradd -ms /bin/bash controller
USER controller
WORKDIR /home/controller

COPY --from=BUILD /root/tke-auth-controller /home/controller/tke-auth-controller
