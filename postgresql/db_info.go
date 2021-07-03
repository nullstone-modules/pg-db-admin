package postgresql

import (
	"database/sql"
	"fmt"
	"github.com/blang/semver"
	_ "github.com/lib/pq"
	"strings"
	"unicode"
)

type DbInfo struct {
	DbVersion         semver.Version
	SupportedFeatures Features
	IsSuperuser       bool
	CurrentUser       string
}

func CalcDbConnectionInfo(db *sql.DB) (*DbInfo, error) {
	dci := &DbInfo{}

	var superuser bool
	if err := db.QueryRow("SELECT rolsuper FROM pg_roles WHERE rolname = CURRENT_USER").Scan(&superuser); err != nil {
		return nil, fmt.Errorf("could not check if current user is superuser: %w", err)
	}

	var err error

	if dci.DbVersion, err = detectDbVersion(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("error detecting capabilities: %w", err)
	}
	dci.SupportedFeatures = CalcSupportedFeatures(dci.DbVersion)
	dci.CurrentUser, err = getCurrentUser(db)

	return dci, nil
}

func detectDbVersion(db *sql.DB) (semver.Version, error) {
	var pgVersion string
	err := db.QueryRow(`SELECT VERSION()`).Scan(&pgVersion)
	if err != nil {
		return semver.Version{}, fmt.Errorf("error PostgreSQL version: %w", err)
	}

	// PostgreSQL 9.2.21 on x86_64-apple-darwin16.5.0, compiled by Apple LLVM version 8.1.0 (clang-802.0.42), 64-bit
	// PostgreSQL 9.6.7, compiled by Visual C++ build 1800, 64-bit
	fields := strings.FieldsFunc(pgVersion, func(c rune) bool {
		return unicode.IsSpace(c) || c == ','
	})
	if len(fields) < 2 {
		return semver.Version{}, fmt.Errorf("error determining the server version: %q", pgVersion)
	}

	version, err := semver.ParseTolerant(fields[1])
	if err != nil {
		return semver.Version{}, fmt.Errorf("error parsing version: %w", err)
	}

	return version, nil
}

func getCurrentUser(db *sql.DB) (string, error) {
	var currentUser string
	err := db.QueryRow("SELECT CURRENT_USER").Scan(&currentUser)
	switch {
	case err == sql.ErrNoRows:
		return "", fmt.Errorf("SELECT CURRENT_USER returns now row, this is quite disturbing")
	case err != nil:
		return "", fmt.Errorf("error while looking for the current user: %w", err)
	}
	return currentUser, nil
}
