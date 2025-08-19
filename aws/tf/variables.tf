variable "name" {
  description = "The name of the lambda function and role"
  type        = string
}

variable "tags" {
  description = "A map of tags that are applied to AWS resources"
  type        = map(string)
}

variable "host" {
  description = "The database cluster host to connect"
  type        = string
}

variable "port" {
  description = "The database cluster port to connect"
  type        = string
  default     = "5432"
}

variable "database" {
  description = "The initial database to connect. By default, uses 'postgres'"
  type        = string
  default     = "postgres"
}

variable "username" {
  description = "Postgres username"
  type        = string
}

variable "password" {
  description = "Postgres password"
  type        = string
}

variable "is_prod_env" {
  type        = bool
  default     = true
  description = <<EOF
When destroying, is_prod_env determines the recovery window for the admin password secret.
If true, a 7-day recovery window will be configured.
If not, secret will be deleted immediately.
EOF
}

variable "network" {
  description = <<EOF
Network configuration.
Do not choose public subnets unless you have configured a VPC Endpoint in the VPC for Secrets Manager.
EOF

  type = object({
    vpc_id : string
    pg_security_group_id : string
    security_group_ids : list(string)
    subnet_ids = list(string)
  })

  default = {
    vpc_id               = ""
    pg_security_group_id = ""
    security_group_ids   = []
    subnet_ids           = []
  }
}
