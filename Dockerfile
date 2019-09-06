FROM golang:alpine as builder
RUN mkdir /build
ADD . /build/
WORKDIR /build
RUN apk update && apk upgrade && \
    apk add --no-cache git
RUN go build -o web ./web_server/main.go

FROM alpine
RUN apk update \
        && apk upgrade \
        && apk add --no-cache \
        ca-certificates \
        && update-ca-certificates 2>/dev/null || true
COPY --from=builder /build/web /app/
COPY --from=builder /build/assets /app/assets
COPY --from=builder /build/raw_importable /app/raw_importable
WORKDIR /app

EXPOSE 1337

CMD ["./web"]