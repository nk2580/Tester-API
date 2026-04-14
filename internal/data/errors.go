package data

import (
    "errors"
    "strings"

    sqlite3 "github.com/mattn/go-sqlite3"
    "gorm.io/gorm"
)

// IsDuplicateError returns true when the provided error indicates a unique constraint violation.
func IsDuplicateError(err error) bool {
    if err == nil {
        return false
    }

    if errors.Is(err, gorm.ErrDuplicatedKey) {
        return true
    }

    var sqliteErr sqlite3.Error
    if errors.As(err, &sqliteErr) {
        return sqliteErr.Code == sqlite3.ErrConstraint && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique
    }

    return strings.Contains(strings.ToLower(err.Error()), "unique constraint")
}
