FROM golang:1.17-alpine3.13 as BUILD

WORKDIR /root

RUN apk add --no-cache git

COPY . .

RUN go mod download

RUN GOOS=linux GOARCH=386 go build -o tke-auth-controller

FROM ubuntu:focal

RUN apt-get update -y
RUN useradd -ms /bin/bash controller

COPY --from=BUILD /root/tke-auth-controller /home/controller/tke-auth-controller

# requires when running docker build on windows(WSL) machine.
RUN chmod 777 /home/controller/tke-auth-controller && chown controller:controller /home/controller/tke-auth-controller

# prevent executable grant previlege
USER controller

ENTRYPOINT /home/controller/tke-auth-controller