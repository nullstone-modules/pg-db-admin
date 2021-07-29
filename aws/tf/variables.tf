variable "image_uri" {
  description = "The docker image to utilize. By default, this uses nullstone's 'latest' publicly available image."
  type        = string
  default     = "public.ecr.aws/nullstone/pg-db-admin"
}

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
  default = "5432"
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

variable "network" {
  description = "Network configuration"
  type = object({
    security_group_ids : list(string)
    subnet_ids = list(string)
  })
  default = {
    security_group_ids = []
    subnet_ids         = []
  }
}
