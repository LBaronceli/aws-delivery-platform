FROM golang:1.26-alpine AS build

WORKDIR /src/app

COPY app/go.mod ./
RUN go mod download

COPY app/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/delivery-api ./cmd/api

FROM alpine:3.23

RUN addgroup -S app && adduser -S -G app app

COPY --from=build /out/delivery-api /usr/local/bin/delivery-api

ENV PORT=8080
EXPOSE 8080

USER app

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q -O - "http://127.0.0.1:${PORT}/health" > /dev/null || exit 1

ENTRYPOINT ["/usr/local/bin/delivery-api"]
