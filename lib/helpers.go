package lib

import (
	"reflect"
)

func Coalesce[T any](params ...T) T {
	for _, param := range params {
		if !reflect.ValueOf(param).IsNil() {
			return param
		}
	}
	return params[0]
}

func CoalesceString(params ...string) string {
	for _, param := range params {
		if param != "" {
			return param
		}
	}
	return ""
}
