# Activate cloud orchestrator in the remote server

This page describes how to run cloud orchestrator managing docker instances
containing Host Orchestrator inside.

## Prepare docker image

Please follow the docker part of
[README.md](https://github.com/google/android-cuttlefish/blob/main/README.md#docker)
in `google/android-cuttlefish` github repository, and check if the docker image
`cuttlefish-orchestration` exists.

## Build cloud orchestrator

To build cloud orchestrator, please execute:
```bash
git clone https://github.com/google/cloud-android-orchestration.git
cd cloud-android-orchestration # Root directory of this repository
go build ./cmd/cloud_orchestrator
```

If build was successful, you would be able to see `cloud_orchestrator` binary at
the root directory of this repository.

## Run cloud orchestrator

To run cloud orchestrator, please execute:
```bash
# In the root directory of this repository
scripts/docker/run.sh
```

<!--
TODO(denniscy1993): Update this section after modifying AccountManager
information of the config file.
-->
If the address of cloud orchestrator is not `localhost`, please modify the
return value of `ChooseNetworkInterface` in `cmd/cloud_orchestrator/main.go` to
the empty string. Note that this is a temporary solution.

<!--
TODO(0405ysj): Update this section after renaming the flag name from
`http_proxy` into `proxy`.
-->
If there's a firewall which blocks accessing cloud orchestrator with HTTP/HTTPS
requests, please try using SOCKS5 proxy. Establishing SOCKS5 proxy by creating
SSH dynamic port forwarding is available with following command.
```bash
ssh -D ${SOCKS5_PORT} -q -C -N ${USERNAME}@${CLOUD_ORCHESTRATOR_IPv4_ADDRESS}
```
Besides, currently `cvdr` supports using SOCKS5 proxy with the flag like
`--http_proxy=socks5://localhost:${SOCKS5_PORT}` for all subcommands.

## Use cloud orchestrator by cvdr

Please follow [cvdr.md](cvdr.md). The URL of running cloud orchestrator should
be `http://${CLOUD_ORCHESTRATOR_IP_ADDRESS}:8080`.
