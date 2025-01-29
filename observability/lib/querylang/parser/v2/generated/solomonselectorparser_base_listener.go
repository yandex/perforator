// Code generated from solomon/libs/java/solomon-grammar/SolomonSelectorParser.g4 by ANTLR 4.13.1. DO NOT EDIT.

package parser // SolomonSelectorParser

import "github.com/antlr4-go/antlr/v4"

// BaseSolomonSelectorParserListener is a complete listener for a parse tree produced by SolomonSelectorParser.
type BaseSolomonSelectorParserListener struct{}

var _ SolomonSelectorParserListener = &BaseSolomonSelectorParserListener{}

// VisitTerminal is called when a terminal node is visited.
func (s *BaseSolomonSelectorParserListener) VisitTerminal(node antlr.TerminalNode) {}

// VisitErrorNode is called when an error node is visited.
func (s *BaseSolomonSelectorParserListener) VisitErrorNode(node antlr.ErrorNode) {}

// EnterEveryRule is called when any rule is entered.
func (s *BaseSolomonSelectorParserListener) EnterEveryRule(ctx antlr.ParserRuleContext) {}

// ExitEveryRule is called when any rule is exited.
func (s *BaseSolomonSelectorParserListener) ExitEveryRule(ctx antlr.ParserRuleContext) {}

// EnterSelectors is called when production selectors is entered.
func (s *BaseSolomonSelectorParserListener) EnterSelectors(ctx *SelectorsContext) {}

// ExitSelectors is called when production selectors is exited.
func (s *BaseSolomonSelectorParserListener) ExitSelectors(ctx *SelectorsContext) {}

// EnterSelectorList is called when production selectorList is entered.
func (s *BaseSolomonSelectorParserListener) EnterSelectorList(ctx *SelectorListContext) {}

// ExitSelectorList is called when production selectorList is exited.
func (s *BaseSolomonSelectorParserListener) ExitSelectorList(ctx *SelectorListContext) {}

// EnterSelector is called when production selector is entered.
func (s *BaseSolomonSelectorParserListener) EnterSelector(ctx *SelectorContext) {}

// ExitSelector is called when production selector is exited.
func (s *BaseSolomonSelectorParserListener) ExitSelector(ctx *SelectorContext) {}

// EnterSelectorOpString is called when production selectorOpString is entered.
func (s *BaseSolomonSelectorParserListener) EnterSelectorOpString(ctx *SelectorOpStringContext) {}

// ExitSelectorOpString is called when production selectorOpString is exited.
func (s *BaseSolomonSelectorParserListener) ExitSelectorOpString(ctx *SelectorOpStringContext) {}

// EnterSelectorOpNumber is called when production selectorOpNumber is entered.
func (s *BaseSolomonSelectorParserListener) EnterSelectorOpNumber(ctx *SelectorOpNumberContext) {}

// ExitSelectorOpNumber is called when production selectorOpNumber is exited.
func (s *BaseSolomonSelectorParserListener) ExitSelectorOpNumber(ctx *SelectorOpNumberContext) {}

// EnterSelectorOpDuration is called when production selectorOpDuration is entered.
func (s *BaseSolomonSelectorParserListener) EnterSelectorOpDuration(ctx *SelectorOpDurationContext) {}

// ExitSelectorOpDuration is called when production selectorOpDuration is exited.
func (s *BaseSolomonSelectorParserListener) ExitSelectorOpDuration(ctx *SelectorOpDurationContext) {}

// EnterSelectorLeftOperand is called when production selectorLeftOperand is entered.
func (s *BaseSolomonSelectorParserListener) EnterSelectorLeftOperand(ctx *SelectorLeftOperandContext) {}

// ExitSelectorLeftOperand is called when production selectorLeftOperand is exited.
func (s *BaseSolomonSelectorParserListener) ExitSelectorLeftOperand(ctx *SelectorLeftOperandContext) {}

// EnterNumberUnary is called when production numberUnary is entered.
func (s *BaseSolomonSelectorParserListener) EnterNumberUnary(ctx *NumberUnaryContext) {}

// ExitNumberUnary is called when production numberUnary is exited.
func (s *BaseSolomonSelectorParserListener) ExitNumberUnary(ctx *NumberUnaryContext) {}

// EnterLabelAbsent is called when production labelAbsent is entered.
func (s *BaseSolomonSelectorParserListener) EnterLabelAbsent(ctx *LabelAbsentContext) {}

// ExitLabelAbsent is called when production labelAbsent is exited.
func (s *BaseSolomonSelectorParserListener) ExitLabelAbsent(ctx *LabelAbsentContext) {}

// EnterIdentOrString is called when production identOrString is entered.
func (s *BaseSolomonSelectorParserListener) EnterIdentOrString(ctx *IdentOrStringContext) {}

// ExitIdentOrString is called when production identOrString is exited.
func (s *BaseSolomonSelectorParserListener) ExitIdentOrString(ctx *IdentOrStringContext) {}
