package querylang

type Parser interface {
	ParseSelector(query string) (*Selector, error)
	ParseExpression(query string) (*Expression, error)
}
