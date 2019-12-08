package main

import (
	"errors"

	"github.com/go-sql-driver/mysql"
)

func deleteFromDatabase(id, table, dbRelationshipErrorMessage string) (string, error) {
	query := "DELETE FROM " + table + " WHERE id=? LIMIT 1"
	res, err := db.Exec(query, id)

	if err == nil {
		rowsAffected, err2 := res.RowsAffected()
		if err2 != nil {
			return "Internal server error", err2
		}
		if rowsAffected == 0 {
			return "Already doesn't exist!", errors.New("no rows deleted") // Success, but no rows were deleted
		}
		return "Success", nil // Success - deleted 1 row
	}

	// Handle expected mysql errors:
	mysqlError, ok := err.(*mysql.MySQLError)
	if !ok {
		return "Internal server error", err
	}
	if mysqlError.Number == 1451 {
		return dbRelationshipErrorMessage, errors.New("database relationship error")
	}
	return "Internal server error", err
}
