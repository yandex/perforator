package parserexp

import (
	"errors"
	"fmt"
)

type ParseErrorType byte

const (
	Syntax ParseErrorType = iota
	Semantic
)

type ParseError struct {
	msg     string
	errType ParseErrorType
}

func (p *ParseError) Error() string {
	switch p.errType {
	case Syntax:
		return fmt.Sprintf("syntax errors: %s", p.msg)
	case Semantic:
		return fmt.Sprintf("semantic errors: %s", p.msg)
	default:
		return fmt.Sprintf("parse errors: %s", p.msg)
	}
}

func NewParseError(syntaxErrors []error, semanticErrors []error) error {
	if len(syntaxErrors) > 0 {
		return NewSyntaxParseError(syntaxErrors)
	}
	if len(semanticErrors) > 0 {
		return NewSemanticParseError(semanticErrors)
	}
	return nil
}

func NewSyntaxParseError(syntaxErrors []error) *ParseError {
	return &ParseError{
		msg:     fmt.Sprintf("%s", errors.Join(syntaxErrors...)),
		errType: Syntax,
	}
}

func NewSemanticParseError(semanticErrors []error) *ParseError {
	return &ParseError{
		msg:     fmt.Sprintf("%s", errors.Join(semanticErrors...)),
		errType: Semantic,
	}
}
