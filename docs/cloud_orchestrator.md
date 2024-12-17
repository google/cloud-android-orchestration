# Cloud Orchestrator

This page describes about `Cloud Orchestrator`.

## What's Cloud Orchestrator?

Cloud Orchestrator is a web service for managing virtual machines or containers
as host instances.
It can create, list, and delete host instances, and provides the way to access
hosts to launch Cuttlefish instances on top of the host instance via Host
Orchestrator.

## Build Cloud Orchestrator

Please execute the command below.
```bash
git clone https://github.com/google/cloud-android-orchestration.git
cd cloud-android-orchestration # Root directory of this repository
docker build \
    --force-rm \
    --no-cache \
    -t cuttlefish-cloud-orchestrator \
    .
```

## Run Cloud Orchestrator

To run cloud orchestrator, prepare your config file(conf.toml) for the cloud
orchestrator and please execute the command below.
```bash
docker run \
    -p 8080:8080 \
    -e CONFIG_FILE="/conf.toml" \
    -v $CLOUD_ORCHESTRATOR_CONFIG_PATH:/conf.toml \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -t cuttlefish-cloud-orchestrator:latest
```

## Use Cloud Orchestrator

We're currently providing using Cloud Orchestrator with Docker instances as
hosts. Please read
[scripts/on-premises/single-server/README.md](/scripts/on-premises/single-server/README.md)
to follow.
<!-- TODO(ser-io): Write how to use CO for GCP. -->
