Immich uses the immich database/user on postgresql-shared-pg-ha.shared.svc (Percona PostgreSQL18), with TLS required and VectorChord selected explicitly. The shared Percona image is built from custom/percona-vectorchord.

The prepare-database init container enables the required extensions and sets the immich role's database-specific search_path to public. Immich's initial migrations introspect public explicitly; Percona's default user-named schema would cause duplicate primary-key migration failures. Other users' schema settings are untouched.

The Immich database user follows this cluster's existing SUPERUSER convention so that extensions and migrations can be managed automatically. Changing to a restricted role would require preparing/updating extensions as an administrator and revisiting Immich's built-in database backups.

Redis queues use redis-shared.shared.svc, with credentials from the existing ExternalSecret and database index2 (other current data was in index1).

The former standalone PostgreSQL deployment/service and Redis deployment/service are removed. The old PostgreSQL PVC and its original credential ExternalSecret are retained for recovery; no old Immich data was copied into the new database and no photo files were deleted. The app starts at the welcome page to create a new administrator account.
