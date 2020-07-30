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
	"sync"
	// "github.com/davecgh/go-spew/spew"
)

var bufPool = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

type undefinedError struct {
	error
}
type nilError struct {
	error
}

func (t *Template) Execute(w io.Writer, val map[string]interface{}) error {
	ctx := NewContext()
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
	if t.maxCallDepth >= 0 && ctx.callDepth > t.maxCallDepth {
		return true, errors.New("call depth exceeded")
	}
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
				b := bufPool.Get().(*bytes.Buffer)
				b.Reset()
				err := vtlPrint(b, v, nil)
				if err != nil {
					return true, err
				}
				b.WriteTo(w)
				bufPool.Put(b)
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
			if len(n.Vals) < len(m.Assign) {
				return false, fmt.Errorf("variable $%s has not been set", m.Assign[len(n.Vals)].Name)
			}
			for i := range m.Assign {
				v, err := t.eval(n.Vals[i], ctx, false)
				if err != nil {
					return true, err
				}
				depth := ctx.Push(m.Assign[i].Name, v)
				defer ctx.Pop(depth, m.Assign[i].Name)
			}
			ctx.callDepth++
			stop, err := t._execute(w, m.Items, ctx)
			ctx.callDepth--
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
			default:
				typ := iter.Type().String()
				if m := iter.MethodByName("Kind"); m.IsValid() && m.Type().NumIn() == 0 {
					ret := m.Call(nil)
					if len(ret) >= 1 && ret[0].Kind() == reflect.String {
						typ = ret[0].String()
					}
				}
				return true, fmt.Errorf("cannot iterate over %s", typ)
			}
			empty := true
			for f.it.HasNext() {
				if t.maxIterations >= 0 && f.Count() > t.maxIterations {
					return true, errors.New("number of iterations exceeded")
				}
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
				var file string
				switch {
				case name.Kind() == reflect.String:
					file = name.String()
				case name.Type().Implements(reflect.TypeOf((*fmt.Stringer)(nil)).Elem()):
					file = fmt.Sprintf("%v", name.Interface())
				default:
					return false, errors.New("invalid include argument")
				}

				data, err := ioutil.ReadFile(filepath.Join(t.root, file))
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
			ctx.callDepth++
			stop, err := tmpl._execute(w, tmpl.tree, ctx)
			ctx.callDepth--
			if stop {
				return true, err
			}
		default:
			log.Printf("unexpected %T, %[1]v", n)
		}
	}
	return false, nil
}

func (t *Template) evalStep(v reflect.Value, m *AccessNode, wrapT bool, ctx Ctx) (reflect.Value, error) {
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
		v, err = t.call(v, m.Name, args...)
	} else {
		v, err = t.property(v, reflect.ValueOf(m.Name))
	}
	if err != nil {
		return reflect.Value{}, err
	}
	if wrapT {
		v = wrapTypes(v)
	}
	return v, nil
}

func (t *Template) evalVar(n *VarNode, ctx Ctx) (reflect.Value, error) {
	v, err := ctx.Get(n.Name)
	if err != nil {
		return v, err
	}
	for _, m := range n.Items {
		if v, err = t.evalStep(v, m, true, ctx); err != nil {
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
	v, err := ctx.Get(n.Name)
	if err != nil && len(n.Items) > 0 {
		return err
	}
	var prev reflect.Value
	for _, m := range n.Items {
		prev = v
		if v, err = t.evalStep(v, m, false, ctx); err != nil {
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
			_, err := t.call(v, "Set", k, val)
			return err
		} else if prev.MethodByName("Put").IsValid() {
			_, err := t.call(prev, "Put", k, val)
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
		b := bufPool.Get().(*bytes.Buffer)
		b.Reset()
		defer bufPool.Put(b)
		_, err := t._execute(b, val.Items, ctx)
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
			if e.IsValid() {
				vv[i] = e.Interface()
			}
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

func toInt(v reflect.Value) int {
	switch v.Kind() {
	case reflect.Float32, reflect.Float64:
		return int(v.Float())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int(v.Uint())
	default:
		return int(v.Int())
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
		b := bufPool.Get().(*bytes.Buffer)
		defer bufPool.Put(b)
		b.Reset()
		for i := 0; i < len(val); i += 2 {
			k, v := val[i], val[i+1]
			err := vtlPrint(b, reflect.ValueOf(k), nil)
			if err != nil {
				return reflect.Value{}, err
			}
			m[b.String()] = v
			b.Reset()
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
		if !isNumber(v1) || !isNumber(v2) {
			return reflect.Value{}, errors.New("NaN")
		}
		return reflect.ValueOf(NewRange(toInt(v1), toInt(v2))), nil
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

func checkCycle(value reflect.Value, path []uintptr) bool {
	if value.Kind() == reflect.Interface {
		value = value.Elem()
	}
	switch value.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Map:
		addr := value.Pointer()
		for i := range path {
			if path[i] == addr {
				return true
			}
		}
	}
	return false
}

func vtlPrint(b *bytes.Buffer, v reflect.Value, path []uintptr) error {
	if checkCycle(v, path) {
		return errors.New("cycle detected")
	}
	switch v.Kind() {
	case reflect.Float64, reflect.Float32:
		buf := bufPool.Get().(*bytes.Buffer)
		buf.Reset()
		fmt.Fprintf(buf, "%G", v.Float())
		bb := buf.Bytes()
		b.Write(bytes.Replace(bb, []byte("+"), nil, 1))
		if !bytes.Contains(bb, []byte(".")) {
			b.WriteString(".0")
		}
		bufPool.Put(buf)
	case reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8, reflect.Int:
		fmt.Fprintf(b, "%d", v.Int())
	case reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8, reflect.Uint:
		fmt.Fprintf(b, "%d", v.Uint())
	case reflect.Ptr:
		path = append(path, v.Pointer())
		switch v.Type() {
		case mapType:
			m := v.Interface().(*Map)
			b.WriteByte('{')
			entries := m.EntrySet().Slice.S
			for i, e := range entries {
				if i > 0 {
					b.WriteString(", ")
				}
				err := vtlPrint(b, reflect.ValueOf(e), path)
				if err != nil {
					return err
				}
			}
			b.WriteByte('}')
		case entryType:
			e := v.Interface().(*MapEntry)
			b.WriteString(e.k)
			b.WriteByte('=')
			return vtlPrint(b, reflect.ValueOf(e.v), path)
		case viewType, keyViewType, entryViewType, valViewType:
			s := v.Elem().FieldByName("Slice")
			return vtlPrint(b, s, path)
		case sliceType, rangeType:
			s := v.Interface().(Collection)
			b.WriteByte('[')
			it := s.Iterator()
			for it.HasNext() {
				err := vtlPrint(b, reflect.ValueOf(it.Next()), path)
				if err != nil {
					return err
				}
				if it.HasNext() {
					b.WriteString(", ")
				}
			}
			b.WriteByte(']')
		default:
			if v.Type().Implements(reflect.TypeOf((*fmt.Stringer)(nil)).Elem()) {
				fmt.Fprintf(b, "%v", v.Interface())
				return nil
			}
			return vtlPrint(b, indirect(v), path)
		}
		return nil
	case reflect.Map:
		return errors.New("use of naked map")
	case reflect.Slice:
		return errors.New("use of naked slice")
	case reflect.Interface:
		return vtlPrint(b, v.Elem(), path)
	case reflect.Invalid:
		b.WriteString("null")
	default:
		fmt.Fprintf(b, "%v", v.Interface())
	}
	return nil
}

func (t *Template) property(v1, v2 reflect.Value) (reflect.Value, error) {
	vv1 := indirect(v1)
	var (
		ret reflect.Value
		err error
	)
	if v2.Kind() == reflect.String {
		if vv1.Kind() == reflect.Struct {
			f, ok := vv1.Type().FieldByName(v2.String())
			if ok && f.PkgPath == "" {
				ret = vv1.FieldByName(v2.String())
			}
		}
		if !ret.IsValid() {
			ret, err = t.call(v1, v2.String())
		}
	}
	if !ret.IsValid() && v1.IsValid() && v1.MethodByName("Get").IsValid() {
		ret, err = t.call(v1, "Get", v2)
	}
	if err == nil {
		return ret, err
	}
	return reflect.Value{}, fmt.Errorf("cannot get property %s of %s value", v2, v1.Kind())
}

func (t *Template) call(v reflect.Value, meth string, args ...reflect.Value) (reflect.Value, error) {
	if !v.IsValid() {
		return reflect.Value{}, fmt.Errorf("cannot call %s on nil value", meth)
	}
	t.cacheMutex.Lock()
	types, ok := t.typeCache[v.Type()]
	if !ok {
		n := v.NumMethod()
		types = make([]methodIdx, n)
		for i := 0; i < n; i++ {
			typ := v.Type().Method(i)
			types[i] = methodIdx{typ.Name, typ.Index}
		}
		sort.Slice(types, func(i, j int) bool { return types[i].name < types[j].name })
		t.typeCache[v.Type()] = types
	}
	t.cacheMutex.Unlock()
	trimm := strings.Title(strings.TrimPrefix(meth, "get"))
	var m reflect.Value
	for _, mm := range []string{meth, trimm, "Get" + trimm, "Is" + trimm} {
		i := sort.Search(len(types), func(i int) bool { return types[i].name >= mm })
		if i < len(types) && types[i].name == mm {
			m = v.Method(types[i].i)
			if m.IsValid() {
				break
			}
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
			return ret[0], nil
		}
	case vv.Kind() == reflect.Struct:
		f := vv.FieldByName(trimm)
		if f.IsValid() {
			return f, nil
		}
	case vv.Type() == reflect.TypeOf(""):
		return reflect.Value{}, errors.New("naked string is not supported")
	case vv.Kind() == reflect.Map:
		return reflect.Value{}, errors.New("naked map is not supported")
	case vv.Kind() == reflect.Slice:
		return reflect.Value{}, errors.New("naked map is not supported")
	default:
		return reflect.Value{}, fmt.Errorf("cannot call %s on %s value", meth, v.Type().String())
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
	if len(args) < f.Type().NumIn() || (!f.Type().IsVariadic() && len(args) > f.Type().NumIn()) {
		return errors.New("incompatible number of arguments")
	}
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
