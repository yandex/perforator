package querylang

// Expression may represent only a subset of possible expressions of Solomon language.
type Expression struct {
	// one of:
	FunctionCall *FunctionCall
	Lambda       *Lambda
	Selector     *Selector
	Identifier   Identifier
	Value        Value
}

type FunctionCall struct {
	Identifier Identifier
	Arguments  []*Expression
}

type Lambda struct {
	Arguments  []Identifier
	Expression *Expression
}

type Identifier string
