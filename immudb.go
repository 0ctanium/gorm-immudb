package gorm_immudb

import (
	"database/sql"
	"fmt"
	"github.com/codenotary/immudb/pkg/client"
	"github.com/codenotary/immudb/pkg/stdlib"
	"regexp"
	"strconv"

	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/migrator"
	"gorm.io/gorm/schema"
)

type Dialector struct {
	*Config
}

type Config struct {
	DriverName           string
	DSN                  string
	Conn                 gorm.ConnPool

	DefaultVarcharSize	uint
	DefaultBlobSize		uint
	DisableDeletion		bool
}

func Open(dsn string) gorm.Dialector {
	return &Dialector{&Config{DSN: dsn}}
}

func New(config Config) gorm.Dialector {
	return &Dialector{Config: &config}
}

func (dialector Dialector) Name() string {
	return "immudb"
}

func (dialector Dialector) Initialize(db *gorm.DB) (err error) {
	// register callbacks
	callbacks.RegisterDefaultCallbacks(db, &callbacks.Config{
		CreateClauses: []string{"INSERT", "VALUES"},
		UpdateClauses: []string{"UPDATE", "SET", "WHERE"},
		DeleteClauses: []string{"DELETE", "FROM", "WHERE"},
	})

	if dialector.Conn != nil {
		db.ConnPool = dialector.Conn
	} else if dialector.DriverName != "" {
		db.ConnPool, err = sql.Open(dialector.DriverName, dialector.Config.DSN)
	} else {
		var config *client.Options

		config, err = stdlib.ParseConfig(dialector.Config.DSN)
		if err != nil {
			return
		}
		if config == nil {
			err = fmt.Errorf("something went wrong when parsing DSN config")
			return
		}

		db.ConnPool = stdlib.OpenDB(config)
	}
	return
}

func (dialector Dialector) Migrator(db *gorm.DB) gorm.Migrator {
	return Migrator{migrator.Migrator{Config: migrator.Config{
		DB:                          db,
		Dialector:                   dialector,
		CreateIndexAfterCreateTable: true,
	}}}
}

func (dialector Dialector) DefaultValueOf(field *schema.Field) clause.Expression {
	return clause.Expr{SQL: "DEFAULT"}
}

func (dialector Dialector) BindVarTo(writer clause.Writer, stmt *gorm.Statement, v interface{}) {
	writer.WriteByte('$')
	writer.WriteString(strconv.Itoa(len(stmt.Vars)))
}

func (dialector Dialector) QuoteTo(writer clause.Writer, str string) {
	writer.WriteString(str)
}

var numericPlaceholder = regexp.MustCompile("\\$(\\d+)")

func (dialector Dialector) Explain(sql string, vars ...interface{}) string {
	return logger.ExplainSQL(sql, numericPlaceholder, `'`, vars...)
}

func (dialector Dialector) DataTypeOf(field *schema.Field) string {
	switch field.DataType {
	case schema.Bool:
		return "BOOLEAN"
	case schema.Int, schema.Uint:
		sqlType := "INTEGER"
		if field.AutoIncrement {
			sqlType += " AUTO_INCREMENT"
		}

		return sqlType
	case schema.Float:
		return "INTEGER"
	case schema.Time:
		return "TIMESTAMP"
	case schema.String:
		if field.Size > 0 {
			return fmt.Sprintf("VARCHAR[%d]", field.Size)
		}
		return "VARCHAR"
	case schema.Bytes:
		if field.Size > 0 {
			return fmt.Sprintf("BLOB[%d]", field.Size)
		}
		return "BLOB"
	}

	return string(field.DataType)
}

func (dialectopr Dialector) SavePoint(tx *gorm.DB, name string) error {
	tx.Exec("SAVEPOINT " + name)
	return nil
}

func (dialectopr Dialector) RollbackTo(tx *gorm.DB, name string) error {
	tx.Exec("ROLLBACK TO SAVEPOINT " + name)
	return nil
}
