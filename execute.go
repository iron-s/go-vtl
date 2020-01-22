package govtl

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
	// "github.com/davecgh/go-spew/spew"
)

type undefinedError struct {
	error
}
type nilError struct {
	error
}

func (t *Template) Execute(w io.Writer, val map[string]interface{}) error {
	ctx := make(Ctx)
	for k, v := range val {
		ctx.Push(k, reflect.ValueOf(v))
	}
	_, err := t._execute(w, t.tree, ctx)
	return err
}

func (t *Template) _execute(w io.Writer, list []Node, ctx Ctx) (bool, error) {
	for _, expr := range list {
		switch n := expr.(type) {
		case TextNode:
			fmt.Fprint(w, n)
		case *IfNode:
			var (
				stop bool
				err  error
			)
			if n.Cond == nil {
				stop, err = t._execute(w, n.Items, ctx)
				if stop {
					return true, err
				}
				break
			}
			cond, err := t.eval(n.Cond, ctx, true)
			if err != nil && !errors.As(err, &undefinedError{}) && !errors.As(err, &nilError{}) {
				return true, err
			}
			if isTrue(cond) {
				stop, err = t._execute(w, n.Items, ctx)
			} else if n.Else != nil {
				stop, err = t._execute(w, []Node{n.Else}, ctx)
			}
			if stop {
				return true, err
			}
		case *SetNode:
			set, err := t.eval(n.Expr, ctx, false)
			if errors.As(err, &nilError{}) {
				// do nothing
				// set = reflect.Value{}
			} else if err != nil {
				return false, err
			}
			if set.IsValid() {
				set = reflect.ValueOf(set.Interface())
			}
			if len(n.Var.Items) == 0 {
				depth := ctx.Push(n.Var.Name, set)
				defer ctx.Pop(depth, n.Var.Name)
			} else if set.IsValid() {
				if err := t.setVar(n.Var, set, ctx); err != nil {
					return false, err
				}
			}
		case *OpNode:
			log.Println("op node should be handled elsewere")
		case *VarNode:
			v, err := t.evalVar(n, ctx)
			if err != nil && !(n.Silent && errors.As(err, &nilError{})) {
				return false, err
			}
			if v.IsValid() {
				fmt.Fprint(w, vtlPrint(v))
			} else if !n.Silent {
				val, _ := ctx.Get(n.Name)
				fmt.Printf("%s %#v %v\n", n.Name, n.Items, val)
			}
		case *MacroNode:
			if _, ok := t.macros[n.Name]; !ok {
				t.macros[n.Name] = n
			}
		case *MacroCall:
			m, ok := t.macros[n.Name]
			if !ok {
				return false, fmt.Errorf("undefined macro '%s' call", n.Name)
			}
			for i := range m.Assign {
				v, err := t.eval(n.Vals[i], ctx, false)
				if err != nil {
					return true, err
				}
				depth := ctx.Push(m.Assign[i].Name, v)
				defer ctx.Pop(depth, m.Assign[i].Name)
			}
			stop, err := t._execute(w, m.Items, ctx)
			if err != nil {
				return true, err
			} else if stop {
				return false, nil
			}
		case *ForeachNode:
			iter, err := t.eval(n.Iter, ctx, false)
			if err != nil {
				return true, err
			}
			if !iter.IsValid() {
				break
			}
			iter = indirect(iter)
			vdepth := ctx.Push(n.Var.Name, reflect.ValueOf(nil))
			f := &foreach{}
			fdepth := ctx.Push("foreach", reflect.ValueOf(f))
			if iter.Kind() == reflect.Map {
				f.c = iter.Len()
				i := 0
				keys := iter.MapKeys()
				sort.Slice(keys, func(i, j int) bool { return lt(keys[i], keys[j]) })
				for _, k := range keys {
					ctx.Set(vdepth, n.Var.Name, iter.MapIndex(k))
					i++
					f.i = i
					_, err := t._execute(w, n.Items, ctx)
					if err != nil {
						return true, err
					}
				}
			} else {
				var it iterator = iter
				if iter.Kind() == reflect.Struct && iter.Type().Name() == "iter" {
					it = iter.Interface().(iterator)
				}
				l := it.Len()
				f.c = l
				for i := 0; i < l; i++ {
					ctx.Set(vdepth, n.Var.Name, it.Index(i))
					f.i = i
					_, err := t._execute(w, n.Items, ctx)
					if err != nil {
						return true, err
					}
				}
			}
			if f.c == 0 && n.Else != nil {
				_, err := t._execute(w, n.Else, ctx)
				if err != nil {
					return true, err
				}
			}
			ctx.Pop(vdepth, n.Var.Name)
			ctx.Pop(fdepth, "foreach")
		// case nil:
		// case string:
		// 	fmt.Fprint(w, n)
		case *StopNode:
			return true, nil
		case *BreakNode:
			return false, nil
		case *IncludeNode:
			for _, v := range n.Names {
				name, err := t.eval(v, ctx, false)
				if err != nil {
					return true, err
				}
				data, err := ioutil.ReadFile(filepath.Join(t.root, name.String()))
				if err != nil {
					return true, err
				}
				w.Write(data)
			}
		case *ParseNode:
			name, err := t.eval(n.Name, ctx, false)
			if err != nil {
				return true, err
			}
			tmpl, err := ParseFile(filepath.Join(t.root, name.String()), t.root, t.lib)
			if err != nil {
				return true, err
			}
			stop, err := tmpl._execute(w, tmpl.tree, ctx)
			if stop {
				return true, err
			}
		default:
			log.Printf("unexpected %T, %[1]v", n)
		}
	}
	return false, nil
}

func (t *Template) evalStep(v reflect.Value, m *AccessNode, ctx Ctx) (reflect.Value, error) {
	var args []reflect.Value
	var err error
	for _, arg := range m.Args {
		a, err := t.eval(arg, ctx, false)
		if err != nil {
			return a, err
		}
		args = append(args, a)
	}
	switch {
	case m.Name == "":
		return idx(v, args[0])
	case m.IsCall:
		v, err = call(v, m.Name, args...)
	default:
		v, err = idx(v, reflect.ValueOf(m.Name))
	}
	if err != nil {
		return reflect.Value{}, err
	}
	return v, nil
}

func (t *Template) evalVar(n *VarNode, ctx Ctx) (reflect.Value, error) {
	v, err := ctx.Get(n.Name)
	if err != nil {
		return v, err
	}
	for _, m := range n.Items {
		if v, err = t.evalStep(v, m, ctx); err != nil {
			return v, err
		}
	}
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			return reflect.Value{}, nilError{fmt.Errorf("nil result $%s", n.Name)}
		}
	}
	return v, nil
}

func (t *Template) setVar(n *VarNode, val reflect.Value, ctx Ctx) error {
	v, _ := ctx.Get(n.Name)
	var err error
	var prev reflect.Value
	for _, m := range n.Items {
		prev = v
		if v, err = t.evalStep(v, m, ctx); err != nil {
			return err
		}
	}
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			return fmt.Errorf("nil resul $%s", n.Name)
		}
	}
	switch {
	case prev.Kind() == reflect.Map:
		prev.SetMapIndex(reflect.ValueOf(n.Items[len(n.Items)-1].Name), val)
	case v.CanSet() && val.Type().ConvertibleTo(v.Type()):
		v.Set(val)
	}
	return nil
}

var reflectValueType = reflect.TypeOf((*reflect.Value)(nil)).Elem()

func (t *Template) eval(e *OpNode, ctx Ctx, undefOk bool) (reflect.Value, error) {
	if e == nil {
		return reflect.Value{}, nil
	}
	if e.Op != "" {
		fn, ok := funcs[e.Op]
		if !ok {
			return reflect.Value{}, fmt.Errorf("unsupported operator: %s", e.Op)
		}
		f := reflect.ValueOf(fn)
		l, err := t.eval(e.Left, ctx, undefOk)
		if err != nil && !(undefOk && errors.As(err, &undefinedError{})) {
			return l, err
		}
		l = indirect(l)
		r, err := t.eval(e.Right, ctx, undefOk)
		if err != nil && !(undefOk && errors.As(err, &undefinedError{})) {
			return r, err
		}
		r = indirect(r)
		ret := f.Call([]reflect.Value{reflect.ValueOf(l), reflect.ValueOf(r)})
		if ret[0].Type() == reflectValueType {
			return ret[0].Interface().(reflect.Value), nil
		}
		return ret[0], nil
	}
	switch val := e.Val.(type) {
	case *InterpolatedNode:
		var b bytes.Buffer
		_, err := t._execute(&b, val.Items, ctx)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(b.String()), nil
	case *VarNode:
		return t.evalVar(val, ctx)
	case nil:
	case int64, float64, bool, string:
		return reflect.ValueOf(val), nil
	case []interface{}:
		vv := make([]interface{}, len(val))
		var err error
		for i := range val {
			if op, ok := val[i].(*OpNode); ok {
				vv[i], err = t.eval(op, ctx, false)
				if err != nil {
					return reflect.Value{}, err
				}
			}
		}
		return reflect.ValueOf(vv), nil
	case *RefNode:
		return ctx.Get(val.Name)
	case []*OpNode:
		m := make(map[string]interface{}, len(val)/2)
		for i := 0; i < len(val); i += 2 {
			k, v := val[i], val[i+1]
			kk, err := t.eval(k, ctx, false)
			if err != nil {
				return kk, err
			}
			var s string
			if isInt(kk) {
				s = fmt.Sprint(kk.Int())
			} else {
				s = kk.String()
			}
			e, err := t.eval(v, ctx, false)
			if err != nil {
				return e, err
			}
			m[s] = e.Interface()
		}
		return reflect.ValueOf(m), nil
	default:
		log.Printf("unsupported type %T: %v", val, val)
		return reflect.ValueOf(val), nil
	}
	return reflect.Value{}, nil
}

func isTrue(v reflect.Value) bool {
	if !v.IsValid() {
		return false
	}
	switch v.Kind() {
	case reflect.Bool:
		return v.Bool()
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() > 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() != 0
	case reflect.Float32, reflect.Float64:
		return v.Float() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() != 0
	case reflect.Chan, reflect.Func, reflect.Ptr, reflect.Interface:
		return !v.IsNil()
	case reflect.Struct:
		return true
	default:
		return false
	}
}

func toFloat(v reflect.Value) float64 {
	switch v.Kind() {
	case reflect.Float32, reflect.Float64:
		return v.Float()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(v.Uint())
	default:
		return float64(v.Int())
	}
}

var funcs = map[string]interface{}{
	"eq": eq, "ne": ne, "le": le, "lt": lt, "ge": ge, "gt": gt,

	"negate": func(v1 reflect.Value, v2 interface{}) (reflect.Value, error) {
		if !isNumber(v1) {
			return reflect.Value{}, errors.New("NaN")
		}
		switch {
		case isInt(v1):
			return reflect.ValueOf(-v1.Int()), nil
		default:
			return reflect.ValueOf(-toFloat(v1)), nil
		}
	},

	"+": func(v1, v2 reflect.Value) (reflect.Value, error) {
		if !isNumber(v1) || !isNumber(v2) {
			return reflect.ValueOf(fmt.Sprint(v1) + fmt.Sprint(v2)), nil
		}
		switch {
		case isInt(v1) && isInt(v2):
			return reflect.ValueOf(v1.Int() + v2.Int()), nil
		default:
			return reflect.ValueOf(toFloat(v1) + toFloat(v2)), nil
		}
	},

	"-": func(v1, v2 reflect.Value) (reflect.Value, error) {
		if !isNumber(v1) || !isNumber(v2) {
			return reflect.Value{}, errors.New("NaN")
		}
		switch {
		case isInt(v1) && isInt(v2):
			return reflect.ValueOf(v1.Int() - v2.Int()), nil
		default:
			return reflect.ValueOf(toFloat(v1) - toFloat(v2)), nil
		}
	},

	"*": func(v1, v2 reflect.Value) (reflect.Value, error) {
		if !isNumber(v1) || !isNumber(v2) {
			return reflect.Value{}, errors.New("NaN")
		}
		switch {
		case isInt(v1) && isInt(v2):
			return reflect.ValueOf(v1.Int() * v2.Int()), nil
		default:
			return reflect.ValueOf(toFloat(v1) * toFloat(v2)), nil
		}
	},

	"/": func(v1, v2 reflect.Value) (reflect.Value, error) {
		if !isNumber(v1) || !isNumber(v2) {
			return reflect.Value{}, errors.New("NaN")
		}
		if toFloat(v2) == 0 {
			return reflect.Value{}, errors.New("division by zero")
		}
		switch {
		case isInt(v1) && isInt(v2):
			return reflect.ValueOf(v1.Int() / v2.Int()), nil
		default:
			return reflect.ValueOf(toFloat(v1) / toFloat(v2)), nil
		}
	},

	"%": func(v1, v2 reflect.Value) (reflect.Value, error) {
		if !isInt(v1) || !isInt(v2) {
			return reflect.Value{}, fmt.Errorf("reminder undefined for %T and %T", v1, v2)
		}
		if v2.Int() == 0 {
			return reflect.Value{}, errors.New("division by zero")
		}
		return reflect.ValueOf(v1.Int() % v2.Int()), nil
	},

	"or":  func(v1, v2 reflect.Value) bool { return (isTrue(v1) || isTrue(v2)) },
	"and": func(v1, v2 reflect.Value) bool { return (isTrue(v1) && isTrue(v2)) },
	"not": func(v1 reflect.Value, v2 interface{}) bool { return !isTrue(v1) },

	"range": func(v1, v2 reflect.Value) ([]interface{}, error) {
		var ret []interface{}
		var diff int64 = 1
		cmp := func(i1, i2 int64) bool { return i1 <= i2 }
		if v1.Kind() == reflect.String {
			i, err := strconv.Atoi(v1.String())
			if err != nil {
				return nil, errors.New("NaN")
			}
			v1 = reflect.ValueOf(i)
		}
		if v2.Kind() == reflect.String {
			i, err := strconv.Atoi(v2.String())
			if err != nil {
				return nil, errors.New("NaN")
			}
			v2 = reflect.ValueOf(i)
		}
		if v1.Int() > v2.Int() {
			diff = -1
			cmp = func(i1, i2 int64) bool { return i2 <= i1 }
		}
		for i := v1.Int(); cmp(i, v2.Int()); i += diff {
			ret = append(ret, int64(i))
		}
		return ret, nil
	},
}

type foreach struct {
	i, c int
}

func (f *foreach) HasNext() bool { return f.c < f.i }
func (f *foreach) First() bool   { return f.i == 0 }
func (f *foreach) Last() bool    { return f.i == f.c }
func (f *foreach) Count() int    { return f.i + 1 }
func (f *foreach) Index() int    { return f.i }

type iterator interface {
	Len() int
	Index(int) reflect.Value
}

type iter struct {
	reflect.Value
	i int
}

func (i *iter) Next() reflect.Value {
	var ret reflect.Value
	if i.i < i.Len() {
		ret = i.Index(i.i)
		i.i++
	}
	return ret
}

func vtlPrint(v reflect.Value) string {
	var ret string
	switch v.Kind() {
	case reflect.Float64, reflect.Float32:
		ret = fmt.Sprintf("%G", v.Float())
		if !strings.Contains(ret, ".") {
			ret += ".0"
		}
		return strings.Replace(ret, "+", "", 1)
	case reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8, reflect.Int:
		return fmt.Sprintf("%d", v.Int())
	case reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8, reflect.Uint:
		return fmt.Sprintf("%d", v.Uint())
	case reflect.Map:
		ret := "{"
		for _, key := range v.MapKeys() {
			ret += fmt.Sprintf("%s=%s", key, vtlPrint(v.MapIndex(key)))
		}
		ret += "}"
		return ret
	case reflect.Slice:
		ret := "["
		for i := 0; i < v.Len(); i++ {
			if i > 0 {
				ret += ", "
			}
			ret += fmt.Sprintf("%s", vtlPrint(v.Index(i)))
		}
		ret += "]"
		return ret
	case reflect.Interface:
		return vtlPrint(v.Elem())
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}

func idx(v1, v2 reflect.Value) (reflect.Value, error) {
	vv1 := indirect(v1)
	v2 = indirect(v2)
	switch vv1.Kind() {
	case reflect.Array, reflect.Slice, reflect.String:
		var x int64
		switch v2.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			x = v2.Int()
		case reflect.Float32, reflect.Float64:
			x = int64(v2.Float())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			x = int64(v2.Uint())
		default:
			return reflect.Value{}, fmt.Errorf("invalid array index: %q %T", v2, v2)
		}
		if x < 0 || x >= int64(vv1.Len()) {
			return reflect.Value{}, fmt.Errorf("index out of range")
		}
		return vv1.Index(int(x)), nil
	case reflect.Map:
		if !v2.Type().AssignableTo(vv1.Type().Key()) {
			return reflect.Value{}, fmt.Errorf("invalid map key: %q %T", v2, v2)
		}
		if x := vv1.MapIndex(v2); x.IsValid() {
			return x, nil
		} else {
			return reflect.Zero(vv1.Type().Elem()), nil
		}
	default:
		return property(v1, v2)
	}
}

func property(v1, v2 reflect.Value) (reflect.Value, error) {
	vv1 := indirect(v1)
	var err error
	if vv1.Kind() == reflect.Struct && v2.Kind() == reflect.String {
		ret := vv1.FieldByName(v2.String())
		if !ret.IsValid() {
			ret, err = call(v1, v2.String())
		}
		if !ret.IsValid() && err == nil && v1.MethodByName("Get").IsValid() {
			ret, err = call(v1, "Get", v2)
		}
		return ret, err
	}

	if v2.Kind() == reflect.String {
		return call(v1, v2.String())
	}
	return reflect.Value{}, fmt.Errorf("cannot get property %s of %s value", v2, v1.Kind())
}

func index(v reflect.Value, indices ...reflect.Value) (reflect.Value, error) {
	if !v.IsValid() {
		return reflect.Value{}, nil
	}
	var err error
	for _, i := range indices {
		v, err = idx(v, i)
		if err != nil {
			return v, err
		}
	}
	return v, nil
}

func call(v reflect.Value, meth string, args ...reflect.Value) (reflect.Value, error) {
	if !v.IsValid() {
		return reflect.Value{}, nil
	}
	trimm := strings.Title(strings.TrimPrefix(meth, "get"))
	var m reflect.Value
	for _, mm := range []string{meth, trimm, "Get" + trimm, "Is" + trimm} {
		m = v.MethodByName(mm)
		if m.IsValid() {
			break
		}
	}
	vv := indirect(v)
	switch {
	case m.IsValid():
		for i, val := range args {
			var argType reflect.Type
			if m.Type().IsVariadic() {
				argType = m.Type().In(m.Type().NumIn() - 1).Elem()
			} else {
				argType = m.Type().In(i)
			}
			if !val.IsValid() {
				return reflect.Value{}, fmt.Errorf("arg %d not valid", i)
			}
			if !val.Type().AssignableTo(argType) {
				if !val.Type().ConvertibleTo(argType) {
					return reflect.Value{}, fmt.Errorf("arg %d: %s %s -> %s", i, "not assignable", val.Type(), m.Type().In(i))
				} else {
					args[i] = val.Convert(argType)
				}
			}
		}
		ret := m.Call(args)
		if len(ret) != 0 {
			return ret[0], nil
		}
	case vv.Kind() == reflect.Struct:
		f := vv.FieldByName(trimm)
		if f.IsValid() {
			return f, nil
		}
	case vv.Kind() == reflect.Map:
		kType, vType := vv.Type().Key(), vv.Type().Elem()
		switch meth {
		case "size":
			return reflect.ValueOf(v.Len()), nil
		case "get":
			if len(args) != 1 {
				return reflect.Value{}, errors.New("no argument for map index specified")
			}
			switch {
			case args[0].Type().AssignableTo(kType):
				val := vv.MapIndex(args[0])
				if !val.IsValid() {
					return reflect.ValueOf(""), nil
				}
				return val, nil
			case isInt(args[0]):
				return vv.MapIndex(reflect.ValueOf(vtlPrint(args[0]))), nil
			}
		case "put":
			if len(args) == 2 && args[0].Type().AssignableTo(kType) && args[1].Type().AssignableTo(vType) {
				prev := vv.MapIndex(args[0])
				vv.SetMapIndex(args[0], args[1])
				return prev, nil
			}
		}
	case vv.Kind() == reflect.String:
		switch meth {
		case "length":
			return reflect.ValueOf(utf8.RuneCountInString(vv.String())), nil
		case "equals":
			if len(args) == 1 && args[0].Kind() == reflect.String {
				return reflect.ValueOf(vv.String() == args[0].String()), nil
			}
		}
	case vv.Kind() == reflect.Slice:
		switch meth {
		case "length":
			return reflect.ValueOf(v.Len()), nil
		case "iterator":
			if len(args) == 0 {
				return reflect.ValueOf(&iter{vv, 0}), nil
			}
		}
	}

	return reflect.Value{}, nil
}

func isInt(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}

func isFloat(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

func isNumber(v reflect.Value) bool {
	return isInt(v) || isFloat(v)
}

func indirect(v reflect.Value) reflect.Value {
	for ; v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface; v = v.Elem() {
		if v.IsNil() {
			return v
		}
	}
	return v
}

func isDirective(n Node) bool {
	switch n.(type) {
	case TextNode, *VarNode:
		return false
	}
	return true
}
