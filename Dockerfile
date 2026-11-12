FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -trimpath -o /out/litloop-go-server ./cmd/litloop-go-server && \
    CGO_ENABLED=1 go build -trimpath -o /out/worker ./cmd/worker && \
    CGO_ENABLED=1 go build -trimpath -o /out/migrate ./cmd/migrate && \
    CGO_ENABLED=1 go build -trimpath -o /out/admin ./cmd/admin

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates ffmpeg && \
    rm -rf /var/lib/apt/lists/* && useradd --uid 10001 --create-home litloop
WORKDIR /app
COPY --from=build /out/ /usr/local/bin/
RUN mkdir /app/media && chown -R litloop:litloop /app
USER litloop
EXPOSE 8000
ENTRYPOINT ["litloop-go-server"]
