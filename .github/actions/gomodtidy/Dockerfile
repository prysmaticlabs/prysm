FROM golang:alpine

RUN apk update && apk install git

COPY entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
