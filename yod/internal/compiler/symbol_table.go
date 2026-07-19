package compiler

type SymbolScope string

const (
	GlobalScope SymbolScope = "GLOBAL"
	LocalScope  SymbolScope = "LOCAL"
	FreeScope   SymbolScope = "FREE"
)

type Symbol struct {
	Name  string
	Scope SymbolScope
	Index int
}

type SymbolTable struct {
	Outer       *SymbolTable
	store       map[string]Symbol
	numDefs     int
	FreeSymbols []Symbol
}

func NewSymbolTable() *SymbolTable {
	return &SymbolTable{store: map[string]Symbol{}}
}

func NewEnclosedSymbolTable(outer *SymbolTable) *SymbolTable {
	s := NewSymbolTable()
	s.Outer = outer
	return s
}

func (s *SymbolTable) Define(name string) Symbol {
	scope := GlobalScope
	if s.Outer != nil {
		scope = LocalScope
	}
	symbol := Symbol{Name: name, Index: s.numDefs, Scope: scope}
	s.store[name] = symbol
	s.numDefs++
	return symbol
}

func (s *SymbolTable) defineFree(original Symbol) Symbol {
	s.FreeSymbols = append(s.FreeSymbols, original)
	symbol := Symbol{
		Name:  original.Name,
		Index: len(s.FreeSymbols) - 1,
		Scope: FreeScope,
	}
	s.store[original.Name] = symbol
	return symbol
}

func (s *SymbolTable) Resolve(name string) (Symbol, bool) {
	obj, ok := s.store[name]
	if ok {
		return obj, true
	}
	if s.Outer == nil {
		return Symbol{}, false
	}
	obj, ok = s.Outer.Resolve(name)
	if !ok {
		return obj, false
	}
	if obj.Scope == GlobalScope {
		return obj, true
	}
	return s.defineFree(obj), true
}

// BuiltinIndex — אינדקס לפונקציות מובנות במכונה
func BuiltinIndex(name string) (int, bool) {
	switch name {
	case "הדפס":
		return 0, true
	case "אורך":
		return 1, true
	case "למספר":
		return 2, true
	case "למחרוזת":
		return 3, true
	case "סוג":
		return 4, true
	case "קלט":
		return 5, true
	case "הוסף":
		return 6, true
	case "טווח":
		return 7, true
	case "אקראי":
		return 8, true
	case "אקראי_בין":
		return 9, true
	case "משימה":
		return 10, true
	case "המתן":
		return 11, true
	case "במקביל":
		return 12, true
	default:
		return 0, false
	}
}
