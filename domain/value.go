package domain

type ValueKind int

const (
	ValNull ValueKind = iota
	ValString
	ValInt
	ValFloat
	ValBool
	ValList
	ValObject
)

type Value struct {
	Kind ValueKind
	Str  string
	Int  int64
	Flt  float64
	Bool bool
	List []Value
	Obj  map[string]Value
}

func Null() Value           { return Value{Kind: ValNull} }
func Str(s string) Value    { return Value{Kind: ValString, Str: s} }
func IntV(i int64) Value    { return Value{Kind: ValInt, Int: i} }
func FloatV(f float64) Value { return Value{Kind: ValFloat, Flt: f} }
func BoolV(b bool) Value    { return Value{Kind: ValBool, Bool: b} }
func ListV(vs []Value) Value { return Value{Kind: ValList, List: vs} }
func ObjV(m map[string]Value) Value { return Value{Kind: ValObject, Obj: m} }
