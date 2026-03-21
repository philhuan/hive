package serializer

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/philhuan/gohive-driver"
	"go/ast"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type ValueWriter struct {
	bytes.Buffer
	Local *time.Location
}

var (
	jsonRawMessageType = reflect.TypeOf(json.RawMessage{})
	bytesType          = reflect.TypeOf([]byte{})
	timeType           = reflect.TypeOf(time.Time{})
)

func (w *ValueWriter) Value(rv reflect.Value) error {
	for rv.Kind() == reflect.Ptr && !rv.IsNil() {
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Interface && !rv.IsNil() {
		rv = rv.Elem()
	}
	rt := rv.Type()

	switch rt {
	case bytesType:
		if rv.IsNil() {
			w.Write([]byte("NULL"))
		} else {
			w.Write([]byte("X'"))
			w.appendEncodedBytes(rv.Bytes())
			w.WriteByte('\'')
		}
		return nil
	case jsonRawMessageType:
		w.WriteByte('\'')
		w.appendEncodedBytes(rv.Bytes())
		w.WriteByte('\'')
		return nil
	case timeType:
		w.appendDateTime(rv.Interface().(time.Time))
		return nil
	}

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
		w.escapeStringBackslash(rv.String())
		w.WriteString("'")
	case reflect.Slice, reflect.Array:
		if rt.Elem().Kind() == reflect.Uint8 { // []byte
			if rv.IsNil() {
				w.Write([]byte("NULL"))
			} else {
				w.Write([]byte("X'"))
				w.appendEncodedBytes(rv.Bytes())
				w.WriteByte('\'')
			}
		}
		length := rv.Len()
		w.Write([]byte("ARRAY("))
		for i := 0; i < length; i++ {
			element := rv.Index(i)
			if i != 0 {
				w.WriteByte(',')
			}
			err := w.Value(element)
			if err != nil {
				return err
			}
		}
		w.WriteByte(')')
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

var escapeStringReplacer = strings.NewReplacer(
	"\\", "\\\\",
	"'", "\\'",
	"\"", "\\\"",
	"\n", "\\n",
	"\r", "\\r",
	"\x00", "\\0",
	"\x1a", "\\Z",
)

func (w *ValueWriter) escapeStringBackslash(v string) {
	n := escapeStringReplacer.Replace(v)
	w.WriteString(n)
}

func (w *ValueWriter) appendEncodedBytes(v []byte) {
	encodedData := make([]byte, hex.EncodedLen(len(v)))
	hex.Encode(encodedData, v)
	w.Write(encodedData)
}

func (w *ValueWriter) appendDateTime(t time.Time) {
	if t.IsZero() {
		w.WriteString("'0000-00-00'")
		return
	}
	t = t.In(w.Local)
	s := t.Format(gohive.TimeStampLayout)
	w.WriteString(s)
}
