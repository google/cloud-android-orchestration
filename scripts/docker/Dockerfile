FROM debian:12 AS cuttlefish-cloud-orchestration

RUN set -x
RUN apt update
RUN apt install -y --no-install-recommends \
    ca-certificates \
    docker.io \
    golang \
    sudo
RUN update-ca-certificates

COPY . /root/cloud-android-orchestration
WORKDIR /root/cloud-android-orchestration

RUN go build ./cmd/cloud_orchestrator

# Expose Cloud Orchestrator Port
EXPOSE 8080

ENTRYPOINT ["/bin/bash", "-c", "./scripts/docker/guest/cloud-orchestrator-runner.sh"]
