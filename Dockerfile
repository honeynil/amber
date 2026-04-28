FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /amber ./cmd/amber

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -h /data amber

COPY --from=builder /amber /usr/local/bin/amber

USER amber
WORKDIR /data
VOLUME /data

EXPOSE 8080 4317

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s \
    CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["amber"]
CMD ["config.yaml"]
