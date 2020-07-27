package govtl

import (
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
