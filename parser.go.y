%{

package govtl

import "strconv"
// import "fmt"

%}

%union {
    t Token
    n Node
    v []Node
}

%type   <v>             vtl directives
%type   <n>             directive reference method interpolated expression term setarg else elseifs range iterable map kvpairs array list primary identifiers identifier args arg literal
%type   <n>             bool_expr bool_and bool_not bool_term
%token  <t>             IDENTIFIER METHOD INDEX TEXT COMMENT STRING FLOAT INT BOOLEAN error
%token  <t>             SET IF ELSEIF ELSE FOREACH INCLUDE PARSE STOP BREAK EVALUATE DEFINE MACRO MACROCALL END
%token  <t>             IN RANGE WS
%left   <t>             OR
%left   <t>             AND NOT
%left   <t>             CMP
%left                   '+' '-'
%left                   '*' '/' '%'
%start vtl

%%

vtl:            directives { yylex.(*Lexer).result = $1 }
                ;

directives:     /*empty */
                { $$ = []Node{} }
        |       directives COMMENT
        |       directives directive
                { $$ = append($1, $2) }
        |       directives interpolated
                { $$ = append($1, $2) }
        |       directives TEXT
                {
                    if len($1) == 0 {
                        $$ = append($1, TextNode($2.literal))
                    } else if t, ok := $1[len($1) - 1].(TextNode); ok {
                        $1[len($1) - 1] = t + TextNode($2.literal)
                        $$ = $1
                    } else {
                        $$ = append($1, TextNode($2.literal))
                    }
                }
                ;

directive:      SET '(' '$' reference '=' setarg ')'
                { $$ = &SetNode{Var: $4.(*VarNode), Expr: $6.(*OpNode)} }
        |       IF '(' bool_expr ')' directives else END
                {
                    elseNode, _ := $6.(*IfNode)
                    $$ = &IfNode{Cond: $3.(*OpNode), Items: $5, Else: elseNode }
                }
        |       FOREACH '(' interpolated IN iterable ')' directives END
                { $$ = &ForeachNode{Var: $3.(*VarNode).RefNode, Iter: $5.(*OpNode), Items: $7} }
        |       FOREACH '(' interpolated IN iterable ')' directives ELSE directives END
                { $$ = &ForeachNode{Var: $3.(*VarNode).RefNode, Iter: $5.(*OpNode), Items: $7, Else: $9} }
        |       INCLUDE '(' args ')'
                { $$ = &IncludeNode{$3.([]*OpNode)} }
        |       PARSE '(' arg ')'
                { $$ = &ParseNode{$3.(*OpNode)} }
        |       EVALUATE '(' STRING ')'
                { $$ = &EvalNode{$3.literal} }
        |       MACRO '(' IDENTIFIER
                {addMacro(yylex, $3.literal) } ')' directives END
                { $$ = &MacroNode{$3.literal, nil, $6} }
        |       MACRO '(' IDENTIFIER identifiers
                {addMacro(yylex, $3.literal) } ')' directives END
                { $$ = &MacroNode{$3.literal, $4.([]*RefNode), $7} }
        |       MACROCALL '(' ')'
                { $$ = &MacroCall{ $1.literal, nil } }
        |       MACROCALL '(' args ')'
                { $$ = &MacroCall{ $1.literal, $3.([]*OpNode) } }
        |       STOP
                { $$ = &StopNode{} }
        |       BREAK
                { $$ = &BreakNode{} }
                ;

setarg:         bool_expr
        |       array
        |       map
                ;

iterable:       interpolated
                { $$ = &OpNode{Val: $1} }
        |       array
        |       map
                ;

else:           elseifs
        |       elseifs ELSE directives
                {
                    ifNode, _ := $1.(*IfNode)
                    if ifNode == nil {
                        $$ = &IfNode{Items: $3}
                    } else {
                        for ifNode.Else != nil {
                            ifNode = ifNode.Else
                        }
                        ifNode.Else = &IfNode{Items: $3}
                    }
                }
                ;

elseifs:        /* nothing */
                { $$ = nil }
        |       elseifs ELSEIF '(' bool_expr ')' directives
                {
                    elseifNode := &IfNode{Cond: $4.(*OpNode), Items: $6}
                    ifNode, _ := $1.(*IfNode)
                    if ifNode == nil {
                        $$ = elseifNode
                    } else {
                        ifNode.Else = elseifNode
                    }
                }
                ;

interpolated:   '$' reference
                { $$ = $2 }
        |       '$' '{' reference '}'
                { $$ = $3 }
        |       '$' '!' reference
                { $3.(*VarNode).Silent = true; $$ = $3 }
        |       '$' '!' '{' reference '}'
                { $4.(*VarNode).Silent = true; $$ = $4 }
                ;

method:         METHOD '(' ')'
                { $$ = &AccessNode{Name: $1.literal, IsCall: true} }
        |       METHOD '(' list ')'
                { $$ = &AccessNode{Name: $1.literal, IsCall: true, Args: $3.([]*OpNode)} }
                ;

reference:      IDENTIFIER
                { $$ = &VarNode{RefNode: &RefNode{Name: $1.literal}} }
        |       reference '.' IDENTIFIER
                {
                    v := $1.(*VarNode)
                    v.Items = append(v.Items, &AccessNode{Name: $3.literal})
                    $$ = $1
                }
        |       reference '[' bool_expr ']'
                {
                    v := $1.(*VarNode)
                    v.Items = append(v.Items, &AccessNode{Name: "get", IsCall: true, Args: []*OpNode{$3.(*OpNode)}})
                    $$ = $1
                }
        |       reference '.' method
                {
                    v := $1.(*VarNode)
                    v.Items = append(v.Items, $3.(*AccessNode))
                    $$ = $1
                }
                ;

array:          '[' ']'
                { $$ = &OpNode{Op: "list", Left: &OpNode{Val: []*OpNode{}}} }
        |       '[' list ']'
                { $$ = &OpNode{Op: "list", Left: &OpNode{Val: $2.([]*OpNode)}} }
        |       '[' range ']'
                { $$ = $2 }
                ;

range:          bool_expr RANGE bool_expr
                { $$ = &OpNode{Op: "range", Left: $1.(*OpNode), Right: $3.(*OpNode)} }
                ;

map:            '{' '}'
                { $$ = &OpNode{Op: "map", Left: &OpNode{Val: []*OpNode{}}} }
        |       '{' kvpairs '}'
                { $$ = &OpNode{Op: "map", Left: $2.(*OpNode) } }
                ;

kvpairs:        bool_expr ':' setarg
                { $$ = &OpNode{Val: []*OpNode{$1.(*OpNode), $3.(*OpNode)}} }
        |       kvpairs ',' bool_expr ':' setarg
                {
                    v := $1.(*OpNode).Val.([]*OpNode)
                    v = append(v, $3.(*OpNode), $5.(*OpNode))
                    $1.(*OpNode).Val = v
                    $$ = $1
                }
                ;

expression:     expression '+' expression
                { $$ = &OpNode{Op: "+", Left: $1.(*OpNode), Right: $3.(*OpNode)} }
        |       expression '-' expression
                { $$ = &OpNode{Op: "-", Left: $1.(*OpNode), Right: $3.(*OpNode)} }
        |       expression '*' expression
                { $$ = &OpNode{Op: "*", Left: $1.(*OpNode), Right: $3.(*OpNode)} }
        |       expression '/' expression
                { $$ = &OpNode{Op: "/", Left: $1.(*OpNode), Right: $3.(*OpNode)} }
        |       expression '%' expression
                { $$ = &OpNode{Op: "%", Left: $1.(*OpNode), Right: $3.(*OpNode)} }
        |       '-' expression
                { $$ = &OpNode{Op: "negate", Left: $2.(*OpNode)} }
        |       term
                ;

term:           interpolated
                { $$ = &OpNode{Val: $1} }
        |       primary
        |       '(' bool_expr ')'
                { $$ = $2 }
                ;

bool_expr:      bool_and
        |       bool_expr OR bool_and
                { $$ = &OpNode{Op: $2.literal, Left: $1.(*OpNode), Right: $3.(*OpNode)} }
                ;

bool_and:       bool_not
        |       bool_and AND bool_not
                { $$ = &OpNode{Op: $2.literal, Left: $1.(*OpNode), Right: $3.(*OpNode)} }
                ;

bool_not:       NOT bool_not
                { $$ = &OpNode{Op: "not", Left: $2.(*OpNode)} }
        |       '!' bool_not
                { $$ = &OpNode{Op: "not", Left: $2.(*OpNode)} }
        |       bool_term
                ;

bool_term:      expression
        |       expression CMP expression
                { $$ = &OpNode{Op: $2.literal, Left: $1.(*OpNode), Right: $3.(*OpNode)} }
        ;

primary:        STRING
                { $$ = &OpNode{Val: $1.literal} }
        |       '"' literal '"'
                { $$ = &OpNode{Val: $2} }
        |       FLOAT
                {
                    f, _ := strconv.ParseFloat($1.literal, 64)
                    $$ = &OpNode{Val: f}
                }
        |       INT
                {
                    i, _ := strconv.ParseInt($1.literal, 10, 64)
                    $$ = &OpNode{Val: i}
                }
        |       BOOLEAN
                {
                    var b bool
                    if $1.literal == "true" {
                        b = true
                    }
                    $$ = &OpNode{Val: b}
                }
                ;

literal:        // empty
                { $$ = &InterpolatedNode{} }
        |       literal TEXT
                {
                    v := $1.(*InterpolatedNode)
                    v.Items = append(v.Items, TextNode($2.literal))
                    $$ = v
                }
        |       literal interpolated
                {
                    v := $1.(*InterpolatedNode)
                    v.Items = append(v.Items, $2.(*VarNode))
                    $$ = v
                }
        |       literal WS
                {
                    v := $1.(*InterpolatedNode)
                    v.Items = append(v.Items, TextNode($2.literal))
                    $$ = v
                }
                ;

arg:            interpolated
                { $$ = &OpNode{Val: $1.(*VarNode)} }
        |       primary
        |       array
        |       map
                ;

args:           arg
                { $$ = []*OpNode{$1.(*OpNode)} }
        |       args ',' arg
                { $$ = append($1.([]*OpNode), $3.(*OpNode)) }
                ;

identifier:     '$' IDENTIFIER
                { $$ = &VarNode{RefNode: &RefNode{Name: $2.literal}} }
        |       '$' '{' IDENTIFIER '}'
                { $$ = &VarNode{RefNode: &RefNode{Name: $3.literal}} }
        |       '$' '!' IDENTIFIER
                { $$ = &VarNode{RefNode: &RefNode{Name: $3.literal}} }
        |       '$' '!' '{' IDENTIFIER '}'
                { $$ = &VarNode{RefNode: &RefNode{Name: $4.literal}} }
                ;

identifiers:    identifier
                { $$ = []*RefNode{$1.(*VarNode).RefNode} }
        |       identifiers ',' identifier
                { $$ = append($1.([]*RefNode), $3.(*VarNode).RefNode) }
                ;

list:           setarg
                { $$ = []*OpNode{$1.(*OpNode)} }
        |       list ',' setarg
                {
                    n := $1.([]*OpNode)
                    $$ = append(n, $3.(*OpNode))
                }
                ;

%% //
