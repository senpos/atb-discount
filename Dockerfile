FROM golang:1.19-alpine as build

ENV CGO_ENABLED=0

RUN : \
    && apk add --no-cache --update \
        tzdata \
        ca-certificates \
        dumb-init\
    && rm -rf /var/cache/apk/*

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN go build -ldflags="-w -s" -o app

FROM scratch

COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /usr/bin/dumb-init /usr/bin/dumb-init
COPY --from=build /build/app /srv/app

USER 1001

ENTRYPOINT ["/usr/bin/dumb-init", "--"]
CMD ["/srv/app"]