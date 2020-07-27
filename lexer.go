package govtl

import (
	"bytes"
	"fmt"
	"strings"
)

type Token struct {
	token   int
	literal string
	line    int
}

type Lexer struct {
	data      []byte
	pos, prev int
	line, col int
	states    []lexState
	macros    map[string]bool
	result    []Node
	err       error
}

var directives = map[string]int{
	"set":      SET,
	"if":       IF,
	"elseif":   ELSEIF,
	"else":     ELSE,
	"end":      END,
	"foreach":  FOREACH,
	"include":  INCLUDE,
	"parse":    PARSE,
	"stop":     STOP,
	"break":    BREAK,
	"evaluate": EVALUATE,
	"define":   DEFINE,
	"macro":    MACRO,
}

func (l *Lexer) Init(s string) {
	l.data = []byte(s)
	l.line = 1
	l.macros = make(map[string]bool)
}

type lexState int

func (l lexState) String() string {
	switch l {
	case sText:
		return "text"
	case sDir:
		return "directive"
	case sRef:
		return "reference"
	case sString:
		return "string"
	case sExpr:
		return "expr"
	case sVar:
		return "var"
	case sFormal:
		return "formal"
	}
	return "unknown"
}

const (
	sText lexState = iota
	sDir
	sRef
	sString
	sExpr
	sVar
	sFormal

	EOF = -(iota + 1)
)

func (l *Lexer) Lex(lval *yySymType) int {
	if l.Peek(0) == EOF {
		return 0
	}
	l.prev = l.pos
	defer func() {
		if l.pos > len(l.data) {
			l.pos = len(l.data)
		}
		l.line += bytes.Count(l.data[l.prev:l.pos], []byte("\n"))
	}()
	switch l.state() {
	case sText:
		line := l.line
		text := l.ScanText("#$")
		var hadComment bool
		if l.Peek(0) == '#' && l.Peek(1) == '#' {
			l.ScanComment("\n")
			l.Skip(1)
			hadComment = true
		}
		if l.pos > l.prev && (len(text) > 0 || hadComment) {
			lval.t = Token{token: TEXT, literal: text, line: line}
			return TEXT
		}
		switch l.Peek(0) {
		case '$':
			p := l.Peek(1)
			switch {
			case p == '!' || p == '{' || p == '_' || (p >= 'A' && p <= 'Z') || (p >= 'a' && p <= 'z'):
				if p == '{' || (p == '!' && l.Peek(2) == '{') {
					pushState(l, sFormal)
				}
				pushState(l, sRef)
				return l.Lex(lval)
			default:
				// return self as text
				lval.t = Token{token: TEXT, literal: string(l.ScanByte()), line: l.line}
				return TEXT
			}
		case '#':
			p := l.Peek(1)
			switch p {
			// case '#':
			// 	// single line comment
			// 	l.ScanComment("\n")
			// 	if l.prev == 0 {
			// 		l.Skip(1)
			// 	}
			// 	return l.Lex(lval)
			case '*':
				// multiline comment
				l.ScanComment("*#")
				// eat ending *#
				l.Skip(2)
				return l.Lex(lval)
			case '{':
				l.Skip(2)
				d := l.ScanIdentifier()
				if directive, ok := directives[d]; ok && l.Peek(0) == '}' {
					l.Skip(1)
					lval.t = Token{token: directive, line: l.line}
					if directive != END && directive != ELSE {
						l.SkipWhitespace()
						pushState(l, sDir)
					}
					return directive
				}
				if l.macros[d] && l.Peek(0) == '}' {
					l.Skip(1)
					l.SkipWhitespace()
					pushState(l, sDir)
					lval.t = Token{token: MACROCALL, literal: d, line: l.line}
					return MACROCALL
				}
				lval.t = Token{token: TEXT, literal: "#{" + d, line: l.line}
				return TEXT
			case '[':
				if l.Peek(2) == '[' {
					// unparsed content until ]]#
				} else {
					// just text
				}
			}
			// directive
			l.Skip(1)
			d := l.ScanIdentifier()
			if directive, ok := directives[d]; ok {
				// fmt.Println("directive", d)
				lval.t = Token{token: directive, line: l.line}
				if directive != END && directive != ELSE {
					l.SkipWhitespace()
					pushState(l, sDir)
				}
				return directive
			}
			if l.macros[d] {
				l.SkipWhitespace()
				pushState(l, sDir)
				lval.t = Token{token: MACROCALL, literal: d, line: l.line}
				return MACROCALL
			}
			// or just text
			lval.t = Token{token: TEXT, literal: "#" + d, line: l.line}
			return TEXT
		}
		lval.t = Token{token: TEXT, literal: "", line: l.line}
		return TEXT
	case sDir:
		switch l.Peek(0) {
		case '(':
			pushState(l, sExpr)
		default:
			popState(l)
			return l.Lex(lval)
		}
	case sFormal:
		if l.Peek(0) != '}' {
			break
		}
		popState(l)
	case sRef:
		switch l.Peek(0) {
		case '$', '!', '{':
		default:
			if isIdent(l.Peek(0)) {
				pushState(l, sVar)
			} else {
				popState(l)
			}
			return l.Lex(lval)
		}
	case sExpr:
		l.SkipWhitespace()
		p := l.Peek(0)
		switch p {
		case '(', '[':
			pushState(l, sExpr)
			return int(l.ScanByte())
		case ')', ']':
			popState(l)
			return int(l.ScanByte())
		case '\'':
			// skip '
			l.Skip(1)
			s := l.ScanString('\'')
			// skip '
			l.Skip(1)
			lval.t = Token{token: STRING, literal: s, line: l.line}
			return STRING
		case '"':
			pushState(l, sString)
			return int(l.ScanByte())
		case '$':
			// l.Skip(1)
			pushState(l, sVar)
			// return l.Lex(lval)
		case '.', '=', '!', '<', '>', '|', '&':
			if op := l.ScanOp(); op != "" {
				lval.t = Token{token: ops[op], literal: altOps[op], line: l.line}
				return ops[op]
			}
			if l.state() == sRef && p == '.' && !isIdent(l.Peek(1)) {
				popState(l)
				lval.t = Token{token: TEXT, literal: l.ScanText("#$"), line: l.line}
				return TEXT
			}
			return int(l.ScanByte())
		default:
			ident := l.ScanIdentifier()
			switch strings.ToLower(ident) {
			case "ge", "le", "gt", "lt", "eq", "ne", "and", "or", "not":
				prev := l.Peek(-len(ident) - 1)
				if prev == ' ' || prev == '\t' || prev == '\n' {
					lval.t = Token{token: ops[ident], literal: ident, line: l.line}
					return ops[ident]
				}
			case "in":
				prev := l.Peek(-len(ident) - 1)
				if prev == ' ' || prev == '\t' || prev == '\n' {
					return IN
				}
			case "true", "false":
				prev := l.Peek(-len(ident) - 1)
				if prev != '.' && prev != '$' {
					lval.t = Token{token: BOOLEAN, literal: ident, line: l.line}
					return BOOLEAN
				}
			}
			if ident != "" {
				lval.t = Token{token: IDENTIFIER, literal: ident, line: l.line}
				return IDENTIFIER
			}
		}
		start := l.pos
		switch {
		case p == '-' && isNum(l.Peek(1)):
			p = l.ScanByte()
			fallthrough
		case isNum(p):
			tok := INT
			l.ScanInt()
			// if there is two dots after number it means it's range
			// if not - we have float
			if l.Peek(0) == '.' && l.Peek(1) != '.' {
				tok = FLOAT
				l.Skip(1)
				l.ScanInt()
			}
			if l.Peek(0) == 'e' {
				tok = FLOAT
				l.Skip(1)
				if l.Peek(0) == '+' || l.Peek(0) == '-' {
					l.Skip(1)
				}
				l.ScanInt()
			}
			lval.t = Token{token: tok, literal: string(l.data[start:l.pos]), line: l.line}
			return tok
		}
	case sVar:
		switch l.Peek(0) {
		case '[', '(':
			pushState(l, sExpr)
		case '.':
			if !isIdent(l.Peek(1)) {
				popState(l)
				return l.Lex(lval)
			}
		default:
			ident := l.ScanIdentifier()
			if ident != "" {
				tok := IDENTIFIER
				switch l.Peek(0) {
				case '(':
					tok = METHOD
					// case '[':
					// 	tok = INDEX
				}
				lval.t = Token{token: tok, literal: ident, line: l.line}
				return tok
			}
			popState(l)
			if l.state() == sRef {
				popState(l)
			}
			return l.Lex(lval)
		}
	case sString:
		// FIXME add support for interpolated values, not just text
		text := l.ScanText("$\"")
		if l.pos > l.prev {
			lval.t = Token{token: TEXT, literal: text, line: l.line}
			return TEXT
		}
		switch l.Peek(0) {
		case '"':
			popState(l)
			return int(l.ScanByte())
		case '$':
			p := l.Peek(1)
			switch {
			case p == '!' || p == '{' || p == '_' || (p >= 'A' && p <= 'Z') || (p >= 'a' && p <= 'z'):
				if p == '{' || (p == '!' && l.Peek(2) == '{') {
					pushState(l, sFormal)
				}
				pushState(l, sRef)
			default:
				lval.t = Token{token: TEXT, literal: string(l.ScanByte()), line: l.line}
				return TEXT
			}
		}
	}
	if l.Peek(0) == EOF {
		return 0
	}
	return int(l.ScanByte())
}

func (l *Lexer) Pos() string {
	return fmt.Sprint(l.pos)
}

func (l *Lexer) Error(s string) {
	start := l.prev - 20
	if start < 0 {
		start = 0
	}
	end := l.prev + 20
	if end > len(l.data) {
		end = len(l.data)
	}
	pos := l.prev
	if pos > len(l.data) {
		pos = len(l.data)
	}
	l.err = fmt.Errorf("%s: line %d (byte %d) (%s|%s)\n", s, l.line, l.prev, l.data[start:pos], l.data[pos:end])
}

func (l *Lexer) state() lexState {
	if len(l.states) == 0 {
		return 0
	}
	return l.states[len(l.states)-1]
}

func addMacro(l interface{}, name string) {
	lex := l.(*Lexer)
	lex.macros[name] = true
}

func eatWSend(t string) string {
	for s := len(t) - 1; s >= 0; s-- {
		if t[s] == ' ' || t[s] == '\t' {
			continue
		}
		if t[s] == '\n' {
			return t[:s+1]
		}
	}
	return t
}

// func gobbleWS(n []Node, text string) string {
// 	wsStart := -1
// 	if len(n) == 1 {
// 		wsStart = 0
// 	} else if len(n) > 1 {
// 		if t, ok := n[len(n)-2].(TextNode); ok {
// 			if len(t) == 0 {
// 				wsStart = 0
// 			}
// 			for s := len(t) - 1; s >= 0; s-- {
// 				if t[s] == ' ' || t[s] == '\t' {
// 					continue
// 				}
// 				if t[s] == '\n' {
// 					wsStart = s
// 				}
// 				break
// 			}
// 		}
// 	}
// 	wsEnd := -1
// 	for s := 0; s < len(text); s++ {
// 		if text[s] == ' ' || text[s] == '\t' {
// 			continue
// 		}
// 		if text[s] == '\n' {
// 			wsEnd = s
// 		}
// 		break
// 	}
// 	if wsStart > -1 && wsEnd > -1 {
// 		if len(n) > 1 && len(n[len(n)-2].(TextNode)) > 0 {
// 			n[len(n)-2] = n[len(n)-2].(TextNode)[:wsStart+1]
// 		}
// 		text = text[wsEnd+1:]
// 	}
// 	return text
// }

func pushState(l interface{}, s lexState) {
	lex := l.(*Lexer)
	lex.states = append(lex.states, s)
}

func popState(l interface{}) {
	lex := l.(*Lexer)
	if len(lex.states) > 0 {
		lex.states = lex.states[:len(lex.states)-1]
	}
}

func (l *Lexer) Peek(n int) rune {
	if l.pos >= 0 && l.pos < len(l.data)-n {
		return rune(l.data[l.pos+n])
	}
	return rune(EOF)
}

func (l *Lexer) Skip(n int) {
	l.pos += n
}

func (l *Lexer) ScanByte() rune {
	l.pos++
	return rune(l.data[l.pos-1])
}

func isNum(r rune) bool   { return r >= '0' && r <= '9' }
func isIdent(r rune) bool { return r == '_' || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') }

func (l *Lexer) ScanIdentifier() string {
	i := l.pos
	for ; i < len(l.data); i++ {
		if isIdent(rune(l.data[i])) || (i > l.pos && isNum(rune(l.data[i]))) {
			continue
		}
		break
	}
	ident := string(l.data[l.pos:i])
	l.pos = i
	return ident
}

func (l *Lexer) ScanInt() string {
	i := l.pos
	for ; i < len(l.data); i++ {
		if !isNum(rune(l.data[i])) {
			break
		}
	}
	num := string(l.data[l.pos:i])
	l.pos = i
	return num
}

func (l *Lexer) ScanText(delim string) string {
	w := l.data[l.pos:]
	idx := bytes.IndexAny(w, delim)
	if idx == -1 {
		l.pos += len(w)
		return string(w)
	}
	s := make([]byte, idx)
	copy(s, w[:idx])
	var c int
	for c < idx && s[idx-c-1] == '\\' {
		c++
	}
	// asdf\$ -> asdf$
	// asdf\\$ -> asdf\
	// asdf\\\$ -> asdf\$
	l.pos += idx + c%2
	if c%2 == 1 {
		s[idx-c/2-1] = w[idx]
	}
	return string(s[:idx-c/2])
}

func (l *Lexer) ScanComment(end string) {
	w := l.data[l.pos:]
	idx := bytes.Index(w, []byte(end))
	if idx == -1 {
		idx = len(w)
	}
	l.pos += idx
}

func (l *Lexer) ScanString(p rune) string {
	i := l.pos
	for ; i < len(l.data); i++ {
		if rune(l.data[i]) == p {
			break
		}
	}
	s := string(l.data[l.pos:i])
	l.pos = i
	return s
}

func (l *Lexer) GobbleWS() {
	i := l.pos
	for ; i < len(l.data); i++ {
		if l.data[i] != ' ' && l.data[i] != '\t' {
			if l.data[i] == '\n' {
				l.pos = i + 1
			}
			break
		}
	}
}

func (l *Lexer) SkipWhitespace() {
	i := l.pos
	for ; i < len(l.data); i++ {
		if l.data[i] != ' ' && l.data[i] != '\t' && l.data[i] != '\n' {
			break
		}
	}
	l.pos = i
}

var ops = map[string]int{
	"..":  RANGE,
	"==":  CMP,
	"eq":  CMP,
	"!=":  CMP,
	"ne":  CMP,
	"<=":  CMP,
	"le":  CMP,
	">=":  CMP,
	"ge":  CMP,
	"<":   CMP,
	"lt":  CMP,
	">":   CMP,
	"gt":  CMP,
	"or":  OR,
	"||":  OR,
	"and": AND,
	"&&":  AND,
	"not": NOT,
	"!":   NOT,
}

var altOps = map[string]string{
	"==": "eq",
	"!=": "ne",
	"<=": "le",
	">=": "ge",
	"<":  "lt",
	">":  "gt",
	"||": "or",
	"&&": "and",
}

func (l *Lexer) ScanOp() string {
	for _, i := range []int{2, 1} {
		if l.pos < len(l.data)-i+1 {
			s := string(l.data[l.pos : l.pos+i])
			if _, ok := ops[s]; ok {
				l.pos += i
				return s
			}
		}
	}
	return ""
}
