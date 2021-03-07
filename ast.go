package govtl

type Node interface{}

type Pos struct {
	line int
}

type PositionedNode interface {
	Position() Pos
}

type NestedNode interface {
	Nested() [][]Node
}

type VarNode struct {
	*RefNode
	Items  []*AccessNode
	Silent bool
	Pos    Pos
}

func (n *VarNode) Position() Pos { return n.Pos }

type RefNode struct {
	Name string
}

type AccessKind int

const (
	AccessNone AccessKind = iota
	AccessProperty
	AccessIndex
	AccessMethod
)

type AccessNode struct {
	Name string
	Args []*OpNode
	Kind AccessKind
	Pos  Pos
}

func (n *AccessNode) Position() Pos { return n.Pos }

type OpNode struct {
	Op    string
	Val   interface{}
	Left  *OpNode
	Right *OpNode
	Pos   Pos
}

func (n *OpNode) Position() Pos { return n.Pos }

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
	Pos  Pos
}

func (n *SetNode) Position() Pos { return n.Pos }

type IfNode struct {
	Cond  *OpNode
	Items []Node
	Else  *IfNode
	Pos   Pos
}

func (n *IfNode) Position() Pos { return n.Pos }

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
	Pos   Pos
}

func (n *ForeachNode) Position() Pos { return n.Pos }

func (n *ForeachNode) Nested() [][]Node { return [][]Node{n.Items, n.Else} }

type MacroCall struct {
	Name string
	Vals []*OpNode
	Pos  Pos
}

func (n *MacroCall) Position() Pos { return n.Pos }

type MacroNode struct {
	Name   string
	Assign []*RefNode
	Items  []Node
	Pos    Pos
}

func (n *MacroNode) Position() Pos { return n.Pos }

func (n *MacroNode) Nested() [][]Node { return [][]Node{n.Items} }

type ParseNode struct {
	Name *OpNode
	Pos  Pos
}

type StopNode struct{}
type BreakNode struct{}

type IncludeNode struct {
	Names []*OpNode
	Pos   Pos
}

func (n *IncludeNode) Position() Pos { return n.Pos }

type EvalNode struct {
	Content string
	Pos     Pos
}

func (n *EvalNode) Position() Pos { return n.Pos }
