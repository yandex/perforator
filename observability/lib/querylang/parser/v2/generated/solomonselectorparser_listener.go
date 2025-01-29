// Code generated from solomon/libs/java/solomon-grammar/SolomonSelectorParser.g4 by ANTLR 4.13.1. DO NOT EDIT.

package parser // SolomonSelectorParser

import "github.com/antlr4-go/antlr/v4"


// SolomonSelectorParserListener is a complete listener for a parse tree produced by SolomonSelectorParser.
type SolomonSelectorParserListener interface {
	antlr.ParseTreeListener

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
