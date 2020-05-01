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
)

type Map struct {
	M map[string]interface{}
}

var errMapExpected = errors.New("map expected")

func (m *Map) Clear() {
	m.M = make(map[string]interface{})
}

func (m *Map) ContainsKey(key string) bool {
	_, ok := m.M[key]
	return ok
}

func (m *Map) ContainsValue(val interface{}) bool {
	for _, v := range m.M {
		if val == v {
			return true
		}
	}
	return false
}

func (m *Map) EntrySet() *EntryView {
	s := &Slice{}
	for k, v := range m.M {
		s.S = append(s.S, &MapEntry{k, v, m})
	}
	sort.Slice(s.S, func(i, j int) bool { return s.S[i].(*MapEntry).k < s.S[j].(*MapEntry).k })
	return &EntryView{Slice: s, m: m}
}

func (m *Map) Equals(v interface{}) (bool, error) {
	vv, ok := v.(*Map)
	if !ok {
		return false, errArrayExpected
	}
	if len(m.M) != len(vv.M) {
		return false, nil
	}
	for k := range vv.M {
		if _, ok := m.M[k]; !ok {
			return false, nil
		}
		if vv.M[k] != m.M[k] {
			return false, nil
		}
	}
	return true, nil
}

func (m *Map) Get(key interface{}) interface{} {
	return m.GetOrDefault(key, nil)
}

func (m *Map) GetOrDefault(key interface{}, deflt interface{}) interface{} {
	k := fmt.Sprint(key)
	v, ok := m.M[k]
	if !ok {
		return deflt
	}
	return v
}

func (m *Map) IsEmpty() bool {
	return len(m.M) == 0
}

func (m *Map) KeySet() *KeyView {
	s := &Slice{}
	for k := range m.M {
		s.S = append(s.S, k)
	}
	sort.Slice(s.S, func(i, j int) bool { return s.S[i].(string) < s.S[j].(string) })
	return &KeyView{Slice: s, m: m}
}

func (m *Map) Put(key string, value interface{}) interface{} {
	v, ok := m.M[key]
	m.M[key] = value
	if !ok {
		return nil
	}
	return v
}

func (m *Map) PutAll(v interface{}) error {
	vv, ok := v.(*Map)
	if !ok {
		return errMapExpected
	}
	for k := range vv.M {
		m.M[k] = vv.M[k]
	}
	return nil
}

func (m *Map) PutIfAbsent(key string, value interface{}) interface{} {
	v := m.Get(key)
	if v == nil {
		m.Put(key, value)
	}
	return v
}

func (m *Map) Remove(key string) interface{} {
	v := m.Get(key)
	delete(m.M, key)
	return v
}

func (m *Map) Replace(key string, val interface{}) interface{} {
	v, ok := m.M[key]
	if ok {
		m.M[key] = val
		return v
	}
	return nil
}

func (m *Map) Size() int {
	return len(m.M)
}

func (m *Map) Values() *ValView {
	s := &Slice{}
	keys := make([]string, 0, len(m.M))
	for k := range m.M {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		s.S = append(s.S, m.M[k])
	}
	return &ValView{View: &View{Slice: s, m: m}, k: keys}
}

type MapEntry struct {
	k string
	v interface{}
	m *Map
}

func (e *MapEntry) Equals(entry *MapEntry) bool {
	return e.k == entry.k && e.v == entry.v
}

func (e *MapEntry) GetKey() string {
	return e.k
}

func (e *MapEntry) GetValue() interface{} {
	return e.v
}

func (e *MapEntry) SetValue(val interface{}) interface{} {
	v := e.v
	e.m.M[e.k] = e.v
	e.v = val
	return v
}

type View struct {
	*Slice
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
	v.Slice.Clear()
	v.m.Clear()
}

type KeyView View

func (view *KeyView) Remove(k string) bool {
	ok := view.Slice.Remove(k)
	if ok {
		view.m.Remove(k)
	}
	return ok
}

type ValView struct {
	*View
	k []string
}

func (view *ValView) Remove(val interface{}) bool {
	ok := view.Slice.Remove(val)
	if ok {
		for i, k := range view.k {
			if view.m.Get(k) == val {
				view.m.Remove(k)
				view.k = append(view.k[:i], view.k[i+1:]...)
				break
			}
		}
	}
	return ok
}

type EntryView View

func (view *EntryView) Remove(entry interface{}) bool {
	ok := view.Slice.Remove(entry)
	if ok {
		k := entry.(*MapEntry).k
		view.m.Remove(k)
	}
	return ok
}

var errArrayExpected = errors.New("array expected")

type Slice struct {
	S []interface{}
}

func (s *Slice) Add(v interface{}) (bool, error) {
	s.S = append(s.S, v)
	return true, nil
}

func (s *Slice) AddAll(v interface{}) (bool, error) {
	vv, ok := v.(*Slice)
	if !ok {
		return false, errArrayExpected
	}
	s.S = append(s.S, vv.S...)
	return true, nil
}

func (s *Slice) Clear() {
	s.S = nil
}

func (s *Slice) Contains(v interface{}) bool {
	for i := range s.S {
		if s.S[i] == v {
			return true
		}
	}
	return false
}

func (s *Slice) ContainsAll(v interface{}) (bool, error) {
	vv, ok := v.(*Slice)
	if !ok {
		return false, errArrayExpected
	}
	for i := range vv.S {
		if !s.Contains(vv.S[i]) {
			return false, nil
		}
	}
	return true, nil
}

func (s *Slice) Equals(v interface{}) (bool, error) {
	vv, ok := v.(*Slice)
	if !ok {
		return false, errArrayExpected
	}
	if len(s.S) != len(vv.S) {
		return false, nil
	}
	for i := range vv.S {
		if vv.S[i] != s.S[i] {
			return false, nil
		}
	}
	return true, nil
}

func (s *Slice) Get(i int) (interface{}, error) {
	if i >= 0 && i < len(s.S) {
		return s.S[i], nil
	}
	return nil, fmt.Errorf("index out of range %d with length %d", i, len(s.S))
}

func (s *Slice) IsEmpty() bool { return len(s.S) == 0 }

func (s *Slice) Iterator() *Iterator { return &Iterator{s: s} }

func (s *Slice) Remove(v interface{}) bool {
	for i := range s.S {
		if s.S[i] == v {
			s.S = append(s.S[:i], s.S[i+1:]...)
			return true
		}
	}
	return false
}

func (s *Slice) RemoveAll(v interface{}) (bool, error) {
	vv, ok := v.(*Slice)
	if !ok {
		return false, errArrayExpected
	}
	var found bool
	for i := range vv.S {
		found = s.Remove(vv.S[i]) || found
	}
	return found, nil
}

func (s *Slice) RetainAll(v interface{}) (bool, error) {
	vv, ok := v.(*Slice)
	if !ok {
		return false, errArrayExpected
	}
	var found bool
	for i := 0; i < len(s.S); i++ {
		if vv.Contains(s.S[i]) {
			continue
		}
		s.Remove(s.S[i])
		found = true
	}
	return found, nil
}

func (s *Slice) Size() int {
	return len(s.S)
}

type Iterator struct {
	s *Slice
	i int
}

func NewIterator(v interface{}) *Iterator {
	vv := wrapTypes(reflect.ValueOf(v))
	switch s := vv.Interface().(type) {
	case *Slice:
		return &Iterator{s: s}
	default:
		return &Iterator{s: &Slice{S: []interface{}{v}}}
	}
}

func (i *Iterator) Next() reflect.Value {
	var ret reflect.Value
	if i.i < i.s.Size() {
		ret = reflect.ValueOf(i.s.S[i.i])
		i.i++
	}
	return ret
}

func (i *Iterator) HasNext() bool { return i.i < i.s.Size() }

var errNotImplemented = errors.New("not implemented")

type Str string

func (s Str) CharAt(i int) rune                   { return []rune(s)[i] }
func (s Str) CodePointAt(i int) error             { return errNotImplemented }
func (s Str) CodePointBefore(i int) error         { return errNotImplemented }
func (s Str) CodePointCount(start, end int) error { return errNotImplemented }

func (s Str) compare(o string, tr func(rune) rune) int {
	rs, ro := []rune(s), []rune(o)
	diff := len(rs) - len(ro)
	if diff != 0 {
		return diff
	}
	for i := range rs {
		diff = int(tr(rs[i])) - int(tr(ro[i]))
		if diff != 0 {
			return diff
		}
	}
	return 0
}

func (s Str) CompareTo(o string) int {
	return s.compare(o, func(r rune) rune { return r })
}

func (s Str) CompareToIgnoreCase(o string) int {
	return s.compare(o, func(r rune) rune { return unicode.ToLower(unicode.ToUpper(r)) })
}

func (s Str) Concat(o string) string      { return string(s) + o }
func (s Str) Contains(o string) bool      { return strings.Contains(string(s), o) }
func (s Str) ContentEquals(o string) bool { return string(s) == o }
func (s Str) EndsWith(suffix string) bool { return strings.HasSuffix(string(s), suffix) }
func (s Str) Equals(o string) bool        { return string(s) == o }
func (s Str) EqualsIgnoreCase(o string) bool {
	return strings.ToLower(strings.ToUpper(string(s))) == strings.ToLower(strings.ToUpper(o))
}
func (s Str) GetBytes() []byte { return []byte(s) }
func (s Str) IndexOf(o string) int {
	i := strings.Index(string(s), o)
	if i > 0 {
		i = utf8.RuneCountInString(string(s[:i+1]))
	}
	return i
}
func (s Str) IsEmpty() bool { return s == "" }
func (s Str) LastIndexOf(o string) int {
	i := strings.LastIndex(string(s), o)
	if i > 0 {
		i = utf8.RuneCountInString(string(s[:i+1]))
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
	loc := r.FindStringIndex(string(s))
	if loc == nil {
		return string(s), nil
	}
	return string(s[:loc[0]]) + replacement + string(s[loc[1]:]), nil
}

func (s Str) Split(regex string) ([]string, error) {
	r, err := regexp.Compile(regex)
	if err != nil {
		return nil, err
	}
	return r.Split(string(s), -1), nil
}

func (s Str) StartsWith(prefix string) bool { return strings.HasPrefix(string(s), prefix) }

func (s Str) SubSequence(start, end int) (string, error) {
	if start < 0 || end < 0 || end > s.Length() || start > end {
		return "", fmt.Errorf("start or end index out of range %d:%d with length %d", start, end, len(s))
	}
	return string([]rune(s)[start:end]), nil
}

func (s Str) Substring(start, end int) (string, error) { return s.SubSequence(start, end) }

func (s Str) ToLowerCase() string { return strings.ToLower(string(s)) }
func (s Str) ToString() string    { return string(s) }
func (s Str) ToUpperCase() string { return strings.ToUpper(string(s)) }
func (s Str) Trim() string        { return strings.TrimSpace(string(s)) }

// TODO add implementation based on the vtl's
type Range struct {
}
