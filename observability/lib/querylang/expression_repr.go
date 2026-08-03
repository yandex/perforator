package querylang

import (
	"strings"
)

func (f *FunctionCall) Repr() string {
	args := make([]string, len(f.Arguments))
	for i, arg := range f.Arguments {
		args[i] = arg.Repr()
	}
	return string(f.Identifier) + "(" + strings.Join(args, ", ") + ")"
}

func (l *Lambda) Repr() string {
	args := make([]string, len(l.Arguments))
	for i, arg := range l.Arguments {
		args[i] = arg.Repr()
	}
	return "(" + strings.Join(args, ", ") + ")" + " -> " + l.Expression.Repr()
}

func (i Identifier) Repr() string {
	return string(i)
}

func (e *Expression) Repr() string {
	switch {
	case e.FunctionCall != nil:
		return e.FunctionCall.Repr()
	case e.Lambda != nil:
		return e.Lambda.Repr()
	case e.Selector != nil:
		return "{" + e.Selector.Repr() + "}"
	case e.Identifier != "":
		return e.Identifier.Repr()
	case e.Value != nil:
		return e.Value.Repr()
	case e.Vector != nil:
		items := make([]string, len(e.Vector))
		for i, item := range e.Vector {
			items[i] = item.Repr()
		}
		return "[" + strings.Join(items, ", ") + "]"
	}
	return "invalid_expression"
}

func (e *Expression) Kind() string {
	switch {
	case e.FunctionCall != nil:
		return "function_call"
	case e.Lambda != nil:
		return "lambda"
	case e.Selector != nil:
		return "selector"
	case e.Identifier != "":
		return "identifier"
	case e.Value != nil:
		return "scalar"
	case e.Vector != nil:
		return "vector"
	}
	return "invalid_expression"
}
