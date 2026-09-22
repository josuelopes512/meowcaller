# Builds examples/web (the browser call console) as a standalone image.
# CGO is required: audio/malgo (miniaudio bindings) links via cgo even though
# mic/speaker devices won't be present in this container - see README-hosting.md.
FROM golang:1.25-bookworm AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src
COPY . .

WORKDIR /src/examples/web
RUN go mod download
RUN CGO_ENABLED=1 GOOS=linux go build -trimpath -o /out/web_console .

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN useradd --system --create-home --home-dir /data meowcaller
WORKDIR /data
COPY --from=builder /out/web_console /usr/local/bin/web_console

USER meowcaller
ENV LISTEN_ADDR=0.0.0.0:8080
EXPOSE 8080
VOLUME ["/data"]

ENTRYPOINT ["/usr/local/bin/web_console"]
