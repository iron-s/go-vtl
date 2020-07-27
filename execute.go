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
	"strconv"
	"strings"
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
		vv := reflect.ValueOf(v)
		ctx.Push(k, wrapTypes(vv))
	}
	_, err := t._execute(w, t.tree, ctx)
	return err
}

func wrapTypes(v reflect.Value) reflect.Value {
	switch v.Kind() {
	case reflect.Slice:
		s := &Slice{}
		for i := 0; i < v.Len(); i++ {
			s.S = append(s.S, v.Index(i).Interface())
		}
		return reflect.ValueOf(s)
	case reflect.Map:
		m := &Map{M: make(map[string]interface{})}
		it := v.MapRange()
		for it.Next() {
			k, v := it.Key(), it.Value()
			m.M[k.String()] = v.Interface()
		}
		return reflect.ValueOf(m)
	case reflect.String:
		return v.Convert(reflect.TypeOf(Str("")))
	case reflect.Interface:
		return wrapTypes(v.Elem())
	case reflect.Ptr:
		vv := wrapTypes(v.Elem())
		if vv == v.Elem() {
			return v
		}
		return vv
	default:
		return v
	}
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
			val, err := t.eval(n.Expr, ctx, false)
			if errors.As(err, &nilError{}) {
			} else if err != nil {
				return false, err
			}
			if len(n.Var.Items) == 0 {
				depth := ctx.Push(n.Var.Name, val)
				defer ctx.Pop(depth, n.Var.Name)
			} else if val.IsValid() {
				if err := t.setVar(n.Var, val, ctx); err != nil {
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
				// } else if !n.Silent {
				// val, _ := ctx.Get(n.Name)
				// fmt.Printf("%s %#v %v %v\n", n.Name, n.Items, val, err)
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
			vdepth := ctx.Push(n.Var.Name, reflect.ValueOf(nil))
			f := &foreach{}
			fdepth := ctx.Push("foreach", reflect.ValueOf(f))
			switch iter.Type() {
			case sliceType:
				f.it = iter.Interface().(*Slice).Iterator()
			case rangeType:
				f.it = iter.Interface().(*Range).Iterator()
			case mapType:
				f.it = iter.Interface().(*Map).Values().Iterator()
			case iteratorType:
				f.it = iter.Interface().(*Iterator)
			}
			empty := true
			for f.it.HasNext() {
				empty = false
				ctx.Set(vdepth, n.Var.Name, reflect.ValueOf(f.it.Next()))
				_, err := t._execute(w, n.Items, ctx)
				if err != nil {
					return true, err
				}
			}
			if empty && n.Else != nil {
				_, err := t._execute(w, n.Else, ctx)
				if err != nil {
					return true, err
				}
			}
			ctx.Pop(vdepth, n.Var.Name)
			ctx.Pop(fdepth, "foreach")
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
	if m.IsCall {
		v, err = call(v, m.Name, args...)
	} else {
		v, err = property(v, reflect.ValueOf(m.Name))
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
	case reflect.Chan, reflect.Func:
		if v.IsNil() {
			return fmt.Errorf("nil resul $%s", n.Name)
		}
	}
	switch {
	case !v.CanSet() && prev.Kind() == reflect.Ptr:
		k := reflect.ValueOf(n.Items[len(n.Items)-1].Name)
		if v.IsValid() && v.MethodByName("Set").IsValid() {
			_, err := call(v, "Set", k, val)
			return err
		} else if prev.MethodByName("Put").IsValid() {
			_, err := call(prev, "Put", k, val)
			return err
		}
		return fmt.Errorf("cannot set %s in $%s", k, n.Name)
	case prev.Kind() == reflect.Map:
		prev.SetMapIndex(reflect.ValueOf(n.Items[len(n.Items)-1].Name), val)
	case v.CanSet() && val.Type().ConvertibleTo(v.Type()):
		v.Set(val.Convert(v.Type()))
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
		switch e.Op {
		case "or":
			if isTrue(l) {
				return reflect.ValueOf(true), nil
			}
		case "and":
			if !isTrue(l) {
				return reflect.ValueOf(false), nil
			}
		}
		r, err := t.eval(e.Right, ctx, undefOk)
		if err != nil && !(undefOk && errors.As(err, &undefinedError{})) {
			return r, err
		}
		ret := f.Call([]reflect.Value{reflect.ValueOf(l), reflect.ValueOf(r)})
		if ret[0].Type() == reflectValueType {
			ret[0] = ret[0].Interface().(reflect.Value)
		}
		err = asError(ret[len(ret)-1].Interface())
		if err != nil {
			return reflect.Value{}, err
		}
		return wrapTypes(ret[0]), nil
	}
	switch val := e.Val.(type) {
	case *InterpolatedNode:
		var b bytes.Buffer
		_, err := t._execute(&b, val.Items, ctx)
		if err != nil {
			return reflect.Value{}, err
		}
		return wrapTypes(reflect.ValueOf(b.String())), nil
	case *VarNode:
		return t.evalVar(val, ctx)
	case nil:
	case int64, float64, bool:
		return reflect.ValueOf(val), nil
	case string:
		return reflect.ValueOf(Str(val)), nil
	case *RefNode:
		return ctx.Get(val.Name)
	case []*OpNode:
		vv := make([]interface{}, len(val))
		for i := range val {
			e, err := t.eval(val[i], ctx, false)
			if err != nil {
				return reflect.Value{}, err
			}
			vv[i] = e.Interface()
		}
		return reflect.ValueOf(&Slice{vv}), nil
	default:
		log.Printf("unsupported type %T: %v", val, val)
		return wrapTypes(reflect.ValueOf(val)), nil
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

	"map": func(v1, v2 reflect.Value) (reflect.Value, error) {
		val := v1.Interface().(*Slice).S
		m := make(map[string]interface{}, len(val)/2)
		for i := 0; i < len(val); i += 2 {
			k, v := val[i], val[i+1]
			s := vtlPrint(reflect.ValueOf(k))
			m[s] = v
		}
		return reflect.ValueOf(&Map{m}), nil
	},
	"list": func(v1, v2 reflect.Value) reflect.Value { return v1 },

	"range": func(v1, v2 reflect.Value) (reflect.Value, error) {
		if v1.Kind() == reflect.String {
			i, err := strconv.Atoi(v1.String())
			if err != nil {
				return reflect.Value{}, errors.New("NaN")
			}
			v1 = reflect.ValueOf(i)
		}
		if v2.Kind() == reflect.String {
			i, err := strconv.Atoi(v2.String())
			if err != nil {
				return reflect.Value{}, errors.New("NaN")
			}
			v2 = reflect.ValueOf(i)
		}
		return reflect.ValueOf(NewRange(int(v1.Int()), int(v2.Int()))), nil
	},
}

type foreach struct {
	it *Iterator
}

func (f *foreach) HasNext() bool { return f.it.HasNext() }
func (f *foreach) First() bool   { return f.it.i == 1 }
func (f *foreach) Last() bool    { return f.it.i == f.it.s.Size()-1 }
func (f *foreach) Count() int    { return f.it.i }
func (f *foreach) Index() int    { return f.it.i - 1 }

var (
	sliceType     = reflect.TypeOf((*Slice)(nil))
	rangeType     = reflect.TypeOf((*Range)(nil))
	mapType       = reflect.TypeOf((*Map)(nil))
	entryType     = reflect.TypeOf((*MapEntry)(nil))
	viewType      = reflect.TypeOf((*View)(nil))
	keyViewType   = reflect.TypeOf((*KeyView)(nil))
	entryViewType = reflect.TypeOf((*EntryView)(nil))
	valViewType   = reflect.TypeOf((*ValView)(nil))
	iteratorType  = reflect.TypeOf((*Iterator)(nil))
)

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
	case reflect.Ptr:
		var b bytes.Buffer
		switch v.Type() {
		case mapType:
			m := v.Interface().(*Map)
			b.WriteByte('{')
			entries := m.EntrySet().Slice.S
			for i, e := range entries {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString(vtlPrint(reflect.ValueOf(e)))
			}
			b.WriteByte('}')
		case entryType:
			e := v.Interface().(*MapEntry)
			b.WriteString(e.k)
			b.WriteByte('=')
			b.WriteString(vtlPrint(reflect.ValueOf(e.v)))
		case viewType, keyViewType, entryViewType, valViewType:
			s := v.Elem().FieldByName("Slice")
			return vtlPrint(s)
		case sliceType, rangeType:
			s := v.Interface().(Collection)
			b.WriteByte('[')
			it := s.Iterator()
			for it.HasNext() {
				b.WriteString(vtlPrint(reflect.ValueOf(it.Next())))
				if it.HasNext() {
					b.WriteString(", ")
				}
			}
			b.WriteByte(']')
		default:
			if v.Type().Implements(reflect.TypeOf((*fmt.Stringer)(nil)).Elem()) {
				return fmt.Sprintf("%v", v.Interface())
			}
			return vtlPrint(indirect(v))
		}
		return b.String()
	case reflect.Map:
		return "use of naked map"
	case reflect.Slice:
		return "use of naked slice"
	case reflect.Interface:
		return vtlPrint(v.Elem())
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}

func property(v1, v2 reflect.Value) (reflect.Value, error) {
	vv1 := indirect(v1)
	var (
		ret reflect.Value
		err error
	)
	if v2.Kind() == reflect.String {
		if vv1.Kind() == reflect.Struct {
			ret = vv1.FieldByName(v2.String())
		}
		if !ret.IsValid() {
			ret, err = call(v1, v2.String())
		}
	}
	if !ret.IsValid() && v1.MethodByName("Get").IsValid() {
		ret, err = call(v1, "Get", v2)
	}
	if err == nil {
		return ret, err
	}
	return reflect.Value{}, fmt.Errorf("cannot get property %s of %s value", v2, v1.Kind())
}

func call(v reflect.Value, meth string, args ...reflect.Value) (reflect.Value, error) {
	if !v.IsValid() {
		return reflect.Value{}, fmt.Errorf("cannot call %s on nil value", meth)
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
		if err := compatible(m, args...); err != nil {
			return reflect.Value{}, err
		}
		ret := m.Call(args)
		if len(ret) > 0 {
			err := asError(ret[len(ret)-1].Interface())
			if err != nil {
				return reflect.Value{}, err
			}
			return wrapTypes(ret[0]), nil
		}
	case vv.Kind() == reflect.Struct:
		f := vv.FieldByName(trimm)
		if f.IsValid() {
			return wrapTypes(f), nil
		}
	case vv.Type() == reflect.TypeOf(""):
		return reflect.Value{}, errors.New("naked string is not supported")
	case vv.Kind() == reflect.Map:
		return reflect.Value{}, errors.New("naked map is not supported")
	case vv.Kind() == reflect.Slice:
		return reflect.Value{}, errors.New("naked map is not supported")
	}

	return reflect.Value{}, nil
}

func asError(v interface{}) error {
	err, ok := v.(error)
	if ok {
		return err
	}
	return nil
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

func isDirective(n Node) bool {
	switch n.(type) {
	case TextNode, *VarNode:
		return false
	}
	return true
}

func comparable(k1, k2 reflect.Value) bool {
	switch {
	case k1.Kind() == k2.Kind():
		return true
	case isNumber(k1) && isNumber(k2):
		return true
	default:
		return false
	}
}

func compatible(f reflect.Value, args ...reflect.Value) error {
	for i, val := range args {
		var argType reflect.Type
		if f.Type().IsVariadic() {
			argType = f.Type().In(f.Type().NumIn() - 1).Elem()
		} else {
			argType = f.Type().In(i)
		}
		if !val.IsValid() {
			return fmt.Errorf("arg %d not valid", i)
		}
		valType := val.Type()
		if !valType.AssignableTo(argType) {
			if valType.ConvertibleTo(argType) {
				args[i] = val.Convert(argType)
			} else {
				return fmt.Errorf("arg %d: %s %s -> %s", i, "not assignable", valType, f.Type().In(i))
			}
		}
	}
	return nil
}
