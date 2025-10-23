// Code generated from ../../../observability/lib/querylang/parser/v2/grammar/SolomonParser.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // SolomonParser

import "github.com/antlr4-go/antlr/v4"

// BaseSolomonParserListener is a complete listener for a parse tree produced by SolomonParser.
type BaseSolomonParserListener struct{}

var _ SolomonParserListener = &BaseSolomonParserListener{}

// VisitTerminal is called when a terminal node is visited.
func (s *BaseSolomonParserListener) VisitTerminal(node antlr.TerminalNode) {}

// VisitErrorNode is called when an error node is visited.
func (s *BaseSolomonParserListener) VisitErrorNode(node antlr.ErrorNode) {}

// EnterEveryRule is called when any rule is entered.
func (s *BaseSolomonParserListener) EnterEveryRule(ctx antlr.ParserRuleContext) {}

// ExitEveryRule is called when any rule is exited.
func (s *BaseSolomonParserListener) ExitEveryRule(ctx antlr.ParserRuleContext) {}

// EnterProgram is called when production program is entered.
func (s *BaseSolomonParserListener) EnterProgram(ctx *ProgramContext) {}

// ExitProgram is called when production program is exited.
func (s *BaseSolomonParserListener) ExitProgram(ctx *ProgramContext) {}

// EnterProgramWithReturn is called when production programWithReturn is entered.
func (s *BaseSolomonParserListener) EnterProgramWithReturn(ctx *ProgramWithReturnContext) {}

// ExitProgramWithReturn is called when production programWithReturn is exited.
func (s *BaseSolomonParserListener) ExitProgramWithReturn(ctx *ProgramWithReturnContext) {}

// EnterPreamble is called when production preamble is entered.
func (s *BaseSolomonParserListener) EnterPreamble(ctx *PreambleContext) {}

// ExitPreamble is called when production preamble is exited.
func (s *BaseSolomonParserListener) ExitPreamble(ctx *PreambleContext) {}

// EnterBlock is called when production block is entered.
func (s *BaseSolomonParserListener) EnterBlock(ctx *BlockContext) {}

// ExitBlock is called when production block is exited.
func (s *BaseSolomonParserListener) ExitBlock(ctx *BlockContext) {}

// EnterStatement is called when production statement is entered.
func (s *BaseSolomonParserListener) EnterStatement(ctx *StatementContext) {}

// ExitStatement is called when production statement is exited.
func (s *BaseSolomonParserListener) ExitStatement(ctx *StatementContext) {}

// EnterAnonymous is called when production anonymous is entered.
func (s *BaseSolomonParserListener) EnterAnonymous(ctx *AnonymousContext) {}

// ExitAnonymous is called when production anonymous is exited.
func (s *BaseSolomonParserListener) ExitAnonymous(ctx *AnonymousContext) {}

// EnterAssignment is called when production assignment is entered.
func (s *BaseSolomonParserListener) EnterAssignment(ctx *AssignmentContext) {}

// ExitAssignment is called when production assignment is exited.
func (s *BaseSolomonParserListener) ExitAssignment(ctx *AssignmentContext) {}

// EnterUse is called when production use is entered.
func (s *BaseSolomonParserListener) EnterUse(ctx *UseContext) {}

// ExitUse is called when production use is exited.
func (s *BaseSolomonParserListener) ExitUse(ctx *UseContext) {}

// EnterExpression is called when production expression is entered.
func (s *BaseSolomonParserListener) EnterExpression(ctx *ExpressionContext) {}

// ExitExpression is called when production expression is exited.
func (s *BaseSolomonParserListener) ExitExpression(ctx *ExpressionContext) {}

// EnterLambda is called when production lambda is entered.
func (s *BaseSolomonParserListener) EnterLambda(ctx *LambdaContext) {}

// ExitLambda is called when production lambda is exited.
func (s *BaseSolomonParserListener) ExitLambda(ctx *LambdaContext) {}

// EnterArglist is called when production arglist is entered.
func (s *BaseSolomonParserListener) EnterArglist(ctx *ArglistContext) {}

// ExitArglist is called when production arglist is exited.
func (s *BaseSolomonParserListener) ExitArglist(ctx *ArglistContext) {}

// EnterExprOr is called when production exprOr is entered.
func (s *BaseSolomonParserListener) EnterExprOr(ctx *ExprOrContext) {}

// ExitExprOr is called when production exprOr is exited.
func (s *BaseSolomonParserListener) ExitExprOr(ctx *ExprOrContext) {}

// EnterExprAnd is called when production exprAnd is entered.
func (s *BaseSolomonParserListener) EnterExprAnd(ctx *ExprAndContext) {}

// ExitExprAnd is called when production exprAnd is exited.
func (s *BaseSolomonParserListener) ExitExprAnd(ctx *ExprAndContext) {}

// EnterExprNot is called when production exprNot is entered.
func (s *BaseSolomonParserListener) EnterExprNot(ctx *ExprNotContext) {}

// ExitExprNot is called when production exprNot is exited.
func (s *BaseSolomonParserListener) ExitExprNot(ctx *ExprNotContext) {}

// EnterExprComp is called when production exprComp is entered.
func (s *BaseSolomonParserListener) EnterExprComp(ctx *ExprCompContext) {}

// ExitExprComp is called when production exprComp is exited.
func (s *BaseSolomonParserListener) ExitExprComp(ctx *ExprCompContext) {}

// EnterExprArith is called when production exprArith is entered.
func (s *BaseSolomonParserListener) EnterExprArith(ctx *ExprArithContext) {}

// ExitExprArith is called when production exprArith is exited.
func (s *BaseSolomonParserListener) ExitExprArith(ctx *ExprArithContext) {}

// EnterExprTerm is called when production exprTerm is entered.
func (s *BaseSolomonParserListener) EnterExprTerm(ctx *ExprTermContext) {}

// ExitExprTerm is called when production exprTerm is exited.
func (s *BaseSolomonParserListener) ExitExprTerm(ctx *ExprTermContext) {}

// EnterExprUnary is called when production exprUnary is entered.
func (s *BaseSolomonParserListener) EnterExprUnary(ctx *ExprUnaryContext) {}

// ExitExprUnary is called when production exprUnary is exited.
func (s *BaseSolomonParserListener) ExitExprUnary(ctx *ExprUnaryContext) {}

// EnterAtomExpressionInParentheses is called when production AtomExpressionInParentheses is entered.
func (s *BaseSolomonParserListener) EnterAtomExpressionInParentheses(ctx *AtomExpressionInParenthesesContext) {
}

// ExitAtomExpressionInParentheses is called when production AtomExpressionInParentheses is exited.
func (s *BaseSolomonParserListener) ExitAtomExpressionInParentheses(ctx *AtomExpressionInParenthesesContext) {
}

// EnterAtomVector is called when production AtomVector is entered.
func (s *BaseSolomonParserListener) EnterAtomVector(ctx *AtomVectorContext) {}

// ExitAtomVector is called when production AtomVector is exited.
func (s *BaseSolomonParserListener) ExitAtomVector(ctx *AtomVectorContext) {}

// EnterAtomSelectors is called when production AtomSelectors is entered.
func (s *BaseSolomonParserListener) EnterAtomSelectors(ctx *AtomSelectorsContext) {}

// ExitAtomSelectors is called when production AtomSelectors is exited.
func (s *BaseSolomonParserListener) ExitAtomSelectors(ctx *AtomSelectorsContext) {}

// EnterAtomDuration is called when production AtomDuration is entered.
func (s *BaseSolomonParserListener) EnterAtomDuration(ctx *AtomDurationContext) {}

// ExitAtomDuration is called when production AtomDuration is exited.
func (s *BaseSolomonParserListener) ExitAtomDuration(ctx *AtomDurationContext) {}

// EnterAtomNumber is called when production AtomNumber is entered.
func (s *BaseSolomonParserListener) EnterAtomNumber(ctx *AtomNumberContext) {}

// ExitAtomNumber is called when production AtomNumber is exited.
func (s *BaseSolomonParserListener) ExitAtomNumber(ctx *AtomNumberContext) {}

// EnterAtomString is called when production AtomString is entered.
func (s *BaseSolomonParserListener) EnterAtomString(ctx *AtomStringContext) {}

// ExitAtomString is called when production AtomString is exited.
func (s *BaseSolomonParserListener) ExitAtomString(ctx *AtomStringContext) {}

// EnterAtomLambda is called when production AtomLambda is entered.
func (s *BaseSolomonParserListener) EnterAtomLambda(ctx *AtomLambdaContext) {}

// ExitAtomLambda is called when production AtomLambda is exited.
func (s *BaseSolomonParserListener) ExitAtomLambda(ctx *AtomLambdaContext) {}

// EnterAtomIdent is called when production AtomIdent is entered.
func (s *BaseSolomonParserListener) EnterAtomIdent(ctx *AtomIdentContext) {}

// ExitAtomIdent is called when production AtomIdent is exited.
func (s *BaseSolomonParserListener) ExitAtomIdent(ctx *AtomIdentContext) {}

// EnterAtomCall is called when production AtomCall is entered.
func (s *BaseSolomonParserListener) EnterAtomCall(ctx *AtomCallContext) {}

// ExitAtomCall is called when production AtomCall is exited.
func (s *BaseSolomonParserListener) ExitAtomCall(ctx *AtomCallContext) {}

// EnterAtomCallByDuration is called when production AtomCallByDuration is entered.
func (s *BaseSolomonParserListener) EnterAtomCallByDuration(ctx *AtomCallByDurationContext) {}

// ExitAtomCallByDuration is called when production AtomCallByDuration is exited.
func (s *BaseSolomonParserListener) ExitAtomCallByDuration(ctx *AtomCallByDurationContext) {}

// EnterAtomCallByLabel is called when production AtomCallByLabel is entered.
func (s *BaseSolomonParserListener) EnterAtomCallByLabel(ctx *AtomCallByLabelContext) {}

// ExitAtomCallByLabel is called when production AtomCallByLabel is exited.
func (s *BaseSolomonParserListener) ExitAtomCallByLabel(ctx *AtomCallByLabelContext) {}

// EnterAtomCallByLabels is called when production AtomCallByLabels is entered.
func (s *BaseSolomonParserListener) EnterAtomCallByLabels(ctx *AtomCallByLabelsContext) {}

// ExitAtomCallByLabels is called when production AtomCallByLabels is exited.
func (s *BaseSolomonParserListener) ExitAtomCallByLabels(ctx *AtomCallByLabelsContext) {}

// EnterCall is called when production call is entered.
func (s *BaseSolomonParserListener) EnterCall(ctx *CallContext) {}

// ExitCall is called when production call is exited.
func (s *BaseSolomonParserListener) ExitCall(ctx *CallContext) {}

// EnterArguments is called when production arguments is entered.
func (s *BaseSolomonParserListener) EnterArguments(ctx *ArgumentsContext) {}

// ExitArguments is called when production arguments is exited.
func (s *BaseSolomonParserListener) ExitArguments(ctx *ArgumentsContext) {}

// EnterSequence is called when production sequence is entered.
func (s *BaseSolomonParserListener) EnterSequence(ctx *SequenceContext) {}

// ExitSequence is called when production sequence is exited.
func (s *BaseSolomonParserListener) ExitSequence(ctx *SequenceContext) {}

// EnterSolomonSelectors is called when production solomonSelectors is entered.
func (s *BaseSolomonParserListener) EnterSolomonSelectors(ctx *SolomonSelectorsContext) {}

// ExitSolomonSelectors is called when production solomonSelectors is exited.
func (s *BaseSolomonParserListener) ExitSolomonSelectors(ctx *SolomonSelectorsContext) {}

// EnterSelectors is called when production selectors is entered.
func (s *BaseSolomonParserListener) EnterSelectors(ctx *SelectorsContext) {}

// ExitSelectors is called when production selectors is exited.
func (s *BaseSolomonParserListener) ExitSelectors(ctx *SelectorsContext) {}

// EnterSelectorList is called when production selectorList is entered.
func (s *BaseSolomonParserListener) EnterSelectorList(ctx *SelectorListContext) {}

// ExitSelectorList is called when production selectorList is exited.
func (s *BaseSolomonParserListener) ExitSelectorList(ctx *SelectorListContext) {}

// EnterSelector is called when production selector is entered.
func (s *BaseSolomonParserListener) EnterSelector(ctx *SelectorContext) {}

// ExitSelector is called when production selector is exited.
func (s *BaseSolomonParserListener) ExitSelector(ctx *SelectorContext) {}

// EnterSelectorOpString is called when production selectorOpString is entered.
func (s *BaseSolomonParserListener) EnterSelectorOpString(ctx *SelectorOpStringContext) {}

// ExitSelectorOpString is called when production selectorOpString is exited.
func (s *BaseSolomonParserListener) ExitSelectorOpString(ctx *SelectorOpStringContext) {}

// EnterSelectorOpNumber is called when production selectorOpNumber is entered.
func (s *BaseSolomonParserListener) EnterSelectorOpNumber(ctx *SelectorOpNumberContext) {}

// ExitSelectorOpNumber is called when production selectorOpNumber is exited.
func (s *BaseSolomonParserListener) ExitSelectorOpNumber(ctx *SelectorOpNumberContext) {}

// EnterSelectorOpDuration is called when production selectorOpDuration is entered.
func (s *BaseSolomonParserListener) EnterSelectorOpDuration(ctx *SelectorOpDurationContext) {}

// ExitSelectorOpDuration is called when production selectorOpDuration is exited.
func (s *BaseSolomonParserListener) ExitSelectorOpDuration(ctx *SelectorOpDurationContext) {}

// EnterSelectorLeftOperand is called when production selectorLeftOperand is entered.
func (s *BaseSolomonParserListener) EnterSelectorLeftOperand(ctx *SelectorLeftOperandContext) {}

// ExitSelectorLeftOperand is called when production selectorLeftOperand is exited.
func (s *BaseSolomonParserListener) ExitSelectorLeftOperand(ctx *SelectorLeftOperandContext) {}

// EnterNumberUnary is called when production numberUnary is entered.
func (s *BaseSolomonParserListener) EnterNumberUnary(ctx *NumberUnaryContext) {}

// ExitNumberUnary is called when production numberUnary is exited.
func (s *BaseSolomonParserListener) ExitNumberUnary(ctx *NumberUnaryContext) {}

// EnterLabelAbsent is called when production labelAbsent is entered.
func (s *BaseSolomonParserListener) EnterLabelAbsent(ctx *LabelAbsentContext) {}

// ExitLabelAbsent is called when production labelAbsent is exited.
func (s *BaseSolomonParserListener) ExitLabelAbsent(ctx *LabelAbsentContext) {}

// EnterIdentOrString is called when production identOrString is entered.
func (s *BaseSolomonParserListener) EnterIdentOrString(ctx *IdentOrStringContext) {}

// ExitIdentOrString is called when production identOrString is exited.
func (s *BaseSolomonParserListener) ExitIdentOrString(ctx *IdentOrStringContext) {}
