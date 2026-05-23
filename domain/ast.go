package domain

// A user-script program is a Block of FuncCalls. Each FuncCall references
// the matched library FuncDef so the evaluator has direct access to its
// template, run snippet, args, and block info.
type Block struct {
	Stmts []FuncCall
}

type FuncCall struct {
	Func     *FuncDef
	Captures map[string]CaptureValue
	Body     *Block    // when Func.Block != nil
	Closer   *FuncCall // when Func.Block != nil
}

// CaptureValue is the bound value for a named capture in a matched FuncCall.
// Identifier/raw captures carry text; expression-typed captures carry an
// unevaluated Expr that the evaluator resolves at render time.
type CaptureValue struct {
	IsExpr bool
	Expr   Expr
	Text   string
}

// --- Expression AST (used by both outer template captures and inner DSL) ---

type Expr interface{ exprNode() }

type NumberLit struct {
	IsInt bool
	I     int64
	F     float64
}
type StringLit struct{ Value string }
type BoolLit struct{ Value bool }
type NullLit struct{}

type VarRef struct{ Path []string }

type CallExpr struct {
	Name []string
	Args []Expr
}

type CompareExpr struct {
	Op    string
	Left  Expr
	Right Expr
}

type NotExpr struct{ X Expr }

type ListLit struct{ Items []Expr }
type ObjLit struct {
	Keys []string
	Vals []Expr
}

func (NumberLit) exprNode()   {}
func (StringLit) exprNode()   {}
func (BoolLit) exprNode()     {}
func (NullLit) exprNode()     {}
func (VarRef) exprNode()      {}
func (CallExpr) exprNode()    {}
func (CompareExpr) exprNode() {}
func (NotExpr) exprNode()     {}
func (ListLit) exprNode()     {}
func (ObjLit) exprNode()      {}

// --- Inner DSL AST (the `run:` language; hardcoded in the engine) ---

type InnerBlock struct {
	Stmts []InnerStmt
}

type InnerStmt interface{ innerStmtNode() }

// Path is a dotted/indexed access chain rooted at a name.
//
//	context.imports          → Path{Root:"context", Steps:[Field("imports")]}
//	context.vars[name]       → Path{Root:"context", Steps:[Field("vars"), Index(VarRef{name})]}
type Path struct {
	Root  string
	Steps []PathStep
}
type PathStep struct {
	IsIndex bool
	Field   string
	Index   Expr
}

type SetStmt struct {
	Target Path
	Value  Expr
}
type AppendStmt struct {
	Target Path
	Value  Expr
}
type PrependStmt struct {
	Target Path
	Value  Expr
}
type MergeStmt struct {
	Target Path
	Value  Expr
}
type DeleteStmt struct {
	Target Path
}
type IfStmt struct {
	Cond Expr
	Body InnerBlock
}
type LoopStmt struct {
	Var  string
	Iter Expr
	Body InnerBlock
}
type CallStmt struct {
	Call CallExpr
}

func (SetStmt) innerStmtNode()     {}
func (AppendStmt) innerStmtNode()  {}
func (PrependStmt) innerStmtNode() {}
func (MergeStmt) innerStmtNode()   {}
func (DeleteStmt) innerStmtNode()  {}
func (IfStmt) innerStmtNode()      {}
func (LoopStmt) innerStmtNode()    {}
func (CallStmt) innerStmtNode()    {}
