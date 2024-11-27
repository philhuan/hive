package serializer

import (
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestValueWriter_Value(t *testing.T) {

	type Addr struct {
		Home string
		Work string
	}

	type User struct {
		Name string
		Addr Addr
	}

	type args struct {
		rv reflect.Value
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "",
			args: args{
				rv: reflect.ValueOf(map[string]string{
					"name": "philhuan",
					"addr": "127.0.0.1",
				}),
			},
			want:    "MAP(addr,127.0.0.1,name,philhuan)",
			wantErr: false,
		},
		{
			name: "",
			args: args{
				rv: reflect.ValueOf(&User{
					Name: "philhuan",
					Addr: Addr{
						Home: "a1",
						Work: "a2",
					},
				}),
			},
			want:    "named_struct(Name,philhuan,Addr,named_struct(Home,a1,Work,a2))",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &ValueWriter{}
			if err := w.Value(tt.args.rv); (err != nil) != tt.wantErr {
				t.Errorf("Value() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.want, w.String())
		})
	}
}
