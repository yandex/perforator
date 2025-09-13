package parserv2

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
)

type errorListener struct {
	antlr.DefaultErrorListener

	parseErrors
}

type parseErrors struct {
	syntaxErrors   []error
	semanticErrors []error
}

func (l *errorListener) hasErrors() bool {
	return len(l.syntaxErrors)+len(l.semanticErrors) > 0
}

func (l *errorListener) getError() error {
	return NewParseError(l.syntaxErrors, l.semanticErrors)
}

func (l *errorListener) onSyntaxError(err error) {
	l.syntaxErrors = append(l.syntaxErrors, err)
}

func (l *errorListener) onSemanticError(err error) {
	l.semanticErrors = append(l.semanticErrors, err)
}

func (l *errorListener) SyntaxError(_ antlr.Recognizer, _ interface{}, line, column int, msg string, _ antlr.RecognitionException) {
	l.onSyntaxError(fmt.Errorf("%s (at line %d, column %d)", msg, line, column))
}

func (l *errorListener) VisitErrorNode(node antlr.ErrorNode) {
	if l.hasErrors() {
		return
	}
	if parent, ok := node.GetParent().GetPayload().(antlr.ParseTree); ok {
		l.onSyntaxError(fmt.Errorf(
			"syntax error at '%s', in token '%s'",
			parent.GetText(),
			node.GetText(),
		))
	} else {
		l.onSyntaxError(fmt.Errorf(
			"syntax error at '%s'",
			node.GetText(),
		))
	}
}
