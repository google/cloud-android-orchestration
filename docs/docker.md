# Activate cloud orchestrator in the remote server

This page describes how to run cloud orchestrator managing docker instances
containing Host Orchestrator inside.

## Prepare docker image

Please follow the docker part of
[README.md](https://github.com/google/android-cuttlefish/blob/main/README.md#docker)
in `google/android-cuttlefish` github repository, and check if the docker image
`cuttlefish-orchestration` exists.

## Building cuttlefish-cloud-orchestration

Docker image `cuttlefish-cloud-orchestration` runs cloud orchestrator inside
container. To build `cuttlefish-cloud-orchestration`, please run
`scripts/docker/image-builder.sh`.

## Running cloud orchestrator with cuttlefish-cloud-orchestration

Please run `scripts/docker/image-runner.sh` to start 
`cuttlefish-cloud-orchestration` docker instance.

If there's a firewall which blocks accessing cloud orchestrator with HTTP/HTTPS
requests, please try using SOCKS5 proxy. Establishing SOCKS5 proxy by creating
SSH dynamic port forwarding is available with following command.
```bash
ssh -D ${SOCKS5_PORT} -q -C -N ${USERNAME}@${CLOUD_ORCHESTRATOR_IPv4_ADDRESS}
```

## Use cloud orchestrator by cvdr

Please check every configuration in `scripts/docker/cvdr.toml` is set well, and
follow the steps at [cvdr.md](cvdr.md).
