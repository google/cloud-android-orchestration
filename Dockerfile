FROM golang:1.22-alpine as builder
RUN apk add --no-cache git
WORKDIR /go/src/github.com/google/cloud-android-orchestrator
COPY . .
RUN GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build \
      -trimpath \
      -o /app \
      cmd/cloud_orchestrator/main.go

FROM gcr.io/distroless/base
COPY --from=builder /app /app
ADD web/intercept /web/intercept
CMD ["/app"]
