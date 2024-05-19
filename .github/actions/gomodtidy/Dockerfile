FROM golang:1.21-alpine

COPY entrypoint.sh /entrypoint.sh

RUN adduser -D -g '' nonroot

USER nonroot

ENTRYPOINT ["/entrypoint.sh"]
