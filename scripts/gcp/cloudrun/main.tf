/**
 * Copyright 2024 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

locals {
  apis = toset([
    "artifactregistry.googleapis.com",
    "compute.googleapis.com",
    "iap.googleapis.com",
    "run.googleapis.com",
    "secretmanager.googleapis.com",
    "vpcaccess.googleapis.com",
  ])
}

provider "google" {
  project = var.project_id
}

provider "google-beta" {
  project = var.project_id
}

resource "google_project_service" "apis" {
  for_each = local.apis
  service  = each.key
}

resource "google_project_service_identity" "iap" {
  provider = google-beta

  project = data.google_project.project.project_id
  service = "iap.googleapis.com"
}

data "google_project" "project" {
}

resource "google_compute_project_metadata_item" "disable-oslogin" {
  key   = "enable-oslogin"
  value = "FALSE"
}

resource "google_artifact_registry_repository" "my-repo" {
  location      = var.region
  repository_id = var.artifact_repository_id
  format        = "DOCKER"
  depends_on = [
    google_project_service.apis
  ]
}

// Secret Manager
resource "google_secret_manager_secret" "co-config" {
  secret_id = "cloud-orchestrator-config"

  replication {
    auto {}
  }
  depends_on = [
    google_project_service.apis
  ]
}

resource "google_secret_manager_secret_version" "secret-version-basic" {
  secret = google_secret_manager_secret.co-config.id
  secret_data = templatefile("conf.toml.tmpl", {
    PROJECT_ID      = var.project_id,
    NETWORK         = google_compute_network.network.id,
    SUBNETWORK      = google_compute_subnetwork.subnetwork.id
    USE_PRIVATE_IPS = var.use_private_ips
  })
}

resource "google_iap_brand" "project_brand" {
  support_email     = var.oauth_support_email
  application_title = "Cloud Orchestrator"
  project           = var.project_id
  depends_on = [
    google_project_service.apis
  ]
}

resource "google_iap_client" "project_client" {
  display_name = "IAP-co-backend-service"
  brand        = google_iap_brand.project_brand.name
}

resource "google_iap_web_backend_service_iam_member" "sa_member" {
  project             = var.project_id
  web_backend_service = module.lb-http.backend_services.default.name
  role                = "roles/iap.httpsResourceAccessor"
  member              = google_service_account.service_account.member
}

resource "google_iap_web_backend_service_iam_member" "member" {
  for_each            = var.service_accessors
  project             = var.project_id
  web_backend_service = module.lb-http.backend_services.default.name
  role                = "roles/iap.httpsResourceAccessor"
  member              = each.key
}

module "lb-http" {
  source  = "terraform-google-modules/lb-http/google//modules/serverless_negs"
  version = "~> 11.0"

  name    = var.lb_name
  project = var.project_id

  load_balancing_scheme           = "EXTERNAL_MANAGED"
  ssl                             = true
  managed_ssl_certificate_domains = [var.domain]
  http_forward                    = false

  backends = {
    default = {
      protocol    = "HTTPS"
      description = null
      groups = [
        {
          group = google_compute_region_network_endpoint_group.serverless_neg.id
        }
      ]
      enable_cdn = false

      iap_config = {
        enable               = true
        oauth2_client_id     = google_iap_client.project_client.client_id
        oauth2_client_secret = google_iap_client.project_client.secret
      }
      log_config = {
        enable = false
      }
    }
  }
  depends_on = [
    google_project_service.apis
  ]
}

resource "google_compute_region_network_endpoint_group" "serverless_neg" {
  provider              = google-beta
  name                  = "serverless-neg"
  network_endpoint_type = "SERVERLESS"
  region                = var.region
  cloud_run {
    service = var.cloud_run_name
  }
  depends_on = [
    google_project_service.apis
  ]
}

resource "google_service_account" "service_account" {
  account_id   = "cloud-orchestrator"
  display_name = "Service Account used for Cloud Orchestrator"
  depends_on = [
    google_project_service.apis
  ]
}

resource "google_service_account_iam_member" "token_creators" {
  for_each           = var.service_accessors
  service_account_id = google_service_account.service_account.id
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = each.key
}

resource "google_secret_manager_secret_iam_member" "member" {
  secret_id = google_secret_manager_secret.co-config.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = google_service_account.service_account.member
}

resource "google_project_iam_member" "member" {
  project = var.project_id
  role    = "roles/compute.admin"
  member  = google_service_account.service_account.member
}

# Networking
resource "google_compute_network" "network" {
  name                    = "co-network"
  auto_create_subnetworks = false
  depends_on = [
    google_project_service.apis
  ]
}

resource "google_compute_subnetwork" "subnetwork" {
  name          = "test-subnetwork"
  ip_cidr_range = "10.2.0.0/16"
  region        = var.region
  network       = google_compute_network.network.id
}

resource "google_vpc_access_connector" "connector" {
  region        = var.region
  name          = var.serverless_connector_name
  ip_cidr_range = "10.8.0.0/28"
  network       = google_compute_network.network.id
}

module "cloud-nat" {
  for_each      = var.use_private_ips ? toset(["1"]) : toset([])
  source        = "terraform-google-modules/cloud-nat/google"
  version       = "~> 5.0"
  project_id    = var.project_id
  region        = var.region
  router        = "cloud-nat-${var.region}"
  create_router = true
  network       = google_compute_network.network.id
}

# Firewall
resource "google_compute_firewall" "default" {
  name    = "allow-cloud-orchestrator"
  network = google_compute_network.network.name

  allow {
    protocol = "tcp"
    ports    = ["1080", "1443", "15550-15599"]
  }

  allow {
    protocol = "udp"
    ports    = ["15550-15599"]
  }

  source_ranges = ["0.0.0.0/0"]
}

resource "local_file" "build_and_deploy" {
  content = templatefile("build-and-deploy.sh.tmpl", {
    IMAGE                = "${google_artifact_registry_repository.my-repo.location}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.my-repo.name}/cloud-orchestrator:latest",
    PROJECT_ID           = var.project_id,
    PROJECT_NUMBER       = data.google_project.project.number,
    REGION               = var.region,
    CLOUD_RUN_NAME       = var.cloud_run_name,
    SA_EMAIL             = google_service_account.service_account.email,
    CONNECTOR_ID         = google_vpc_access_connector.connector.id,
    BACKEND_SERVICE_NAME = nonsensitive(module.lb-http.backend_services.default.name),
    BACKEND_SERVICE_ID   = nonsensitive(module.lb-http.backend_services.default.generated_id),
    OAUTH_CLIENT_ID      = google_iap_client.project_client.client_id,
  })
  filename = "${path.root}/build-and-deploy.sh"
}
