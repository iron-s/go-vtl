package govtl

import (
	"testing"
)

func TestLex(t *testing.T) {
	tests := []string{
		`schmoo #set($x = $list.size() * ( 3 + $list[0].length() )) and then some
#set($some = {"var": {"index": $list}})
$some.var["index"].add( $x )
$x. ${some}and${x.}`,
		`$x$x back to back`,
		`${x}$x formal then normal`,
		`$x${x} normal then formal`,
		`$!{x}$x silent formal, then normal`,
	}
	for _, test := range tests {
		l := &Lexer{}
		l.Init(test)
		var stuck int
		for {
			s := &yySymType{}
			ret := l.Lex(s)
			if ret == 0 {
				break
			}
			if l.pos == l.prev {
				stuck++
			}
			if stuck > 1 {
				t.Fatal("lexer stuck")
			}
		}
	}
}

// func TestScanText(t *testing.T) {
// 	tests := []struct{
// 		in, out string
// 		pos int
// 	}{
// 		{`
// #{if} (true)
//     this is a test
// #end
// `}
// 	}
// }
