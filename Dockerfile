FROM golang:1.22-alpine as builder
RUN apk add --no-cache git
WORKDIR /go/src/github.com/google/cloud-android-orchestration
COPY . .
RUN GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build \
      -trimpath \
      -o /cloud_orchestrator \
      cmd/cloud_orchestrator/main.go

FROM gcr.io/distroless/base as runner-base
COPY --from=builder /cloud_orchestrator /cloud_orchestrator
ADD web/intercept /web/intercept

FROM runner-base as runner-gcp
CMD ["/cloud_orchestrator"]

FROM runner-base as runner-docker
ENV CONFIG_FILE /config/conf.toml
ADD scripts/docker/conf.toml $CONFIG_FILE
CMD ["/cloud_orchestrator"]
