package hive

import (
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/philhuan/gohive-driver"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (d *Dialector) ClauseBuilders() map[string]clause.ClauseBuilder {
	clauseBuilders := map[string]clause.ClauseBuilder{
		"VALUES": func(c clause.Clause, builder clause.Builder) {
			statement, ok := builder.(*gorm.Statement)
			if !ok {
				_ = statement.DB.AddError(fmt.Errorf("hive VALUES ClauseBuilder, builder is not statement"))
				return
			}

			err := d.clauseBuilderValues(c, builder, statement)
			if err != nil {
				_ = statement.DB.AddError(fmt.Errorf("hive VALUES ClauseBuilder, %w", err))
			}
		},
	}

	return clauseBuilders
}

func (d *Dialector) clauseBuilderValues(c clause.Clause, builder clause.Builder, statement *gorm.Statement) error {
	c.Build(builder)
	values, err := d.valuesConvert(statement.Vars...)
	if err != nil {
		return fmt.Errorf("valuesConvert failed, %w", err)
	}

	if statement.Schema != nil {
		for i, field := range statement.Schema.Fields {
			if strings.HasPrefix(string(field.DataType), "MAP") {
				values[i] = gohive.NewValueArgsWriter(values[i])
			}
		}
	}

	sql, err := d.paramsInterpolator.Interpolate(statement.SQL.String(), values)
	if err != nil {
		return fmt.Errorf("paramsInterpolator.Interpolate failed, %w", err)
	}

	statement.SQL.Reset()
	statement.SQL.WriteString(sql)
	statement.Vars = nil
	return nil
}

func (d *Dialector) valuesConvert(vars ...interface{}) ([]driver.Value, error) {
	var res = make([]driver.Value, 0, len(vars))
	for _, v := range vars {
		value, err := driver.DefaultParameterConverter.ConvertValue(v)
		if err != nil {
			return nil, fmt.Errorf("failed to convert value: %w,  %v", err, v)
		}
		res = append(res, value)
	}
	return res, nil
}
