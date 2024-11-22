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

variable "project_id" {
  type = string
}

variable "region" {
  description = "Location for load balancer and Cloud Run resources"
  type        = string
  default     = "europe-west3"
}

variable "domain" {
  description = "Domain name to run the load balancer on."
  type        = string
}

variable "lb_name" {
  description = "Name for load balancer and associated resources"
  type        = string
  default     = "tf-cr-lb"
}

variable "oauth_support_email" {
  description = "eMail address displayed to users regarding questions about their consent"
  type        = string
}

variable "artifact_repository_id" {
  description = "The name of the Artificat Repository"
  type        = string
  default     = "cloud-android-orchestration"
}

variable "service_accessors" {
  description = "List of principals which should be able to call the cloud orchestrator"
  type        = set(string)
  default     = []
}

variable "cloud_run_name" {
  description = "The name of the Cloud Run service"
  type        = string
  default     = "cloud-orchestrator"
}

variable "serverless_connector_name" {
  description = "The name of the Serverless VPC Connector"
  type        = string
  default     = "co-vpc-connector"
}

variable "use_private_ips" {
  description = "Toggle to use private IPs in GCP"
  type        = bool
  default     = true
}
