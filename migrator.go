package gorm_immudb

import (
	"database/sql"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/migrator"
	"gorm.io/gorm/schema"
)

type Migrator struct {
	migrator.Migrator
}

func (m Migrator) AutoMigrate(values ...interface{}) error {
	for _, value := range m.ReorderModels(values, true) {
		tx := m.DB.Session(&gorm.Session{})

		if err := tx.Migrator().CreateTable(value); err != nil {
			return err
		}

		if err := m.RunWithValue(value, func(stmt *gorm.Statement) (errr error) {
			for _, _ = range stmt.Schema.Relationships.Relations {

				for _, chk := range stmt.Schema.ParseCheckConstraints() {
					if err := tx.Migrator().CreateConstraint(value, chk.Name); err != nil {
						return err
					}
				}
			}

			for _, idx := range stmt.Schema.ParseIndexes() {
				if err := tx.Migrator().CreateIndex(value, idx.Name); err != nil {
					return err
				}
			}

			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}

type Column struct {
	name              string
	nullable          sql.NullString
	datatype          string
	maxlen            sql.NullInt64
	precision         sql.NullInt64 // float not yet supported
	scale             sql.NullInt64 // float not yet supported
	typlen            sql.NullInt64
}

func (c Column) Name() string {
	return c.name
}

func (c Column) DatabaseTypeName() string {
	return c.datatype
}

func (c Column) Length() (length int64, ok bool) {
	ok = c.typlen.Valid
	if ok && c.typlen.Int64 > 0 {
		length = c.typlen.Int64
	} else {
		ok = c.maxlen.Valid
		if ok {
			length = c.maxlen.Int64
		} else {
			length = 0
		}
	}
	return
}

func (c Column) Nullable() (nullable bool, ok bool) {
	if c.nullable.Valid {
		nullable, ok = c.nullable.String == "YES", true
	} else {
		nullable, ok = false, false
	}
	return
}

func (c Column) DecimalSize() (precision int64, scale int64, ok bool) {
	panic(ErrNotImplemented)
}

func (m Migrator) CurrentDatabase() (name string) {
	panic(ErrNotImplemented)
}

func (m Migrator) BuildIndexOptions(opts []schema.IndexOption, stmt *gorm.Statement) (results []interface{}) {
	for _, opt := range opts {
		str := stmt.Quote(opt.DBName)
		if opt.Expression != "" {
			str = opt.Expression
		}

		if opt.Collate != "" {
			str += " COLLATE " + opt.Collate
		}

		if opt.Sort != "" {
			str += " " + opt.Sort
		}
		results = append(results, clause.Expr{SQL: str})
	}
	return
}

func (m Migrator) HasIndex(value interface{}, name string) bool {
	panic(ErrNotImplemented)
}

func (m Migrator) CreateIndex(value interface{}, name string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if idx := stmt.Schema.LookIndex(name); idx != nil {
			opts := m.BuildIndexOptions(idx.Fields, stmt)
			values := []interface{}{m.CurrentTable(stmt), opts}
			//values := []interface{}{clause.Column{Name: idx.Name}, m.CurrentTable(stmt), opts}

			createIndexSQL := "CREATE "
			if idx.Class != "" {
				createIndexSQL += idx.Class + " "
			}
			createIndexSQL += "INDEX IF NOT EXISTS "

			if strings.TrimSpace(strings.ToUpper(idx.Option)) == "CONCURRENTLY" {
				createIndexSQL += "CONCURRENTLY "
			}

			createIndexSQL += "ON ?"

			if idx.Type != "" {
				createIndexSQL += " USING " + idx.Type + "(?)"
			} else {
				createIndexSQL += " ?"
			}

			if idx.Where != "" {
				createIndexSQL += " WHERE " + idx.Where
			}

			return m.DB.Exec(createIndexSQL, values...).Error
		}

		return fmt.Errorf("failed to create index with name %v", name)
	})
}

func (m Migrator) RenameIndex(value interface{}, oldName, newName string) error {
	panic(ErrNotSupported)
}

func (m Migrator) DropIndex(value interface{}, name string) error {
	panic(ErrNotSupported)
}

func (m Migrator) GetTables() (tableList []string, err error) {
	// TODO: Try to list table via gRPC immuclient
	panic(ErrNotImplemented)
}


// CreateTable from gorm@v1.22.3/migrator/migrator.go
func (m Migrator) CreateTable(values ...interface{}) error {
	for _, value := range m.ReorderModels(values, false) {

		tx := m.DB.Session(&gorm.Session{})

		if err := m.RunWithValue(value, func(stmt *gorm.Statement) (errr error) {
			var (
				createTableSQL          = "CREATE TABLE IF NOT EXISTS ? ("
				values                  = []interface{}{m.CurrentTable(stmt)}
				hasPrimaryKeyInDataType bool
			)

			for _, dbName := range stmt.Schema.DBNames {
				field := stmt.Schema.FieldsByDBName[dbName]
				if !field.IgnoreMigration {
					createTableSQL += "? ?"
					hasPrimaryKeyInDataType = hasPrimaryKeyInDataType || strings.Contains(strings.ToUpper(string(field.DataType)), "PRIMARY KEY")
					values = append(values, clause.Column{Name: dbName}, m.DB.Migrator().FullDataTypeOf(field))
					createTableSQL += ","
				}
			}

			if !hasPrimaryKeyInDataType && len(stmt.Schema.PrimaryFields) > 0 {
				createTableSQL += "PRIMARY KEY ?,"
				primaryKeys := []interface{}{}
				for _, field := range stmt.Schema.PrimaryFields {
					primaryKeys = append(primaryKeys, clause.Column{Name: field.DBName})
				}

				values = append(values, primaryKeys)
			}

			for _, idx := range stmt.Schema.ParseIndexes() {
				if m.CreateIndexAfterCreateTable {
					defer func(value interface{}, name string) {
						if errr == nil {
							errr = m.CreateIndex(value, name)
						}
					}(value, idx.Name)
				} else {
					if idx.Class != "" {
						createTableSQL += idx.Class + " "
					}
					createTableSQL += "INDEX ? ?"

					// Comments not supported
					// if idx.Comment != "" {
					// 	createTableSQL += fmt.Sprintf(" COMMENT '%s'", idx.Comment)
					// }

					if idx.Option != "" {
						createTableSQL += " " + idx.Option
					}

					createTableSQL += ","
					values = append(values, clause.Expr{SQL: idx.Name}, tx.Migrator().(migrator.BuildIndexOptionsInterface).BuildIndexOptions(idx.Fields, stmt))
				}
			}

			createTableSQL = strings.TrimSuffix(createTableSQL, ",")

			createTableSQL += ")"

			if tableOption, ok := m.DB.Get("gorm:table_options"); ok {
				createTableSQL += fmt.Sprint(tableOption)
			}


			errr = tx.Exec(createTableSQL, values...).Error
			return errr
		}); err != nil {
			return err
		}
	}
	return nil
}

func (m Migrator) HasTable(value interface{}) bool {
	panic(ErrNotImplemented)
}

func (m Migrator) DropTable(values ...interface{}) error {
	return ErrNotSupported
}

func (m Migrator) AddColumn(value interface{}, field string) error {
	return ErrNotSupported
}

func (m Migrator) HasColumn(value interface{}, field string) bool {
	panic(ErrNotImplemented)
}

func (m Migrator) MigrateColumn(value interface{}, field *schema.Field, columnType gorm.ColumnType) error {
	panic(ErrNotImplemented)
}

func (m Migrator) HasConstraint(value interface{}, name string) bool {
	panic(ErrNotImplemented)
}

func (m Migrator) ColumnTypes(value interface{}) (columnTypes []gorm.ColumnType, err error) {
	panic(ErrNotImplemented)
}

func (m Migrator) CurrentSchema(stmt *gorm.Statement, table string) (interface{}, interface{}) {
	panic(ErrNotImplemented)
}