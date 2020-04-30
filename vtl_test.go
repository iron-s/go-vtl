package govtl_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/sergi/go-diff/diffmatchpatch"

	govtl "github.com/iron-s/go-vtl"
)

var templates = []string{"arithmetic", "array", "block", "commas", "comment-eof", "comment", "curly-directive", "diabolical", "encodingtest", "encodingtest2", "encodingtest3", "encodingtest_KOI8-R", "equality", "escape2", "escape", "foreach-array", "foreach-introspect", "foreach-map", "foreach-method", "foreach-null-list", "foreach-type", "foreach-variable", "formal", "get", "if", "ifstatement", "include", "interpolation", "logical2", "logical", "loop", "map", "math", "method", "newline", "parse", "pedantic", "quotes", "range", "sample", "settest", "shorthand", "stop1", "stop2", "stop3", "string", "subclass", "test", "velocimacro2", "velocimacro", "vm_test1"}

func TestTemplates(t *testing.T) {
	dmp := diffmatchpatch.New()
	for _, f := range templates {
		t.Run(f, func(t *testing.T) {
			f = "templates/" + f + ".vm"
			data, err := ioutil.ReadFile(f)
			if err != nil {
				t.Fatal("Error reading", f, err)
			}
			tmpl, err := govtl.Parse(string(data), "templates", "VM_global_library.vm")
			if err != nil {
				t.Fatal("parsing", f, "error", err)
			}
			var b bytes.Buffer
			err = tmpl.Execute(&b, SetupTest())
			if err != nil && err.Error() != "stop" {
				t.Fatal("Error running", f, err)
			}
			base := strings.TrimSuffix(path.Base(f), ".vm")
			cmp := "templates/compare/" + base + ".cmp"
			expect, err := ioutil.ReadFile(cmp)
			if err != nil {
				t.Fatal("Error reading", cmp, err)
			}
			if string(expect) != b.String() {
				ioutil.WriteFile(f+".result", b.Bytes(), os.FileMode(0660))
				diffs := dmp.DiffMain(string(expect), b.String(), false)
				t.Error("not equal", DiffPrettyTextWS(diffs))
			}
		})
	}
}

func DiffPrettyTextWS(diffs []diffmatchpatch.Diff) string {
	var buff bytes.Buffer
	for _, diff := range diffs {
		text := diff.Text

		switch diff.Type {
		case diffmatchpatch.DiffInsert:
			_, _ = buff.WriteString("\x1b[32m\x1b[7m")
			_, _ = buff.WriteString(text)
			_, _ = buff.WriteString("\x1b[0m")
		case diffmatchpatch.DiffDelete:
			_, _ = buff.WriteString("\x1b[31m\x1b[7m")
			_, _ = buff.WriteString(text)
			_, _ = buff.WriteString("\x1b[0m")
		case diffmatchpatch.DiffEqual:
			_, _ = buff.WriteString(text)
		}
	}

	return buff.String()
}

func SetupTest() map[string]interface{} {
	p := &provider{"lunatic", false}
	h := map[string]string{"Bar": "this is from a hashtable!", "Foo": "this is from a hashtable too!"}
	v := &vec{"string1", "string2"}
	var n *nullToString

	return map[string]interface{}{
		"name":        "jason",
		"name2":       "jason",
		"name3":       "geoge",
		"provider":    p,
		"list":        p.GetCustomers(),
		"stringarray": p.GetArray(),
		"hashtable":   h,
		"hashmap":     map[string]string{},
		"vector":      v,
		"mystring":    "",
		"Floog":       "floogie woogie",
		"boolobj":     &boolean{},

		"int1":            1000,
		"long1":           10000000000,
		"float1":          1000.1234,
		"double1":         10000000000,
		"templatenumber1": num(999.125),
		"obarr":           []string{"a", "b", "c", "d"},
		"intarr":          []int{10, 20, 30, 40, 50},
		"map":             h,
		"collection":      v,
		"iterator":        govtl.NewIterator(v),
		"enumerator":      v,
		"nullList":        []interface{}{"a", "b", nil, "d"},
		"emptyarr":        []interface{}{},
		"nullToString":    n,
	}
}

type provider struct {
	Title string
	State bool
}

func (p *provider) Get(k string) string {
	return k
}

func (p *provider) GetName() string {
	return "jason"
}

func (p *provider) GetList() []string {
	return []string{"list element 1", "list element 2", "list element 3"}
}

func (p *provider) GetCustomers() []string {
	return []string{"ArrayList element 1", "ArrayList element 2", "ArrayList element 3", "ArrayList element 4"}
}

func (p *provider) GetHashtable() map[string]string {
	return map[string]string{"key0": "value0", "key1": "value1", "key2": "value2"}
}

func (p *provider) Concat(s ...interface{}) string {
	var ret, space string
	slice, ok := s[0].(*govtl.Slice)
	if len(s) == 1 && ok {
		s = slice.S
		// quirk: if we concat array - we need space, if it's just strings - no need for space
		space = " "
	}
	for _, v := range s {
		ret += fmt.Sprintf("%v", v) + space
	}
	return ret
}

func (p *provider) ObjConcat(s *govtl.Slice) string {
	var ret string
	for _, v := range s.S {
		ret += fmt.Sprintf("%v", v) + " "
	}
	return ret
}

func (p *provider) GetEmptyList() []interface{} {
	return []interface{}{}
}

func (p *provider) GetArray() []string {
	return []string{"first element", "second element"}
}

func (p *provider) TheAPLRules() bool {
	return true
}

type vec []string

func (v *vec) FirstElement() string {
	return (*v)[0]
}

func (p *provider) GetVector() *vec {
	return &vec{"vector element 1", "vector element 2"}
}

func (p *provider) String() string {
	return "test provider"
}

func (p *provider) ShowPerson(pp *person) string {
	return pp.name
}

func (p *provider) Person() *person {
	return &person{"Person"}
}

func (p *provider) Child() *person {
	return &person{"Child"}
}

func (p *provider) Chop(s string, i int) string {
	return s[:len(s)-i]
}

type person struct {
	name string
}

func (p *person) GetName() string {
	return p.name
}

type num float64

func (n num) AsNumber() float64 {
	return float64(n)
}

type boolean struct{}

func (*boolean) IsBoolean() bool { return true }

type nullToString struct{}

func (*nullToString) ToString() interface{} { return nil }
