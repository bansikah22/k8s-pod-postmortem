# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app


RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /postmortem ./cmd/postmortem

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -g 1000 postmortem && \
    adduser -u 1000 -G postmortem -s /bin/sh -D postmortem

WORKDIR /app

COPY --from=builder /postmortem /app/postmortem

RUN chown -R postmortem:postmortem /app

USER postmortem

ENTRYPOINT ["/app/postmortem"]