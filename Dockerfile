FROM golang:1.17-alpine3.13 as BUILD

WORKDIR /root

RUN apk add --no-cache git

COPY . .

RUN go mod download

# CGO_ENABLED=0 set compiler links library static
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o tke-auth-controller

FROM ubuntu:focal

# tencent client requires curl to run
RUN apt-get update -y && apt-get install -y ca-certificates
RUN useradd -ms /bin/bash controller

COPY --from=BUILD /root/tke-auth-controller /home/controller/tke-auth-controller

# requires when running docker build on windows(WSL) machine.
RUN chmod 777 /home/controller/tke-auth-controller && chown controller:controller /home/controller/tke-auth-controller

# prevent executable grant previlege
USER controller

ENTRYPOINT ["/home/controller/tke-auth-controller"]
