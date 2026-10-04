Immich uses the immich database/user on postgresql-shared-pg-ha.shared.svc (Percona PostgreSQL18), with TLS required and VectorChord selected explicitly. The shared Percona image is built from custom/percona-vectorchord.

The prepare-database init container enables the required extensions and sets the immich role's database-specific search_path to public. Immich's initial migrations introspect public explicitly; Percona's default user-named schema would cause duplicate primary-key migration failures. Other users' schema settings are untouched.

The Immich database user follows this cluster's existing SUPERUSER convention so that extensions and migrations can be managed automatically. Changing to a restricted role would require preparing/updating extensions as an administrator and revisiting Immich's built-in database backups.

Redis queues use redis-shared.shared.svc, with credentials from the existing ExternalSecret and database index 2, chosen because it was unused while other current data was in index 1. The number is not an Immich requirement; it keeps queue keys separate but shares the same Redis process and resources.

The former standalone PostgreSQL and Redis resources are removed. Immich started with a fresh shared database; no old database records were migrated. Photo storage remains at /media/martin/Data/Photos/immich. Database backups from before the cutover remain under /home/martin/ubuntu-diagnostics-2026-10-04/immich-upgrade.
