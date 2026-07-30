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
