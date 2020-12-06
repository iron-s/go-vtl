package govtl

import (
	"fmt"
	"reflect"
)

type Ctx struct {
	s map[string][]reflect.Value

	callDepth int
}

func NewContext() Ctx {
	return Ctx{make(map[string][]reflect.Value), 0}
}

func (ctx Ctx) Push(k string, v reflect.Value) int {
	ctx.s[k] = append(ctx.s[k], v)
	return len(ctx.s[k]) - 1
}

func (ctx Ctx) Pop(i int, k string) {
	if len(ctx.s[k]) > 0 && i >= 0 {
		ctx.s[k] = append(ctx.s[k][:i], ctx.s[k][i+1:]...)
	}
}

func (ctx Ctx) Get(k string) (reflect.Value, error) {
	s := ctx.s[k]
	if len(s) > 0 {
		return s[len(s)-1], nil
	}
	return reflect.Value{}, undefinedError{fmt.Errorf("undefined var $%s", k)}
}

func (ctx Ctx) Set(i int, k string, v reflect.Value) {
	s := ctx.s[k]
	if len(s) > i {
		s[i] = v
	}
}
