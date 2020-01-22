package govtl

type Node interface {
	// Orig() string
}

// type ListNode struct {
// 	Nodes []Node
// }

// func (l *ListNode) Orig() string {
// 	var ret string
// 	for _, n := range l.Nodes {
// 		ret += n.Orig()
// 	}
// 	return ret
// }

type NestedNode interface {
	Nested() [][]Node
}

type VarNode struct {
	*RefNode
	Items  []*AccessNode
	Silent bool
}

type RefNode struct {
	Name string
}

type AccessNode struct {
	Name   string
	Args   []*OpNode
	IsCall bool
}

type OpNode struct {
	Op    string
	Val   interface{}
	Left  *OpNode
	Right *OpNode
}

type TextNode string

func (t TextNode) Orig() string {
	return string(t)
}

type InterpolatedNode struct {
	Items []Node
}

type SetNode struct {
	Var  *VarNode
	Expr *OpNode
}

type IfNode struct {
	Cond  *OpNode
	Items []Node
	Else  *IfNode
}

func (n *IfNode) Nested() [][]Node {
	var nested [][]Node
	for ; n != nil; n = n.Else {
		nested = append(nested, n.Items)
	}
	return nested
}

type ForeachNode struct {
	Var   *RefNode
	Iter  *OpNode
	Items []Node
	Else  []Node
}

func (n *ForeachNode) Nested() [][]Node { return [][]Node{n.Items, n.Else} }

type MacroCall struct {
	Name string
	Vals []*OpNode
}

type MacroNode struct {
	Name   string
	Assign []*RefNode
	Items  []Node
}

func (n *MacroNode) Nested() [][]Node { return [][]Node{n.Items} }

type ParseNode struct {
	Name *OpNode
}

type StopNode struct{}
type BreakNode struct{}

type IncludeNode struct {
	Names []*OpNode
}

type EvalNode struct {
	Content string
}
