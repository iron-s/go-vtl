package govtl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func emptyMap() map[string]interface{} {
	return map[string]interface{}{}
}

func TestMap_Clear(t *testing.T) {
	type fields struct {
		M map[string]interface{}
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{"full map should be cleared", fields{M: map[string]interface{}{"some": "field"}}},
		{"and even nil map should be replaced", fields{M: nil}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Map{
				M: tt.fields.M,
			}
			m.Clear()
			assert.NotNil(t, m.M, "map is not nil")
			assert.Len(t, m.M, 0, "and of zero length")
		})
	}
}

func TestMap_ContainsKey(t *testing.T) {
	type fields struct {
		M map[string]interface{}
	}
	type args struct {
		key string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{"key in map exists", fields{M: map[string]interface{}{"some": "field"}}, args{key: "some"}, true},
		{"no key in map exists", fields{M: map[string]interface{}{"some": "field"}}, args{key: "other"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Map{
				M: tt.fields.M,
			}
			assert.Equal(t, tt.want, m.ContainsKey(tt.args.key))
		})
	}
}

func TestMap_ContainsValue(t *testing.T) {
	type fields struct {
		M map[string]interface{}
	}
	type args struct {
		val interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{"value does not exist", fields{M: map[string]interface{}{"some": "field"}}, args{val: "some"}, false},
		{"value exists and of comparable type", fields{M: map[string]interface{}{"some": "field"}}, args{val: "field"}, true},
		{"value exists and is nil", fields{M: map[string]interface{}{"some": nil}}, args{val: nil}, true},
		{"value exists and of non-comparable type", fields{M: map[string]interface{}{"some": []string{"field"}}}, args{val: []string{"field"}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Map{
				M: tt.fields.M,
			}
			assert.Equal(t, tt.want, m.ContainsValue(tt.args.val))
		})
	}
}

func TestMap_EntrySet(t *testing.T) {
	type fields struct {
		M map[string]interface{}
	}
	m := map[string]interface{}{"second": "value", "first": []int{1}}
	tests := []struct {
		name   string
		fields fields
		want   *EntryView
	}{
		{"works on empty map", fields{emptyMap()}, &EntryView{&Slice{nil}, &Map{emptyMap()}}},
		{"returns entryset sorted", fields{m}, &EntryView{&Slice{[]interface{}{&MapEntry{"first", []int{1}, &Map{m}}, &MapEntry{"second", "value", &Map{m}}}}, &Map{m}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Map{
				M: tt.fields.M,
			}
			assert.Equal(t, tt.want, m.EntrySet())
		})
	}
}

func TestMap_Equals(t *testing.T) {
	type fields struct {
		M map[string]interface{}
	}
	type args struct {
		v interface{}
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      bool
		assertion assert.ErrorAssertionFunc
	}{
		{"expects map", fields{emptyMap()}, args{[]int{1}}, false,
			assert.Error},
		{"empty map", fields{map[string]interface{}{}}, args{&Map{emptyMap()}}, true, assert.NoError},
		{"different length", fields{emptyMap()}, args{&Map{map[string]interface{}{"a": 1}}}, false, assert.NoError},
		{"different length the other way", fields{map[string]interface{}{"a": 1}}, args{&Map{emptyMap()}}, false, assert.NoError},
		{"same comparable content", fields{map[string]interface{}{"a": 1, "and": "string"}}, args{&Map{map[string]interface{}{"a": 1, "and": "string"}}}, true, assert.NoError},
		{"same non-comparable content", fields{map[string]interface{}{"a": &Map{map[string]interface{}{"inner": "map"}}, "and": &Slice{[]interface{}{"with", "items"}}}}, args{&Map{map[string]interface{}{"a": &Map{map[string]interface{}{"inner": "map"}}, "and": &Slice{[]interface{}{"with", "items"}}}}}, true, assert.NoError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Map{
				M: tt.fields.M,
			}
			got, err := m.Equals(tt.args.v)
			tt.assertion(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMap_Get(t *testing.T) {
	type fields struct {
		M map[string]interface{}
	}
	type args struct {
		key interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   interface{}
	}{
		{"empty map", fields{emptyMap()}, args{"key"}, nil},
		{"absent key as string", fields{map[string]interface{}{"other": "key"}}, args{"key"}, nil},
		{"absent key as int", fields{map[string]interface{}{"1": 2}}, args{"3"}, nil},
		{"present key as string", fields{map[string]interface{}{"1": 2}}, args{"1"}, 2},
		{"present key as int", fields{map[string]interface{}{"1": "2"}}, args{1}, "2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Map{
				M: tt.fields.M,
			}
			assert.Equal(t, tt.want, m.Get(tt.args.key))
		})
	}
}

func TestMap_GetOrDefault(t *testing.T) {
	type fields struct {
		M map[string]interface{}
	}
	type args struct {
		key   interface{}
		deflt interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   interface{}
	}{
		{"empty map", fields{emptyMap()}, args{"key", "default"}, "default"},
		{"absent key as string", fields{map[string]interface{}{"other": "key"}}, args{"key", nil}, nil},
		{"absent key as int", fields{map[string]interface{}{"1": 2}}, args{"3", 4}, 4},
		{"present key as string", fields{map[string]interface{}{"1": 2}}, args{"1", 4}, 2},
		{"present key as int", fields{map[string]interface{}{"1": "2"}}, args{1, 3}, "2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Map{
				M: tt.fields.M,
			}
			assert.Equal(t, tt.want, m.GetOrDefault(tt.args.key, tt.args.deflt))
		})
	}
}

func TestMap_IsEmpty(t *testing.T) {
	type fields struct {
		M map[string]interface{}
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{"empty map", fields{map[string]interface{}{}}, true},
		{"non-empty", fields{map[string]interface{}{"other": "key"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Map{
				M: tt.fields.M,
			}
			assert.Equal(t, tt.want, m.IsEmpty())
		})
	}
}

func TestMap_KeySet(t *testing.T) {
	type fields struct {
		M map[string]interface{}
	}
	m := map[string]interface{}{"second": "value", "first": []int{1}}
	empty := emptyMap()
	tests := []struct {
		name   string
		fields fields
		want   *KeyView
	}{
		{"works on empty map", fields{empty}, &KeyView{&Slice{nil}, &Map{empty}}},
		{"returns keyset sorted", fields{m}, &KeyView{&Slice{[]interface{}{"first", "second"}}, &Map{m}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Map{
				M: tt.fields.M,
			}
			assert.Equal(t, tt.want, m.KeySet())
		})
	}
}

func TestMap_Put(t *testing.T) {
	type fields struct {
		M map[string]interface{}
	}
	m := emptyMap()
	type args struct {
		key   string
		value interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   interface{}
	}{
		{"works on empty map", fields{m}, args{"k", 1}, nil},
		{"returns replaced value", fields{m}, args{"k", 2}, 1},
		{"and adds another", fields{m}, args{"other", "key"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Map{
				M: tt.fields.M,
			}
			assert.Equal(t, tt.want, m.Put(tt.args.key, tt.args.value))
			assert.Equal(t, tt.args.value, m.Get(tt.args.key))
		})
	}
}

func TestMap_PutAll(t *testing.T) {
	type fields struct {
		M map[string]interface{}
	}
	type args struct {
		v interface{}
	}
	m := emptyMap()
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      map[string]interface{}
		assertion assert.ErrorAssertionFunc
	}{
		{"expects map", fields{emptyMap()}, args{[]int{1}}, map[string]interface{}{}, assert.Error},
		{"works on empty map", fields{m}, args{&Map{map[string]interface{}{"a": "b", "c": 2}}}, map[string]interface{}{"a": "b", "c": 2}, assert.NoError},
		{"works on empty map as argument", fields{m}, args{&Map{emptyMap()}}, map[string]interface{}{"a": "b", "c": 2}, assert.NoError},
		{"adds and replaces value", fields{m}, args{&Map{map[string]interface{}{"c": "d", "e": []int{1}}}}, map[string]interface{}{"a": "b", "c": "d", "e": []int{1}}, assert.NoError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Map{
				M: tt.fields.M,
			}
			tt.assertion(t, m.PutAll(tt.args.v))
			assert.Equal(t, m.M, tt.want)
		})
	}
}

func TestMap_PutIfAbsent(t *testing.T) {
	type fields struct {
		M map[string]interface{}
	}
	type args struct {
		key   string
		value interface{}
	}
	m := emptyMap()
	tests := []struct {
		name   string
		fields fields
		args   args
		want   interface{}
		wantM  map[string]interface{}
	}{
		{"works on empty map", fields{m}, args{"k", 1}, nil, map[string]interface{}{"k": 1}},
		{"does not replace existing and returns it's value", fields{m}, args{"k", 2}, 1, map[string]interface{}{"k": 1}},
		{"and adds another", fields{m}, args{"other", "key"}, nil, map[string]interface{}{"k": 1, "other": "key"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Map{
				M: tt.fields.M,
			}
			assert.Equal(t, tt.want, m.PutIfAbsent(tt.args.key, tt.args.value))
			assert.Equal(t, m.M, tt.wantM)
		})
	}
}

func TestMap_Remove(t *testing.T) {
	type fields struct {
		M map[string]interface{}
	}
	type args struct {
		key string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   interface{}
		wantM  map[string]interface{}
	}{
		{"works on empty map", fields{emptyMap()}, args{"k"}, nil, map[string]interface{}{}},
		{"removes last element", fields{map[string]interface{}{"k": 1}}, args{"k"}, 1, map[string]interface{}{}},
		{"and removes some other", fields{map[string]interface{}{"k": 1, "other": "key"}}, args{"other"}, "key", map[string]interface{}{"k": 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Map{
				M: tt.fields.M,
			}
			assert.Equal(t, tt.want, m.Remove(tt.args.key))
			assert.Equal(t, tt.wantM, m.M)
		})
	}
}

func TestMap_Replace(t *testing.T) {
	type fields struct {
		M map[string]interface{}
	}
	type args struct {
		key string
		val interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   interface{}
		wantM  map[string]interface{}
	}{
		{"does nothing on empty map", fields{emptyMap()}, args{"k", 1}, nil, map[string]interface{}{}},
		{"replaces only element", fields{map[string]interface{}{"k": 1}}, args{"k", 2}, 1, map[string]interface{}{"k": 2}},
		{"replaces some other", fields{map[string]interface{}{"k": 1, "other": "key"}}, args{"other", "new"}, "key", map[string]interface{}{"k": 1, "other": "new"}},
		{"does nothing if not found", fields{map[string]interface{}{"k": 1, "other": "key"}}, args{"K", "new"}, nil, map[string]interface{}{"k": 1, "other": "key"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Map{
				M: tt.fields.M,
			}
			assert.Equal(t, tt.want, m.Replace(tt.args.key, tt.args.val))
		})
	}
}

func TestMap_Size(t *testing.T) {
	type fields struct {
		M map[string]interface{}
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{"works on empty map", fields{emptyMap()}, 0},
		{"map with single element", fields{map[string]interface{}{"k": 1}}, 1},
		{"and longer map", fields{map[string]interface{}{"k": 1, "some": "other", "and": "more"}}, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Map{
				M: tt.fields.M,
			}
			assert.Equal(t, tt.want, m.Size())
		})
	}
}

func TestMap_Values(t *testing.T) {
	type fields struct {
		M map[string]interface{}
	}
	empty := emptyMap()
	m := map[string]interface{}{"second": "value", "first": []int{1}}
	tests := []struct {
		name   string
		fields fields
		want   *ValView
	}{
		{"works on empty map", fields{empty}, &ValView{&View{&Slice{nil}, &Map{empty}}, []string{}}},
		{"returns values in order of sorted keys", fields{m}, &ValView{&View{&Slice{[]interface{}{[]int{1}, "value"}}, &Map{m}}, []string{"first", "second"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Map{
				M: tt.fields.M,
			}
			assert.Equal(t, tt.want, m.Values())
		})
	}
}

func TestMapEntry_Equals(t *testing.T) {
	type fields struct {
		k string
		v interface{}
		m *Map
	}
	type args struct {
		entry *MapEntry
	}
	empty := &Map{emptyMap()}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{"equal map entries from different maps",
			fields{"k", 1, empty}, args{&MapEntry{"k", 1, &Map{}}}, true},
		{"equal map entries from nil maps",
			fields{"k", 1, nil}, args{&MapEntry{"k", 1, nil}}, true},
		{"unequal map entries from same map",
			fields{"k", 1, empty}, args{&MapEntry{"k", 2, empty}}, false},
		{"unequal map entries from different maps",
			fields{"K", 1, empty}, args{&MapEntry{"k", 1, &Map{map[string]interface{}{"k": 1}}}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &MapEntry{
				k: tt.fields.k,
				v: tt.fields.v,
				m: tt.fields.m,
			}
			assert.Equal(t, tt.want, e.Equals(tt.args.entry))
		})
	}
}

func TestMapEntry_GetKey(t *testing.T) {
	type fields struct {
		k string
		v interface{}
		m *Map
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{"regular map entry",
			fields{"k", 1, nil}, "k"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &MapEntry{
				k: tt.fields.k,
				v: tt.fields.v,
				m: tt.fields.m,
			}
			assert.Equal(t, tt.want, e.GetKey())
		})
	}
}

func TestMapEntry_GetValue(t *testing.T) {
	type fields struct {
		k string
		v interface{}
		m *Map
	}
	tests := []struct {
		name   string
		fields fields
		want   interface{}
	}{
		{"regular map entry",
			fields{"k", 1, nil}, 1},
		{"works with nil",
			fields{"k", nil, nil}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &MapEntry{
				k: tt.fields.k,
				v: tt.fields.v,
				m: tt.fields.m,
			}
			assert.Equal(t, tt.want, e.GetValue())
		})
	}
}

func TestMapEntry_SetValue(t *testing.T) {
	type fields struct {
		k string
		v interface{}
		m *Map
	}
	type args struct {
		val interface{}
	}
	m := &Map{map[string]interface{}{"k": 1, "other": "value"}}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   interface{}
		wantM  map[string]interface{}
	}{
		{"regular map entry",
			fields{"k", 1, m}, args{"str"}, 1, map[string]interface{}{"k": "str", "other": "value"}},
		{"works with nil",
			fields{"other", nil, m}, args{nil}, nil, map[string]interface{}{"k": "str", "other": nil}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &MapEntry{
				k: tt.fields.k,
				v: tt.fields.v,
				m: tt.fields.m,
			}
			assert.Equal(t, tt.want, e.SetValue(tt.args.val))
		})
	}
}

func TestView_Add(t *testing.T) {
	type fields struct {
		Slice *Slice
		m     *Map
	}
	type args struct {
		in0 interface{}
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      bool
		assertion assert.ErrorAssertionFunc
		wantErr   error
	}{
		{"should return unsupported", fields{}, args{}, false, assert.Error, errUnsupported},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &View{
				Slice: tt.fields.Slice,
				m:     tt.fields.m,
			}
			got, err := v.Add(tt.args.in0)
			if tt.assertion(t, err) {
				assert.Equal(t, err, tt.wantErr)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestView_AddAll(t *testing.T) {
	type fields struct {
		Slice *Slice
		m     *Map
	}
	type args struct {
		in0 interface{}
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      bool
		assertion assert.ErrorAssertionFunc
		wantErr   error
	}{
		{"should return unsupported", fields{}, args{}, false, assert.Error, errUnsupported},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &View{
				Slice: tt.fields.Slice,
				m:     tt.fields.m,
			}
			got, err := v.AddAll(tt.args.in0)
			if tt.assertion(t, err) {
				assert.Equal(t, err, tt.wantErr)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestView_Clear(t *testing.T) {
	type fields struct {
		Slice *Slice
		m     *Map
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{"works with empty", fields{&Slice{}, &Map{emptyMap()}}},
		{"clears full", fields{&Slice{[]interface{}{1, "two", []int{3}}}, &Map{map[string]interface{}{"a": 1, "b": "two"}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &View{
				Slice: tt.fields.Slice,
				m:     tt.fields.m,
			}
			v.Clear()
			assert.Equal(t, emptyMap(), v.m.M)
			assert.Equal(t, []interface{}(nil), v.Slice.S)
		})
	}
}

func TestKeyView_Remove(t *testing.T) {
	type fields struct {
		Slice *Slice
		m     *Map
	}
	type args struct {
		k string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
		wantS  []interface{}
		wantM  map[string]interface{}
	}{
		{"works with empty", fields{&Slice{}, &Map{emptyMap()}}, args{"key"}, false, []interface{}(nil), emptyMap()},
		{"does nothing for non-existant", fields{&Slice{[]interface{}{"k"}}, &Map{map[string]interface{}{"k": 1}}}, args{"key"}, false, []interface{}{"k"}, map[string]interface{}{"k": 1}},
		{"removes from slice and map", fields{&Slice{[]interface{}{"k", "key"}}, &Map{map[string]interface{}{"k": 1, "key": "value"}}}, args{"key"}, true, []interface{}{"k"}, map[string]interface{}{"k": 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := &KeyView{
				Slice: tt.fields.Slice,
				m:     tt.fields.m,
			}
			assert.Equal(t, tt.want, view.Remove(tt.args.k))
			assert.Equal(t, tt.wantM, view.m.M)
			assert.Equal(t, tt.wantS, view.Slice.S)
		})
	}
}

func TestKeyView_RemoveAll(t *testing.T) {
	type fields struct {
		Slice *Slice
		m     *Map
	}
	type args struct {
		v interface{}
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      bool
		assertion assert.ErrorAssertionFunc
		wantS     []interface{}
		wantM     map[string]interface{}
	}{
		{"works with empty", fields{&Slice{}, &Map{emptyMap()}}, args{&Slice{[]interface{}{"key", 1}}}, false, assert.NoError, []interface{}(nil), emptyMap()},
		{"does nothing for non-existant", fields{&Slice{[]interface{}{"k"}}, &Map{map[string]interface{}{"k": 1}}}, args{&Slice{[]interface{}{"key"}}}, false, assert.NoError, []interface{}{"k"}, map[string]interface{}{"k": 1}},
		{"removes all found from slice and map", fields{&Slice{[]interface{}{"k", "key"}}, &Map{map[string]interface{}{"k": 1, "key": "value"}}}, args{&Slice{[]interface{}{"key", 1}}}, true, assert.NoError, []interface{}{"k"}, map[string]interface{}{"k": 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := &KeyView{
				Slice: tt.fields.Slice,
				m:     tt.fields.m,
			}
			got, err := view.RemoveAll(tt.args.v)
			tt.assertion(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantM, view.m.M)
			assert.Equal(t, tt.wantS, view.Slice.S)
		})
	}
}

func TestKeyView_RetainAll(t *testing.T) {
	type fields struct {
		Slice *Slice
		m     *Map
	}
	type args struct {
		v interface{}
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      bool
		assertion assert.ErrorAssertionFunc
		wantS     []interface{}
		wantM     map[string]interface{}
	}{
		{"works with empty", fields{&Slice{}, &Map{emptyMap()}}, args{&Slice{[]interface{}{"key", 1}}}, false, assert.NoError, []interface{}(nil), emptyMap()},
		{"does nothing for existing", fields{&Slice{[]interface{}{"k"}}, &Map{map[string]interface{}{"k": 1}}}, args{&Slice{[]interface{}{"k"}}}, false, assert.NoError, []interface{}{"k"}, map[string]interface{}{"k": 1}},
		{"retains only existing", fields{&Slice{[]interface{}{"k", "and"}}, &Map{map[string]interface{}{"k": 1, "and": 2}}}, args{&Slice{[]interface{}{"some", "keys"}}}, true, assert.NoError, []interface{}{}, map[string]interface{}{}},
		{"removes all not found in other from slice and map", fields{&Slice{[]interface{}{"k", "key"}}, &Map{map[string]interface{}{"k": 1, "key": "value"}}}, args{&Slice{[]interface{}{"key", 1}}}, true, assert.NoError, []interface{}{"key"}, map[string]interface{}{"key": "value"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := &KeyView{
				Slice: tt.fields.Slice,
				m:     tt.fields.m,
			}
			got, err := view.RetainAll(tt.args.v)
			tt.assertion(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantM, view.m.M)
			assert.Equal(t, tt.wantS, view.Slice.S)
		})
	}
}

func TestValView_Remove(t *testing.T) {
	type fields struct {
		View *View
		k    []string
	}
	type args struct {
		val interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
		wantS  []interface{}
		wantM  map[string]interface{}
		wantK  []string
	}{
		{"works with empty", fields{&View{&Slice{}, &Map{emptyMap()}}, []string{}}, args{2}, false, []interface{}(nil), emptyMap(), []string{}},
		{"does nothing for non-existant", fields{&View{&Slice{[]interface{}{1}}, &Map{map[string]interface{}{"k": 1}}}, []string{"k"}}, args{2}, false, []interface{}{1}, map[string]interface{}{"k": 1}, []string{"k"}},
		{"removes from slice, map, and key slice", fields{&View{&Slice{[]interface{}{1, "value"}}, &Map{map[string]interface{}{"k": 1, "key": "value"}}}, []string{"k", "key"}}, args{1}, true, []interface{}{"value"}, map[string]interface{}{"key": "value"}, []string{"key"}},
		{"removes even non-comparable values", fields{&View{&Slice{[]interface{}{[]int{1}, "value"}}, &Map{map[string]interface{}{"k": []int{1}, "key": "value"}}}, []string{"k", "key"}}, args{[]int{1}}, true, []interface{}{"value"}, map[string]interface{}{"key": "value"}, []string{"key"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := &ValView{
				View: tt.fields.View,
				k:    tt.fields.k,
			}
			assert.Equal(t, tt.want, view.Remove(tt.args.val))
			assert.Equal(t, tt.wantM, view.m.M)
			assert.Equal(t, tt.wantS, view.Slice.S)
			assert.Equal(t, tt.wantK, view.k)
		})
	}
}

func TestValView_RemoveAll(t *testing.T) {
	type fields struct {
		View *View
		k    []string
	}
	type args struct {
		v interface{}
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      bool
		assertion assert.ErrorAssertionFunc
		wantS     []interface{}
		wantM     map[string]interface{}
		wantK     []string
	}{
		{"works with empty", fields{&View{&Slice{}, &Map{emptyMap()}}, []string{}}, args{&Slice{[]interface{}{2, "value"}}}, false, assert.NoError, []interface{}(nil), emptyMap(), []string{}},
		{"does nothing for non-existant", fields{&View{&Slice{[]interface{}{1}}, &Map{map[string]interface{}{"k": 1}}}, []string{"k"}}, args{&Slice{[]interface{}{2, "value"}}}, false, assert.NoError, []interface{}{1}, map[string]interface{}{"k": 1}, []string{"k"}},
		{"removes from slice, map, and key slice", fields{&View{&Slice{[]interface{}{1, "value"}}, &Map{map[string]interface{}{"k": 1, "key": "value"}}}, []string{"k", "key"}}, args{&Slice{[]interface{}{1, "v"}}}, true, assert.NoError, []interface{}{"value"}, map[string]interface{}{"key": "value"}, []string{"key"}},
		{"removes even non-comparable values", fields{&View{&Slice{[]interface{}{[]int{1}, "value"}}, &Map{map[string]interface{}{"k": []int{1}, "key": "value"}}}, []string{"k", "key"}}, args{&Slice{[]interface{}{[]int{1}, 1}}}, true, assert.NoError, []interface{}{"value"}, map[string]interface{}{"key": "value"}, []string{"key"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := &ValView{
				View: tt.fields.View,
				k:    tt.fields.k,
			}
			got, err := view.RemoveAll(tt.args.v)
			tt.assertion(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantM, view.m.M)
			assert.Equal(t, tt.wantS, view.Slice.S)
			assert.Equal(t, tt.wantK, view.k)
		})
	}
}

func TestValView_RetainAll(t *testing.T) {
	type fields struct {
		View *View
		k    []string
	}
	type args struct {
		v interface{}
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      bool
		assertion assert.ErrorAssertionFunc
		wantS     []interface{}
		wantM     map[string]interface{}
		wantK     []string
	}{
		{"works with empty", fields{&View{&Slice{}, &Map{emptyMap()}}, []string{}}, args{&Slice{[]interface{}{2, "value"}}}, false, assert.NoError, []interface{}(nil), emptyMap(), []string{}},
		{"does nothing for existing", fields{&View{&Slice{[]interface{}{1}}, &Map{map[string]interface{}{"k": 1}}}, []string{"k"}}, args{&Slice{[]interface{}{1, "value"}}}, false, assert.NoError, []interface{}{1}, map[string]interface{}{"k": 1}, []string{"k"}},
		{"retains only existing", fields{&View{&Slice{[]interface{}{1, 2}}, &Map{map[string]interface{}{"k": 1, "and": 2}}}, []string{"k", "and"}}, args{&Slice{[]interface{}{"some", "value"}}}, true, assert.NoError, []interface{}{}, map[string]interface{}{}, []string{}},
		{"removes from slice, map, and key slice", fields{&View{&Slice{[]interface{}{1, "value"}}, &Map{map[string]interface{}{"k": 1, "key": "value"}}}, []string{"k", "key"}}, args{&Slice{[]interface{}{2, "value"}}}, true, assert.NoError, []interface{}{"value"}, map[string]interface{}{"key": "value"}, []string{"key"}},
		{"retains even non-comparable values", fields{&View{&Slice{[]interface{}{[]int{1}, "value"}}, &Map{map[string]interface{}{"k": []int{1}, "key": "value"}}}, []string{"k", "key"}}, args{&Slice{[]interface{}{[]int{1}, 1}}}, true, assert.NoError, []interface{}{[]int{1}}, map[string]interface{}{"k": []int{1}}, []string{"k"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := &ValView{
				View: tt.fields.View,
				k:    tt.fields.k,
			}
			got, err := view.RetainAll(tt.args.v)
			tt.assertion(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantM, view.m.M)
			assert.Equal(t, tt.wantS, view.Slice.S)
			assert.Equal(t, tt.wantK, view.k)
		})
	}
}

func TestEntryView_Remove(t *testing.T) {
	type fields struct {
		Slice *Slice
		m     *Map
	}
	type args struct {
		entry *MapEntry
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
		wantS  []interface{}
		wantM  map[string]interface{}
	}{
		{"works with empty", fields{&Slice{}, &Map{emptyMap()}}, args{&MapEntry{"k", 1, nil}}, false, []interface{}(nil), emptyMap()},
		{"does nothing for non-existant", fields{&Slice{[]interface{}{&MapEntry{"k", 1, nil}}}, &Map{map[string]interface{}{"k": 1}}}, args{&MapEntry{"k", 2, nil}}, false, []interface{}{&MapEntry{"k", 1, nil}}, map[string]interface{}{"k": 1}},
		{"removes from slice and map", fields{&Slice{[]interface{}{&MapEntry{"k", 1, nil}, &MapEntry{"key", "value", nil}}}, &Map{map[string]interface{}{"k": 1, "key": "value"}}}, args{&MapEntry{"k", 1, nil}}, true, []interface{}{&MapEntry{"key", "value", nil}}, map[string]interface{}{"key": "value"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := &EntryView{
				Slice: tt.fields.Slice,
				m:     tt.fields.m,
			}
			assert.Equal(t, tt.want, view.Remove(tt.args.entry))
			assert.Equal(t, tt.wantM, view.m.M)
			assert.Equal(t, tt.wantS, view.Slice.S)
		})
	}
}

func TestEntryView_RemoveAll(t *testing.T) {
	type fields struct {
		Slice *Slice
		m     *Map
	}
	type args struct {
		v interface{}
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      bool
		assertion assert.ErrorAssertionFunc
		wantS     []interface{}
		wantM     map[string]interface{}
	}{
		{"works with empty", fields{&Slice{}, &Map{emptyMap()}}, args{&Slice{[]interface{}{&MapEntry{"k", 1, nil}, &MapEntry{"key", "value", nil}}}}, false, assert.NoError, []interface{}(nil), emptyMap()},
		{"does nothing for non-existant", fields{&Slice{[]interface{}{&MapEntry{"k", 1, nil}}}, &Map{map[string]interface{}{"k": 1}}}, args{&Slice{[]interface{}{&MapEntry{"k", 2, nil}, &MapEntry{"key", "value", nil}}}}, false, assert.NoError, []interface{}{&MapEntry{"k", 1, nil}}, map[string]interface{}{"k": 1}},
		{"removes from slice and map", fields{&Slice{[]interface{}{&MapEntry{"k", 1, nil}, &MapEntry{"key", "value", nil}, &MapEntry{"some", []string{"more"}, nil}}}, &Map{map[string]interface{}{"k": 1, "key": "value", "some": []string{"more"}}}}, args{&Slice{[]interface{}{&MapEntry{"k", 1, nil}, &MapEntry{"some", []string{"more"}, nil}}}}, true, assert.NoError, []interface{}{&MapEntry{"key", "value", nil}}, map[string]interface{}{"key": "value"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := &EntryView{
				Slice: tt.fields.Slice,
				m:     tt.fields.m,
			}
			got, err := view.RemoveAll(tt.args.v)
			tt.assertion(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantM, view.m.M)
			assert.Equal(t, tt.wantS, view.Slice.S)
		})
	}
}

func TestEntryView_RetainAll(t *testing.T) {
	type fields struct {
		Slice *Slice
		m     *Map
	}
	type args struct {
		v interface{}
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      bool
		assertion assert.ErrorAssertionFunc
		wantS     []interface{}
		wantM     map[string]interface{}
	}{
		{"works with empty", fields{&Slice{}, &Map{emptyMap()}}, args{&Slice{[]interface{}{&MapEntry{"k", 1, nil}, &MapEntry{"key", "value", nil}}}}, false, assert.NoError, []interface{}(nil), emptyMap()},
		{"does nothing for existing", fields{&Slice{[]interface{}{&MapEntry{"k", 1, nil}}}, &Map{map[string]interface{}{"k": 1}}}, args{&Slice{[]interface{}{&MapEntry{"k", 1, nil}, &MapEntry{"key", "value", nil}}}}, false, assert.NoError, []interface{}{&MapEntry{"k", 1, nil}}, map[string]interface{}{"k": 1}},
		{"retains only existing", fields{&Slice{[]interface{}{&MapEntry{"k", 1, nil}, &MapEntry{"and", 2, nil}}}, &Map{map[string]interface{}{"k": 1, "and": 2}}}, args{&Slice{[]interface{}{&MapEntry{"some", "entry", nil}, &MapEntry{"key", "value", nil}}}}, true, assert.NoError, []interface{}{}, map[string]interface{}{}},
		{"retains", fields{&Slice{[]interface{}{&MapEntry{"k", 1, nil}, &MapEntry{"key", "value", nil}, &MapEntry{"some", []string{"more"}, nil}}}, &Map{map[string]interface{}{"k": 1, "key": "value", "some": []string{"more"}}}}, args{&Slice{[]interface{}{&MapEntry{"k", 1, nil}, &MapEntry{"some", []string{"more"}, nil}}}}, true, assert.NoError, []interface{}{&MapEntry{"k", 1, nil}, &MapEntry{"some", []string{"more"}, nil}}, map[string]interface{}{"k": 1, "some": []string{"more"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := &EntryView{
				Slice: tt.fields.Slice,
				m:     tt.fields.m,
			}
			got, err := view.RetainAll(tt.args.v)
			tt.assertion(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantM, view.m.M)
			assert.Equal(t, tt.wantS, view.Slice.S)
		})
	}
}

func TestSlice_Add(t *testing.T) {
	type fields struct {
		S []interface{}
	}
	type args struct {
		v interface{}
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      bool
		assertion assert.ErrorAssertionFunc
		wantS     []interface{}
	}{
		{"add to empty slice", fields{nil}, args{1}, true, assert.NoError, []interface{}{1}},
		{"add to slice", fields{[]interface{}{"1"}}, args{1}, true, assert.NoError, []interface{}{"1", 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Slice{
				S: tt.fields.S,
			}
			got, err := s.Add(tt.args.v)
			tt.assertion(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantS, s.S)
		})
	}
}

func TestSlice_AddAll(t *testing.T) {
	type fields struct {
		S []interface{}
	}
	type args struct {
		v interface{}
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      bool
		assertion assert.ErrorAssertionFunc
		wantS     []interface{}
	}{
		{"add to empty slice", fields{nil}, args{&Slice{[]interface{}{1, "2"}}}, true, assert.NoError, []interface{}{1, "2"}},
		{"add to slice", fields{[]interface{}{1}}, args{&Slice{[]interface{}{"2", []int{3}}}}, true, assert.NoError, []interface{}{1, "2", []int{3}}},
		{"error adding regular go slice", fields{[]interface{}{1}}, args{[]interface{}{"2", []int{3}}}, false, assert.Error, []interface{}{1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Slice{
				S: tt.fields.S,
			}
			got, err := s.AddAll(tt.args.v)
			tt.assertion(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantS, s.S)
		})
	}
}

func TestSlice_Clear(t *testing.T) {
	type fields struct {
		S []interface{}
	}
	tests := []struct {
		name   string
		fields fields
		wantS  []interface{}
	}{
		{"clear nil slice", fields{nil}, nil},
		{"clear non-nil slice", fields{[]interface{}{1, 2}}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Slice{
				S: tt.fields.S,
			}
			s.Clear()
			assert.Equal(t, tt.wantS, s.S)
		})
	}
}

func TestSlice_Contains(t *testing.T) {
	type fields struct {
		S []interface{}
	}
	type args struct {
		v interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{"value does not exist", fields{[]interface{}{"some", "slice"}}, args{"not exists"}, false},
		{"value is nil and does not exist", fields{[]interface{}{"some", "slice"}}, args{nil}, false},
		{"value exists and of comparable type", fields{[]interface{}{"some", "slice"}}, args{"slice"}, true},
		{"value exists and is nil", fields{[]interface{}{"some", nil}}, args{nil}, true},
		{"value exists and of non-comparable type", fields{[]interface{}{"some", []string{"slice"}}}, args{[]string{"slice"}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Slice{
				S: tt.fields.S,
			}
			assert.Equal(t, tt.want, s.Contains(tt.args.v))
		})
	}
}

func TestSlice_ContainsAll(t *testing.T) {
	type fields struct {
		S []interface{}
	}
	type args struct {
		v interface{}
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      bool
		assertion assert.ErrorAssertionFunc
	}{
		{"error checking regular go slice", fields{[]interface{}{"some", "slice"}}, args{[]interface{}{"some"}}, false, assert.Error},
		{"no value exists", fields{[]interface{}{"some", "slice"}}, args{&Slice{[]interface{}{"not", "exists"}}}, false, assert.NoError},
		{"some values exist", fields{[]interface{}{"some", "slice"}}, args{&Slice{[]interface{}{"some", "not exists"}}}, false, assert.NoError},
		{"all values exist and of comparable type", fields{[]interface{}{1, 2, "slice"}}, args{&Slice{[]interface{}{1, "slice"}}}, true, assert.NoError},
		{"values exist and of non-comparable type", fields{[]interface{}{1, map[string]interface{}{"a": []int{1}}, "some", []string{"slice"}}}, args{&Slice{[]interface{}{[]string{"slice"}, map[string]interface{}{"a": []int{1}}}}}, true, assert.NoError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Slice{
				S: tt.fields.S,
			}
			got, err := s.ContainsAll(tt.args.v)
			tt.assertion(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSlice_Equals(t *testing.T) {
	type fields struct {
		S []interface{}
	}
	type args struct {
		v interface{}
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      bool
		assertion assert.ErrorAssertionFunc
	}{
		{"error checking regular go slice", fields{[]interface{}{"some", "slice"}}, args{[]interface{}{"some"}}, false, assert.Error},
		{"some values exist", fields{[]interface{}{"some", "slice"}}, args{&Slice{[]interface{}{"some", "not exists"}}}, false, assert.NoError},
		{"same values, different order", fields{[]interface{}{"some", "slice"}}, args{&Slice{[]interface{}{"slice", "some"}}}, false, assert.NoError},
		{"all values exist and of comparable type", fields{[]interface{}{1, 2, "slice"}}, args{&Slice{[]interface{}{1, 2, "slice"}}}, true, assert.NoError},
		{"slices are equal and values of comparable type", fields{[]interface{}{1, "slice"}}, args{&Slice{[]interface{}{1, "slice"}}}, true, assert.NoError},
		{"slices are equal and of non-comparable type", fields{[]interface{}{1, map[string]interface{}{"a": []int{1}}, "some", []string{"slice"}}}, args{&Slice{[]interface{}{1, map[string]interface{}{"a": []int{1}}, "some", []string{"slice"}}}}, true, assert.NoError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Slice{
				S: tt.fields.S,
			}
			got, err := s.Equals(tt.args.v)
			tt.assertion(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSlice_Get(t *testing.T) {
	type fields struct {
		S []interface{}
	}
	type args struct {
		i int
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      interface{}
		assertion assert.ErrorAssertionFunc
	}{
		{"negative index", fields{[]interface{}{1, "2"}}, args{-1}, nil, assert.Error},
		{"index equals length", fields{[]interface{}{1, "2"}}, args{2}, nil, assert.Error},
		{"zero index", fields{[]interface{}{1, "2"}}, args{0}, 1, assert.NoError},
		{"index is length-1", fields{[]interface{}{1, "2"}}, args{1}, "2", assert.NoError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Slice{
				S: tt.fields.S,
			}
			got, err := s.Get(tt.args.i)
			tt.assertion(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSlice_IsEmpty(t *testing.T) {
	type fields struct {
		S []interface{}
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{"nil slice is empty", fields{nil}, true},
		{"zero length slice is empty", fields{[]interface{}{}}, true},
		{"non-zero length slice is not empty", fields{[]interface{}{1}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Slice{
				S: tt.fields.S,
			}
			assert.Equal(t, tt.want, s.IsEmpty())
		})
	}
}

func TestSlice_Iterator(t *testing.T) {
	type fields struct {
		S []interface{}
	}
	tests := []struct {
		name   string
		fields fields
		want   *Iterator
	}{
		{"should return iterator", fields{[]interface{}{1, 2}}, &Iterator{s: &Slice{[]interface{}{1, 2}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Slice{
				S: tt.fields.S,
			}
			assert.Equal(t, tt.want, s.Iterator())
		})
	}
}

func TestSlice_Remove(t *testing.T) {
	type fields struct {
		S []interface{}
	}
	type args struct {
		v interface{}
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      bool
		assertion assert.ErrorAssertionFunc
		wantS     []interface{}
	}{
		{"works with nil", fields{nil}, args{1}, false, assert.NoError, nil},
		{"works with empty", fields{[]interface{}{}}, args{1}, false, assert.NoError, []interface{}{}},
		{"does nothing for non-existant", fields{[]interface{}{1}}, args{2}, false, assert.NoError, []interface{}{1}},
		{"removes comparable", fields{[]interface{}{1, 2}}, args{2}, true, assert.NoError, []interface{}{1}},
		{"removes non comparable", fields{[]interface{}{1, []int{2}}}, args{[]int{2}}, true, assert.NoError, []interface{}{1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Slice{
				S: tt.fields.S,
			}
			removed, err := s.Remove(tt.args.v)
			assert.Equal(t, tt.want, removed)
			assert.Equal(t, tt.wantS, s.S)
			tt.assertion(t, err)
		})
	}
}

func TestSlice_RemoveAll(t *testing.T) {
	type fields struct {
		S []interface{}
	}
	type args struct {
		v interface{}
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      bool
		assertion assert.ErrorAssertionFunc
		wantS     []interface{}
	}{
		{"works with nil", fields{nil}, args{&Slice{[]interface{}{1}}}, false, assert.NoError, nil},
		{"works with empty", fields{[]interface{}{}}, args{&Slice{[]interface{}{1}}}, false, assert.NoError, []interface{}{}},
		{"does nothing for non-existant", fields{[]interface{}{1, 3}}, args{&Slice{[]interface{}{2}}}, false, assert.NoError, []interface{}{1, 3}},
		{"removes comparable", fields{[]interface{}{1, 2, 3}}, args{&Slice{[]interface{}{1, 3}}}, true, assert.NoError, []interface{}{2}},
		{"removes all comparable", fields{[]interface{}{1, 2, 3}}, args{&Slice{[]interface{}{1, 2, 3}}}, true, assert.NoError, []interface{}{}},
		{"removes non comparable", fields{[]interface{}{1, []int{2}, 3}}, args{&Slice{[]interface{}{1, []int{2}}}}, true, assert.NoError, []interface{}{3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Slice{
				S: tt.fields.S,
			}
			got, err := s.RemoveAll(tt.args.v)
			tt.assertion(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantS, s.S)
		})
	}
}

func TestSlice_RetainAll(t *testing.T) {
	type fields struct {
		S []interface{}
	}
	type args struct {
		v interface{}
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      bool
		assertion assert.ErrorAssertionFunc
		wantS     []interface{}
	}{
		{"works with nil", fields{nil}, args{&Slice{[]interface{}{1}}}, false, assert.NoError, nil},
		{"works with empty", fields{[]interface{}{}}, args{&Slice{[]interface{}{1}}}, false, assert.NoError, []interface{}{}},
		{"does nothing for existant", fields{[]interface{}{1, 3}}, args{&Slice{[]interface{}{1, 3}}}, false, assert.NoError, []interface{}{1, 3}},
		{"retains comparable", fields{[]interface{}{1, 2, 3}}, args{&Slice{[]interface{}{1, 3}}}, true, assert.NoError, []interface{}{1, 3}},
		{"retains only existant", fields{[]interface{}{1, 2, 3}}, args{&Slice{[]interface{}{4, 5}}}, true, assert.NoError, []interface{}{}},
		{"retains non comparable", fields{[]interface{}{1, []int{2}, 3}}, args{&Slice{[]interface{}{1, []int{2}}}}, true, assert.NoError, []interface{}{1, []int{2}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Slice{
				S: tt.fields.S,
			}
			got, err := s.RetainAll(tt.args.v)
			tt.assertion(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantS, s.S)
		})
	}
}

func TestSlice_Size(t *testing.T) {
	type fields struct {
		S []interface{}
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{"nil slice", fields{nil}, 0},
		{"empty slice", fields{[]interface{}{}}, 0},
		{"short slice", fields{[]interface{}{""}}, 1},
		{"some slice", fields{[]interface{}{1, "2", []int{3}}}, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Slice{
				S: tt.fields.S,
			}
			assert.Equal(t, tt.want, s.Size())
		})
	}
}

func TestSlice_Set(t *testing.T) {
	type fields struct {
		S []interface{}
	}
	type args struct {
		i int
		v interface{}
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      interface{}
		assertion assert.ErrorAssertionFunc
		wantS     []interface{}
	}{
		{"empty", fields{[]interface{}{}}, args{0, "a"}, nil, assert.Error, []interface{}{}},
		{"negative index", fields{[]interface{}{1}}, args{-1, "a"}, nil, assert.Error, []interface{}{1}},
		{"past end", fields{[]interface{}{1}}, args{1, "a"}, nil, assert.Error, []interface{}{1}},
		{"single element", fields{[]interface{}{1}}, args{0, "a"}, interface{}(1), assert.NoError, []interface{}{"a"}},
		{"last element", fields{[]interface{}{1, "x", "y"}}, args{2, 3}, interface{}("y"), assert.NoError, []interface{}{1, "x", 3}},
		{"first element", fields{[]interface{}{1, "x", "y"}}, args{0, 3}, interface{}(1), assert.NoError, []interface{}{3, "x", "y"}},
		{"middle element", fields{[]interface{}{1, "x", "y"}}, args{1, 3}, interface{}("x"), assert.NoError, []interface{}{1, 3, "y"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Slice{
				S: tt.fields.S,
			}
			got, err := s.Set(tt.args.i, tt.args.v)
			tt.assertion(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantS, s.S)
		})
	}
}

func TestSlice_ToArray(t *testing.T) {
	type fields struct {
		S []interface{}
	}
	tests := []struct {
		name   string
		fields fields
		want   *Slice
	}{
		{"works with nil", fields{nil}, &Slice{}},
		{"works with empty", fields{[]interface{}{}}, &Slice{[]interface{}{}}},
		{"returns copy", fields{[]interface{}{1, "2", []int{3}}}, &Slice{[]interface{}{1, "2", []int{3}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Slice{
				S: tt.fields.S,
			}
			got := s.ToArray()
			assert.Equal(t, tt.want, got)
			if len(s.S) > 0 {
				s.S[0] = "some new value"
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestNewIterator(t *testing.T) {
	type args struct {
		v interface{}
	}
	tests := []struct {
		name string
		args args
		want *Iterator
	}{
		{"creates iterator from go slice", args{[]interface{}{1, 2, 3}}, &Iterator{&Slice{[]interface{}{1, 2, 3}}, 0}},
		{"creates iterator from vtl slice", args{&Slice{[]interface{}{4, 5, 6}}}, &Iterator{&Slice{[]interface{}{4, 5, 6}}, 0}},
		{"creates iterator from scalar", args{1}, &Iterator{&Slice{[]interface{}{1}}, 0}},
		{"creates iterator from map as scalar", args{map[string]interface{}{"1": 2, "3": 4, "5": 6}}, &Iterator{&Slice{[]interface{}{map[string]interface{}{"1": 2, "3": 4, "5": 6}}}, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NewIterator(tt.args.v))
		})
	}
}

func TestIterator_Next(t *testing.T) {
	type fields struct {
		s Collection
		i int
	}
	tests := []struct {
		name   string
		fields fields
		want   interface{}
	}{
		{"slice from the beginning", fields{&Slice{[]interface{}{1, 2, 3}}, 0}, 1},
		{"slice middle", fields{&Slice{[]interface{}{1, 2, 3}}, 1}, 2},
		{"slice last", fields{&Slice{[]interface{}{1, 2, 3}}, 2}, 3},
		{"slice after last", fields{&Slice{[]interface{}{1, 2, 3}}, 3}, nil},
		{"range from the beginning", fields{NewRange(1, 3), 0}, 1},
		{"range middle", fields{NewRange(1, 3), 1}, 2},
		{"range last", fields{NewRange(1, 3), 2}, 3},
		{"range after last", fields{NewRange(1, 3), 3}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &Iterator{
				s: tt.fields.s,
				i: tt.fields.i,
			}
			assert.Equal(t, tt.want, i.Next())
		})
	}
}

func TestIterator_HasNext(t *testing.T) {
	type fields struct {
		s Collection
		i int
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{"slice in the beginning", fields{&Slice{[]interface{}{1, 2, 3}}, 0}, true},
		{"slice middle", fields{&Slice{[]interface{}{1, 2, 3}}, 1}, true},
		{"slice last", fields{&Slice{[]interface{}{1, 2, 3}}, 2}, true},
		{"slice after last", fields{&Slice{[]interface{}{1, 2, 3}}, 3}, false},
		{"range in the beginning", fields{NewRange(1, 3), 0}, true},
		{"range middle", fields{NewRange(1, 3), 1}, true},
		{"range last", fields{NewRange(1, 3), 2}, true},
		{"range after last", fields{NewRange(1, 3), 3}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &Iterator{
				s: tt.fields.s,
				i: tt.fields.i,
			}
			assert.Equal(t, tt.want, i.HasNext())
		})
	}
}

func TestStr_CharAt(t *testing.T) {
	type args struct {
		i int
	}
	tests := []struct {
		name      string
		s         Str
		args      args
		want      rune
		assertion assert.ErrorAssertionFunc
	}{
		{"empty string", Str(""), args{0}, 0, assert.Error},
		{"ascii negative index", Str("123"), args{-1}, 0, assert.Error},
		{"ascii out of bounds", Str("123"), args{3}, 0, assert.Error},
		{"ascii beginning", Str("123"), args{0}, '1', assert.NoError},
		{"ascii middle", Str("123"), args{1}, '2', assert.NoError},
		{"ascii last", Str("123"), args{2}, '3', assert.NoError},
		{"utf negative", Str("日本語"), args{-1}, 0, assert.Error},
		{"utf out of bounds", Str("日本語"), args{3}, 0, assert.Error},
		{"utf beginning", Str("日本語"), args{0}, '日', assert.NoError},
		{"utf middle", Str("日本語"), args{1}, '本', assert.NoError},
		{"utf last", Str("日本語"), args{2}, '語', assert.NoError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.CharAt(tt.args.i)
			tt.assertion(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStr_CodePointAt(t *testing.T) {
	type args struct {
		i int
	}
	tests := []struct {
		name      string
		s         Str
		args      args
		assertion assert.ErrorAssertionFunc
		wantErr   error
	}{
		{"should return not implemented", Str(""), args{0}, assert.Error, errNotImplemented},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.s.CodePointAt(tt.args.i)
			tt.assertion(t, err)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func TestStr_CodePointBefore(t *testing.T) {
	type args struct {
		i int
	}
	tests := []struct {
		name      string
		s         Str
		args      args
		assertion assert.ErrorAssertionFunc
		wantErr   error
	}{
		{"should return not implemented", Str(""), args{0}, assert.Error, errNotImplemented},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.s.CodePointBefore(tt.args.i)
			tt.assertion(t, err)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func TestStr_CodePointCount(t *testing.T) {
	type args struct {
		start int
		end   int
	}
	tests := []struct {
		name      string
		s         Str
		args      args
		assertion assert.ErrorAssertionFunc
		wantErr   error
	}{
		{"should return not implemented", Str(""), args{0, 1}, assert.Error, errNotImplemented},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.s.CodePointCount(tt.args.start, tt.args.end)
			tt.assertion(t, err)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func TestStr_CompareTo(t *testing.T) {
	type args struct {
		o string
	}
	tests := []struct {
		name string
		s    Str
		args args
		want int
	}{
		{"empty string", Str(""), args{""}, 0},
		{"ascii length different", Str("abcd"), args{"ab"}, 2},
		{"ascii different", Str("abca"), args{"abcd"}, -3},
		{"ascii eq", Str("abcd"), args{"abcd"}, 0},
		{"utf length different", Str("日本語"), args{"a"}, 25988},
		{"utf different", Str("日本語"), args{"日本誙"}, 5},
		{"utf eq", Str("日本語"), args{"日本語"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.s.CompareTo(tt.args.o))
		})
	}
}

func TestStr_CompareToIgnoreCase(t *testing.T) {
	type args struct {
		o string
	}
	tests := []struct {
		name string
		s    Str
		args args
		want int
	}{
		{"empty string", Str(""), args{""}, 0},
		{"ascii length different", Str("abcd"), args{"ab"}, 2},
		{"ascii different", Str("abca"), args{"ABCD"}, -3},
		{"ascii eq", Str("abcd"), args{"AbCD"}, 0},
		{"utf length different", Str("日本語"), args{"a"}, 25988},
		{"utf different", Str("STRASSE"), args{"straße"}, -108},
		{"utf eq", Str("STRAßE"), args{"straße"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.s.CompareToIgnoreCase(tt.args.o))
		})
	}
}

func TestStr_Concat(t *testing.T) {
	type args struct {
		o string
	}
	tests := []struct {
		name  string
		s     Str
		args  args
		want  string
		wantS string
	}{
		{"empty strings", Str(""), args{""}, "", ""},
		{"non-empty strings", Str("ab"), args{"日本語"}, "ab日本語", "ab"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.s.Concat(tt.args.o))
			assert.Equal(t, tt.wantS, string(tt.s), "keep receiver unchanged")
		})
	}
}

func TestStr_Contains(t *testing.T) {
	type args struct {
		o string
	}
	tests := []struct {
		name string
		s    Str
		args args
		want bool
	}{
		{"empty strings", Str(""), args{""}, true},
		{"contains ascii", Str("abcd"), args{"ab"}, true},
		{"doesn't contain ascii", Str("abcd"), args{"ef"}, false},
		{"contains utf", Str("日本語"), args{"本"}, true},
		{"doesn't contain utf", Str("日本語"), args{"札"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.s.Contains(tt.args.o))
		})
	}
}

func TestStr_ContentEquals(t *testing.T) {
	type args struct {
		o string
	}
	tests := []struct {
		name string
		s    Str
		args args
		want bool
	}{
		{"empty strings", Str(""), args{""}, true},
		{"not equals ascii", Str("abcde"), args{"abcdf"}, false},
		{"not equals utf - russian e", Str("abcde"), args{"abcdе"}, false},
		{"equals ascii", Str("abcde"), args{"abcde"}, true},
		{"equals utf", Str("日本語"), args{"日本語"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.s.ContentEquals(tt.args.o))
		})
	}
}

func TestStr_EndsWith(t *testing.T) {
	type args struct {
		suffix string
	}
	tests := []struct {
		name string
		s    Str
		args args
		want bool
	}{
		{"empty strings", Str(""), args{""}, true},
		{"ascii ends", Str("asdf"), args{"df"}, true},
		{"ascii doesn't end", Str("asdf"), args{"de"}, false},
		{"utf ends", Str("日本語"), args{"語"}, true},
		{"utf doesn't end - russian e", Str("asde"), args{"dе"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.s.EndsWith(tt.args.suffix))
		})
	}
}

func TestStr_Equals(t *testing.T) {
	type args struct {
		o string
	}
	tests := []struct {
		name string
		s    Str
		args args
		want bool
	}{
		{"empty strings", Str(""), args{""}, true},
		{"not equals ascii", Str("abcde"), args{"abcdf"}, false},
		{"not equals utf - russian e", Str("abcde"), args{"abcdе"}, false},
		{"equals ascii", Str("abcde"), args{"abcde"}, true},
		{"equals utf", Str("日本語"), args{"日本語"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.s.Equals(tt.args.o))
		})
	}
}

func TestStr_EqualsIgnoreCase(t *testing.T) {
	type args struct {
		o string
	}
	tests := []struct {
		name string
		s    Str
		args args
		want bool
	}{
		{"empty string", Str(""), args{""}, true},
		{"ascii length different", Str("abcd"), args{"ab"}, false},
		{"ascii different", Str("abca"), args{"ABCD"}, false},
		{"ascii eq", Str("abcd"), args{"AbCD"}, true},
		{"utf length different", Str("日本語"), args{"a"}, false},
		{"utf different", Str("STRASSE"), args{"straße"}, false},
		{"utf eq", Str("STRAßE"), args{"straße"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.s.EqualsIgnoreCase(tt.args.o))
		})
	}
}

func TestStr_GetBytes(t *testing.T) {
	tests := []struct {
		name string
		s    Str
		want []byte
	}{
		{"empty string", Str(""), []byte{}},
		{"ascii", Str("abcd"), []byte{0x61, 0x62, 0x63, 0x64}},
		{"utf", Str("日本語"), []byte{0xe6, 0x97, 0xa5, 0xe6, 0x9c, 0xac, 0xe8, 0xaa, 0x9e}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.s.GetBytes())
		})
	}
}

func TestStr_IndexOf(t *testing.T) {
	type args struct {
		o string
	}
	tests := []struct {
		name string
		s    Str
		args args
		want int
	}{
		{"empty string in empty string", Str(""), args{""}, 0},
		{"empty string in non-empty string", Str("asd"), args{""}, 0},
		{"non-empty string in empty string", Str(""), args{"a"}, -1},
		{"ascii match", Str("asdf"), args{"df"}, 2},
		{"ascii no match", Str("asdf"), args{"de"}, -1},
		{"utf match", Str("日本語"), args{"語"}, 2},
		{"utf no match", Str("日本語"), args{"札"}, -1},
		{"mixed match", Str("asdф"), args{"dф"}, 2},
		{"mixed no match", Str("asdф"), args{"dе"}, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.s.IndexOf(tt.args.o))
		})
	}
}

func TestStr_IsEmpty(t *testing.T) {
	tests := []struct {
		name string
		s    Str
		want bool
	}{
		{"empty string", Str(""), true},
		{"non-empty string", Str(string([]byte{0})), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.s.IsEmpty())
		})
	}
}

func TestStr_LastIndexOf(t *testing.T) {
	type args struct {
		o string
	}
	tests := []struct {
		name string
		s    Str
		args args
		want int
	}{
		{"empty string in empty string", Str(""), args{""}, 0},
		{"empty string in non-empty string", Str("asd"), args{""}, 3},
		{"non-empty string in empty string", Str(""), args{"a"}, -1},
		{"ascii match", Str("asdfdf"), args{"df"}, 4},
		{"ascii no match", Str("asdf"), args{"de"}, -1},
		{"utf match", Str("日本語語"), args{"語"}, 3},
		{"utf no match", Str("日本語"), args{"札"}, -1},
		{"mixed match", Str("asdфdф"), args{"dф"}, 4},
		{"mixed no match", Str("asdф"), args{"dе"}, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.s.LastIndexOf(tt.args.o))
		})
	}
}

func TestStr_Length(t *testing.T) {
	tests := []struct {
		name string
		s    Str
		want int
	}{
		{"empty string", Str(""), 0},
		{"ascii", Str("abcd"), 4},
		{"utf", Str("日本語"), 3},
		{"mixed", Str("asd日本語"), 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.s.Length())
		})
	}
}

func TestStr_Matches(t *testing.T) {
	type args struct {
		regex string
	}
	tests := []struct {
		name      string
		s         Str
		args      args
		want      bool
		assertion assert.ErrorAssertionFunc
	}{
		{"empty string matches", Str(""), args{``}, true, assert.NoError},
		{"invalid regex", Str(""), args{`(`}, false, assert.Error},
		{"no match", Str("gopher"), args{`(gopher){2}`}, false, assert.NoError},
		{"match 1", Str("gophergopher"), args{`(gopher){2}`}, true, assert.NoError},
		{"match 2", Str("gophergophergopher"), args{`(gopher){2}`}, true, assert.NoError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.Matches(tt.args.regex)
			tt.assertion(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStr_Replace(t *testing.T) {
	type args struct {
		old string
		new string
	}
	tests := []struct {
		name  string
		s     Str
		args  args
		want  string
		wantS string
	}{
		{"empty string", Str(""), args{"", "asd"}, "asd", ""},
		{"whole string", Str("ab"), args{"ab", "asd"}, "asd", "ab"},
		{"single entry", Str("ab"), args{"b", "sd"}, "asd", "ab"},
		{"multiple entries", Str("abab"), args{"b", "sd"}, "asdasd", "abab"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.s.Replace(tt.args.old, tt.args.new))
			assert.Equal(t, tt.wantS, string(tt.s), "receiver kept intact")
		})
	}
}

func TestStr_ReplaceAll(t *testing.T) {
	type args struct {
		regex       string
		replacement string
	}
	tests := []struct {
		name      string
		s         Str
		args      args
		want      string
		assertion assert.ErrorAssertionFunc
		wantS     string
	}{
		{"empty string", Str(""), args{"", "asd"}, "asd", assert.NoError, ""},
		{"whole string literally", Str("ab"), args{"ab", "asd"}, "asd", assert.NoError, "ab"},
		{"whole string via regex", Str("ab"), args{".*", "asd"}, "asd", assert.NoError, "ab"},
		{"single entry", Str("ab"), args{".", "sd"}, "sdsd", assert.NoError, "ab"},
		{"multiple entries", Str("abab"), args{"(a)b", "${1}sd"}, "asdasd", assert.NoError, "abab"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.ReplaceAll(tt.args.regex, tt.args.replacement)
			tt.assertion(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantS, string(tt.s), "receiver kept intact")
		})
	}
}

func TestStr_ReplaceFirst(t *testing.T) {
	type args struct {
		regex       string
		replacement string
	}
	tests := []struct {
		name      string
		s         Str
		args      args
		want      string
		assertion assert.ErrorAssertionFunc
		wantS     string
	}{
		{"empty string", Str(""), args{"", "asd"}, "asd", assert.NoError, ""},
		{"whole string literally", Str("ab"), args{"ab", "asd"}, "asd", assert.NoError, "ab"},
		{"whole string via regex", Str("ab"), args{".*", "asd"}, "asd", assert.NoError, "ab"},
		{"single entry", Str("ab"), args{".", "sd"}, "sdb", assert.NoError, "ab"},
		{"multiple entries", Str("abab"), args{"(a)b", "${1}sd"}, "asdab", assert.NoError, "abab"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.ReplaceFirst(tt.args.regex, tt.args.replacement)
			tt.assertion(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantS, string(tt.s), "receiver kept intact")
		})
	}
}

func TestStr_Split(t *testing.T) {
	type args struct {
		regex string
	}
	tests := []struct {
		name      string
		s         Str
		args      args
		want      []string
		assertion assert.ErrorAssertionFunc
		wantS     string
	}{
		{"empty string", Str(""), args{""}, []string{}, assert.NoError, ""},
		{"boo:and:foo", Str("boo:and:foo"), args{"o"}, []string{"b", "", ":and:f"}, assert.NoError, "boo:and:foo"},
		{"boo:and:foo", Str("boo:and:foo"), args{":"}, []string{"boo", "and", "foo"}, assert.NoError, "boo:and:foo"},
		{"regex", Str("abc"), args{"."}, []string{}, assert.NoError, "abc"},
		{"regex 2", Str("ab-c"), args{"b.?"}, []string{"a", "c"}, assert.NoError, "ab-c"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.Split(tt.args.regex)
			tt.assertion(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantS, string(tt.s), "receiver kept intact")
		})
	}
}

func TestStr_StartsWith(t *testing.T) {
	type args struct {
		prefix string
	}
	tests := []struct {
		name string
		s    Str
		args args
		want bool
	}{
		{"empty strings", Str(""), args{""}, true},
		{"ascii starts", Str("asdf"), args{"as"}, true},
		{"ascii doesn't start", Str("asdf"), args{"aw"}, false},
		{"utf starts", Str("日本語"), args{"日"}, true},
		{"utf doesn't end - russian a", Str("asde"), args{"аs"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.s.StartsWith(tt.args.prefix))
		})
	}
}

func TestStr_SubSequence(t *testing.T) {
	type args struct {
		start int
		end   int
	}
	tests := []struct {
		name      string
		s         Str
		args      args
		want      string
		assertion assert.ErrorAssertionFunc
	}{
		{"negative start", Str("asd"), args{-1, 0}, "", assert.Error},
		{"negative end", Str("asd"), args{0, -1}, "", assert.Error},
		{"end before start", Str("asd"), args{1, 0}, "", assert.Error},
		{"end past len", Str("asd"), args{1, 4}, "", assert.Error},
		{"empty string and all zero", Str(""), args{0, 0}, "", assert.NoError},
		{"ascii match", Str("asdf"), args{2, 4}, "df", assert.NoError},
		{"utf match", Str("日本語"), args{0, 2}, "日本", assert.NoError},
		{"mixed match", Str("asdф"), args{2, 4}, "dф", assert.NoError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.SubSequence(tt.args.start, tt.args.end)
			tt.assertion(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStr_Substring(t *testing.T) {
	type args struct {
		start int
		end   int
	}
	tests := []struct {
		name      string
		s         Str
		args      args
		want      string
		assertion assert.ErrorAssertionFunc
	}{
		{"negative start", Str("asd"), args{-1, 0}, "", assert.Error},
		{"negative end", Str("asd"), args{0, -1}, "", assert.Error},
		{"end before start", Str("asd"), args{1, 0}, "", assert.Error},
		{"end past len", Str("asd"), args{1, 4}, "", assert.Error},
		{"empty string and all zero", Str(""), args{0, 0}, "", assert.NoError},
		{"ascii match", Str("asdf"), args{2, 4}, "df", assert.NoError},
		{"utf match", Str("日本語"), args{0, 2}, "日本", assert.NoError},
		{"mixed match", Str("asdф"), args{2, 4}, "dф", assert.NoError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.Substring(tt.args.start, tt.args.end)
			tt.assertion(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStr_ToLowerCase(t *testing.T) {
	tests := []struct {
		name string
		s    Str
		want string
	}{
		{"empty string", Str(""), ""},
		{"ascii", Str("ABcd"), "abcd"},
		{"utf", Str("STraßE"), "straße"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.s.ToLowerCase())
		})
	}
}

func TestStr_ToString(t *testing.T) {
	tests := []struct {
		name string
		s    Str
		want string
	}{
		{"empty string", Str(""), ""},
		{"ascii", Str("ABcd"), "ABcd"},
		{"utf", Str("STraßE"), "STraßE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.s.ToString())
		})
	}
}

func TestStr_ToUpperCase(t *testing.T) {
	tests := []struct {
		name string
		s    Str
		want string
	}{
		{"empty string", Str(""), ""},
		{"ascii", Str("ABcd"), "ABCD"},
		{"utf", Str("STraßE"), "STRASSE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.s.ToUpperCase())
		})
	}
}

func TestStr_Trim(t *testing.T) {
	tests := []struct {
		name string
		s    Str
		want string
	}{
		{"empty string", Str(""), ""},
		{"whitespace", Str(" \t \n"), ""},
		{"ascii", Str("   AB cd   "), "AB cd"},
		{"utf", Str("\t 日本語\n"), "日本語"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.s.Trim())
		})
	}
}

func TestNewRange(t *testing.T) {
	type args struct {
		start int
		end   int
	}
	tests := []struct {
		name string
		args args
		want *Range
	}{
		{"single element", args{0, 0}, &Range{0, 0, 1}},
		{"up positive", args{0, 5}, &Range{0, 5, 1}},
		{"down positive", args{5, 0}, &Range{5, 0, -1}},
		{"up negative", args{-5, -3}, &Range{-5, -3, 1}},
		{"down negative", args{-3, -5}, &Range{-3, -5, -1}},
		{"up cross", args{-3, 5}, &Range{-3, 5, 1}},
		{"down cross", args{3, -5}, &Range{3, -5, -1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NewRange(tt.args.start, tt.args.end))
		})
	}
}

func TestRange_Get(t *testing.T) {
	type fields struct {
		start int
		end   int
		diff  int
	}
	type args struct {
		i int
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      interface{}
		assertion assert.ErrorAssertionFunc
	}{
		{"negative index", fields{0, 3, 1}, args{-1}, nil, assert.Error},
		{"after last", fields{0, 3, 1}, args{5}, nil, assert.Error},
		{"single element", fields{2, 2, 1}, args{0}, 2, assert.NoError},
		{"first up", fields{1, 3, 1}, args{0}, 1, assert.NoError},
		{"middle up", fields{1, 3, 1}, args{1}, 2, assert.NoError},
		{"last up", fields{1, 3, 1}, args{2}, 3, assert.NoError},
		{"first down", fields{3, 1, -1}, args{0}, 3, assert.NoError},
		{"middle down", fields{3, 1, -1}, args{1}, 2, assert.NoError},
		{"last down", fields{3, 1, -1}, args{2}, 1, assert.NoError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Range{
				start: tt.fields.start,
				end:   tt.fields.end,
				diff:  tt.fields.diff,
			}
			got, err := r.Get(tt.args.i)
			tt.assertion(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRange_IndexOf(t *testing.T) {
	type fields struct {
		start int
		end   int
		diff  int
	}
	type args struct {
		i int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{"single element no match", fields{2, 2, 1}, args{0}, -1},
		{"single element match", fields{2, 2, 1}, args{2}, 0},
		{"first up", fields{1, 3, 1}, args{1}, 0},
		{"middle up", fields{1, 3, 1}, args{2}, 1},
		{"last up", fields{1, 3, 1}, args{3}, 2},
		{"first down", fields{3, 1, -1}, args{3}, 0},
		{"middle down", fields{3, 1, -1}, args{2}, 1},
		{"last down", fields{3, 1, -1}, args{1}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Range{
				start: tt.fields.start,
				end:   tt.fields.end,
				diff:  tt.fields.diff,
			}
			assert.Equal(t, tt.want, r.IndexOf(tt.args.i))
		})
	}
}

func TestRange_Iterator(t *testing.T) {
	type fields struct {
		start int
		end   int
		diff  int
	}
	tests := []struct {
		name   string
		fields fields
		want   *Iterator
	}{
		{"just a range", fields{-5, 2, 1}, &Iterator{&Range{-5, 2, 1}, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Range{
				start: tt.fields.start,
				end:   tt.fields.end,
				diff:  tt.fields.diff,
			}
			assert.Equal(t, tt.want, r.Iterator())
		})
	}
}

func TestRange_LastIndexOf(t *testing.T) {
	type fields struct {
		start int
		end   int
		diff  int
	}
	type args struct {
		i int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		// same as IndexOf
		{"single element no match", fields{2, 2, 1}, args{0}, -1},
		{"single element match", fields{2, 2, 1}, args{2}, 0},
		{"first up", fields{1, 3, 1}, args{1}, 0},
		{"middle up", fields{1, 3, 1}, args{2}, 1},
		{"last up", fields{1, 3, 1}, args{3}, 2},
		{"first down", fields{3, 1, -1}, args{3}, 0},
		{"middle down", fields{3, 1, -1}, args{2}, 1},
		{"last down", fields{3, 1, -1}, args{1}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Range{
				start: tt.fields.start,
				end:   tt.fields.end,
				diff:  tt.fields.diff,
			}
			assert.Equal(t, tt.want, r.LastIndexOf(tt.args.i))
		})
	}
}

func TestRange_Set(t *testing.T) {
	type fields struct {
		start int
		end   int
		diff  int
	}
	type args struct {
		i int
		v interface{}
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      interface{}
		assertion assert.ErrorAssertionFunc
	}{
		{"index in range", fields{-2, 2, 1}, args{0, 2}, nil, assert.Error},
		{"index outside range", fields{-2, 2, 1}, args{8, 1}, nil, assert.Error},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Range{
				start: tt.fields.start,
				end:   tt.fields.end,
				diff:  tt.fields.diff,
			}
			prev, err := r.Set(tt.args.i, tt.args.v)
			assert.Equal(t, tt.want, prev)
			tt.assertion(t, err)
		})
	}
}

func TestRange_Size(t *testing.T) {
	type fields struct {
		start int
		end   int
		diff  int
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{"single element", fields{0, 0, 1}, 1},
		{"up positive", fields{0, 5, 1}, 6},
		{"down positive", fields{5, 0, -1}, 6},
		{"up negative", fields{-5, -3, 1}, 3},
		{"down negative", fields{-3, -5, -1}, 3},
		{"up cross", fields{-3, 5, 1}, 9},
		{"down cross", fields{3, -5, -1}, 9},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Range{
				start: tt.fields.start,
				end:   tt.fields.end,
				diff:  tt.fields.diff,
			}
			assert.Equal(t, tt.want, r.Size())
		})
	}
}
