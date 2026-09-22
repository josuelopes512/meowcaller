# Builds examples/web (the browser call console) as a standalone image.
# Both audio and video are bridged through the browser (WebCodecs + Web Audio), so the
# binary itself never opens a local mic/speaker/camera and needs no CGO.
FROM golang:1.25-bookworm AS builder

WORKDIR /src
COPY . .

WORKDIR /src/examples/web
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/web_console .

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
