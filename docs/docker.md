# Activate cloud orchestrator in the remote server

This page describes how to run cloud orchestrator managing docker instances
containing Host Orchestrator inside.

## Prepare docker image

Please follow the docker part of
[README.md](https://github.com/google/android-cuttlefish/blob/main/README.md#docker)
in `google/android-cuttlefish` github repository, and check if the docker image
`cuttlefish-orchestration` exists.

## Build and run cloud orchestrator

To build and run cloud orchestrator, please execute below commands in the server
machine.
```bash
git clone https://github.com/google/cloud-android-orchestration.git
cd cloud-android-orchestration # Root directory of this repository
scripts/docker/run.sh
```

If the address of cloud orchestrator is not `localhost`, please modify the
return value of `ChooseNetworkInterface` in `cmd/cloud_orchestrator/main.go` to
the empty string. Note that this is a temporary solution.

If there's a firewall which blocks accessing cloud orchestrator with HTTP/HTTPS
requests, please try using SOCKS5 proxy. Establishing SOCKS5 proxy by creating
SSH dynamic port forwarding is available with following command.
```bash
ssh -D ${SOCKS5_PORT} -q -C -N ${USERNAME}@${CLOUD_ORCHESTRATOR_IPv4_ADDRESS}
```

## Use cloud orchestrator by cvdr

Please check every configuration in `scripts/docker/cvdr.toml` is set well, and
follow the steps at [cvdr.md](cvdr.md).
