FROM golang:1.26.10-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates make

COPY go.mod ./
RUN go mod download

COPY . .
RUN make generate && go mod tidy
RUN go test ./...
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /marketplace ./cmd/api

FROM alpine:3.22

RUN apk add --no-cache ca-certificates wget \
    && addgroup -S app \
    && adduser -S app -G app

WORKDIR /app
COPY --from=builder /marketplace /app/marketplace

USER app
EXPOSE 8080

CMD ["/app/marketplace"]
