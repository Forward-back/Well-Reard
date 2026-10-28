package service

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// CheckEmployee verifies an employee ID (工号) against the business MySQL
// employees table. Returns true when the ID exists and the employee is active.
// A database error is returned to the caller so registration can decide
// whether to degrade to a public account.
func CheckEmployee(ctx context.Context, empNo string) (bool, error) {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&timeout=5s",
		getenv("YLSW_MYSQL_USER", "root"),
		getenv("YLSW_MYSQL_PASSWORD", ""),
		getenv("YLSW_MYSQL_HOST", "host.docker.internal"),
		getenv("YLSW_MYSQL_PORT", "3306"),
		getenv("YLSW_MYSQL_DATABASE", "ums_employee"),
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return false, fmt.Errorf("open business mysql: %w", err)
	}
	defer db.Close()

	db.SetConnMaxLifetime(time.Minute)

	queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var status int
	err = db.QueryRowContext(
		queryCtx,
		"SELECT status FROM employees WHERE emp_no = ? LIMIT 1",
		empNo,
	).Scan(&status)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query employees table: %w", err)
	}
	return status == 1, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
