# 0.10.0 (Sep 01, 2026)
* Added ensure password (`POST /roles/{name}/ensure_password`, lambda event `ensure_role_password`): alters the role's password only when a login attempt with it fails.
  Idempotent, so Terraform data sources can call it on every apply to heal a role created passwordless or reset outside the caller's control. A missing role errors; creation stays with `/roles`.
* Role updates now probe a login before `ALTER ROLE ... WITH PASSWORD` and skip the ALTER when the password already works, keeping the password literal out of server logs under `log_statement=ddl`.
  Only a confirmed login skips; any probe failure falls through to the ALTER.

# 0.9.4 (Aug 10, 2026)
* Added conditional IAM to GCP bucket access to resolve IAM permission issues for Google-managed build buckets.

# 0.9.3 (Aug 03, 2026)
* Added parameter privileges (`/roles/{role}/parameter_privileges`) granting a role `SET` on a superuser-context parameter such as `session_replication_role`.
  Membership in `cloudsqlsuperuser`/`rds_superuser` does not confer this, because postgres never inherits role attributes through membership.
  Requires postgres 15+; older servers fail with a clear error rather than silently granting nothing.

# 0.9.2 (Jul 30, 2026)
* Publications now acquire the database owner's privileges, which `CREATE PUBLICATION` requires.
* `GRANT CONNECT` and `GRANT USAGE ON SCHEMA` acquire the database and schema owner's privileges, and verify the result.
  Postgres downgrades a grant the grantor cannot make to a warning, so these could silently grant nothing.

# 0.9.1 (Jul 30, 2026)
* Fixed temporary role membership on postgres 16+, where a `CREATEROLE` admin holds `ADMIN OPTION` without `INHERIT`.
  Membership is now detected with `pg_has_role` and granted `WITH INHERIT TRUE`.
* Table privileges now acquire every relation owner's privileges before granting, covering views and materialized views.

# 0.9.0 (Jul 30, 2026)
* Added `replication` to role attributes. This emits `REPLICATION` on `CREATE ROLE` and adds it to an existing role through `ALTER ROLE ... WITH REPLICATION`.
  Role attributes are now only ever added, never removed.
* Added publications (`/databases/{database}/publications`) supporting `FOR ALL TABLES`, `FOR TABLES IN SCHEMA` (postgres 15+), and `FOR TABLE`.
  `PUT` reconciles membership with `ALTER PUBLICATION ... ADD|DROP` so that an active subscription is never invalidated.
* Added logical replication slots (`/databases/{database}/replication_slots`).
  `DELETE` really drops the slot, an orphaned slot pins WAL indefinitely and disconnects a lingering consumer to do so.
* Added table privileges (`/databases/{database}/table_privileges`) to grant a specific (typically read-only) set of privileges on a schema,
  covering `GRANT USAGE ON SCHEMA`, `GRANT ... ON ALL TABLES IN SCHEMA`, and `ALTER DEFAULT PRIVILEGES ... GRANT ... ON TABLES`.
* Added `GET /settings` reporting `wal_level`, the current user, and replication slot usage.
* Role reads now report role attributes (`createDb`, `createRole`, `replication`).
* Scoped GCP invocation to the cloud function instead of the entire project.
* Scoped GCP cloud function access to binaries bucket instead of the entire project.
