package govtl

import (
	"bytes"
	"html/template"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
		return assert.EqualError(t, err, "division by zero", msg)
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
			`#set($arr=[])$arr.iterator.s`, "", ""},
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
					assert.EqualError(t, err, test.expectErr)
				}
				assert.Equal(t, test.expect, b.String())
			}
		})
	}
}

func TestExecuteSet(t *testing.T) {
	type S struct {
		Int      int
		Str      string
		Slice    []string
		Map      map[string]interface{}
		SliceMap []map[string]interface{}
		SliceAny []interface{}
	}
	tests := []struct {
		name      string
		tmpl      string
		context   map[string]interface{}
		expect    string
		expectErr string
	}{
		{"undefined variable",
			`#set($some = "text")$some`, nil, "text", ""},
		{"existing variable",
			`#set($some = "text")$some`, map[string]interface{}{"some": 123}, "text", ""},
		{"redefine existing #set variable",
			`#set($some = 123)#set($some = "text")$some`, nil, "text", ""},
		{"redefine existing #set variable inside other directive",
			`#set($some = "text")#if(true)#set($some = 123)$some#end$some`, nil, "123text", ""},
		{"property of a struct to the wrong type",
			`#set($s.int = "qwe")$s.int`, map[string]interface{}{"s": &S{Int: 42}}, "", "cannot set int (int) to string"},
		{"property of a struct",
			`#set($s.str = "some text")$s.str`, map[string]interface{}{"s": &S{Str: "orig"}}, "some text", ""},
		{"index in new array",
			`#set($arr = [1, 2, 3])#set($arr[0] = 0)$arr`, nil, "[0, 2, 3]", ""},
		// FIXME redo types as simple interface values ???
		// {"index in existing array",
		// 	`#set($s.Slice[2] = 0)$s.Slice`, map[string]interface{}{"s": &S{Slice: []string{"a", "b", "c"}}}, "[a, b, 0]", ""},
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
					assert.EqualError(t, err, test.expectErr)
				}
				assert.Equal(t, test.expect, b.String())
			}
		})
	}
}
