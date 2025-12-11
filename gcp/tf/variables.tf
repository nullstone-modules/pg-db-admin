variable "name" {
  description = "The name of the cloud function function and "
  type        = string
}

variable "labels" {
  description = "A map of labels that are applied to GCP resources"
  type        = map(string)
}

variable "host" {
  description = "The database cluster host to connect"
  type        = string
}

variable "port" {
  description = "The database cluster port to connect"
  type        = string
  default     = "3306"
}

variable "database" {
  description = "The initial database to connect"
  type        = string
  default     = ""
}

variable "username" {
  description = "Postgresql username"
  type        = string
}

variable "password" {
  description = "Postgresql password"
  type        = string
}

variable "vpc_access_connector_name" {
  type        = string
  description = <<EOF
This module requires a VPC Serverless Access Connector to reach the Cloud SQL instance in a private network.
This variable configures the function to use an existing access connector.
EOF
}

variable "invoker_impersonators" {
  type        = set(string)
  default     = []
  description = <<EOF
A set of IDs for service accounts that should have permission to impersonate the invoker service account.
EOF
}
