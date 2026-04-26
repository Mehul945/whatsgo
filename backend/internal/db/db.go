package db

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// DB wraps *sql.DB and exposes the Driver name so callers can branch on dialect
// when needed (e.g. upserts). All query/exec methods automatically rewrite "?"
// placeholders to "$1,$2,..." when the driver is PostgreSQL.
type DB struct {
	*sql.DB
	Driver string // "sqlite3", "postgres", "mysql"
}

// Connect parses databaseURL and returns a ready-to-use *DB.
//
// Supported schemes:
//
//	sqlite://path           – SQLite  (file path after "sqlite://")
//	postgres://...          – PostgreSQL (standard DSN)
//	postgresql://...        – PostgreSQL (alias)
//	mysql://user:pass@host:port/dbname  – MariaDB / MySQL
//	mariadb://...           – MariaDB (alias, treated as mysql)
func Connect(databaseURL string) (*DB, error) {
	switch {
	case strings.HasPrefix(databaseURL, "sqlite://"):
		return connectSQLite(databaseURL)
	case strings.HasPrefix(databaseURL, "postgres://") || strings.HasPrefix(databaseURL, "postgresql://"):
		return connectPostgres(databaseURL)
	case strings.HasPrefix(databaseURL, "mysql://") || strings.HasPrefix(databaseURL, "mariadb://"):
		return connectMySQL(databaseURL)
	default:
		return nil, fmt.Errorf("unsupported database URL scheme: %s (expected sqlite://, postgres://, postgresql://, mysql://, or mariadb://)", databaseURL)
	}
}

// ─── SQLite ─────────────────────────────────────────────────────────────────

func connectSQLite(databaseURL string) (*DB, error) {
	dbPath := strings.TrimPrefix(databaseURL, "sqlite://")
	if strings.HasPrefix(dbPath, "/./") || dbPath == "/." {
		dbPath = strings.TrimPrefix(dbPath, "/")
	}
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}
	raw, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	if err := raw.Ping(); err != nil {
		return nil, err
	}
	if _, err := raw.Exec(sqliteSchema()); err != nil {
		return nil, fmt.Errorf("migrate sqlite: %w", err)
	}
	return &DB{DB: raw, Driver: "sqlite3"}, nil
}

// ─── PostgreSQL ─────────────────────────────────────────────────────────────

func connectPostgres(databaseURL string) (*DB, error) {
	raw, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	if err := raw.Ping(); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	if _, err := raw.Exec(postgresSchema()); err != nil {
		return nil, fmt.Errorf("migrate postgres: %w", err)
	}
	return &DB{DB: raw, Driver: "postgres"}, nil
}

// ─── MySQL / MariaDB ───────────────────────────────────────────────────────

func connectMySQL(databaseURL string) (*DB, error) {
	dsn, err := mysqlURLToDSN(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse mysql url: %w", err)
	}
	raw, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	if err := raw.Ping(); err != nil {
		return nil, fmt.Errorf("ping mysql: %w", err)
	}
	// MySQL requires executing statements one at a time for multi-statement DDL
	// when using the default multiStatements=false. We enable multiStatements
	// in the DSN so the schema can be applied in one Exec call.
	if _, err := raw.Exec(mysqlSchema()); err != nil {
		return nil, fmt.Errorf("migrate mysql: %w", err)
	}
	return &DB{DB: raw, Driver: "mysql"}, nil
}

// mysqlURLToDSN converts a mysql://user:pass@host:port/dbname URL into the
// go-sql-driver/mysql DSN format: user:pass@tcp(host:port)/dbname?parseTime=true&multiStatements=true
func mysqlURLToDSN(rawURL string) (string, error) {
	// Normalise mariadb:// to mysql:// so url.Parse handles it
	rawURL = strings.Replace(rawURL, "mariadb://", "mysql://", 1)

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	user := u.User.Username()
	pass, _ := u.User.Password()

	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "3306"
	}

	dbname := strings.TrimPrefix(u.Path, "/")

	// Carry over any query params from the original URL
	q := u.Query()
	q.Set("parseTime", "true")
	q.Set("multiStatements", "true")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?%s", user, pass, host, port, dbname, q.Encode())
	return dsn, nil
}

// ─── Query wrappers (placeholder rewriting for Postgres) ────────────────────

// Query wraps sql.DB.Query, rewriting "?" placeholders for Postgres.
func (d *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return d.DB.Query(d.rewrite(query), args...)
}

// QueryRow wraps sql.DB.QueryRow, rewriting "?" placeholders for Postgres.
func (d *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	return d.DB.QueryRow(d.rewrite(query), args...)
}

// Exec wraps sql.DB.Exec, rewriting "?" placeholders for Postgres.
func (d *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return d.DB.Exec(d.rewrite(query), args...)
}

// rewrite converts "?" placeholders to "$N" when using the postgres driver.
// For sqlite3 and mysql the query is returned unchanged.
func (d *DB) rewrite(query string) string {
	if d.Driver != "postgres" {
		return query
	}
	var b strings.Builder
	b.Grow(len(query) + 16)
	n := 0
	inString := false
	for i := 0; i < len(query); i++ {
		ch := query[i]
		if ch == '\'' {
			inString = !inString
			b.WriteByte(ch)
			continue
		}
		if ch == '?' && !inString {
			n++
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(n))
		} else {
			b.WriteByte(ch)
		}
	}
	return b.String()
}
