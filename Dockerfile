# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install dependencies first for better caching
RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build with TARGETOS and TARGETARCH for multi-platform support
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-w -s" -o /postmortem ./cmd/postmortem

FROM alpine:3.23

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -g 1000 postmortem && \
    adduser -u 1000 -G postmortem -s /bin/sh -D postmortem

WORKDIR /app

COPY --from=builder /postmortem /app/postmortem

RUN chown -R postmortem:postmortem /app

USER postmortem

ENTRYPOINT ["/app/postmortem"]