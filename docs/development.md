# Building

From the top directory run:

```
go build ./... && go test ./...
```

To build the the executables run:

```
go build ./cmd/<executablename>
```

Where `<executablename>` is one of `cvdr` or `cloud_orchestrator`

# Running

Running the cloud orchestrator is as easy as executing the `cloud_orchestrator` binary with no
arguments. It will read the *conf.toml* configuration file and listen on port 8080 for plain HTTP
requests. The provided configuration file uses *fakes* and *stubs* for every module that requires a
production environment.

In development, the cloud orchestrator is configured to interact with a single host orchestrator
listening on 1081.

## The host orchestrator

> WARNING: The host orchestrator is developed in a different repository, so the following
instructions may become obsolete without notice.

1. Get The source code
    ```
    git clone https://github.com/google/android-cuttlefish
    ```

2. Build the host orchestrator binary:
    ```
    cd frontend/src/host_orchestrator
    go build ./...
    ```

3. Run the host orchestrator:

    ```
    # This is the user the host orchestrator will use to execute cvd as.
    sudo adduser _cvd-executor
    HO_RUN_DIR=/tmp/host_orchestrator_runtime
    mkdir $HO_RUN_DIR

    ORCHESTRATOR_SOCKET_PATH=$HO_RUN_DIR/operator \
    ORCHESTRATOR_HTTP_PORT=1081 \
    ORCHESTRATOR_HTTPS_PORT=1444 \
    ORCHESTRATOR_CVD_ARTIFACTS_DIR=$HO_RUN_DIR \
    ./host_orchestrator
    ```
