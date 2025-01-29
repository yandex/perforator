package querylang

type Parser interface {
	ParseSelector(query string) (*Selector, error)
}
