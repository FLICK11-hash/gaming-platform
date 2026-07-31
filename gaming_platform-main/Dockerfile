FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

FROM alpine:3.20
# stockfish lives in Alpine's community repo, not main - explicit
# --repository covers Alpine base images where community isn't enabled by
# default. See internal/domain/bot.ConfigureStockfish: if this ever fails
# to install, the server still starts fine and real-match bots fall back
# to their built-in move logic instead of a real engine.
RUN apk add --no-cache ca-certificates \
    && apk add --no-cache --repository=https://dl-cdn.alpinelinux.org/alpine/v3.20/community stockfish
WORKDIR /app
COPY --from=build /server ./server
COPY --from=build /app/internal/db/migrations ./internal/db/migrations

EXPOSE 8080
CMD ["./server"]
