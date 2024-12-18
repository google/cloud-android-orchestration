# Activate cloud orchestrator in the remote server

This page describes how to run cloud orchestrator managing docker instances
containing Host Orchestrator inside.

## Prepare docker image

Please follow the docker part of
[README.md](https://github.com/google/android-cuttlefish/blob/main/README.md#docker)
in `google/android-cuttlefish` github repository, and check if the docker image
`cuttlefish-orchestration` exists.

## Build and run cloud orchestrator

Config file for the cloud orchestrator is at
[scripts/on-premises/single-server/conf.toml](conf.toml). Follow the steps at
[cloud_orchestrator.md](/docs/cloud_orchestrator.md).

If there's a firewall which blocks accessing cloud orchestrator with HTTP/HTTPS
requests, please try using SOCKS5 proxy. Establishing SOCKS5 proxy by creating
SSH dynamic port forwarding is available with following command.
```bash
ssh -D ${SOCKS5_PORT} -q -C -N ${USERNAME}@${CLOUD_ORCHESTRATOR_IPv4_ADDRESS}
```

## Use cloud orchestrator by cvdr

Please check every configuration in
`scripts/on-premises/single-server/cvdr.toml` is set well, and follow the steps
at [cvdr.md](/docs/cvdr.md).
