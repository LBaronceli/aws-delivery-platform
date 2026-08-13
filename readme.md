# AWS Delivery Platform

A small delivery-tracking platform for learning AWS networking, ECS/Fargate,
queues, events, authentication, and deployment.

## Run the API locally

The API requires Go 1.26 or later:

```bash
cd app
go run ./cmd/api
```

It listens on port `8080` by default. Set `PORT` to use a different port.

```bash
curl http://localhost:8080/health
curl http://localhost:8080/hello
```

## Run with Docker

From the repository root:

```bash
docker build -t delivery-api .
docker run --rm -p 8080:8080 delivery-api
curl http://localhost:8080/health
```

The health response is:

```json
{"status":"ok"}
```

## Test

```bash
cd app
go test ./...
```
