# E2E test of cvdr

## Prerequisite

- Test runner should run Cloud Orchestrator on the machine where testing machine
can access.
- Test runner should prepare `cvdr` runnable in the testing machine.
- Test runner should prepare the configuration file of `cvdr` including
accessible URL of Cloud Orchestrator.

## Test procedure

```
cd /path/to/cloud-android-orchestration
cd e2etests
bazel test //cvdr/... \
  --test_env=CVDR_PATH=${CVDR_PATH} \
  --test_env=CVDR_CONFIG_PATH=${CVDR_CONFIG_PATH}
```
