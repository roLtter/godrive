// Package dbmigrate pins golang-migrate drivers in the module graph.
package dbmigrate

import (
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)
