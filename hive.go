package hive

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"runtime"
	"strings"

	"github.com/philhuan/gohive-driver"
	"gorm.io/driver/hive/serializer"

	"gorm.io/gorm/migrator"

	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func init() {
	schema.RegisterSerializer("map", serializer.MapSerializer{})
}

const (
	DriverName = "hive"
)

type Config struct {
	DriverName    string
	ServerVersion string
	DSN           string
	DSNConfig     *DSNConfig
	Conn          gorm.ConnPool

	DefaultStringSize uint
}

type Dialector struct {
	*Config

	logger             logger.Interface
	paramsInterpolator *gohive.ParamsInterpolator
}

var (
	// CreateClauses create clauses
	CreateClauses = []string{"INSERT", "VALUES", "ON CONFLICT"}
	// QueryClauses query clauses
	QueryClauses = []string{"SELECT", "FROM", "WHERE", "GROUP BY", "ORDER BY", "LIMIT", "FOR"}
	// UpdateClauses update clauses
	UpdateClauses = []string{"UPDATE", "SET", "WHERE"}
	// DeleteClauses delete clauses
	DeleteClauses = []string{"DELETE", "FROM", "WHERE"}
)

func Open(dsn string) gorm.Dialector {
	dsnConf, _ := ParseDSN(dsn)
	dsn = dsnConf.Complete().FormatDSN()
	return &Dialector{
		Config:             &Config{DSN: dsn, DSNConfig: dsnConf},
		paramsInterpolator: gohive.NewParamsInterpolator(),
	}
}

func New(config Config) gorm.Dialector {
	switch {
	case config.DSN == "" && config.DSNConfig != nil:
		config.DSN = config.DSNConfig.Complete().FormatDSN()
	case config.DSN != "" && config.DSNConfig == nil:
		config.DSNConfig, _ = ParseDSN(config.DSN)
		config.DSNConfig.Complete()
	}
	return &Dialector{
		Config:             &config,
		paramsInterpolator: gohive.NewParamsInterpolator(),
	}
}

func (d *Dialector) Name() string {
	return DriverName
}

func (d *Dialector) Initialize(db *gorm.DB) (err error) {
	d.logger = db.Logger
	if d.DriverName == "" {
		d.DriverName = DriverName
	}

	if d.Conn != nil {
		db.ConnPool = d.Conn
	} else {
		dsn := d.DSNConfig.FormatDSN()
		db.ConnPool, err = sql.Open(d.DriverName, dsn)
		if err != nil {
			return err
		}
	}

	callbackConfig := &callbacks.Config{
		CreateClauses: CreateClauses,
		QueryClauses:  QueryClauses,
		UpdateClauses: UpdateClauses,
		DeleteClauses: DeleteClauses,
	}

	d.RegisterCallbacks(db, callbackConfig)

	if db.Config != nil {
		db.Config.PrepareStmt = false
		db.Config.DisableNestedTransaction = false
		db.Config.SkipDefaultTransaction = true
	}

	for k, v := range d.ClauseBuilders() {
		db.ClauseBuilders[k] = v
	}

	if d.paramsInterpolator == nil {
		d.paramsInterpolator = gohive.NewParamsInterpolator()
	}

	return nil
}

func (d *Dialector) RegisterCallbacks(db *gorm.DB, config *callbacks.Config) {

	if len(config.CreateClauses) == 0 {
		config.CreateClauses = CreateClauses
	}
	if len(config.QueryClauses) == 0 {
		config.QueryClauses = QueryClauses
	}
	if len(config.DeleteClauses) == 0 {
		config.DeleteClauses = DeleteClauses
	}
	if len(config.UpdateClauses) == 0 {
		config.UpdateClauses = UpdateClauses
	}

	createCallback := db.Callback().Create()
	d.handleError(createCallback.Register("gorm:before_create", callbacks.BeforeCreate))
	d.handleError(createCallback.Register("gorm:save_before_associations", callbacks.SaveBeforeAssociations(true)))
	d.handleError(createCallback.Register("gorm:create", callbacks.Create(config)))
	d.handleError(createCallback.Register("gorm:save_after_associations", callbacks.SaveAfterAssociations(true)))
	d.handleError(createCallback.Register("gorm:after_create", callbacks.AfterCreate))
	createCallback.Clauses = config.CreateClauses

	queryCallback := db.Callback().Query()
	d.handleError(queryCallback.Register("gorm:query", callbacks.Query))
	d.handleError(queryCallback.Register("gorm:preload", callbacks.Preload))
	d.handleError(queryCallback.Register("gorm:after_query", callbacks.AfterQuery))
	queryCallback.Clauses = config.QueryClauses

	deleteCallback := db.Callback().Delete()
	d.handleError(deleteCallback.Register("gorm:before_delete", callbacks.BeforeDelete))
	d.handleError(deleteCallback.Register("gorm:delete_before_associations", callbacks.DeleteBeforeAssociations))
	d.handleError(deleteCallback.Register("gorm:delete", callbacks.Delete(config)))
	d.handleError(deleteCallback.Register("gorm:after_delete", callbacks.AfterDelete))
	deleteCallback.Clauses = config.DeleteClauses

	updateCallback := db.Callback().Update()
	d.handleError(updateCallback.Register("gorm:setup_reflect_value", callbacks.SetupUpdateReflectValue))
	d.handleError(updateCallback.Register("gorm:before_update", callbacks.BeforeUpdate))
	d.handleError(updateCallback.Register("gorm:save_before_associations", callbacks.SaveBeforeAssociations(false)))
	d.handleError(updateCallback.Register("gorm:update", callbacks.Update(config)))
	d.handleError(updateCallback.Register("gorm:save_after_associations", callbacks.SaveAfterAssociations(false)))
	d.handleError(updateCallback.Register("gorm:after_update", callbacks.AfterUpdate))
	updateCallback.Clauses = config.UpdateClauses

	rowCallback := db.Callback().Row()
	d.handleError(rowCallback.Register("gorm:row", callbacks.RowQuery))
	rowCallback.Clauses = config.QueryClauses

	rawCallback := db.Callback().Raw()
	d.handleError(rawCallback.Register("gorm:raw", callbacks.RawExec))
	rawCallback.Clauses = config.QueryClauses
}

func (d *Dialector) Migrator(db *gorm.DB) gorm.Migrator {
	return Migrator{
		Migrator: migrator.Migrator{
			Config: migrator.Config{
				DB:        db,
				Dialector: d,
			},
		},
		Dialector: d,
	}
}

func (d *Dialector) DataTypeOf(field *schema.Field) string {
	switch field.DataType {
	case schema.Bool:
		return "boolean"
	case schema.Int, schema.Uint:
		return d.getSchemaIntAndUnitType(field)
	case schema.Float:
		return d.getSchemaFloatType(field)
	case schema.String:
		return d.getSchemaStringType(field)
	case schema.Time:
		return d.getSchemaTimeType(field)
	case schema.Bytes:
		return d.getSchemaBytesType(field)
	default:
		return d.getSchemaCustomType(field)
	}
}

func (d *Dialector) getSchemaIntAndUnitType(field *schema.Field) string {
	switch {
	case field.Size <= 8:
		return "tinyint"
	case field.Size <= 16:
		return "smallint"
	case field.Size <= 32:
		return "int"
	default:
		return "bigint"
	}
}

func (d *Dialector) getSchemaFloatType(field *schema.Field) string {
	if field.Precision > 0 {
		return fmt.Sprintf("decimal(%d, %d)", field.Precision, field.Scale)
	}

	if field.Size <= 32 {
		return "float"
	}

	return "double"
}

func (d *Dialector) getSchemaStringType(field *schema.Field) string {
	return "String" // TODO: varchar?
}

func (d *Dialector) getSchemaTimeType(field *schema.Field) string {
	return "Timestamp" // TODO: DATE?
}

func (d *Dialector) getSchemaBytesType(field *schema.Field) string {
	return "BINARY"
}

func (d *Dialector) getSchemaCustomType(field *schema.Field) string {
	sqlType := string(field.DataType)
	return sqlType
}

func (d *Dialector) DefaultValueOf(field *schema.Field) clause.Expression {
	return clause.Expr{SQL: "DEFAULT"}
}

func (d *Dialector) BindVarTo(writer clause.Writer, stmt *gorm.Statement, v interface{}) {
	_ = writer.WriteByte('?')
}

func (d *Dialector) QuoteTo(writer clause.Writer, str string) {
	_ = writer.WriteByte('`')
	if strings.Contains(str, ".") {
		for idx, str := range strings.Split(str, ".") {
			if idx > 0 {
				_, _ = writer.WriteString(".`")
			}
			_, _ = writer.WriteString(str)
			_ = writer.WriteByte('`')
		}
	} else {
		_, _ = writer.WriteString(str)
		_ = writer.WriteByte('`')
	}
}

func (d *Dialector) Explain(sql string, vars ...interface{}) string {
	return logger.ExplainSQL(sql, nil, `'`, vars...)
}

func (d *Dialector) getLogger() logger.Interface {
	if d.logger == nil {
		return logger.Default
	}
	return d.logger
}

func (d *Dialector) handleError(err error, ignoreErrors ...error) {
	if err != nil {
		for _, except := range ignoreErrors {
			if errors.Is(err, except) {
				return
			}
		}
		_, file, line, _ := runtime.Caller(1)
		d.getLogger().Warn(context.Background(), "%s:%d %v", file, line, err)
	}
}
