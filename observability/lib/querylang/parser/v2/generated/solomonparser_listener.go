// Code generated from ../../../solomon/libs/java/solomon-grammar/SolomonParser.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // SolomonParser

import "github.com/antlr4-go/antlr/v4"

// SolomonParserListener is a complete listener for a parse tree produced by SolomonParser.
type SolomonParserListener interface {
	antlr.ParseTreeListener

	// EnterProgram is called when entering the program production.
	EnterProgram(c *ProgramContext)

	// EnterProgramWithReturn is called when entering the programWithReturn production.
	EnterProgramWithReturn(c *ProgramWithReturnContext)

	// EnterPreamble is called when entering the preamble production.
	EnterPreamble(c *PreambleContext)

	// EnterBlock is called when entering the block production.
	EnterBlock(c *BlockContext)

	// EnterStatement is called when entering the statement production.
	EnterStatement(c *StatementContext)

	// EnterAnonymous is called when entering the anonymous production.
	EnterAnonymous(c *AnonymousContext)

	// EnterAssignment is called when entering the assignment production.
	EnterAssignment(c *AssignmentContext)

	// EnterUse is called when entering the use production.
	EnterUse(c *UseContext)

	// EnterExpression is called when entering the expression production.
	EnterExpression(c *ExpressionContext)

	// EnterLambda is called when entering the lambda production.
	EnterLambda(c *LambdaContext)

	// EnterArglist is called when entering the arglist production.
	EnterArglist(c *ArglistContext)

	// EnterExprOr is called when entering the exprOr production.
	EnterExprOr(c *ExprOrContext)

	// EnterExprAnd is called when entering the exprAnd production.
	EnterExprAnd(c *ExprAndContext)

	// EnterExprNot is called when entering the exprNot production.
	EnterExprNot(c *ExprNotContext)

	// EnterExprComp is called when entering the exprComp production.
	EnterExprComp(c *ExprCompContext)

	// EnterExprArith is called when entering the exprArith production.
	EnterExprArith(c *ExprArithContext)

	// EnterExprTerm is called when entering the exprTerm production.
	EnterExprTerm(c *ExprTermContext)

	// EnterExprUnary is called when entering the exprUnary production.
	EnterExprUnary(c *ExprUnaryContext)

	// EnterAtomExpressionInParentheses is called when entering the AtomExpressionInParentheses production.
	EnterAtomExpressionInParentheses(c *AtomExpressionInParenthesesContext)

	// EnterAtomVector is called when entering the AtomVector production.
	EnterAtomVector(c *AtomVectorContext)

	// EnterAtomSelectors is called when entering the AtomSelectors production.
	EnterAtomSelectors(c *AtomSelectorsContext)

	// EnterAtomDuration is called when entering the AtomDuration production.
	EnterAtomDuration(c *AtomDurationContext)

	// EnterAtomNumber is called when entering the AtomNumber production.
	EnterAtomNumber(c *AtomNumberContext)

	// EnterAtomString is called when entering the AtomString production.
	EnterAtomString(c *AtomStringContext)

	// EnterAtomLambda is called when entering the AtomLambda production.
	EnterAtomLambda(c *AtomLambdaContext)

	// EnterAtomIdent is called when entering the AtomIdent production.
	EnterAtomIdent(c *AtomIdentContext)

	// EnterAtomCall is called when entering the AtomCall production.
	EnterAtomCall(c *AtomCallContext)

	// EnterAtomCallByDuration is called when entering the AtomCallByDuration production.
	EnterAtomCallByDuration(c *AtomCallByDurationContext)

	// EnterAtomCallByLabel is called when entering the AtomCallByLabel production.
	EnterAtomCallByLabel(c *AtomCallByLabelContext)

	// EnterAtomCallByLabels is called when entering the AtomCallByLabels production.
	EnterAtomCallByLabels(c *AtomCallByLabelsContext)

	// EnterCall is called when entering the call production.
	EnterCall(c *CallContext)

	// EnterArguments is called when entering the arguments production.
	EnterArguments(c *ArgumentsContext)

	// EnterSequence is called when entering the sequence production.
	EnterSequence(c *SequenceContext)

	// EnterSolomonSelectors is called when entering the solomonSelectors production.
	EnterSolomonSelectors(c *SolomonSelectorsContext)

	// EnterSelectors is called when entering the selectors production.
	EnterSelectors(c *SelectorsContext)

	// EnterSelectorList is called when entering the selectorList production.
	EnterSelectorList(c *SelectorListContext)

	// EnterSelector is called when entering the selector production.
	EnterSelector(c *SelectorContext)

	// EnterSelectorOpString is called when entering the selectorOpString production.
	EnterSelectorOpString(c *SelectorOpStringContext)

	// EnterSelectorOpNumber is called when entering the selectorOpNumber production.
	EnterSelectorOpNumber(c *SelectorOpNumberContext)

	// EnterSelectorOpDuration is called when entering the selectorOpDuration production.
	EnterSelectorOpDuration(c *SelectorOpDurationContext)

	// EnterSelectorLeftOperand is called when entering the selectorLeftOperand production.
	EnterSelectorLeftOperand(c *SelectorLeftOperandContext)

	// EnterNumberUnary is called when entering the numberUnary production.
	EnterNumberUnary(c *NumberUnaryContext)

	// EnterLabelAbsent is called when entering the labelAbsent production.
	EnterLabelAbsent(c *LabelAbsentContext)

	// EnterIdentOrString is called when entering the identOrString production.
	EnterIdentOrString(c *IdentOrStringContext)

	// ExitProgram is called when exiting the program production.
	ExitProgram(c *ProgramContext)

	// ExitProgramWithReturn is called when exiting the programWithReturn production.
	ExitProgramWithReturn(c *ProgramWithReturnContext)

	// ExitPreamble is called when exiting the preamble production.
	ExitPreamble(c *PreambleContext)

	// ExitBlock is called when exiting the block production.
	ExitBlock(c *BlockContext)

	// ExitStatement is called when exiting the statement production.
	ExitStatement(c *StatementContext)

	// ExitAnonymous is called when exiting the anonymous production.
	ExitAnonymous(c *AnonymousContext)

	// ExitAssignment is called when exiting the assignment production.
	ExitAssignment(c *AssignmentContext)

	// ExitUse is called when exiting the use production.
	ExitUse(c *UseContext)

	// ExitExpression is called when exiting the expression production.
	ExitExpression(c *ExpressionContext)

	// ExitLambda is called when exiting the lambda production.
	ExitLambda(c *LambdaContext)

	// ExitArglist is called when exiting the arglist production.
	ExitArglist(c *ArglistContext)

	// ExitExprOr is called when exiting the exprOr production.
	ExitExprOr(c *ExprOrContext)

	// ExitExprAnd is called when exiting the exprAnd production.
	ExitExprAnd(c *ExprAndContext)

	// ExitExprNot is called when exiting the exprNot production.
	ExitExprNot(c *ExprNotContext)

	// ExitExprComp is called when exiting the exprComp production.
	ExitExprComp(c *ExprCompContext)

	// ExitExprArith is called when exiting the exprArith production.
	ExitExprArith(c *ExprArithContext)

	// ExitExprTerm is called when exiting the exprTerm production.
	ExitExprTerm(c *ExprTermContext)

	// ExitExprUnary is called when exiting the exprUnary production.
	ExitExprUnary(c *ExprUnaryContext)

	// ExitAtomExpressionInParentheses is called when exiting the AtomExpressionInParentheses production.
	ExitAtomExpressionInParentheses(c *AtomExpressionInParenthesesContext)

	// ExitAtomVector is called when exiting the AtomVector production.
	ExitAtomVector(c *AtomVectorContext)

	// ExitAtomSelectors is called when exiting the AtomSelectors production.
	ExitAtomSelectors(c *AtomSelectorsContext)

	// ExitAtomDuration is called when exiting the AtomDuration production.
	ExitAtomDuration(c *AtomDurationContext)

	// ExitAtomNumber is called when exiting the AtomNumber production.
	ExitAtomNumber(c *AtomNumberContext)

	// ExitAtomString is called when exiting the AtomString production.
	ExitAtomString(c *AtomStringContext)

	// ExitAtomLambda is called when exiting the AtomLambda production.
	ExitAtomLambda(c *AtomLambdaContext)

	// ExitAtomIdent is called when exiting the AtomIdent production.
	ExitAtomIdent(c *AtomIdentContext)

	// ExitAtomCall is called when exiting the AtomCall production.
	ExitAtomCall(c *AtomCallContext)

	// ExitAtomCallByDuration is called when exiting the AtomCallByDuration production.
	ExitAtomCallByDuration(c *AtomCallByDurationContext)

	// ExitAtomCallByLabel is called when exiting the AtomCallByLabel production.
	ExitAtomCallByLabel(c *AtomCallByLabelContext)

	// ExitAtomCallByLabels is called when exiting the AtomCallByLabels production.
	ExitAtomCallByLabels(c *AtomCallByLabelsContext)

	// ExitCall is called when exiting the call production.
	ExitCall(c *CallContext)

	// ExitArguments is called when exiting the arguments production.
	ExitArguments(c *ArgumentsContext)

	// ExitSequence is called when exiting the sequence production.
	ExitSequence(c *SequenceContext)

	// ExitSolomonSelectors is called when exiting the solomonSelectors production.
	ExitSolomonSelectors(c *SolomonSelectorsContext)

	// ExitSelectors is called when exiting the selectors production.
	ExitSelectors(c *SelectorsContext)

	// ExitSelectorList is called when exiting the selectorList production.
	ExitSelectorList(c *SelectorListContext)

	// ExitSelector is called when exiting the selector production.
	ExitSelector(c *SelectorContext)

	// ExitSelectorOpString is called when exiting the selectorOpString production.
	ExitSelectorOpString(c *SelectorOpStringContext)

	// ExitSelectorOpNumber is called when exiting the selectorOpNumber production.
	ExitSelectorOpNumber(c *SelectorOpNumberContext)

	// ExitSelectorOpDuration is called when exiting the selectorOpDuration production.
	ExitSelectorOpDuration(c *SelectorOpDurationContext)

	// ExitSelectorLeftOperand is called when exiting the selectorLeftOperand production.
	ExitSelectorLeftOperand(c *SelectorLeftOperandContext)

	// ExitNumberUnary is called when exiting the numberUnary production.
	ExitNumberUnary(c *NumberUnaryContext)

	// ExitLabelAbsent is called when exiting the labelAbsent production.
	ExitLabelAbsent(c *LabelAbsentContext)

	// ExitIdentOrString is called when exiting the identOrString production.
	ExitIdentOrString(c *IdentOrStringContext)
}
