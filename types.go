package govtl

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var Nil = reflect.Value{}

var errMapExpected = errors.New("map expected")

type Map struct {
	M interface{}
}

func (m *Map) Kind() string { return "map" }

func (m *Map) Clear() {
	m.M = reflect.MakeMap(reflect.TypeOf(m.M)).Interface()
}

func (m *Map) ContainsKey(key interface{}) bool {
	v := reflect.ValueOf(m.M).MapIndex(reflect.ValueOf(key))
	return v != Nil
}

func (m *Map) ContainsValue(val interface{}) bool {
	iter := reflect.ValueOf(m.M).MapRange()
	for iter.Next() {
		v := iter.Value()
		if reflect.DeepEqual(v.Interface(), val) {
			return true
		}
	}
	return false
}

func (m *Map) EntrySet() *EntryView {
	return &EntryView{m: m}
}

func (m *Map) Equals(v interface{}) (bool, error) {
	vv, ok := v.(*Map)
	if !ok {
		return false, errMapExpected
	}
	mM, vM := reflect.ValueOf(m.M), reflect.ValueOf(vv.M)
	if mM.Len() != vM.Len() {
		return false, nil
	}
	iter := mM.MapRange()
	for iter.Next() {
		vV := vM.MapIndex(iter.Key())
		if vV == Nil || !reflect.DeepEqual(vV.Interface(), iter.Value().Interface()) {
			return false, nil
		}
	}
	return true, nil
}

func (m *Map) Get(key interface{}) interface{} {
	return m.GetOrDefault(key, nil)
}

func (m *Map) GetOrDefault(key interface{}, deflt interface{}) interface{} {
	mM := reflect.ValueOf(m.M)
	k := reflect.ValueOf(key)
	switch {
	case k.Type().AssignableTo(mM.Type().Key()):
		k = k.Convert(mM.Type().Key())
	case mM.Type().Key() != k.Type() && mM.Type().Key().Kind() == reflect.String:
		k = reflect.ValueOf(fmt.Sprint(key))
	}
	v := mM.MapIndex(k)
	if v == Nil {
		return deflt
	}
	return v.Interface()
}

func (m *Map) IsEmpty() bool {
	return reflect.ValueOf(m.M).Len() == 0
}

func (m *Map) KeySet() *KeyView {
	return &KeyView{m: m}
}

func (m *Map) Put(key interface{}, value interface{}) interface{} {
	mM := reflect.ValueOf(m.M)
	kk := reflect.ValueOf(key)
	vv := reflect.ValueOf(value)
	if kk.Type().ConvertibleTo(mM.Type().Elem()) {
		kk = kk.Convert(mM.Type().Key())
	}
	if vv.Type().ConvertibleTo(mM.Type().Elem()) {
		vv = vv.Convert(mM.Type().Elem())
	}
	was := mM.MapIndex(kk)
	mM.SetMapIndex(kk, vv)
	if was == Nil {
		return nil
	}
	return was.Interface()
}

func (m *Map) PutAll(v interface{}) error {
	vv, ok := v.(*Map)
	if !ok {
		return errMapExpected
	}
	mM, vM := reflect.ValueOf(m.M), reflect.ValueOf(vv.M)
	iter := vM.MapRange()
	for iter.Next() {
		mM.SetMapIndex(iter.Key(), iter.Value())
	}
	return nil
}

func (m *Map) PutIfAbsent(key interface{}, value interface{}) interface{} {
	v := m.Get(key)
	if v == nil {
		m.Put(key, value)
	}
	return v
}

func (m *Map) Remove(key interface{}) interface{} {
	mM := reflect.ValueOf(m.M)
	v := m.Get(key)
	mM.SetMapIndex(reflect.ValueOf(key), Nil)
	return v
}

func (m *Map) Replace(key interface{}, val interface{}) interface{} {
	mM := reflect.ValueOf(m.M)
	v := mM.MapIndex(reflect.ValueOf(key))
	if v != Nil {
		m.Put(key, val)
		return v.Interface()
	}
	return nil
}

func (m *Map) Size() int {
	return reflect.ValueOf(m.M).Len()
}

func (m *Map) Values() *ValView {
	return &ValView{m: m}
}

type MapEntry struct {
	k interface{}
	v interface{}
	m *Map
}

func (e *MapEntry) Kind() string { return "mapEntry" }

func (e *MapEntry) Equals(entry *MapEntry) bool {
	return reflect.DeepEqual(e.k, entry.k) && reflect.DeepEqual(e.v, entry.v)
}

func (e *MapEntry) GetKey() interface{} {
	return e.k
}

func (e *MapEntry) GetValue() interface{} {
	return e.v
}

func (e *MapEntry) SetValue(val interface{}) interface{} {
	v := e.v
	vv := reflect.ValueOf(val)
	reflect.ValueOf(e.m.M).SetMapIndex(reflect.ValueOf(e.k), vv)
	e.v = vv
	return v
}

type View struct {
	m *Map
}

var errUnsupported = errors.New("unsupported operation")

func (v *View) Add(interface{}) (bool, error) {
	return false, errUnsupported
}

func (v *View) AddAll(interface{}) (bool, error) {
	return false, errUnsupported
}

func (v *View) Clear() {
	v.m.Clear()
}

type KeyView View

func (view *KeyView) Contains(val interface{}) bool {
	return view.m.ContainsKey(val)
}

func (view *KeyView) Iterator() Iterator {
	return NewMapIterator(view.m,
		func(m, k reflect.Value) interface{} {
			return k.Interface()
		})
}

func (v *KeyView) Kind() string { return "keyView" }

func (view *KeyView) Remove(k interface{}) bool {
	ok := view.m.ContainsKey(k)
	if ok {
		view.m.Remove(k)
	}
	return ok
}

func (view *KeyView) RemoveAll(v interface{}) (bool, error) {
	vv, ok := v.(*Slice)
	if !ok {
		return false, errArrayExpected
	}
	return removeAll(view.Iterator(), vv)
}

func (view *KeyView) RetainAll(v interface{}) (bool, error) {
	vv, ok := v.(*Slice)
	if !ok {
		return false, errArrayExpected
	}
	return retainAll(view.Iterator(), vv)
}

type ValView View

func (view *ValView) Contains(val interface{}) bool {
	return view.m.ContainsValue(val)
}

func (view *ValView) Iterator() Iterator {
	return NewMapIterator(view.m,
		func(m, k reflect.Value) interface{} {
			return m.MapIndex(k).Interface()
		})
}

func (view *ValView) Kind() string { return "valView" }

func (view *ValView) Remove(val interface{}) bool {
	mM := reflect.ValueOf(view.m.M)
	var found bool
	iter := mM.MapRange()
	for iter.Next() {
		v := iter.Value()
		if reflect.DeepEqual(v.Interface(), val) {
			mM.SetMapIndex(iter.Key(), Nil)
			found = true
			break
		}
	}
	return found
}

func (view *ValView) RemoveAll(v interface{}) (bool, error) {
	vv, ok := v.(*Slice)
	if !ok {
		return false, errArrayExpected
	}
	return removeAll(view.Iterator(), vv)
}

func (view *ValView) RetainAll(v interface{}) (bool, error) {
	vv, ok := v.(*Slice)
	if !ok {
		return false, errArrayExpected
	}
	return retainAll(view.Iterator(), vv)
}

type EntryView View

func (view *EntryView) Contains(val interface{}) bool {
	me, ok := val.(*MapEntry)
	if !ok {
		return false
	}
	return view.m.ContainsKey(me.k) && reflect.DeepEqual(view.m.Get(me.k), me.v)
}

func (view *EntryView) Iterator() Iterator {
	return NewMapIterator(view.m,
		func(m, k reflect.Value) interface{} {
			return &MapEntry{k: k.Interface(), v: m.MapIndex(k).Interface(), m: view.m}
		})
}

func (v *EntryView) Kind() string { return "entryView" }

func (view *EntryView) Remove(val *MapEntry) bool {
	mM := reflect.ValueOf(view.m.M)
	v := mM.MapIndex(reflect.ValueOf(val.k))
	if v != Nil && reflect.DeepEqual(v.Interface(), val.v) {
		mM.SetMapIndex(reflect.ValueOf(val.k), Nil)
		return true
	}
	return false
}

func (view *EntryView) RemoveAll(v interface{}) (bool, error) {
	vv, ok := v.(*Slice)
	if !ok {
		return false, errArrayExpected
	}
	return removeAll(view.Iterator(), vv)
}

func (view *EntryView) RetainAll(v interface{}) (bool, error) {
	vv, ok := v.(*Slice)
	if !ok {
		return false, errArrayExpected
	}
	return retainAll(view.Iterator(), vv)
}

var errArrayExpected = errors.New("array expected")

type Slice struct {
	S interface{}
}

func (s *Slice) Kind() string { return "slice" }

func (s *Slice) Add(v interface{}) (bool, error) {
	s.S = reflect.Append(reflect.ValueOf(s.S), reflect.ValueOf(v)).Interface()
	return true, nil
}

func (s *Slice) AddAll(v interface{}) (bool, error) {
	vv, ok := v.(*Slice)
	if !ok {
		return false, errArrayExpected
	}
	s.S = reflect.AppendSlice(reflect.ValueOf(s.S), reflect.ValueOf(vv.S)).Interface()
	return true, nil
}

func (s *Slice) Clear() error {
	sS := reflect.ValueOf(s.S)
	if sS.IsValid() && sS.Kind() == reflect.Slice {
		s.S = reflect.Zero(sS.Type()).Interface()
	} else {
		s.S = reflect.ValueOf([]interface{}(nil)).Interface()
	}
	return nil
}

func (s *Slice) Contains(v interface{}) bool {
	sS := reflect.ValueOf(s.S)
	comparer := reflect.DeepEqual
	// special case - if this is slice of MapEntry we should use Equals method
	vv := reflect.ValueOf(v)
	for vv.Kind() == reflect.Interface {
		vv = vv.Elem()
	}

	if vv.IsValid() && vv.Type() == entryType {
		comparer = func(x, y interface{}) bool {
			ye, ok := y.(*MapEntry)
			return ok && x.(*MapEntry).Equals(ye)
		}
	}
	for i := 0; i < sS.Len(); i++ {
		elt := sS.Index(i)
		for elt.Kind() == reflect.Interface && elt.Elem().IsValid() {
			elt = elt.Elem()
		}
		if comparer(elt.Interface(), v) {
			return true
		}
	}
	return false
}

func (s *Slice) ContainsAll(v interface{}) (bool, error) {
	return containsAll(s, v)
}

func (s *Slice) Equals(v interface{}) (bool, error) {
	return equals(s, v)
}

func (s *Slice) Get(i int) (interface{}, error) {
	sS := reflect.ValueOf(s.S)
	if i >= 0 && i < sS.Len() {
		return sS.Index(i).Interface(), nil
	}
	return nil, fmt.Errorf("index out of range %d with length %d", i, sS.Len())
}

func (s *Slice) IsEmpty() bool { return reflect.ValueOf(s.S).Len() == 0 }

func (s *Slice) Iterator() Iterator { return &CollectionIterator{s: s} }

func (s *Slice) Remove(v interface{}) (bool, error) {
	sS := reflect.ValueOf(s.S)
	if !sS.IsValid() {
		return false, nil
	}
	for i := 0; i < sS.Len(); i++ {
		if reflect.DeepEqual(sS.Index(i).Interface(), v) {
			s.S = reflect.AppendSlice(sS.Slice(0, i), sS.Slice(i+1, sS.Len())).Interface()
			return true, nil
		}
	}
	return false, nil
}

func (s *Slice) removeAt(i int) error {
	sS := reflect.ValueOf(s.S)
	if !sS.IsValid() {
		return fmt.Errorf("index out of range %d with length %d", i, 0)
	}
	s.S = reflect.AppendSlice(sS.Slice(0, i), sS.Slice(i+1, sS.Len())).Interface()
	return nil
}

func (s *Slice) RemoveAll(v interface{}) (bool, error) {
	vv, ok := v.(Collection)
	if !ok {
		return false, errArrayExpected
	}
	return removeAll(s.Iterator(), vv)
}

func (s *Slice) RetainAll(v interface{}) (bool, error) {
	vv, ok := v.(*Slice)
	if !ok {
		return false, errArrayExpected
	}
	return retainAll(s.Iterator(), vv)
}

func (s *Slice) Set(i int, v interface{}) (interface{}, error) {
	sS := reflect.ValueOf(s.S)
	if i >= 0 && i < sS.Len() {
		r := sS.Index(i).Interface()
		sS.Index(i).Set(reflect.ValueOf(v))
		return r, nil
	}
	return nil, fmt.Errorf("index out of range %d with length %d", i, sS.Len())
}

func (s *Slice) Size() int {
	return reflect.ValueOf(s.S).Len()
}

func (s *Slice) ToArray() (*Slice, error) {
	sS := reflect.ValueOf(s.S)
	if sS.IsNil() {
		return &Slice{}, nil
	}
	ss := reflect.MakeSlice(sS.Type(), sS.Len(), sS.Len())
	reflect.Copy(ss, sS)
	return &Slice{ss.Interface()}, nil
}

type Iterator interface {
	Next() interface{}
	HasNext() bool
	Remove()
}

type CollectionIterator struct {
	s Collection
	i int
}

func NewIterator(v interface{}) Iterator {
	vv := wrapTypes(reflect.ValueOf(v))
	switch s := vv.Interface().(type) {
	case *Slice:
		return &CollectionIterator{s: s}
	default:
		ss := reflect.MakeSlice(reflect.SliceOf(vv.Type()), 1, 1)
		ss.Index(0).Set(vv)
		return &CollectionIterator{s: &Slice{S: ss.Interface()}}
	}
}

func (it *CollectionIterator) Kind() string { return "iterator" }

func (it *CollectionIterator) Next() interface{} {
	if it.i < it.s.Size() {
		it.i++
		r, _ := it.s.Get(it.i - 1)
		return r
	}
	return nil
}

func (it *CollectionIterator) HasNext() bool { return it.i < it.s.Size() }

func (it *CollectionIterator) Remove() {
	it.s.removeAt(it.i - 1)
	it.i--
}

type MapIterator struct {
	mM     reflect.Value
	mapper func(m, k reflect.Value) interface{}
	k      []reflect.Value
	i      int
}

func NewMapIterator(m *Map, mapper func(m, k reflect.Value) interface{}) *MapIterator {
	mM := reflect.ValueOf(m.M)
	keys := mM.MapKeys()
	kind := basicKind(reflect.Zero(mM.Type().Key()))
	sort.Slice(keys, func(i, j int) bool {
		switch kind {
		case reflect.String:
			return keys[i].String() < keys[j].String()
		case reflect.Int64:
			return keys[i].Int() < keys[j].Int()
		case reflect.Uint64:
			return keys[i].Uint() < keys[j].Uint()
		case reflect.Float64:
			return keys[i].Float() < keys[j].Float()
		case reflect.Bool:
			return keys[i].Bool() && !keys[j].Bool()
		default:
			return true
		}
	})
	return &MapIterator{mM: mM, k: keys, mapper: mapper}
}
func (it *MapIterator) HasNext() bool { return it.i < len(it.k) }
func (it *MapIterator) Next() interface{} {
	it.i++
	return it.mapper(it.mM, it.k[it.i-1])
}
func (it *MapIterator) Remove() { it.mM.SetMapIndex(it.k[it.i-1], Nil) }

type Collection interface {
	Add(v interface{}) (bool, error)
	AddAll(interface{}) (bool, error)
	Clear() error
	Contains(interface{}) bool
	ContainsAll(interface{}) (bool, error)
	Equals(v interface{}) (bool, error)
	Get(i int) (interface{}, error)
	IsEmpty() bool
	Iterator() Iterator
	Remove(interface{}) (bool, error)
	removeAt(int) error
	RemoveAll(interface{}) (bool, error)
	RetainAll(interface{}) (bool, error)
	Set(int, interface{}) (interface{}, error)
	Size() int
	ToArray() (*Slice, error)
}

func containsAll(c Collection, v interface{}) (bool, error) {
	vv, ok := v.(Collection)
	if !ok {
		return false, errArrayExpected
	}
	if err := checkSize(vv); err != nil {
		return false, err
	}
	for i := 0; i < vv.Size(); i++ {
		o, _ := vv.Get(i)
		if !c.Contains(o) {
			return false, nil
		}
	}
	return true, nil
}

func equals(c Collection, v interface{}) (bool, error) {
	vv, ok := v.(Collection)
	if !ok {
		return false, errArrayExpected
	}
	if c.Size() != vv.Size() {
		return false, nil
	}
	if err := checkSize(vv); err != nil {
		return false, err
	}
	for i := 0; i < vv.Size(); i++ {
		o, _ := vv.Get(i)
		t, _ := c.Get(i)
		if !reflect.DeepEqual(o, t) {
			return false, nil
		}
	}
	return true, nil
}

func retainAll(it Iterator, c Collection) (bool, error) {
	var changed bool
	for it.HasNext() {
		if !c.Contains(it.Next()) {
			it.Remove()
			changed = true
		}
	}
	return changed, nil
}

func removeAll(it Iterator, c Collection) (bool, error) {
	var changed bool
	for it.HasNext() {
		v := it.Next()
		if c.Contains(v) {
			it.Remove()
			changed = true
		}
	}
	return changed, nil
}

var errNotImplemented = errors.New("not implemented")

var lower = cases.Lower(language.Und)
var upper = cases.Upper(language.Und)

type Str string

func (s Str) Kind() string { return "string" }

func (s Str) CharAt(i int) (rune, error) {
	r := []rune(s)
	if i < 0 || i >= len(r) {
		return 0, fmt.Errorf("index out of range %d with length %d", i, len(r))
	}
	return r[i], nil
}
func (s Str) CodePointAt(i int) error             { return errNotImplemented }
func (s Str) CodePointBefore(i int) error         { return errNotImplemented }
func (s Str) CodePointCount(start, end int) error { return errNotImplemented }

func (s Str) compare(o string, tr func(rune) rune) int {
	rs, ro := []rune(s), []rune(o)
	l := len(rs)
	if len(ro) < l {
		l = len(ro)
	}
	for i := 0; i < l; i++ {
		diff := int(tr(rs[i])) - int(tr(ro[i]))
		if diff != 0 {
			return diff
		}
	}
	diff := len(rs) - len(ro)
	return diff
}

func (s Str) CompareTo(o string) int {
	return s.compare(o, func(r rune) rune { return r })
}

func (s Str) CompareToIgnoreCase(o string) int {
	return s.compare(o, func(r rune) rune { return unicode.ToLower(unicode.ToUpper(r)) })
}

func (s Str) Concat(o string) string         { return string(s) + o }
func (s Str) Contains(o string) bool         { return strings.Contains(string(s), o) }
func (s Str) ContentEquals(o string) bool    { return string(s) == o }
func (s Str) EndsWith(suffix string) bool    { return strings.HasSuffix(string(s), suffix) }
func (s Str) Equals(o string) bool           { return string(s) == o }
func (s Str) EqualsIgnoreCase(o string) bool { return s.CompareToIgnoreCase(o) == 0 }
func (s Str) GetBytes() []byte               { return []byte(s) }
func (s Str) IndexOf(o string) int {
	i := strings.Index(string(s), o)
	if i > 0 {
		i = utf8.RuneCountInString(string(s[:i]))
	}
	return i
}
func (s Str) IsEmpty() bool { return s == "" }
func (s Str) LastIndexOf(o string) int {
	i := strings.LastIndex(string(s), o)
	if i > 0 {
		i = utf8.RuneCountInString(string(s[:i]))
	}
	return i
}
func (s Str) Length() int                        { return utf8.RuneCountInString(string(s)) }
func (s Str) Matches(regex string) (bool, error) { return regexp.MatchString(regex, string(s)) }
func (s Str) Replace(old, new string) string     { return strings.ReplaceAll(string(s), old, new) }

func (s Str) ReplaceAll(regex, replacement string) (string, error) {
	r, err := regexp.Compile(regex)
	if err != nil {
		return "", err
	}
	return string(r.ReplaceAllString(string(s), replacement)), nil
}

func (s Str) ReplaceFirst(regex, replacement string) (string, error) {
	r, err := regexp.Compile(regex)
	if err != nil {
		return "", err
	}
	ss := string(s)
	match := r.FindStringSubmatchIndex(ss)
	if match == nil {
		return ss, nil
	}
	return string(s[:match[0]]) + string(r.ExpandString(nil, replacement, ss, match)) + string(s[match[1]:]), nil
}

func (s Str) Split(regex string) ([]string, error) {
	r, err := regexp.Compile(regex)
	if err != nil {
		return nil, err
	}
	result := r.Split(string(s), -1)
	i := len(result) - 1
	for ; i >= 0; i-- {
		if result[i] != "" {
			break
		}
	}
	return result[:i+1], nil
}

func (s Str) StartsWith(prefix string) bool { return strings.HasPrefix(string(s), prefix) }

func (s Str) SubSequence(start, end int) (string, error) {
	if start < 0 || end < 0 || end > s.Length() || start > end {
		return "", fmt.Errorf("start or end index out of range %d:%d with length %d", start, end, len(s))
	}
	return string([]rune(s)[start:end]), nil
}

func (s Str) Substring(start, end int) (string, error) { return s.SubSequence(start, end) }

func (s Str) ToLowerCase() string { return lower.String(string(s)) }
func (s Str) ToString() string    { return string(s) }
func (s Str) ToUpperCase() string { return upper.String(string(s)) }
func (s Str) Trim() string        { return strings.TrimSpace(string(s)) }

type Range struct {
	start, end, diff int
}

func (r *Range) Kind() string { return "range" }

func NewRange(start, end int) *Range {
	r := &Range{start, end, 1}
	if start > end {
		r.diff = -1
	}
	return r
}

func (r *Range) Add(v interface{}) (bool, error)    { return false, errUnsupported }
func (r *Range) AddAll(v interface{}) (bool, error) { return false, errUnsupported }
func (r *Range) Clear() error                       { return errUnsupported }

func (r *Range) Contains(v interface{}) bool {
	switch v.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32:
		vv := int(reflect.ValueOf(v).Int())
		return r.start <= vv && vv <= r.end
	}
	return false
}

func (r *Range) ContainsAll(v interface{}) (bool, error) {
	return containsAll(r, v)
}

func (r *Range) Equals(v interface{}) (bool, error) {
	if r2, ok := v.(*Range); ok {
		return r.start == r2.start && r.end == r2.end, nil
	}
	return equals(r, v)
}

func (r *Range) Get(i int) (interface{}, error) {
	if i < 0 || i >= r.Size() {
		return nil, fmt.Errorf("index out of range %d with length %d", i, r.Size())
	}
	return r.start + i*r.diff, nil
}

func (r *Range) IndexOf(i int) int {
	ret := (i - r.start) * r.diff
	if ret < 0 || ret >= r.Size() {
		return -1
	}
	return ret
}

func (r *Range) IsEmpty() bool { return r.Size() > 0 }

func (r *Range) Iterator() Iterator {
	return &CollectionIterator{s: r}
}

func (r *Range) LastIndexOf(i int) int {
	return r.IndexOf(i)
}

func (r *Range) Remove(interface{}) (bool, error)    { return false, errUnsupported }
func (r *Range) removeAt(int) error                  { return errUnsupported }
func (r *Range) RemoveAll(interface{}) (bool, error) { return false, errUnsupported }
func (r *Range) RetainAll(interface{}) (bool, error) { return false, errUnsupported }

func (r *Range) Set(int, interface{}) (interface{}, error) { return nil, errUnsupported }

func (r *Range) Size() int {
	return (r.end-r.start)*r.diff + 1
}

func (r *Range) ToArray() (*Slice, error) {
	if err := checkSize(r); err != nil {
		return nil, err
	}
	s := make([]int, r.Size())
	it := r.Iterator()
	var i int
	for it.HasNext() {
		s[i] = it.Next().(int)
		i++
	}
	return &Slice{s}, nil
}

func checkSize(c Collection) error {
	if c.Size() > DefaultMaxArrayRenderSize {
		return errors.New("size is too large")
	}
	return nil
}
