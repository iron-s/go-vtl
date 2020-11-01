package govtl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Test struct {
	name     string
	template string
	expected []Node
}

func TestParse(t *testing.T) {
	tests := []Test{
		{"empty",
			"",
			[]Node{},
		},
		// FIXME spaces and comments
		// should be tested separately
		// as they could be eaten
		// {"space",
		// 	" ",
		// 	[]Node{TextNode(" ")},
		// },
		// {"spaces",
		// 	" \t\n",
		// 	[]Node{TextNode(" \t\n")},
		// },

		// {"simple comment",
		// 	" \t##comment\n",
		// 	[]Node{TextNode("")},
		// },
		// {"multiline single line comment",
		// 	" \t#*comment*#",
		// 	[]Node{TextNode(" \t")},
		// },
		// {"multiline comment",
		// 	" \t#*comment\n*#\n##\n",
		// 	[]Node{TextNode(" \t\n")},
		// },

		{"short var reference",
			"$var_1",
			[]Node{&VarNode{&RefNode{"var_1"}, nil, false, Pos{1}}},
		},
		{"short and formal reference",
			"$var${var}",
			[]Node{&VarNode{&RefNode{"var"}, nil, false, Pos{1}}, &VarNode{&RefNode{"var"}, nil, false, Pos{1}}},
		},
		{"formal var reference",
			"${var_1}",
			[]Node{&VarNode{&RefNode{"var_1"}, nil, false, Pos{1}}},
		},
		{"silent short var reference",
			"$!var",
			[]Node{&VarNode{&RefNode{"var"}, nil, true, Pos{1}}},
		},
		{"silent formal var reference",
			"$!{var}",
			[]Node{&VarNode{&RefNode{"var"}, nil, true, Pos{1}}},
		},

		{"regular property notation",
			"$customer1.Address",
			[]Node{&VarNode{
				&RefNode{"customer1"},
				[]*AccessNode{{"Address", nil, AccessProperty, Pos{1}}},
				false, Pos{1},
			}},
		},
		{"formal property notation",
			"${customer1.Address}",
			[]Node{&VarNode{
				&RefNode{"customer1"},
				[]*AccessNode{{"Address", nil, AccessProperty, Pos{1}}},
				false, Pos{1},
			}},
		},

		{"regular method notation",
			"$customer1.getAddress()",
			[]Node{&VarNode{
				&RefNode{"customer1"},
				[]*AccessNode{{"getAddress", nil, AccessMethod, Pos{1}}},
				false, Pos{1},
			}},
		},
		{"formal method notation",
			"${customer1.getAddress()}",
			[]Node{&VarNode{
				&RefNode{"customer1"},
				[]*AccessNode{{"getAddress", nil, AccessMethod, Pos{1}}},
				false, Pos{1},
			}},
		},
		{"formal method notation with params",
			`${customer1.setAddress("Somewhere")}`,
			[]Node{&VarNode{
				&RefNode{"customer1"},
				[]*AccessNode{{"setAddress", []*OpNode{{Val: &InterpolatedNode{Items: []Node{TextNode("Somewhere")}}}}, AccessMethod, Pos{1}}},
				false, Pos{1},
			}},
		},
		{"regular method notation with expression in params",
			`${customer1.setAddress("Somewhere" + 1)}`,
			[]Node{&VarNode{
				&RefNode{"customer1"},
				[]*AccessNode{{"setAddress", []*OpNode{{Op: "+", Left: &OpNode{Val: &InterpolatedNode{Items: []Node{TextNode("Somewhere")}}, Pos: Pos{0}}, Right: &OpNode{Val: int64(1), Pos: Pos{1}}, Pos: Pos{0}}}, AccessMethod, Pos{1}}},
				false, Pos{1},
			}},
		},

		// {"method call on method call",
		// 	"$customer1.getAddress()()",
		// 	[]Node{&VarNode{
		// 		&RefNode{"customer1"},
		// 		[]*AccessNode{{"getAddress", nil, true}},
		// 		false,
		// 	}},
		// },

		{"set dirctive with var reference",
			`#set( $monkey = $bill )`,
			[]Node{&SetNode{
				&VarNode{&RefNode{"monkey"}, nil, false, Pos{1}},
				&OpNode{Val: &VarNode{&RefNode{"bill"}, nil, false, Pos{1}}}, Pos{1},
			}},
		},
		{"set directive with string literal",
			`#set( $monkey.Friend = 'monica' )`,
			[]Node{&SetNode{
				&VarNode{&RefNode{"monkey"}, []*AccessNode{{"Friend", nil, AccessProperty, Pos{1}}}, false, Pos{1}},
				&OpNode{Val: "monica", Pos: Pos{1}}, Pos{1},
			}},
		},
		{"set directive with number literal",
			`#set( $monkey.Number = 123 )`,
			[]Node{&SetNode{
				&VarNode{&RefNode{"monkey"}, []*AccessNode{{"Number", nil, AccessProperty, Pos{1}}}, false, Pos{1}},
				&OpNode{Val: int64(123), Pos: Pos{1}}, Pos{1},
			}},
		},
		{"set directive with property reference",
			`#set( $monkey.Blame = $whitehouse.Leak )`,
			[]Node{&SetNode{
				&VarNode{&RefNode{"monkey"}, []*AccessNode{{"Blame", nil, AccessProperty, Pos{1}}}, false, Pos{1}},
				&OpNode{Val: &VarNode{&RefNode{"whitehouse"}, []*AccessNode{{"Leak", nil, AccessProperty, Pos{1}}}, false, Pos{1}}}, Pos{1},
			}},
		},
		{"set directive with method reference",
			`#set( $monkey.Plan = $spindoctor.weave($web) )`,
			[]Node{&SetNode{
				&VarNode{&RefNode{"monkey"}, []*AccessNode{{"Plan", nil, AccessProperty, Pos{1}}}, false, Pos{1}},
				&OpNode{Val: &VarNode{
					&RefNode{"spindoctor"},
					[]*AccessNode{{"weave", []*OpNode{{Val: &VarNode{&RefNode{"web"}, nil, false, Pos{1}}}}, AccessMethod, Pos{1}}}, false, Pos{1}}}, Pos{1},
			}},
		},
		{"set directive with range operator",
			`#set( $monkey.Numbers = [1..3] )`,
			[]Node{&SetNode{
				&VarNode{&RefNode{"monkey"}, []*AccessNode{{"Numbers", nil, AccessProperty, Pos{1}}}, false, Pos{1}},
				&OpNode{Op: "range", Left: &OpNode{Val: int64(1), Pos: Pos{1}}, Right: &OpNode{Val: int64(3), Pos: Pos{1}}, Pos: Pos{1}}, Pos{1},
			}},
		},
		{"set directive with object list",
			`#set( $monkey.Say = ["Not", $my, "fault"] )`,
			[]Node{&SetNode{
				&VarNode{&RefNode{"monkey"}, []*AccessNode{{"Say", nil, AccessProperty, Pos{1}}}, false, Pos{1}},
				&OpNode{Op: "list", Left: &OpNode{Val: []*OpNode{
					{Val: &InterpolatedNode{Items: []Node{TextNode("Not")}}},
					{Val: &VarNode{&RefNode{"my"}, nil, false, Pos{1}}},
					{Val: &InterpolatedNode{Items: []Node{TextNode("fault")}}}}},
				}, Pos{1}}},
		},
		{"set directive with object map",
			`#set( $monkey.Map = {"banana" : "good", "roast beef" : "bad"})`,
			[]Node{&SetNode{
				&VarNode{&RefNode{"monkey"}, []*AccessNode{{"Map", nil, AccessProperty, Pos{1}}}, false, Pos{1}},
				&OpNode{Op: "map", Left: &OpNode{Val: []*OpNode{
					{Val: &InterpolatedNode{Items: []Node{TextNode("banana")}}},
					{Val: &InterpolatedNode{Items: []Node{TextNode("good")}}},
					{Val: &InterpolatedNode{Items: []Node{TextNode("roast beef")}}},
					{Val: &InterpolatedNode{Items: []Node{TextNode("bad")}}}}},
				}, Pos{1}}},
		},
		{"set directive with arithmetic RHS",
			`#set( $value = $foo + 1 )`,
			[]Node{&SetNode{
				&VarNode{&RefNode{"value"}, nil, false, Pos{1}},
				&OpNode{Op: "+", Left: &OpNode{Val: &VarNode{&RefNode{"foo"}, nil, false, Pos{1}}}, Right: &OpNode{Val: int64(1), Pos: Pos{1}}}, Pos{1},
			}},
		},
		{"set directive with complex arithmetic RHS",
			`#set( $value = $foo * (3 + 1) )`,
			[]Node{&SetNode{
				&VarNode{&RefNode{"value"}, nil, false, Pos{1}},
				&OpNode{Op: "*", Left: &OpNode{Val: &VarNode{&RefNode{"foo"}, nil, false, Pos{1}}}, Right: &OpNode{Op: "+", Left: &OpNode{Val: int64(3), Pos: Pos{1}}, Right: &OpNode{Val: int64(1), Pos: Pos{1}}}}, Pos{1},
			}},
		},

		{"condition simple",
			`#if( !$foo )42#end`,
			[]Node{&IfNode{
				&OpNode{Op: "not", Left: &OpNode{Val: &VarNode{&RefNode{"foo"}, nil, false, Pos{1}}}, Pos: Pos{1}},
				[]Node{TextNode("42")},
				nil, Pos{1},
			}},
		},

		{"condition with else",
			`#if( $foo == 42 )42#{else}not!#end`,
			[]Node{&IfNode{
				&OpNode{Op: "eq", Left: &OpNode{Val: &VarNode{&RefNode{"foo"}, nil, false, Pos{1}}}, Right: &OpNode{Val: int64(42), Pos: Pos{1}}, Pos: Pos{1}},
				[]Node{TextNode("42")},
				&IfNode{nil, []Node{TextNode("not!")}, nil, Pos{1}}, Pos{1},
			}},
		},

		{"condition with elseif",
			`#{if}( $foo == 42 )42#{elseif}($foo > 3)\$foo > 3#{else}#{end}`,
			[]Node{&IfNode{
				&OpNode{Op: "eq", Left: &OpNode{Val: &VarNode{&RefNode{"foo"}, nil, false, Pos{1}}}, Right: &OpNode{Val: int64(42), Pos: Pos{1}}, Pos: Pos{1}},
				[]Node{TextNode("42")},
				&IfNode{
					&OpNode{Op: "gt", Left: &OpNode{Val: &VarNode{&RefNode{"foo"}, nil, false, Pos{1}}}, Right: &OpNode{Val: int64(3), Pos: Pos{1}}, Pos: Pos{1}},
					[]Node{TextNode(`$foo > 3`)},
					&IfNode{nil, []Node{}, nil, Pos{1}}, Pos{1}}, Pos{1},
			}},
		},
	}

	perm(t, tests, 1, 3, make([]bool, len(tests)), make([]Test, len(tests)), func(t *testing.T, sub []Test) {
		var (
			name, template string
			expected       = []Node{}
		)

		for i := range sub {
			if i > 0 {
				name += ","
			}
			name += sub[i].name
			template += sub[i].template
			l := len(expected)
			if l > 0 && len(sub[i].expected) > 0 {
				e1, ok1 := expected[l-1].(TextNode)
				e2, ok2 := sub[i].expected[0].(TextNode)
				if ok1 && ok2 {
					expected[l-1] = e1 + e2
					expected = append(expected, sub[i].expected[1:]...)
					continue
				}
			}
			expected = append(expected, sub[i].expected...)
		}
		t.Run(name, func(t *testing.T) {
			tmpl, err := Parse(template, "", "")
			require.NoError(t, err)
			assert.EqualValues(t, expected, tmpl.tree, "template AST")
		})
	})
}

func TestParseConditions(t *testing.T) {
	tests := []struct {
		operator string
		expected string
		err      string
	}{
		{"==", "eq", ""},
		{"eq", "eq", ""},
		{"!=", "ne", ""},
		{"ne", "ne", ""},
		{">=", "ge", ""},
		{"ge", "ge", ""},
		{"<=", "le", ""},
		{"le", "le", ""},
		{">", "gt", ""},
		{"gt", "gt", ""},
		{"<", "lt", ""},
		{"lt", "lt", ""},
		{"lte", "", "error parsing"},
	}
	for _, test := range tests {
		t.Run(test.operator, func(t *testing.T) {
			tmpl := `#if( $foo ` + test.operator + ` 42 )42#end`
			template, err := Parse(tmpl, "", "")
			if test.err != "" {
				if assert.Error(t, err) {
					assert.Regexp(t, "^unexpected IDENTIFIER,", err.Error())
				}
			} else if assert.NoError(t, err) {
				expected := []Node{&IfNode{
					&OpNode{Op: test.expected, Left: &OpNode{Val: &VarNode{&RefNode{"foo"}, nil, false, Pos{1}}}, Right: &OpNode{Val: int64(42), Pos: Pos{1}}, Pos: Pos{1}},
					[]Node{TextNode("42")},
					nil, Pos{1},
				}}

				assert.EqualValues(t, expected, template.tree, "template AST")
			}
		})
	}

}

func perm(t *testing.T, tests []Test, n, limit int, used []bool, rec []Test, f func(*testing.T, []Test)) {
	if n > limit {
		return
	}
	for i := 0; i < len(tests); i++ {
		if used[i] {
			continue
		}
		rec[n-1] = tests[i]
		f(t, rec[:n])
		used[i] = true
		perm(t, tests, n+1, limit, used, rec, f)
		used[i] = false
	}
}

func TestGobble(t *testing.T) {
	tests := []struct {
		name     string
		ast      []Node
		expected []Node
	}{
		{"spaces directive",
			[]Node{TextNode("  "), &SetNode{}},
			[]Node{TextNode(""), &SetNode{}},
		},
		{"directive spaces",
			[]Node{&SetNode{}, TextNode("   ")},
			[]Node{&SetNode{}, TextNode("")},
		},
		{"spaces directive spaces",
			[]Node{TextNode("  "), &SetNode{}, TextNode("   ")},
			[]Node{TextNode(""), &SetNode{}, TextNode("")},
		},
		{"nl spaces directive spaces",
			[]Node{TextNode("\n  "), &SetNode{}, TextNode("   ")},
			[]Node{TextNode("\n"), &SetNode{}, TextNode("")},
		},
		{"spaces directive spaces nl",
			[]Node{TextNode("  "), &SetNode{}, TextNode("   \n")},
			[]Node{TextNode(""), &SetNode{}, TextNode("")},
		},
		{"nl spaces directive spaces nl",
			[]Node{TextNode("\n  "), &SetNode{}, TextNode("   \n")},
			[]Node{TextNode("\n"), &SetNode{}, TextNode("")},
		},
		{"nl spaces directive spaces nl directive",
			[]Node{TextNode("\n  "), &SetNode{}, TextNode("   \n"), &SetNode{}},
			[]Node{TextNode("\n"), &SetNode{}, TextNode(""), &SetNode{}},
		},
		{"nl spaces directive spaces nl directive spaces",
			[]Node{TextNode("\n  "), &SetNode{}, TextNode("   \n"), &SetNode{}, TextNode("   ")},
			[]Node{TextNode("\n"), &SetNode{}, TextNode(""), &SetNode{}, TextNode("")},
		},
		{"nl spaces directive spaces nl directive spaces nl",
			[]Node{TextNode("\n  "), &SetNode{}, TextNode("   \n"), &SetNode{}, TextNode("   \n")},
			[]Node{TextNode("\n"), &SetNode{}, TextNode(""), &SetNode{}, TextNode("")},
		},

		{"text spaces directive",
			[]Node{TextNode("asd  "), &SetNode{}},
			[]Node{TextNode("asd  "), &SetNode{}},
		},
		{"spaces text directive",
			[]Node{TextNode("  asd"), &SetNode{}},
			[]Node{TextNode("  asd"), &SetNode{}},
		},
		{"directive text spaces",
			[]Node{&SetNode{}, TextNode("asd  ")},
			[]Node{&SetNode{}, TextNode("asd  ")},
		},
		{"directive spaces text",
			[]Node{&SetNode{}, TextNode("  asd")},
			[]Node{&SetNode{}, TextNode("  asd")},
		},
		{"text spaces directive spaces text",
			[]Node{TextNode("asd  "), &SetNode{}, TextNode("  asd")},
			[]Node{TextNode("asd  "), &SetNode{}, TextNode("  asd")},
		},
		{"spaces text directive spaces text",
			[]Node{TextNode("  asd"), &SetNode{}, TextNode("  asd")},
			[]Node{TextNode("  asd"), &SetNode{}, TextNode("  asd")},
		},
		{"text spaces directive text spaces",
			[]Node{TextNode("asd  "), &SetNode{}, TextNode("  asd")},
			[]Node{TextNode("asd  "), &SetNode{}, TextNode("  asd")},
		},
		{"spaces text directive text spaces",
			[]Node{TextNode("  asd"), &SetNode{}, TextNode("  asd")},
			[]Node{TextNode("  asd"), &SetNode{}, TextNode("  asd")},
		},

		{"nl spaces nested spaces nl",
			[]Node{TextNode("\n  "), &IfNode{Items: []Node{TextNode("  \n  asd\n  ")}, Else: &IfNode{Items: []Node{TextNode("  \n  asd\n  ")}}}, TextNode("   \n")},
			[]Node{TextNode("\n"), &IfNode{Items: []Node{TextNode("  asd\n")}, Else: &IfNode{Items: []Node{TextNode("  asd\n")}}}, TextNode("")},
		},

		{"nested nl spaces nested",
			[]Node{&IfNode{Items: []Node{TextNode("  \n  asd\n  ")}, Else: &IfNode{Items: []Node{TextNode("  \n  asd\n  ")}}}, TextNode("\n  "), &IfNode{Items: []Node{TextNode("  \n  asd\n  ")}, Else: &IfNode{Items: []Node{TextNode("  \n  asd\n  ")}}}},
			[]Node{&IfNode{Items: []Node{TextNode("  asd\n")}, Else: &IfNode{Items: []Node{TextNode("  asd\n")}}}, TextNode(""), &IfNode{Items: []Node{TextNode("  asd\n")}, Else: &IfNode{Items: []Node{TextNode("  asd\n")}}}},
		},
		{"nl spaces nested nl spaces nested",
			[]Node{TextNode("\n   "), &IfNode{Items: []Node{TextNode("  \n  asd\n  ")}, Else: &IfNode{Items: []Node{TextNode("  \n  asd\n  ")}}}, TextNode("\n  "), &IfNode{Items: []Node{TextNode("  \n  asd\n  ")}, Else: &IfNode{Items: []Node{TextNode("  \n  asd\n  ")}}}},
			[]Node{TextNode("\n"), &IfNode{Items: []Node{TextNode("  asd\n")}, Else: &IfNode{Items: []Node{TextNode("  asd\n")}}}, TextNode(""), &IfNode{Items: []Node{TextNode("  asd\n")}, Else: &IfNode{Items: []Node{TextNode("  asd\n")}}}},
		},
	}
	for _, test := range tests {
		gobble(test.ast, false)
		assert.EqualValues(t, test.expected, test.ast, test.name)
	}
}
