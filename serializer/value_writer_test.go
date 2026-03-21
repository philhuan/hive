package serializer

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
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
		{
			name: "",
			args: args{
				rv: reflect.ValueOf([]int{1, 2, 3, 4, 5}),
			},
			want:    "ARRAY(1,2,3,4,5)",
			wantErr: false,
		},
		{
			name: "",
			args: args{
				rv: reflect.ValueOf([]map[string]string{
					map[string]string{
						"name": "philhuan",
						"addr": "127.0.0.1",
					},
					map[string]string{
						"name": "hjw",
						"addr": "hjwblog.com",
					},
				}),
			},
			want:    "ARRAY(MAP('name','philhuan','addr','127.0.0.1'),MAP('name','hjw','addr','hjwblog.com'))",
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
