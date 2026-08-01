FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

FROM debian:bookworm-slim
# Alpine's own repos don't carry a stockfish package (confirmed against
# v3.20's main+community indexes), so the final image is Debian instead,
# which does. The package installs to /usr/games/stockfish - not on this
# base image's default PATH - so it's symlinked into /usr/local/bin, which
# is, letting bot.ConfigureStockfish's default STOCKFISH_PATH ("stockfish",
# resolved via PATH) find it with no extra env var needed. See
# internal/domain/bot.ConfigureStockfish: if this ever fails to install,
# the server still starts fine and real-match bots fall back to their
# built-in move logic instead of a real engine.
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates stockfish \
    && ln -s /usr/games/stockfish /usr/local/bin/stockfish \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=build /server ./server
COPY --from=build /app/internal/db/migrations ./internal/db/migrations

EXPOSE 8080
CMD ["./server"]
