package govtl

import (
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
)

const DefaultMaxCallDepth = 20
const DefaultMaxIterations = -1
const DefaultMaxArrayRenderSize = 1024 * 1024

type methodIdx struct {
	name string
	i    int
}

type Template struct {
	root, lib     string
	tree          []Node
	macros        map[string]*MacroNode
	typeCache     map[reflect.Type][]methodIdx
	cacheMutex    sync.Mutex
	maxCallDepth  int
	maxIterations int
	maxArraySize  int
	pos           Pos
}

func Must(t *Template, err error) *Template {
	if err != nil {
		panic(err)
	}
	return t
}

func ParseFile(f, root, lib string) (*Template, error) {
	data, err := ioutil.ReadFile(f)
	if err != nil {
		return nil, err
	}

	return Parse(string(data), root, lib)
}

func Parse(vtl, root, lib string) (*Template, error) {
	macros := make(map[string]*MacroNode)
	if lib != "" {
		libAST, err := ParseFile(filepath.Join(root, lib), root, "")
		if err != nil {
			return nil, err
		}
		libAST.Execute(ioutil.Discard, nil)
		macros = libAST.macros
	}
	l := new(Lexer)
	l.Init(vtl)
	for k := range macros {
		l.macros[k] = true
	}
	ret := yyParse(l)
	if ret != 0 {
		return nil, l.err
	}
	ast := l.result
	gobble(ast, false)
	return &Template{root, lib, ast, macros, make(map[reflect.Type][]methodIdx), sync.Mutex{}, DefaultMaxCallDepth, DefaultMaxIterations, DefaultMaxArrayRenderSize, Pos{}}, nil
}

func (t *Template) WithMaxCallDepth(n int) *Template {
	t.maxCallDepth = n
	return t
}

func (t *Template) WithMaxIterations(n int) *Template {
	t.maxIterations = n
	return t
}

func (t *Template) WithMaxArrayRenderSize(n int) *Template {
	t.maxArraySize = n
	return t
}

func gobble(ast []Node, nested bool) {
	// Text Directive Text
	// Directive Text
	// Directive Text Directive
	// Nested Text
	// Text Nested
	// Text Nested Text

	// if current is directive and (prev is text or none) and (next is text or none)
	//    if (prev is none) or (prev is text and ends with newlineAndSpaces)
	//       if (next is none) or (next is text and starts with spacesAndNewLine)
	//          gobble start
	//          gobble end
	//          if current is nested and inner end is text and ends with newLineAndSpaces
	//             gobble inner end
	//       if current is nested and inner start is text and starts with spacesAndNewLine
	//          gobble inner start

	// Or, maybe???
	// if cur is text
	//  cur starts with nl spaces
	//    [ cur dir ]
	//    [ cur dir nl spaces ]
	//    [ dir cur ]
	//    [ nl spaces dir cur ]
	//
	//    if prev is none and cur starts with newLineAndSpaces or is justWS and next is directive before the end of line or at the very end
	//       gobble start
	//    if prev is directive at the very beginning and cur starts with newLineAndSpaces
	//       gobble start
	//    if prev is directive at the beginning of line and cur starts with newLineAndSpaces
	//       gobble start
	//    if next is none and cur ends with spacesAndNewLine and prev is directive at the beginning of line or at the very beginning
	//       gobble start
	//    if next is directive at the end and cur starts with spacesAndNewLine
	//       gobble end
	//    if next is directive before the end of line and cur ends with spacesAndNewLine
	//       gobble end

	changes := make(map[int][]func(t TextNode) string)
	for i := range ast {
		switch cur := ast[i].(type) {
		case NestedNode:
			for _, nest := range cur.Nested() {
				if len(nest) == 0 {
					continue
				}
				if n, ok := nest[0].(TextNode); ok && startsWithSpacesAndNewline(n) {
					s := strings.TrimLeft(string(n), " \t")
					if len(s) > 0 && s[0] == '\n' {
						s = s[1:]
					}
					nest[0] = TextNode(s)
				}
				if n, ok := nest[len(nest)-1].(TextNode); ok && endsWithNewlineAndSpace(n) {
					nest[len(nest)-1] = TextNode(strings.TrimRight(string(n), " \t"))
				}
				gobble(nest, true)
			}
		}
		cur, ok := ast[i].(TextNode)
		if !ok {
			continue
		}

		switch {
		// if first node and there is a node after
		case justText(ast, i) && i < len(ast)-1 && directiveBeforeNewline(ast, i+1) && (i == 0 && justWS(cur) || endsWithNewlineAndSpace(cur)):
			changes[i] = append(changes[i], trimAfter)
		case i > 0 && i < len(ast)-1 && directiveAtNewline(ast, i-1) && endsWithNewlineAndSpace(cur) && directiveBeforeNewline(ast, i+1):
			changes[i] = append(changes[i], trimAfter)
		case i > 0 && i == len(ast)-1 && directiveBeforeNewline(ast, i-1) && !(nested && justWS(cur)):
			changes[i] = append(changes[i], trimAfter)
		}
		switch {
		case i > 0 && i < len(ast)-1 && directiveAtNewline(ast, i-1) && directiveBeforeNewline(ast, i-1) && startsWithSpacesAndNewline(cur):
			changes[i] = append(changes[i], trimBefore)
		case i > 0 && i == len(ast)-1 && directiveAtNewline(ast, i-1) && endsWithNewlineAndSpace(cur):
			changes[i] = append(changes[i], trimBefore)
		}
	}
	for i, v := range changes {
		n := ast[i].(TextNode)
		for _, f := range v {
			n = TextNode(f(n))
		}
		ast[i] = n
	}
}

func trimBefore(n TextNode) string {
	s := strings.TrimLeft(string(n), " \t")
	if len(s) > 0 && s[0] == '\n' {
		s = s[1:]
	}
	return s
}

func trimAfter(n TextNode) string {
	return strings.TrimRight(string(n), " \t")
}

func justText(ast []Node, i int) bool {
	for ; i >= 0; i-- {
		switch ast[i].(type) {
		case TextNode, *VarNode:
		default:
			return true
		}
	}
	return true
}

func directiveAtNewline(ast []Node, i int) bool {
	switch ast[i].(type) {
	case TextNode, *VarNode:
	default:
		if i == 0 {
			return true
		}
		t, ok := ast[i-1].(TextNode)
		if !ok {
			return false
		}
		if endsWithNewlineAndSpace(t) || (i == 1 && justWS(t)) || len(t) == 0 {
			return true
		}
	}
	return false
}

func directiveBeforeNewline(ast []Node, i int) bool {
	switch ast[i].(type) {
	case TextNode, *VarNode:
	default:
		if i == len(ast)-1 {
			return true
		}
		t, ok := ast[i+1].(TextNode)
		if !ok {
			return false
		}
		if startsWithSpacesAndNewline(t) || (i == len(ast)-2 && justWS(t)) {
			return true
		}
	}
	return false
}

func endsWithNewlineAndSpace(s TextNode) bool {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == ' ' || s[i] == '\t' {
			continue
		}
		if s[i] == '\n' {
			return true
		}
		break
	}
	return false
}

func startsWithSpacesAndNewline(s TextNode) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\t' {
			continue
		}
		if s[i] == '\n' {
			return true
		}
		break
	}
	return false
}

func justWS(s TextNode) bool {
	for _, r := range s {
		if r != ' ' && r != '\t' {
			return false
		}
	}
	return true
}
