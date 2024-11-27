package serializer

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"gorm.io/gorm/schema"
)

// MapSerializer map serializer
type MapSerializer struct{}

// Scan implements serializer interface
func (MapSerializer) Scan(ctx context.Context, field *schema.Field, dst reflect.Value, dbValue interface{}) (err error) {
	fieldValue := reflect.New(field.FieldType)

	if dbValue != nil {
		var bytes []byte
		switch v := dbValue.(type) {
		case []byte:
			bytes = v
		case string:
			bytes = []byte(v)
		default:
			return fmt.Errorf("failed to unmarshal JSONB value: %#v", dbValue)
		}

		if len(bytes) > 0 {
			err = json.Unmarshal(bytes, fieldValue.Interface())
		}
	}

	field.ReflectValueOf(ctx, dst).Set(fieldValue.Elem())
	return
}

// Value MAP('math',120,'english', 123)
func (MapSerializer) Value(ctx context.Context, field *schema.Field, dst reflect.Value, fieldValue interface{}) (interface{}, error) {

	if fieldValue == nil {
		if field.TagSettings["NOT NULL"] != "" {
			return "", nil
		}
		return nil, nil
	}

	w := ValueWriter{}
	err := w.Value(reflect.ValueOf(fieldValue))
	if err != nil {
		return nil, err
	}
	result := w.Bytes()
	return result, nil
}
