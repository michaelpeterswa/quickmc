# -=-=-=-=-=-=- Compile Image -=-=-=-=-=-=-

FROM golang:1 AS stage-compile

WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./... && CGO_ENABLED=0 GOOS=linux go build ./cmd/quickmc

# -=-=-=-=- Final Debian 11 Slim Image -=-=-=-=-

FROM debian:bullseye as stage-final

COPY --from=stage-compile /go/src/app/quickmc /
COPY ./server.properties /

RUN apt-get update --fix-missing && \
    apt-get install -yqq --no-install-recommends \
    ca-certificates \
    curl \
    tzdata \
    openjdk-17-jre-headless \
    && \
    apt-get autoclean -yqq && \
    apt-get clean -yqq

CMD ["/quickmc"]