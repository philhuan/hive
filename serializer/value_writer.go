package serializer

import (
	"bytes"
	"fmt"
	"go/ast"
	"reflect"
	"strconv"
)

type ValueWriter struct {
	bytes.Buffer
}

func (w *ValueWriter) Value(rv reflect.Value) error {
	for rv.Kind() == reflect.Ptr && !rv.IsNil() {
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Interface && !rv.IsNil() {
		rv = rv.Elem()
	}
	rt := rv.Type()

	switch rt.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		w.WriteString(strconv.FormatInt(rv.Int(), 10))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		w.WriteString(strconv.FormatUint(rv.Uint(), 10))
	case reflect.Float32, reflect.Float64:
		w.WriteString(strconv.FormatFloat(rv.Float(), 'f', -1, 64))
	case reflect.Bool:
		w.WriteString(strconv.FormatBool(rv.Bool()))
	case reflect.String:
		w.WriteString("'")
		w.WriteString(rv.String())
		w.WriteString("'")
	case reflect.Map:
		w.WriteString("MAP(")
		for i, k := range rv.MapKeys() {
			if i > 0 {
				w.WriteString(",")
			}
			err := w.Value(k)
			if err != nil {
				return err
			}
			w.WriteString(",")
			err = w.Value(rv.MapIndex(k))
			if err != nil {
				return err
			}
		}
		w.WriteString(")")
	case reflect.Struct:
		w.WriteString("named_struct(")
		for i := 0; i < rv.NumField(); i++ {
			fieldStruct := rt.Field(i)
			if !ast.IsExported(fieldStruct.Name) {
				continue
			}
			if i > 0 {
				w.WriteString(",")
			}

			w.WriteString(fieldStruct.Name)
			w.WriteString(",")
			err := w.Value(rv.Field(i))
			if err != nil {
				return err
			}

		}
		w.WriteString(")")
	default: // TODO: slice array, etc.
		return fmt.Errorf("unsupported type: %s", rt.String())
	}
	return nil
}
