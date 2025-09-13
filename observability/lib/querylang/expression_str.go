package querylang

import (
	"fmt"
	"strings"
)

func (f *FunctionCall) ToString() (string, error) {
	var err error

	args, err := arrayToString(f.Arguments)
	if err != nil {
		return "", err
	}

	identifier, err := f.Identifier.ToString()
	if err != nil {
		return "", err
	}

	return identifier + "(" + strings.Join(args, ", ") + ")", nil
}

func (l *Lambda) ToString() (string, error) {
	var err error

	args, err := arrayToString(l.Arguments)
	if err != nil {
		return "", err
	}

	expression, err := l.Expression.ToString()
	if err != nil {
		return "", err
	}

	if len(args) == 0 {
		return "", fmt.Errorf("lambda must have at least one argument")
	} else if len(args) == 1 {
		return args[0] + " -> " + expression, nil
	} else {
		return "(" + strings.Join(args, ", ") + ")" + " -> " + expression, nil
	}
}

func (i Identifier) ToString() (string, error) {
	if strings.ContainsRune(string(i), '"') || strings.ContainsRune(string(i), '\'') {
		return "", fmt.Errorf("invalid identifier: %s", string(i))
	}
	return string(i), nil
}

func (e *Expression) ToString() (string, error) {
	switch {
	case e.FunctionCall != nil:
		return e.FunctionCall.ToString()
	case e.Lambda != nil:
		return e.Lambda.ToString()
	case e.Selector != nil:
		return e.Selector.ToString()
	case e.Identifier != "":
		return e.Identifier.ToString()
	case e.Value != nil:
		return e.Value.ToString()
	}
	return "", fmt.Errorf("empty expression")
}
