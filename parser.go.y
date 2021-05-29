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
                { $$ = &SetNode{Var: $4.(*VarNode), Expr: $6.(*OpNode), Pos: Pos{$1.line}} }
        |       IF '(' setarg ')' directives else END
                {
                    elseNode, _ := $6.(*IfNode)
                    $$ = &IfNode{Cond: $3.(*OpNode), Items: $5, Else: elseNode, Pos: Pos{$1.line} }
                }
        |       FOREACH '(' interpolated IN iterable ')' directives END
                { $$ = &ForeachNode{Var: $3.(*VarNode).RefNode, Iter: $5.(*OpNode), Items: $7, Pos: Pos{$1.line}} }
        |       FOREACH '(' interpolated IN iterable ')' directives ELSE directives END
                { $$ = &ForeachNode{Var: $3.(*VarNode).RefNode, Iter: $5.(*OpNode), Items: $7, Else: $9, Pos: Pos{$1.line}} }
        |       INCLUDE '(' args ')'
                { $$ = &IncludeNode{Names: $3.([]*OpNode), Pos: Pos{$1.line}} }
        |       PARSE '(' arg ')'
                { $$ = &ParseNode{Name: $3.(*OpNode), Pos: Pos{$1.line}} }
        |       EVALUATE '(' STRING ')'
                { $$ = &EvalNode{Content: $3.literal, Pos: Pos{$1.line}} }
        |       MACRO '(' IDENTIFIER
                {addMacro(yylex, $3.literal) } ')' directives END
                { $$ = &MacroNode{Name: $3.literal, Assign: nil, Items: $6, Pos: Pos{$1.line}} }
        |       MACRO '(' IDENTIFIER identifiers
                {addMacro(yylex, $3.literal) } ')' directives END
                { $$ = &MacroNode{Name: $3.literal, Assign: $4.([]*RefNode), Items: $7, Pos: Pos{$1.line}} }
        |       MACROCALL '(' ')'
                { $$ = &MacroCall{ Name: $1.literal, Vals: nil, Pos: Pos{$1.line} } }
        |       MACROCALL '(' args ')'
                { $$ = &MacroCall{ Name: $1.literal, Vals: $3.([]*OpNode), Pos: Pos{$1.line} } }
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
                        $$ = &IfNode{Items: $3, Pos: Pos{$2.line}}
                    } else {
                        for ifNode.Else != nil {
                            ifNode = ifNode.Else
                        }
                        ifNode.Else = &IfNode{Items: $3, Pos: Pos{$2.line}}
                    }
                }
                ;

elseifs:        /* nothing */
                { $$ = nil }
        |       elseifs ELSEIF '(' bool_expr ')' directives
                {
                    elseifNode := &IfNode{Cond: $4.(*OpNode), Items: $6, Pos: Pos{$2.line}}
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
                { $$ = &AccessNode{Name: $1.literal, Kind: AccessMethod, Pos: Pos{$1.line}} }
        |       METHOD '(' list ')'
                { $$ = &AccessNode{Name: $1.literal, Kind: AccessMethod, Args: $3.([]*OpNode), Pos: Pos{$1.line}} }
                ;

reference:      IDENTIFIER
                { $$ = &VarNode{RefNode: &RefNode{Name: $1.literal}, Pos: Pos{$1.line}} }
        |       reference '.' IDENTIFIER
                {
                    v := $1.(*VarNode)
                    v.Items = append(v.Items, &AccessNode{Name: $3.literal, Kind: AccessProperty, Pos: Pos{$3.line}})
                    $$ = $1
                }
        |       reference '[' bool_expr ']'
                {
                    v := $1.(*VarNode)
                    v.Items = append(v.Items, &AccessNode{Kind: AccessIndex, Args: []*OpNode{$3.(*OpNode)}})
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
                { $$ = &OpNode{Op: "range", Left: $1.(*OpNode), Right: $3.(*OpNode), Pos: Pos{$2.line}} }
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
                { $$ = &OpNode{Op: $2.literal, Left: $1.(*OpNode), Right: $3.(*OpNode), Pos: Pos{$2.line}} }
                ;

bool_and:       bool_not
        |       bool_and AND bool_not
                { $$ = &OpNode{Op: $2.literal, Left: $1.(*OpNode), Right: $3.(*OpNode), Pos: Pos{$2.line}} }
                ;

bool_not:       NOT bool_not
                { $$ = &OpNode{Op: "not", Left: $2.(*OpNode), Pos: Pos{$1.line}} }
        |       bool_term
                ;

bool_term:      expression
        |       expression CMP expression
                { $$ = &OpNode{Op: $2.literal, Left: $1.(*OpNode), Right: $3.(*OpNode), Pos: Pos{$2.line}} }
        ;

primary:        STRING
                { $$ = &OpNode{Val: $1.literal, Pos: Pos{$1.line}} }
        |       '"' literal '"'
                { $$ = &OpNode{Val: $2} }
        |       FLOAT
                {
                    f, err := strconv.ParseFloat($1.literal, 64)
                    if err != nil {
                        yylex.(*Lexer).Error(err.Error())
                        return yyError
                    }
                    $$ = &OpNode{Val: f, Pos: Pos{$1.line}}
                }
        |       INT
                {
                    i, err := strconv.ParseInt($1.literal, 10, 64)
                    if err != nil {
                        yylex.(*Lexer).Error(err.Error())
                        return yyError
                    }
                    $$ = &OpNode{Val: i, Pos: Pos{$1.line}}
                }
        |       BOOLEAN
                {
                    var b bool
                    if $1.literal == "true" {
                        b = true
                    }
                    $$ = &OpNode{Val: b, Pos: Pos{$1.line}}
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
                { $$ = &VarNode{RefNode: &RefNode{Name: $2.literal}, Pos: Pos{$2.line}} }
        |       '$' '{' IDENTIFIER '}'
                { $$ = &VarNode{RefNode: &RefNode{Name: $3.literal}, Pos: Pos{$3.line}} }
        |       '$' '!' IDENTIFIER
                { $$ = &VarNode{RefNode: &RefNode{Name: $3.literal}, Pos: Pos{$3.line}} }
        |       '$' '!' '{' IDENTIFIER '}'
                { $$ = &VarNode{RefNode: &RefNode{Name: $4.literal}, Pos: Pos{$4.line}} }
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
