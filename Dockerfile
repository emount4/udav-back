FROM golang:1.26-alpine3.24 AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN CGO_ENABLED=0 go build -trimpath -o /out/udav-api ./cmd/api \
    && CGO_ENABLED=0 go build -trimpath -o /out/udav-migrate ./cmd/migrate

FROM alpine:3.24 AS runtime

RUN addgroup -S udav \
    && adduser -S -G udav udav \
    && mkdir -p /var/lib/udav/logs \
    && chown udav:udav /var/lib/udav /var/lib/udav/logs

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
WORKDIR /var/lib/udav
USER udav:udav

FROM runtime AS migrate

COPY --from=build /out/udav-migrate /usr/local/bin/udav-migrate
COPY migrations/ ./migrations/
ENTRYPOINT ["/usr/local/bin/udav-migrate"]

FROM runtime AS api

COPY --from=build /out/udav-api /usr/local/bin/udav-api
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/udav-api"]
