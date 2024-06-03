# Cloud Android Orchestration

This repo holds source for a Cuttlefish Cloud Orchestration application. The
Orchestration application allows to create, delete and connect to Cuttlefish
devices in the cloud.

The application is developed for and tested on Google Cloud Platform, but it's
designed in a way that should allow its deployment in other Cloud solutions
without much trouble.

## Links for each component

To know details, please refer to these documents below.

- [Cloud Orchestrator](docs/cloud_orchestrator.md): A web service for hosting
VMs or containers for running Cuttlefish instances on top of.
- [cvdr](docs/cvdr.md): CLI wrapper of Cloud Orchestrator providing
user-friendly interface.
- [Cuttlefish Web Launcher](web/page0/README.md): Navigator & web-based wrapper
of Cloud Orchestrator providing user-friendly interface.
