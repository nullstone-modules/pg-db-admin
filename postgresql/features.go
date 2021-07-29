package postgresql

import (
	"github.com/blang/semver"
)

type FeatureName uint

const (
	FeatureCreateRoleWith FeatureName = iota
	FeatureDBAllowConnections
	FeatureDBIsTemplate
	FeatureFallbackApplicationName
	FeatureRLS
	FeatureSchemaCreateIfNotExist
	FeatureReplication
	FeatureExtension
	FeaturePrivileges
	FeatureForceDropDatabase
	FeaturePid
)

type Features map[FeatureName]bool

func CalcSupportedFeatures(dbVersion semver.Version) Features {
	return Features{
		// CREATE ROLE WITH
		FeatureCreateRoleWith: semver.MustParseRange(">=8.1.0")(dbVersion),

		// CREATE DATABASE has ALLOW_CONNECTIONS support
		FeatureDBAllowConnections: semver.MustParseRange(">=9.5.0")(dbVersion),

		// CREATE DATABASE has IS_TEMPLATE support
		FeatureDBIsTemplate: semver.MustParseRange(">=9.5.0")(dbVersion),

		// https://www.postgresql.org/docs/9.0/static/libpq-connect.html
		FeatureFallbackApplicationName: semver.MustParseRange(">=9.0.0")(dbVersion),

		// CREATE SCHEMA IF NOT EXISTS
		FeatureSchemaCreateIfNotExist: semver.MustParseRange(">=9.3.0")(dbVersion),

		// row-level security
		FeatureRLS: semver.MustParseRange(">=9.5.0")(dbVersion),

		// CREATE ROLE has REPLICATION support.
		FeatureReplication: semver.MustParseRange(">=9.1.0")(dbVersion),

		// CREATE EXTENSION support.
		FeatureExtension: semver.MustParseRange(">=9.1.0")(dbVersion),

		// We do not support postgresql_grant and postgresql_default_privileges
		// for Postgresql < 9.
		FeaturePrivileges: semver.MustParseRange(">=9.0.0")(dbVersion),

		// DROP DATABASE WITH FORCE
		// for Postgresql >= 13
		FeatureForceDropDatabase: semver.MustParseRange(">=13.0.0")(dbVersion),

		// Column procpid was replaced by pid in pg_stat_activity
		// for Postgresql >= 9.2 and above
		FeaturePid: semver.MustParseRange(">=9.2.0")(dbVersion),
	}
}

func (f Features) IsSupported(name FeatureName) bool {
	val, ok := f[name]
	return ok && val
}
