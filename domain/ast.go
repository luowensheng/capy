package domain

// A user-script program is a Block of FuncCalls. Each FuncCall references
// the matched library FuncDef so the evaluator has direct access to its
// template, run snippet, args, and block info.
type Block struct {
	Stmts []FuncCall
	// IsVerbatim marks a body that wasn't parsed as nested statements
	// — instead the body's raw source bytes were captured into
	// VerbatimText. Used by functions declared `block_verbatim`. The
	// renderer surfaces VerbatimText via `${body}` exactly like a
	// parsed block body's rendered output.
	IsVerbatim   bool
	VerbatimText string
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
	Else *InnerBlock // optional; populated by `else` / `else if` arms
}
type LoopStmt struct {
	// Var is the value variable name. For two-var loops over maps,
	// this is the value side; for two-var loops over lists, this is
	// the element side.
	Var string
	// KeyVar is the optional key/index variable. Empty for one-var
	// loops. When set:
	//   - iterating a list: KeyVar is the integer index
	//   - iterating a map:  KeyVar is the key
	KeyVar string
	Iter   Expr
	Body   InnerBlock
}
type CallStmt struct {
	Call CallExpr
}

// WriteStmt appends the rendered value of Value to the current
// function's output buffer. Used by the unified `write` block design;
// the translator in orchestrator/features/make_library_loader.go
// processes these out before the engine sees them.
type WriteStmt struct {
	Value Expr
}

func (SetStmt) innerStmtNode()     {}
func (AppendStmt) innerStmtNode()  {}
func (PrependStmt) innerStmtNode() {}
func (MergeStmt) innerStmtNode()   {}
func (DeleteStmt) innerStmtNode()  {}
func (IfStmt) innerStmtNode()      {}
func (LoopStmt) innerStmtNode()    {}
func (CallStmt) innerStmtNode()    {}
func (WriteStmt) innerStmtNode()   {}
