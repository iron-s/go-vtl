package govtl

import (
	"bytes"
	"html/template"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type _m map[string]interface{}

const result = `
 There are customers: 
    First,
    Second,
    Third,
    Fourth,
    Fifth,
    First,
    Second,
    Third,
    Fourth,
    Fifth,
    First,
    Second,
    Third,
    Fourth,
    Fifth,
    First,
    Second,
    Third,
    Fourth,
    Fifth,
    First,
    Second,
    Third,
    Fourth,
    Fifth
`

const goTemplate = `
{{range $i, $c := $.customerList -}}
{{if eq $i 0}} There are customers: {{end}}
    {{- if gt $i 0}},{{end}}
    {{ $c.Name}}
{{- else}}
    Nobody around
{{- end}}
`

var goTmpl = template.Must(template.New("test").Parse(goTemplate))

const vtlTemplate = `
#foreach( $customer in $customerList )
    #if( $foreach.first ) There are customers: 
    #end
    $customer.Name#if( $foreach.hasNext ),#end
#else
    Nobody around
#end
`

var vtlTmpl = Must(Parse(vtlTemplate, "", ""))

var m = map[string]interface{}{
	"customerList": []struct {
		Name, Address string
	}{
		{"First", "Address1"},
		{"Second", "Address2"},
		{"Third", "Address3"},
		{"Fourth", "Address4"},
		{"Fifth", "Address5"},
		{"First", "Address1"},
		{"Second", "Address2"},
		{"Third", "Address3"},
		{"Fourth", "Address4"},
		{"Fifth", "Address5"},
		{"First", "Address1"},
		{"Second", "Address2"},
		{"Third", "Address3"},
		{"Fourth", "Address4"},
		{"Fifth", "Address5"},
		{"First", "Address1"},
		{"Second", "Address2"},
		{"Third", "Address3"},
		{"Fourth", "Address4"},
		{"Fifth", "Address5"},
		{"First", "Address1"},
		{"Second", "Address2"},
		{"Third", "Address3"},
		{"Fourth", "Address4"},
		{"Fifth", "Address5"},
	},
}

func TestExecuteGo(t *testing.T) {
	var b strings.Builder
	err := goTmpl.Execute(&b, m)
	if err != nil {
		t.Error("expect nil error, got", err)
	}
	if b.String() != result {
		t.Error("expect correct output", result, ", got", b.String(), ".")
	}
}

func TestExecuteVtl(t *testing.T) {
	var b strings.Builder
	err := vtlTmpl.Execute(&b, m)
	if err != nil {
		t.Error("expect nil error, got", err)
	}
	if b.String() != result {
		t.Error("expect correct output", result, ", got", b.String(), ".")
	}
}

func TestExecuteShortCircuit(t *testing.T) {
	divByZero := assert.ErrorAssertionFunc(func(t assert.TestingT, err error, msg ...interface{}) bool {
		return assert.EqualError(t, unposErr(err), "division by zero", msg)
	})
	tests := []struct {
		name      string
		tmpl      string
		want      string
		assertion assert.ErrorAssertionFunc
	}{
		{"no short circuit with false in or",
			"#if(false or 1/0)true#{else}false#end",
			"", divByZero},
		{"no short circuit with true in and",
			"#if(true and 1/0)true#{else}false#end",
			"", divByZero},
		{"short circuit naked bool in or",
			"#if(true or 1/0)true#end",
			"true", assert.NoError},
		{"short circuit naked bool in and",
			"#if(false and 1/0)true#{else}false#end",
			"false", assert.NoError},
		{"short circuit bool result in or",
			"#if(5 > 3 or 1/0)true#end",
			"true", assert.NoError},
		{"short circuit bool result in and",
			"#if(5 == 3 and 1/0)true#{else}false#end",
			"false", assert.NoError},
	}
	var b strings.Builder
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vtl := Must(Parse(tt.tmpl, "", ""))
			err := vtl.Execute(&b, nil)
			assert.Equal(t, tt.want, b.String())
			tt.assertion(t, err)
			b.Reset()
		})
	}
}

func BenchmarkExecuteGo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		goTmpl.Execute(ioutil.Discard, m)
	}
}

func BenchmarkExecuteVtl(b *testing.B) {
	for i := 0; i < b.N; i++ {
		vtlTmpl.Execute(ioutil.Discard, m)
	}
}

func TestExecuteFuzzCrashes(t *testing.T) {
	tests := []struct {
		name      string
		tmpl      string
		expect    string
		expectErr string
	}{
		{"set property of nil value",
			`#set($o.h={})`, "", "undefined var $o"},
		{"get property same as method with non-zero argument count",
			`#set($_foo="")#if($_foo.equals)#end`, "", "cannot get property equals of string value"},
		{"infinite recursion",
			`#macro(test)asd #test()#end#test()`, "asd asd asd asd asd asd asd asd asd asd asd asd asd asd asd asd asd asd asd asd ", "call depth exceeded"},
		{"iteration over string",
			`#set($y="")#foreach($a in$y)#end`, "", "cannot iterate over string"},
		{"property of string",
			`#set($_foo="")#if($_foo.t.o)#end`, "", "cannot get property t of string value"},
		{"range with float",
			`#foreach($i in[0..0.])#end`, "", ""},
		{"create slice from non-existent map property",
			`#set($woog={})#set($o=[$woog.r])$o`, "[null]", ""},
		{"macro called without arguments",
			`#macro(setthing$a)#end#setthing()`, "", "variable $a has not been set"},
		{"range with bool",
			`#macro(dirarg$a)#end#dirarg([0..!0])`, "", "NaN"},
		{"unexported field",
			`#set($arr=[])$arr.iterator().s`, "", "cannot get property s of iterator value"},
		{"cyclic reference",
			`#set($p={})#set($p.p=$p)$p.p`, "", "cycle detected"},
		{"include range",
			`#include([[[[0e..0]]]])`, "", "invalid include argument"},
		{"use $foreach as #foreach reference inside first foreach",
			`#foreach($foreach in [0])$foreach#end`, "0", ""},
		{"use $foreach as #foreach reference inside second foreach",
			`#foreach($m in [0])#foreach($foreach in[0])$foreach#end#end`, "0", ""},
		{"set property of the string",
			`#set($e="")#set($e.p="")`, "", "cannot set p on string value"},
		{"set non-existent property", `#set($p={})#set($p.p.x={})`, "", "cannot set x on nil value"},
		{"huge range",
			`#set($r=[2e30..0])$r.ToArray`, "", "start overflows int64"},
		{"huge negative range",
			`#set($r=[0..-2e30])$r.ToArray`, "", "end overflows int64"},
		{"toArray after Clear",
			`#set($r=[])$r.Clear()$r.ToArray()`, "[]", ""},
		{"unexported struct fields",
			`#foreach($i in [0])$foreach#end`, "{}", ""},
		{"$foreach var outside #foreach",
			`#foreach($i in [0])#end$foreach`, "", "undefined var $foreach"},
		{"set method",
			`#set($p={})#set($p.p=[])#set($p.p.Kind="")`, "", "cannot set Kind on slice value"},
		{"append string to byte slice",
			`#set($r='')$r.Bytes.Add($r)`, "", "cannot convert argument string to uint8"},
		{"remove in the beginning of iterator",
			`#set($array=[])#set($it=$array.iterator())#foreach($m in[0])$it.Remove()#end`, "", "next hasn't yet been called on iterator"},
		{"invalid conversion",
			`#foreach($m in[0])#if(1)#set($p={0:0})$p.EntrySet().RetainAll([[0..0]])#end#end`, "true", ""},
		{"try to set valid string property - has method IsEmpty",
			`#set($p='')#set($p.empty=true)`, "", "cannot set empty on string value"},
		{"set property of the slice",
			`#set($r=[])#if(1)#set($r.s='')$r.Add($r)#end`, "", ""},
		{"map with undefined var as key inside if",
			`#if({$p:0})0#end`, "", ""},
		{"map with undefined var as value inside if",
			`#foreach($m in[0])#if({'':$p})0#end#end`, "", ""},
		{"map equals with nil value", `#set($p={8:0})#foreach($i in[0])$p.equals({8:$p.p})#end`, "false", ""},
		{"map equals with nil key", `#set($p={8:0})#foreach($i in[0])$p.equals({$p.p:0})#end`, "false", ""},
		{"map with the name of previous var",
			`#set($e="")#set($p='')#set($p={0:00,'':$p})#foreach($i in$p)#foreach($i in$p)$p.KeySet().RetainAll([])#end#end`, "truefalse", ""},
		{"set on empty value",
			`#set($p={})#set($p={'':$p.r})#if(1)$p.Values().ToArray()#end`, "[null]", ""},
		{"interator next",
			`#foreach($m in[0])#if(1)#set($p={})$p.Values().Iterator().Next()#end#end`, "", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tmpl, err := Parse(test.tmpl, "", "")
			var b bytes.Buffer
			if assert.NoError(t, err) {
				err := tmpl.Execute(&b, nil)
				if test.expectErr == "" {
					assert.NoError(t, err)
				} else {
					assert.EqualError(t, unposErr(err), test.expectErr)
				}
				assert.Equal(t, test.expect, b.String())
			}
		})
	}
}

func TestExecuteSet_LHS(t *testing.T) {
	type S struct {
		Int      int
		Str      string
		Slice    []string
		Map      map[string]int
		SliceMap []map[string]string
		SliceAny []interface{}
	}
	tests := []struct {
		name      string
		tmpl      string
		context   _m
		expect    string
		expectErr string
	}{
		{"undefined variable",
			`#set($some = "text")$some`, nil, "text", ""},
		{"existing variable",
			`#set($some = "text")$some`, _m{"some": 123}, "text", ""},
		{"redefine existing #set variable",
			`#set($some = 123)#set($some = "text")$some`, nil, "text", ""},
		{"redefine existing #set variable inside other directive",
			`#set($some = "text")#if(true)#set($some = 123)$some#end$some`, nil, "123text", ""},
		{"property of a struct to the wrong type",
			`#set($s.int = "qwe")$s.int`, _m{"s": &S{Int: 42}}, "", "cannot set int (int) to string"},
		{"property of a struct",
			`#set($s.str = "some text")$s.str`, _m{"s": &S{Str: "orig"}}, "some text", ""},
		{"property of the existing map",
			`#set($s.map.a = 5)$s.map`, _m{"s": &S{Map: map[string]int{"a": 1, "b": 2}}}, "{a=5, b=2}", ""},
		{"property of the existing map to the wrong type",
			`#set($s.map.a = "b")$s.map`, _m{"s": &S{Map: map[string]int{"a": 1, "b": 2}}}, "", "cannot convert value string to int"},
		{"property of a new map",
			`#set($m = {})#set($m.num = 1)#set($m.str = "str")$m`, nil, "{num=1, str=str}", ""},
		{"index in new array - same type",
			`#set($arr = [1, 2, 3])#set($arr[0] = 0)$arr`, nil, "[0, 2, 3]", ""},
		{"index in new array - different type",
			`#set($arr = [1, 2, 3])#set($arr[0] = "0")$arr`, nil, "[0, 2, 3]", ""},
		{"index in new array - composite type",
			`#set($arr = [1, 2, 3])#set($arr[0] = {"0": 1})$arr`, nil, "[{0=1}, 2, 3]", ""},
		{"index in existing array",
			`#set($s.Slice[2] = "0")$s.Slice`, _m{"s": &S{Slice: []string{"a", "b", "c"}}}, "[a, b, 0]", ""},
		{"index in existing array to incompatible type",
			`#set($s.Slice[2] = {})$s.Slice`, _m{"s": &S{Slice: []string{"a", "b", "c"}}}, "", "cannot convert map to string"},
		{"index in new map - same type",
			`#set($map = {1:"a", 2:"b", 3:"c"})#set($map["2"] = "d")$map`, nil, "{1=a, 2=d, 3=c}", ""},
		{"index in new map - different type",
			`#set($map = {1:"a", 2:"b", 3:"c"})#set($map["1"] = 1)$map`, nil, "{1=1, 2=b, 3=c}", ""},
		{"index in new map - composite type",
			`#set($map = {1:"a", 2:"b", 3:"c"})#set($map["1"] = {"0": 1})$map`, nil, "{1={0=1}, 2=b, 3=c}", ""},
		{"index in existing map",
			`#set($s.Map["c"] = 5)$s.Map`, _m{"s": &S{Map: map[string]int{"a": 1, "b": 2, "c": 3}}}, "{a=1, b=2, c=5}", ""},
		{"index in existing map to incompatible type",
			`#set($s.Map["1"] = {})$s.Map`, _m{"s": &S{Map: map[string]int{"a": 1, "b": 2, "c": 3}}}, "", "cannot convert value map to int"},
		{"set struct property with chained access - map property, array",
			`#set($m.a[0].str = "new")$m`, _m{"m": map[string][]*S{"a": {&S{Slice: []string{"a", "b", "c"}}}}}, "{a=[{Int:0, Str:new, Slice:[a, b, c], Map:{}, SliceMap:[], SliceAny:[]}]}", ""},
		{"set struct property with chained access - map property, array",
			`#set($m.a[0].str = "new")$m`, _m{"m": map[string][]*S{"a": {&S{Slice: []string{"a", "b", "c"}}}}}, "{a=[{Int:0, Str:new, Slice:[a, b, c], Map:{}, SliceMap:[], SliceAny:[]}]}", ""},
		{"set struct property with chained access - map index, array",
			`#set($m["a"][0].str = "new")$m`, _m{"m": map[string][]*S{"a": {&S{Slice: []string{"a", "b", "c"}}}}}, "{a=[{Int:0, Str:new, Slice:[a, b, c], Map:{}, SliceMap:[], SliceAny:[]}]}", ""},
		{"set struct property with chained access - array, map property",
			`#set($a[0].a.str = "new")$a`, _m{"a": []map[string]*S{{"a": &S{Slice: []string{"a", "b", "c"}}}}}, "[{a={Int:0, Str:new, Slice:[a, b, c], Map:{}, SliceMap:[], SliceAny:[]}}]", ""},
		{"set struct property with chained access - array, map index",
			`#set($a[0]["a"].str = "new")$a`, _m{"a": []map[string]*S{{"a": &S{Slice: []string{"a", "b", "c"}}}}}, "[{a={Int:0, Str:new, Slice:[a, b, c], Map:{}, SliceMap:[], SliceAny:[]}}]", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tmpl, err := Parse(test.tmpl, "", "")
			var b bytes.Buffer
			if assert.NoError(t, err) {
				err := tmpl.Execute(&b, test.context)
				if test.expectErr == "" {
					assert.NoError(t, err)
				} else {
					assert.EqualError(t, unposErr(err), test.expectErr)
				}
				assert.Equal(t, test.expect, b.String())
			}
		})
	}
}

func TestExecuteSet_RHS(t *testing.T) {
	tests := []struct {
		name      string
		tmpl      string
		context   _m
		expect    string
		expectErr string
	}{
		{"undefined variable",
			`#set($some = $var)$some`, nil, "", "undefined var $var"},
		{"defined variable - new",
			`#set($some = 1)#set($var = $some)$var`, nil, "1", ""},
		{"defined variable - existing",
			`#set($var = $some)$var`, _m{"some": "text"}, "text", ""},
		{"literal string",
			`#set($var = 'literal$string')$var`, nil, "literal$string", ""},
		{"interpolated string",
			`#set($var = "interpolated $str")$var`, _m{"str": "string"}, "interpolated string", ""},
		{"number",
			`#set($var = 0.2e-2)$var`, _m{"str": "string"}, "0.002", ""},
		{"range",
			`#set($r = [3..1])$r`, nil, "[3, 2, 1]", ""},
		{"list",
			`#set($l = ["a", 1, {}, []])$l`, nil, "[a, 1, {}, []]", ""},
		{"map",
			`#set($m = {"a": 1, "b": [], "c": {}})$m`, nil, "{a=1, b=[], c={}}", ""},
		{"struct property",
			`#set($v = $s.prop)$v`, _m{"s": struct{ Prop string }{"val"}}, "val", ""},
		{"math - 1",
			`#set($v = 10 - 3 * 2)$v`, nil, "4", ""},
		{"math - 2",
			`#set($v = 2 * (10 - 5))$v`, nil, "10", ""},
		{"math - 3",
			`#set($v = 3/2)$v`, nil, "1", ""},
		{"math - 4",
			`#set($v = 5%3)$v`, nil, "2", ""},
		{"bool",
			`#set($v = ((5 > 2) || !6/0) && 3-2)$v`, nil, "true", ""},
		{"Java's array type method invocation",
			`#set($l = $arr.size())$l`, _m{"arr": []int{1, 2, 3}}, "3", ""},
		{"Java's map type method invocation on newly constructed map",
			`#set($m = {"a":1, "b": 2})#set($l=$m.replace("a", 3))$m$l`, _m{"arr": []int{1, 2, 3}}, "{a=3, b=2}1", ""},
		{"method/index/property on external map",
			`#set($k=$m.entrySet().toArray()[1].key)$k`, _m{"m": map[string]string{"first": "abc", "second": "def", "third": "ghi"}}, "second", ""},
		{"method/property/index on external slice of map",
			`#set($k=$s.Get(0).first[0])$k`, _m{"s": []map[string][]string{{"first": []string{"abc"}, "second": []string{"def"}, "third": []string{"ghi"}}}}, "abc", ""},
		{"index/method/property on external map",
			`#set($e=$m["first"].ToUpperCase().empty)$e`, _m{"m": map[string]string{"first": "abc", "second": "def", "third": "ghi"}}, "false", ""},
		{"index/property/method on external map",
			`#set($l=$m["second"].def.Size())$l`, _m{"m": map[string]map[string]interface{}{"first": {"abc": []int{1}}, "second": {"def": []int{2}}, "third": {"ghi": []int{3}}}}, "1", ""},
		{"property/index/method on external map",
			`#set($l=$m.second["def"].Size())$l`, _m{"m": map[string]map[string]interface{}{"first": {"abc": []int{1}}, "second": {"def": []int{2}}, "third": {"ghi": []int{3}}}}, "1", ""},
		{"property/method/index on external map",
			`#set($l=$m.second.Get("def")[0])$l`, _m{"m": map[string]map[string]interface{}{"first": {"abc": []int{1}}, "second": {"def": []int{2}}, "third": {"ghi": []int{3}}}}, "2", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tmpl, err := Parse(test.tmpl, "", "")
			var b bytes.Buffer
			if assert.NoError(t, err) {
				err := tmpl.Execute(&b, test.context)
				if test.expectErr == "" {
					assert.NoError(t, err)
				} else {
					assert.EqualError(t, unposErr(err), test.expectErr)
				}
				assert.Equal(t, test.expect, b.String())
			}
		})
	}
}

func TestExecuteIf_Condition(t *testing.T) {
	tests := []struct {
		name      string
		tmpl      string
		context   _m
		expect    string
		expectErr string
	}{
		// null result
		{"undefined variable",
			`#if($some)$some#end`, nil, "", ""},
		{"defined null variable",
			`#if($some)$some#end`, _m{"some": nil}, "", ""},
		{"evals to null",
			`#if($x.some)$some#end`, _m{"x": map[string]interface{}{"some": nil}}, "", ""},
		// literals
		{"empty interpolated string literal",
			`#if("")false#{else}not#end`, nil, "not", ""},
		{"non-empty interpolated string literal",
			`#if("0")true#{else}not#end`, nil, "true", ""},
		{"empty array literal",
			`#if([])false#{else}not#end`, nil, "not", ""},
		{"non-empty array literal",
			`#if([0])true#{else}not#end`, nil, "true", ""},
		{"empty map literal",
			`#if({})false#{else}not#end`, nil, "not", ""},
		{"non-empty map literal",
			`#if({0:0})true#{else}not#end`, nil, "true", ""},
		{"quoted string literal",
			`#if('')false#{else}not#end`, nil, "not", ""},
		{"non-empty quoted string literal",
			`#if('0')true#{else}not#end`, nil, "true", ""},
		{"bool false literal",
			`#if(false)false#{else}not#end`, nil, "not", ""},
		{"bool true literal",
			`#if(true)true#{else}not#end`, nil, "true", ""},
		{"zero int literal",
			`#if(0)false#{else}not#end`, nil, "not", ""},
		{"negative zero float literal",
			`#if(-0.0)false#{else}not#end`, nil, "not", ""},
		{"two int literal",
			`#if(2)true#{else}not#end`, nil, "true", ""},
		{"negative one float literal",
			`#if(-1.0)true#{else}not#end`, nil, "true", ""},
		// vars
		{"empty string var",
			`#if($some)false#{else}not#end`, _m{"some": ""}, "not", ""},
		{"non-empty string var",
			`#if($some)true#{else}not#end`, _m{"some": "0"}, "true", ""},
		{"empty array var",
			`#if($some)false#{else}not#end`, _m{"some": []int{}}, "not", ""},
		{"non-empty array var",
			`#if($some)true#{else}not#end`, _m{"some": []string{""}}, "true", ""},
		{"empty map var",
			`#if($some)false#{else}not#end`, _m{"some": map[string]int{}}, "not", ""},
		{"non-empty map var",
			`#if($some)true#{else}not#end`, _m{"some": map[string]string{"": ""}}, "true", ""},
		{"bool false var",
			`#if($some)false#{else}not#end`, _m{"some": false}, "not", ""},
		{"bool true var",
			`#if($some)true#{else}not#end`, _m{"some": true}, "true", ""},
		{"zero int var",
			`#if($some)false#{else}not#end`, _m{"some": 0}, "not", ""},
		{"negative zero float var",
			`#if($some)false#{else}not#end`, _m{"some": -0.0}, "not", ""},
		{"two int var",
			`#if($some)true#{else}not#end`, _m{"some": 2}, "true", ""},
		{"negative one float var",
			`#if($some)true#{else}not#end`, _m{"some": -1.0}, "true", ""},
		// comparison
		{"false comparison - gt",
			`#if($some.a.length() > 1)false#{else}not#end`, _m{"some": map[string]string{"a": "2"}}, "not", ""},
		{"false comparison - lt",
			`#if($some.a.length() < 1)false#{else}not#end`, _m{"some": map[string]string{"a": "2"}}, "not", ""},
		{"false comparison - gte",
			`#if($some.a.length() >= 2)false#{else}not#end`, _m{"some": map[string]string{"a": "2"}}, "not", ""},
		{"false comparison - lte",
			`#if($some.a.length() <= 0)false#{else}not#end`, _m{"some": map[string]string{"a": "2"}}, "not", ""},
		{"false comparison - eq",
			`#if($some.a.length() == 0)false#{else}not#end`, _m{"some": map[string]string{"a": "2"}}, "not", ""},
		{"false comparison - ne",
			`#if($some.a.length() != 1)false#{else}not#end`, _m{"some": map[string]string{"a": "2"}}, "not", ""},
		{"true comparison - gt",
			`#if($some.a.length() > 0)true#{else}not#end`, _m{"some": map[string]string{"a": "2"}}, "true", ""},
		{"true comparison - lt",
			`#if($some.a.length() < 2)true#{else}not#end`, _m{"some": map[string]string{"a": "2"}}, "true", ""},
		{"true comparison - gte",
			`#if($some.a.length() >= 1)true#{else}not#end`, _m{"some": map[string]string{"a": "2"}}, "true", ""},
		{"true comparison - lte",
			`#if($some.a.length() <= 1)true#{else}not#end`, _m{"some": map[string]string{"a": "2"}}, "true", ""},
		{"true comparison - eq",
			`#if($some.a.length() == 1)true#{else}not#end`, _m{"some": map[string]string{"a": "2"}}, "true", ""},
		{"true comparison - ne",
			`#if($some.a.length() != 0)true#{else}not#end`, _m{"some": map[string]string{"a": "2"}}, "true", ""},
		// logical
		{"true logical - and",
			`#if($some and true)true#{else}not#end`, _m{"some": true, "other": false}, "true", ""},
		{"true logical - or",
			`#if($other or $some)true#{else}not#end`, _m{"some": true, "other": false}, "true", ""},
		{"true logical - not",
			`#if(not $other)true#{else}not#end`, _m{"some": true, "other": false}, "true", ""},
		{"true logical - and not",
			`#if($some and not $other)true#{else}not#end`, _m{"some": true, "other": false}, "true", ""},
		{"true logical - or not",
			`#if($other or not 1<0)true#{else}not#end`, _m{"some": true, "other": false}, "true", ""},
		{"true logical - or and",
			`#if($other or $some and true)true#{else}not#end`, _m{"some": true, "other": false}, "true", ""},
		{"true logical - and or",
			`#if($some and $other or true)true#{else}not#end`, _m{"some": true, "other": false}, "true", ""},
		{"true logical - not and",
			`#if(not ($some and ($other or 1<0)))true#{else}not#end`, _m{"some": true, "other": false}, "true", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tmpl, err := Parse(test.tmpl, "", "")
			var b bytes.Buffer
			if assert.NoError(t, err) {
				err := tmpl.Execute(&b, test.context)
				if test.expectErr == "" {
					assert.NoError(t, err)
				} else {
					assert.EqualError(t, unposErr(err), test.expectErr)
				}
				assert.Equal(t, test.expect, b.String())
			}
		})
	}
}

func TestExecuteIf_Body(t *testing.T) {
	tests := []struct {
		name      string
		tmpl      string
		context   _m
		expect    string
		expectErr string
	}{
		{"just string",
			`#if(true)string#end`, nil, "string", ""},
		{"with elseif",
			`#if(false)#elseif("1")string#end`, nil, "string", ""},
		{"else",
			`#if(false)if#elseif(1<0)elseif#{else}string#end`, nil, "string", ""},
		{"nested ifs",
			`#if(false)if#elseif(1<0)elseif#{else}#if(1>0)inner#{end}outer#end`, nil, "innerouter", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tmpl, err := Parse(test.tmpl, "", "")
			var b bytes.Buffer
			if assert.NoError(t, err) {
				err := tmpl.Execute(&b, test.context)
				if test.expectErr == "" {
					assert.NoError(t, err)
				} else {
					assert.EqualError(t, unposErr(err), test.expectErr)
				}
				assert.Equal(t, test.expect, b.String())
			}
		})
	}
}

func TestExecuteForeach_Reference(t *testing.T) {
	tests := []struct {
		name      string
		tmpl      string
		context   _m
		expect    string
		expectErr string
	}{
		// null expression
		{"undefined reference",
			`#foreach($x in [1])x: $x#end`, nil, "x: 1", ""},
		{"existing variable",
			`#foreach($x in [1])x: $x#end, after: $x`, _m{"x": 42}, "x: 1, after: 42", ""},
		{"reserved variable foreach will be shadowed",
			`#foreach($foreach in [1])foreach: $foreach#end`, nil, "foreach: 1", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tmpl, err := Parse(test.tmpl, "", "")
			var b bytes.Buffer
			if assert.NoError(t, err) {
				err := tmpl.Execute(&b, test.context)
				if test.expectErr == "" {
					assert.NoError(t, err)
				} else {
					assert.EqualError(t, unposErr(err), test.expectErr)
				}
				assert.Equal(t, test.expect, b.String())
			}
		})
	}
}

func TestExecuteForeach_Argument(t *testing.T) {
	tests := []struct {
		name      string
		tmpl      string
		context   _m
		expect    string
		expectErr string
	}{
		// null expression
		{"undefined variable",
			`#foreach($x in $anything)x: $x#end`, nil, "", "undefined var $anything"},
		{"defined null variable",
			`#foreach($x in $some)x: $x#end`, _m{"some": nil}, "", ""},
		{"evals to null",
			`#foreach($x in $y.some)x: $x#end`, _m{"y": map[string]interface{}{"some": nil}}, "", ""},
		// literals
		{"range literal",
			`#foreach($x in [0..0])x: $x#end`, nil, "x: 0", ""},
		{"empty array literal",
			`#foreach($x in [])x: $x#end`, nil, "", ""},
		{"non-empty array literal",
			`#foreach($x in [0])x: $x#end`, nil, "x: 0", ""},
		{"empty map literal",
			`#foreach($x in {})x: $x#end`, nil, "", ""},
		{"non-empty map literal",
			`#foreach($x in {0:1})x: $x#end`, nil, "x: 1", ""},
		// non-iterable vars
		{"empty string var",
			`#foreach($x in $some)x: $x#end`, _m{"some": ""}, "", "cannot iterate over string"},
		{"non-empty string var",
			`#foreach($x in $some)x: $x#end`, _m{"some": "0"}, "", "cannot iterate over string"},
		{"bool false var",
			`#foreach($x in $some)x: $x#end`, _m{"some": false}, "", "cannot iterate over bool"},
		{"bool true var",
			`#foreach($x in $some)x: $x#end`, _m{"some": true}, "", "cannot iterate over bool"},
		{"zero int var",
			`#foreach($x in $some)x: $x#end`, _m{"some": 0}, "", "cannot iterate over int"},
		{"negative zero float var",
			`#foreach($x in $some)x: $x#end`, _m{"some": -0.0}, "", "cannot iterate over float64"},
		{"two int var",
			`#foreach($x in $some)x: $x#end`, _m{"some": 2}, "", "cannot iterate over int"},
		{"negative one float var",
			`#foreach($x in $some)x: $x#end`, _m{"some": -1.0}, "", "cannot iterate over float64"},
		// iterable vars
		{"empty array var",
			`#foreach($x in $some)x: $x#end`, _m{"some": []int{}}, "", ""},
		{"non-empty array var",
			`#foreach($x in $some)x: $x#end`, _m{"some": []string{""}}, "x: ", ""},
		{"empty map var",
			`#foreach($x in $some)x: $x#end`, _m{"some": map[string]int{}}, "", ""},
		{"non-empty map var",
			`#foreach($x in $some)x: $x#end`, _m{"some": map[string]string{"": ""}}, "x: ", ""},
		{"empty iterator",
			`#foreach($x in $some)x: $x#end`, _m{"some": (&Map{map[string]int{}}).EntrySet()}, "", ""},
		{"non-empty iterator",
			`#foreach($x in $some)x: $x#end`, _m{"some": (&Map{map[string]string{"": ""}}).EntrySet()}, "x: =", ""},
		// expressions
		{"non-empty iterator from array",
			`#foreach($x in $some.iterator())x: $x#end`, _m{"some": []string{""}}, "x: ", ""},
		{"non-empty values from map",
			`#foreach($x in $some.values())x: $x#end`, _m{"some": map[string]string{"": ""}}, "x: ", ""},
		{"entrySet from map",
			`#foreach($x in $some.entrySet())$x.key - $x.value#if($foreach.hasNext), #end#end`, _m{"some": map[string]int{"a": 1, "b": 2}}, "a - 1, b - 2", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tmpl, err := Parse(test.tmpl, "", "")
			var b bytes.Buffer
			if assert.NoError(t, err) {
				err := tmpl.Execute(&b, test.context)
				if test.expectErr == "" {
					assert.NoError(t, err)
				} else {
					assert.EqualError(t, unposErr(err), test.expectErr)
				}
				assert.Equal(t, test.expect, b.String())
			}
		})
	}
}

func TestExecuteForeach_Body(t *testing.T) {
	tests := []struct {
		name      string
		tmpl      string
		context   _m
		expect    string
		expectErr string
	}{
		{"foreach properties",
			`#foreach($x in [1..2])
x: $x
first: $foreach.first
last: $foreach.last
index: $foreach.index
count: $foreach.count
hasNext: $foreach.hasNext
#end`, nil, `x: 1
first: true
last: false
index: 0
count: 1
hasNext: true
x: 2
first: false
last: true
index: 1
count: 2
hasNext: false
`, ""},
		{"if, set and other foreach",
			`#foreach($x in [1..2])x: $x#if($foreach.hasNext), #end#set($l = [2, 3])#if($x > 1)!#foreach($y in $l)$y#end#end#end`, nil, "x: 1, x: 2!23", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tmpl, err := Parse(test.tmpl, "", "")
			var b bytes.Buffer
			if assert.NoError(t, err) {
				err := tmpl.Execute(&b, test.context)
				if test.expectErr == "" {
					assert.NoError(t, err)
				} else {
					assert.EqualError(t, unposErr(err), test.expectErr)
				}
				assert.Equal(t, test.expect, b.String())
			}
		})
	}
}

func TestExpressions(t *testing.T) {
	tests := []struct {
		tmpl   string
		expect string
	}{
		{`1+2`, "3"},
		{`1+2+3`, "6"},
		{`1-2`, "-1"},
		{`1-2-3`, "-4"},
		{`2*3`, "6"},
		{`2*3*4`, "24"},
		{`4/2`, "2"},
		{`3/2`, "1"},
		{`3/2.0`, "1.5"},
		{`6/(((3)))/2`, "1"},
		{`-6/3/2`, "-1"},
		{`3*(2.0/4)`, "1.5"},
		{`1+1+(1+(1+(1+(1+(1+(1+(1+(1+(1+(1+(1+(1+(1+1.0/15)/14)/13)/12)/11)/10)/9)/8)/7)/6)/5)/4)/3)/2`, "2.7182818284589945"},
		{`3-1.0/(4-2.0/(5-3.0/(6-4.0/(7-5.0/(8-6.0/(9-7.0/(10-8.0/(11-9.0/(12-10.0/(13-11.0/(14-12.0/(15-13.0/(16-14.0/(17-15.0))))))))))))))`, "2.7182818284589945"},
		{`-(2)`, "-2"},
		{` 3   -1  `, "2"},
		{` 3  -  2.0 `, "1.0"},
		{`15.0+ 2`, "17.0"},
		{`11 +2.0`, "13.0"},
		{`2 *4`, "8"},
		{`5.0005 + 0.0095`, "5.01"},
		{`1--1`, "2"},
		{`1---1`, "0"},
		{`2*-4`, "-8"},
		{`2./-4`, "-0.5"},
		{`(3+3/3)*(5-5)`, "0"},
		{`-0+1`, "1"},
		{`3%2`, "1"},
		{`-3%2`, "-1"},
		{`-3%-2`, "-1"},
		{`3%-2`, "1"},
		{`2+6-7%4*6/3`, "2"},
		{`true or true and false`, "true"},
		{`(true or true) and false`, "false"},
		{`true or (true and false)`, "true"},
		{`false and true or true`, "true"},
		{`(false and true) or true`, "true"},
		{`false and (true or true)`, "false"},
		{`3<4.0 && true`, "true"},
		{`1 && "string"`, "true"},
		{`0 && "string"`, "false"},
		{`"" and 1`, "false"},
		{`1 + "a"`, "1a"},
		{`"a" + 1`, "a1"},
		{`1.0 + "a"`, "1.0a"},
		{`"a" + 1.0`, "a1.0"},
	}
	for _, test := range tests {
		t.Run(test.tmpl, func(t *testing.T) {
			tmpl, err := Parse(`#set($x = `+test.tmpl+`)$x`, "", "")
			var b bytes.Buffer
			if assert.NoError(t, err) {
				err := tmpl.Execute(&b, nil)
				assert.NoError(t, err)
				assert.Equal(t, test.expect, b.String())
			}
		})
	}
}

func TestExpressionFailures(t *testing.T) {
	tests := []struct {
		tmpl      string
		expect    string
		expectErr string
	}{
		{`"" < 1`, "", "left side of comparison operation is not a number"},
		{`1 < ""`, "", "right side of comparison operation is not a number"},
	}
	for _, test := range tests {
		t.Run(test.tmpl, func(t *testing.T) {
			tmpl, err := Parse(`#set($x = `+test.tmpl+`)$x`, "", "")
			var b bytes.Buffer
			if assert.NoError(t, err) {
				err := tmpl.Execute(&b, nil)
				assert.EqualError(t, unposErr(err), test.expectErr)
				assert.Equal(t, test.expect, b.String())
			}
		})
	}
}

func unposErr(err error) error {
	if poser, ok := err.(posError); ok {
		return poser.error
	}
	return err
}
