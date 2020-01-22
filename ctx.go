package govtl

import (
	"fmt"
	"reflect"
)

type Ctx map[string][]reflect.Value

func (ctx Ctx) Push(k string, v reflect.Value) int {
	ctx[k] = append(ctx[k], v)
	return len(ctx[k]) - 1
}

func (ctx Ctx) Pop(i int, k string) {
	if len(ctx[k]) > 0 && i > 0 {
		ctx[k] = append(ctx[k][:i], ctx[k][i+1:]...)
	}
}

func (ctx Ctx) Get(k string) (reflect.Value, error) {
	s := ctx[k]
	if len(s) > 0 {
		return s[len(s)-1], nil
	}
	return reflect.Value{}, undefinedError{fmt.Errorf("undefined var $%s", k)}
}

func (ctx Ctx) Set(i int, k string, v reflect.Value) {
	s := ctx[k]
	if len(s) > i {
		s[i] = v
	}
}
