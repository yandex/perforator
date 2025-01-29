// Code generated from solomon/libs/java/solomon-grammar/SolomonSelectorParser.g4 by ANTLR 4.13.1. DO NOT EDIT.

package parser // SolomonSelectorParser

import (
	"fmt"
	"strconv"
  	"sync"

	"github.com/antlr4-go/antlr/v4"
)

// Suppress unused import errors
var _ = fmt.Printf
var _ = strconv.Itoa
var _ = sync.Once{}


type SolomonSelectorParser struct {
	*antlr.BaseParser
}

var SolomonSelectorParserParserStaticData struct {
  once                   sync.Once
  serializedATN          []int32
  LiteralNames           []string
  SymbolicNames          []string
  RuleNames              []string
  PredictionContextCache *antlr.PredictionContextCache
  atn                    *antlr.ATN
  decisionToDFA          []*antlr.DFA
}

func solomonselectorparserParserInit() {
  staticData := &SolomonSelectorParserParserStaticData
  staticData.LiteralNames = []string{
    "", "'let'", "'by'", "'return'", "'{'", "'}'", "'('", "','", "')'", 
    "'['", "']'", "'->'", "'+'", "'-'", "'/'", "'*'", "'!'", "'&&'", "'||'", 
    "'<'", "'>'", "'<='", "'>='", "'=='", "'!='", "'!=='", "'=~'", "'!~'", 
    "'?'", "':'", "'='", "';'",
  }
  staticData.SymbolicNames = []string{
    "", "KW_LET", "KW_BY", "KW_RETURN", "OPENING_BRACE", "CLOSING_BRACE", 
    "OPENING_PAREN", "COMMA", "CLOSING_PAREN", "OPENING_BRACKET", "CLOSING_BRACKET", 
    "ARROW", "PLUS", "MINUS", "DIV", "MUL", "NOT", "AND", "OR", "LT", "GT", 
    "LE", "GE", "EQ", "NE", "NOT_EQUIV", "REGEX", "NOT_REGEX", "QUESTION", 
    "COLON", "ASSIGNMENT", "SEMICOLON", "IDENT_WITH_DOTS", "IDENT", "DURATION", 
    "NUMBER", "STRING", "WS", "COMMENTS",
  }
  staticData.RuleNames = []string{
    "selectors", "selectorList", "selector", "selectorOpString", "selectorOpNumber", 
    "selectorOpDuration", "selectorLeftOperand", "numberUnary", "labelAbsent", 
    "identOrString",
  }
  staticData.PredictionContextCache = antlr.NewPredictionContextCache()
  staticData.serializedATN = []int32{
	4, 1, 38, 74, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2, 4, 7, 
	4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 1, 0, 1, 
	0, 1, 0, 1, 0, 1, 0, 1, 0, 3, 0, 27, 8, 0, 1, 1, 1, 1, 1, 1, 5, 1, 32, 
	8, 1, 10, 1, 12, 1, 35, 9, 1, 1, 2, 1, 2, 1, 2, 1, 2, 3, 2, 41, 8, 2, 1, 
	2, 1, 2, 1, 2, 1, 2, 1, 2, 1, 2, 1, 2, 1, 2, 1, 2, 1, 2, 1, 2, 1, 2, 3, 
	2, 55, 8, 2, 1, 3, 1, 3, 1, 4, 1, 4, 1, 5, 1, 5, 1, 6, 1, 6, 1, 7, 3, 7, 
	66, 8, 7, 1, 7, 1, 7, 1, 8, 1, 8, 1, 9, 1, 9, 1, 9, 0, 0, 10, 0, 2, 4, 
	6, 8, 10, 12, 14, 16, 18, 0, 6, 2, 0, 19, 27, 30, 30, 2, 0, 19, 25, 30, 
	30, 1, 0, 19, 22, 2, 0, 32, 33, 36, 36, 1, 0, 12, 13, 2, 0, 33, 33, 36, 
	36, 70, 0, 26, 1, 0, 0, 0, 2, 28, 1, 0, 0, 0, 4, 54, 1, 0, 0, 0, 6, 56, 
	1, 0, 0, 0, 8, 58, 1, 0, 0, 0, 10, 60, 1, 0, 0, 0, 12, 62, 1, 0, 0, 0, 
	14, 65, 1, 0, 0, 0, 16, 69, 1, 0, 0, 0, 18, 71, 1, 0, 0, 0, 20, 21, 5, 
	4, 0, 0, 21, 22, 3, 2, 1, 0, 22, 23, 5, 5, 0, 0, 23, 27, 1, 0, 0, 0, 24, 
	25, 5, 4, 0, 0, 25, 27, 5, 5, 0, 0, 26, 20, 1, 0, 0, 0, 26, 24, 1, 0, 0, 
	0, 27, 1, 1, 0, 0, 0, 28, 33, 3, 4, 2, 0, 29, 30, 5, 7, 0, 0, 30, 32, 3, 
	4, 2, 0, 31, 29, 1, 0, 0, 0, 32, 35, 1, 0, 0, 0, 33, 31, 1, 0, 0, 0, 33, 
	34, 1, 0, 0, 0, 34, 3, 1, 0, 0, 0, 35, 33, 1, 0, 0, 0, 36, 37, 3, 12, 6, 
	0, 37, 40, 3, 6, 3, 0, 38, 41, 5, 32, 0, 0, 39, 41, 3, 18, 9, 0, 40, 38, 
	1, 0, 0, 0, 40, 39, 1, 0, 0, 0, 41, 55, 1, 0, 0, 0, 42, 43, 3, 12, 6, 0, 
	43, 44, 3, 8, 4, 0, 44, 45, 3, 14, 7, 0, 45, 55, 1, 0, 0, 0, 46, 47, 3, 
	12, 6, 0, 47, 48, 3, 10, 5, 0, 48, 49, 5, 34, 0, 0, 49, 55, 1, 0, 0, 0, 
	50, 51, 3, 12, 6, 0, 51, 52, 5, 30, 0, 0, 52, 53, 3, 16, 8, 0, 53, 55, 
	1, 0, 0, 0, 54, 36, 1, 0, 0, 0, 54, 42, 1, 0, 0, 0, 54, 46, 1, 0, 0, 0, 
	54, 50, 1, 0, 0, 0, 55, 5, 1, 0, 0, 0, 56, 57, 7, 0, 0, 0, 57, 7, 1, 0, 
	0, 0, 58, 59, 7, 1, 0, 0, 59, 9, 1, 0, 0, 0, 60, 61, 7, 2, 0, 0, 61, 11, 
	1, 0, 0, 0, 62, 63, 7, 3, 0, 0, 63, 13, 1, 0, 0, 0, 64, 66, 7, 4, 0, 0, 
	65, 64, 1, 0, 0, 0, 65, 66, 1, 0, 0, 0, 66, 67, 1, 0, 0, 0, 67, 68, 5, 
	35, 0, 0, 68, 15, 1, 0, 0, 0, 69, 70, 5, 13, 0, 0, 70, 17, 1, 0, 0, 0, 
	71, 72, 7, 5, 0, 0, 72, 19, 1, 0, 0, 0, 5, 26, 33, 40, 54, 65,
}
  deserializer := antlr.NewATNDeserializer(nil)
  staticData.atn = deserializer.Deserialize(staticData.serializedATN)
  atn := staticData.atn
  staticData.decisionToDFA = make([]*antlr.DFA, len(atn.DecisionToState))
  decisionToDFA := staticData.decisionToDFA
  for index, state := range atn.DecisionToState {
    decisionToDFA[index] = antlr.NewDFA(state, index)
  }
}

// SolomonSelectorParserInit initializes any static state used to implement SolomonSelectorParser. By default the
// static state used to implement the parser is lazily initialized during the first call to
// NewSolomonSelectorParser(). You can call this function if you wish to initialize the static state ahead
// of time.
func SolomonSelectorParserInit() {
  staticData := &SolomonSelectorParserParserStaticData
  staticData.once.Do(solomonselectorparserParserInit)
}

// NewSolomonSelectorParser produces a new parser instance for the optional input antlr.TokenStream.
func NewSolomonSelectorParser(input antlr.TokenStream) *SolomonSelectorParser {
	SolomonSelectorParserInit()
	this := new(SolomonSelectorParser)
	this.BaseParser = antlr.NewBaseParser(input)
  staticData := &SolomonSelectorParserParserStaticData
	this.Interpreter = antlr.NewParserATNSimulator(this, staticData.atn, staticData.decisionToDFA, staticData.PredictionContextCache)
	this.RuleNames = staticData.RuleNames
	this.LiteralNames = staticData.LiteralNames
	this.SymbolicNames = staticData.SymbolicNames
	this.GrammarFileName = "SolomonSelectorParser.g4"

	return this
}


// SolomonSelectorParser tokens.
const (
	SolomonSelectorParserEOF = antlr.TokenEOF
	SolomonSelectorParserKW_LET = 1
	SolomonSelectorParserKW_BY = 2
	SolomonSelectorParserKW_RETURN = 3
	SolomonSelectorParserOPENING_BRACE = 4
	SolomonSelectorParserCLOSING_BRACE = 5
	SolomonSelectorParserOPENING_PAREN = 6
	SolomonSelectorParserCOMMA = 7
	SolomonSelectorParserCLOSING_PAREN = 8
	SolomonSelectorParserOPENING_BRACKET = 9
	SolomonSelectorParserCLOSING_BRACKET = 10
	SolomonSelectorParserARROW = 11
	SolomonSelectorParserPLUS = 12
	SolomonSelectorParserMINUS = 13
	SolomonSelectorParserDIV = 14
	SolomonSelectorParserMUL = 15
	SolomonSelectorParserNOT = 16
	SolomonSelectorParserAND = 17
	SolomonSelectorParserOR = 18
	SolomonSelectorParserLT = 19
	SolomonSelectorParserGT = 20
	SolomonSelectorParserLE = 21
	SolomonSelectorParserGE = 22
	SolomonSelectorParserEQ = 23
	SolomonSelectorParserNE = 24
	SolomonSelectorParserNOT_EQUIV = 25
	SolomonSelectorParserREGEX = 26
	SolomonSelectorParserNOT_REGEX = 27
	SolomonSelectorParserQUESTION = 28
	SolomonSelectorParserCOLON = 29
	SolomonSelectorParserASSIGNMENT = 30
	SolomonSelectorParserSEMICOLON = 31
	SolomonSelectorParserIDENT_WITH_DOTS = 32
	SolomonSelectorParserIDENT = 33
	SolomonSelectorParserDURATION = 34
	SolomonSelectorParserNUMBER = 35
	SolomonSelectorParserSTRING = 36
	SolomonSelectorParserWS = 37
	SolomonSelectorParserCOMMENTS = 38
)

// SolomonSelectorParser rules.
const (
	SolomonSelectorParserRULE_selectors = 0
	SolomonSelectorParserRULE_selectorList = 1
	SolomonSelectorParserRULE_selector = 2
	SolomonSelectorParserRULE_selectorOpString = 3
	SolomonSelectorParserRULE_selectorOpNumber = 4
	SolomonSelectorParserRULE_selectorOpDuration = 5
	SolomonSelectorParserRULE_selectorLeftOperand = 6
	SolomonSelectorParserRULE_numberUnary = 7
	SolomonSelectorParserRULE_labelAbsent = 8
	SolomonSelectorParserRULE_identOrString = 9
)

// ISelectorsContext is an interface to support dynamic dispatch.
type ISelectorsContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	OPENING_BRACE() antlr.TerminalNode
	SelectorList() ISelectorListContext
	CLOSING_BRACE() antlr.TerminalNode

	// IsSelectorsContext differentiates from other interfaces.
	IsSelectorsContext()
}

type SelectorsContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptySelectorsContext() *SelectorsContext {
	var p = new(SelectorsContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonSelectorParserRULE_selectors
	return p
}

func InitEmptySelectorsContext(p *SelectorsContext)  {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonSelectorParserRULE_selectors
}

func (*SelectorsContext) IsSelectorsContext() {}

func NewSelectorsContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SelectorsContext {
	var p = new(SelectorsContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonSelectorParserRULE_selectors

	return p
}

func (s *SelectorsContext) GetParser() antlr.Parser { return s.parser }

func (s *SelectorsContext) OPENING_BRACE() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserOPENING_BRACE, 0)
}

func (s *SelectorsContext) SelectorList() ISelectorListContext {
	var t antlr.RuleContext;
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISelectorListContext); ok {
			t = ctx.(antlr.RuleContext);
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISelectorListContext)
}

func (s *SelectorsContext) CLOSING_BRACE() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserCLOSING_BRACE, 0)
}

func (s *SelectorsContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SelectorsContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}


func (s *SelectorsContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonSelectorParserListener); ok {
		listenerT.EnterSelectors(s)
	}
}

func (s *SelectorsContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonSelectorParserListener); ok {
		listenerT.ExitSelectors(s)
	}
}




func (p *SolomonSelectorParser) Selectors() (localctx ISelectorsContext) {
	localctx = NewSelectorsContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 0, SolomonSelectorParserRULE_selectors)
	p.SetState(26)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 0, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(20)
			p.Match(SolomonSelectorParserOPENING_BRACE)
			if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
			}
		}
		{
			p.SetState(21)
			p.SelectorList()
		}
		{
			p.SetState(22)
			p.Match(SolomonSelectorParserCLOSING_BRACE)
			if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
			}
		}


	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(24)
			p.Match(SolomonSelectorParserOPENING_BRACE)
			if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
			}
		}
		{
			p.SetState(25)
			p.Match(SolomonSelectorParserCLOSING_BRACE)
			if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
			}
		}

	case antlr.ATNInvalidAltNumber:
		goto errorExit
	}


errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	if false { goto errorExit }
	return localctx
}


// ISelectorListContext is an interface to support dynamic dispatch.
type ISelectorListContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllSelector() []ISelectorContext
	Selector(i int) ISelectorContext
	AllCOMMA() []antlr.TerminalNode
	COMMA(i int) antlr.TerminalNode

	// IsSelectorListContext differentiates from other interfaces.
	IsSelectorListContext()
}

type SelectorListContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptySelectorListContext() *SelectorListContext {
	var p = new(SelectorListContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonSelectorParserRULE_selectorList
	return p
}

func InitEmptySelectorListContext(p *SelectorListContext)  {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonSelectorParserRULE_selectorList
}

func (*SelectorListContext) IsSelectorListContext() {}

func NewSelectorListContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SelectorListContext {
	var p = new(SelectorListContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonSelectorParserRULE_selectorList

	return p
}

func (s *SelectorListContext) GetParser() antlr.Parser { return s.parser }

func (s *SelectorListContext) AllSelector() []ISelectorContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(ISelectorContext); ok {
			len++
		}
	}

	tst := make([]ISelectorContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(ISelectorContext); ok {
			tst[i] = t.(ISelectorContext)
			i++
		}
	}

	return tst
}

func (s *SelectorListContext) Selector(i int) ISelectorContext {
	var t antlr.RuleContext;
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISelectorContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext);
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISelectorContext)
}

func (s *SelectorListContext) AllCOMMA() []antlr.TerminalNode {
	return s.GetTokens(SolomonSelectorParserCOMMA)
}

func (s *SelectorListContext) COMMA(i int) antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserCOMMA, i)
}

func (s *SelectorListContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SelectorListContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}


func (s *SelectorListContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonSelectorParserListener); ok {
		listenerT.EnterSelectorList(s)
	}
}

func (s *SelectorListContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonSelectorParserListener); ok {
		listenerT.ExitSelectorList(s)
	}
}




func (p *SolomonSelectorParser) SelectorList() (localctx ISelectorListContext) {
	localctx = NewSelectorListContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 2, SolomonSelectorParserRULE_selectorList)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(28)
		p.Selector()
	}
	p.SetState(33)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)


	for _la == SolomonSelectorParserCOMMA {
		{
			p.SetState(29)
			p.Match(SolomonSelectorParserCOMMA)
			if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
			}
		}
		{
			p.SetState(30)
			p.Selector()
		}


		p.SetState(35)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
	    	goto errorExit
	    }
		_la = p.GetTokenStream().LA(1)
	}



errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	if false { goto errorExit }
	return localctx
}


// ISelectorContext is an interface to support dynamic dispatch.
type ISelectorContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	SelectorLeftOperand() ISelectorLeftOperandContext
	SelectorOpString() ISelectorOpStringContext
	IDENT_WITH_DOTS() antlr.TerminalNode
	IdentOrString() IIdentOrStringContext
	SelectorOpNumber() ISelectorOpNumberContext
	NumberUnary() INumberUnaryContext
	SelectorOpDuration() ISelectorOpDurationContext
	DURATION() antlr.TerminalNode
	ASSIGNMENT() antlr.TerminalNode
	LabelAbsent() ILabelAbsentContext

	// IsSelectorContext differentiates from other interfaces.
	IsSelectorContext()
}

type SelectorContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptySelectorContext() *SelectorContext {
	var p = new(SelectorContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonSelectorParserRULE_selector
	return p
}

func InitEmptySelectorContext(p *SelectorContext)  {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonSelectorParserRULE_selector
}

func (*SelectorContext) IsSelectorContext() {}

func NewSelectorContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SelectorContext {
	var p = new(SelectorContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonSelectorParserRULE_selector

	return p
}

func (s *SelectorContext) GetParser() antlr.Parser { return s.parser }

func (s *SelectorContext) SelectorLeftOperand() ISelectorLeftOperandContext {
	var t antlr.RuleContext;
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISelectorLeftOperandContext); ok {
			t = ctx.(antlr.RuleContext);
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISelectorLeftOperandContext)
}

func (s *SelectorContext) SelectorOpString() ISelectorOpStringContext {
	var t antlr.RuleContext;
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISelectorOpStringContext); ok {
			t = ctx.(antlr.RuleContext);
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISelectorOpStringContext)
}

func (s *SelectorContext) IDENT_WITH_DOTS() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserIDENT_WITH_DOTS, 0)
}

func (s *SelectorContext) IdentOrString() IIdentOrStringContext {
	var t antlr.RuleContext;
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IIdentOrStringContext); ok {
			t = ctx.(antlr.RuleContext);
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IIdentOrStringContext)
}

func (s *SelectorContext) SelectorOpNumber() ISelectorOpNumberContext {
	var t antlr.RuleContext;
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISelectorOpNumberContext); ok {
			t = ctx.(antlr.RuleContext);
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISelectorOpNumberContext)
}

func (s *SelectorContext) NumberUnary() INumberUnaryContext {
	var t antlr.RuleContext;
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INumberUnaryContext); ok {
			t = ctx.(antlr.RuleContext);
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(INumberUnaryContext)
}

func (s *SelectorContext) SelectorOpDuration() ISelectorOpDurationContext {
	var t antlr.RuleContext;
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISelectorOpDurationContext); ok {
			t = ctx.(antlr.RuleContext);
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISelectorOpDurationContext)
}

func (s *SelectorContext) DURATION() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserDURATION, 0)
}

func (s *SelectorContext) ASSIGNMENT() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserASSIGNMENT, 0)
}

func (s *SelectorContext) LabelAbsent() ILabelAbsentContext {
	var t antlr.RuleContext;
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ILabelAbsentContext); ok {
			t = ctx.(antlr.RuleContext);
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ILabelAbsentContext)
}

func (s *SelectorContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SelectorContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}


func (s *SelectorContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonSelectorParserListener); ok {
		listenerT.EnterSelector(s)
	}
}

func (s *SelectorContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonSelectorParserListener); ok {
		listenerT.ExitSelector(s)
	}
}




func (p *SolomonSelectorParser) Selector() (localctx ISelectorContext) {
	localctx = NewSelectorContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 4, SolomonSelectorParserRULE_selector)
	p.SetState(54)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 3, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(36)
			p.SelectorLeftOperand()
		}
		{
			p.SetState(37)
			p.SelectorOpString()
		}
		p.SetState(40)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}

		switch p.GetTokenStream().LA(1) {
		case SolomonSelectorParserIDENT_WITH_DOTS:
			{
				p.SetState(38)
				p.Match(SolomonSelectorParserIDENT_WITH_DOTS)
				if p.HasError() {
						// Recognition error - abort rule
						goto errorExit
				}
			}


		case SolomonSelectorParserIDENT, SolomonSelectorParserSTRING:
			{
				p.SetState(39)
				p.IdentOrString()
			}



		default:
			p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
			goto errorExit
		}


	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(42)
			p.SelectorLeftOperand()
		}
		{
			p.SetState(43)
			p.SelectorOpNumber()
		}
		{
			p.SetState(44)
			p.NumberUnary()
		}


	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(46)
			p.SelectorLeftOperand()
		}
		{
			p.SetState(47)
			p.SelectorOpDuration()
		}
		{
			p.SetState(48)
			p.Match(SolomonSelectorParserDURATION)
			if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
			}
		}


	case 4:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(50)
			p.SelectorLeftOperand()
		}
		{
			p.SetState(51)
			p.Match(SolomonSelectorParserASSIGNMENT)
			if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
			}
		}
		{
			p.SetState(52)
			p.LabelAbsent()
		}

	case antlr.ATNInvalidAltNumber:
		goto errorExit
	}


errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	if false { goto errorExit }
	return localctx
}


// ISelectorOpStringContext is an interface to support dynamic dispatch.
type ISelectorOpStringContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	ASSIGNMENT() antlr.TerminalNode
	NE() antlr.TerminalNode
	EQ() antlr.TerminalNode
	NOT_EQUIV() antlr.TerminalNode
	REGEX() antlr.TerminalNode
	NOT_REGEX() antlr.TerminalNode
	GT() antlr.TerminalNode
	LT() antlr.TerminalNode
	GE() antlr.TerminalNode
	LE() antlr.TerminalNode

	// IsSelectorOpStringContext differentiates from other interfaces.
	IsSelectorOpStringContext()
}

type SelectorOpStringContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptySelectorOpStringContext() *SelectorOpStringContext {
	var p = new(SelectorOpStringContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonSelectorParserRULE_selectorOpString
	return p
}

func InitEmptySelectorOpStringContext(p *SelectorOpStringContext)  {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonSelectorParserRULE_selectorOpString
}

func (*SelectorOpStringContext) IsSelectorOpStringContext() {}

func NewSelectorOpStringContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SelectorOpStringContext {
	var p = new(SelectorOpStringContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonSelectorParserRULE_selectorOpString

	return p
}

func (s *SelectorOpStringContext) GetParser() antlr.Parser { return s.parser }

func (s *SelectorOpStringContext) ASSIGNMENT() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserASSIGNMENT, 0)
}

func (s *SelectorOpStringContext) NE() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserNE, 0)
}

func (s *SelectorOpStringContext) EQ() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserEQ, 0)
}

func (s *SelectorOpStringContext) NOT_EQUIV() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserNOT_EQUIV, 0)
}

func (s *SelectorOpStringContext) REGEX() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserREGEX, 0)
}

func (s *SelectorOpStringContext) NOT_REGEX() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserNOT_REGEX, 0)
}

func (s *SelectorOpStringContext) GT() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserGT, 0)
}

func (s *SelectorOpStringContext) LT() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserLT, 0)
}

func (s *SelectorOpStringContext) GE() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserGE, 0)
}

func (s *SelectorOpStringContext) LE() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserLE, 0)
}

func (s *SelectorOpStringContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SelectorOpStringContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}


func (s *SelectorOpStringContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonSelectorParserListener); ok {
		listenerT.EnterSelectorOpString(s)
	}
}

func (s *SelectorOpStringContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonSelectorParserListener); ok {
		listenerT.ExitSelectorOpString(s)
	}
}




func (p *SolomonSelectorParser) SelectorOpString() (localctx ISelectorOpStringContext) {
	localctx = NewSelectorOpStringContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 6, SolomonSelectorParserRULE_selectorOpString)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(56)
		_la = p.GetTokenStream().LA(1)

		if !(((int64(_la) & ^0x3f) == 0 && ((int64(1) << _la) & 1341652992) != 0)) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}



errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	if false { goto errorExit }
	return localctx
}


// ISelectorOpNumberContext is an interface to support dynamic dispatch.
type ISelectorOpNumberContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	ASSIGNMENT() antlr.TerminalNode
	NE() antlr.TerminalNode
	EQ() antlr.TerminalNode
	NOT_EQUIV() antlr.TerminalNode
	GT() antlr.TerminalNode
	LT() antlr.TerminalNode
	GE() antlr.TerminalNode
	LE() antlr.TerminalNode

	// IsSelectorOpNumberContext differentiates from other interfaces.
	IsSelectorOpNumberContext()
}

type SelectorOpNumberContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptySelectorOpNumberContext() *SelectorOpNumberContext {
	var p = new(SelectorOpNumberContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonSelectorParserRULE_selectorOpNumber
	return p
}

func InitEmptySelectorOpNumberContext(p *SelectorOpNumberContext)  {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonSelectorParserRULE_selectorOpNumber
}

func (*SelectorOpNumberContext) IsSelectorOpNumberContext() {}

func NewSelectorOpNumberContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SelectorOpNumberContext {
	var p = new(SelectorOpNumberContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonSelectorParserRULE_selectorOpNumber

	return p
}

func (s *SelectorOpNumberContext) GetParser() antlr.Parser { return s.parser }

func (s *SelectorOpNumberContext) ASSIGNMENT() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserASSIGNMENT, 0)
}

func (s *SelectorOpNumberContext) NE() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserNE, 0)
}

func (s *SelectorOpNumberContext) EQ() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserEQ, 0)
}

func (s *SelectorOpNumberContext) NOT_EQUIV() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserNOT_EQUIV, 0)
}

func (s *SelectorOpNumberContext) GT() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserGT, 0)
}

func (s *SelectorOpNumberContext) LT() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserLT, 0)
}

func (s *SelectorOpNumberContext) GE() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserGE, 0)
}

func (s *SelectorOpNumberContext) LE() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserLE, 0)
}

func (s *SelectorOpNumberContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SelectorOpNumberContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}


func (s *SelectorOpNumberContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonSelectorParserListener); ok {
		listenerT.EnterSelectorOpNumber(s)
	}
}

func (s *SelectorOpNumberContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonSelectorParserListener); ok {
		listenerT.ExitSelectorOpNumber(s)
	}
}




func (p *SolomonSelectorParser) SelectorOpNumber() (localctx ISelectorOpNumberContext) {
	localctx = NewSelectorOpNumberContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 8, SolomonSelectorParserRULE_selectorOpNumber)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(58)
		_la = p.GetTokenStream().LA(1)

		if !(((int64(_la) & ^0x3f) == 0 && ((int64(1) << _la) & 1140326400) != 0)) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}



errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	if false { goto errorExit }
	return localctx
}


// ISelectorOpDurationContext is an interface to support dynamic dispatch.
type ISelectorOpDurationContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	GT() antlr.TerminalNode
	LT() antlr.TerminalNode
	GE() antlr.TerminalNode
	LE() antlr.TerminalNode

	// IsSelectorOpDurationContext differentiates from other interfaces.
	IsSelectorOpDurationContext()
}

type SelectorOpDurationContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptySelectorOpDurationContext() *SelectorOpDurationContext {
	var p = new(SelectorOpDurationContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonSelectorParserRULE_selectorOpDuration
	return p
}

func InitEmptySelectorOpDurationContext(p *SelectorOpDurationContext)  {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonSelectorParserRULE_selectorOpDuration
}

func (*SelectorOpDurationContext) IsSelectorOpDurationContext() {}

func NewSelectorOpDurationContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SelectorOpDurationContext {
	var p = new(SelectorOpDurationContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonSelectorParserRULE_selectorOpDuration

	return p
}

func (s *SelectorOpDurationContext) GetParser() antlr.Parser { return s.parser }

func (s *SelectorOpDurationContext) GT() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserGT, 0)
}

func (s *SelectorOpDurationContext) LT() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserLT, 0)
}

func (s *SelectorOpDurationContext) GE() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserGE, 0)
}

func (s *SelectorOpDurationContext) LE() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserLE, 0)
}

func (s *SelectorOpDurationContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SelectorOpDurationContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}


func (s *SelectorOpDurationContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonSelectorParserListener); ok {
		listenerT.EnterSelectorOpDuration(s)
	}
}

func (s *SelectorOpDurationContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonSelectorParserListener); ok {
		listenerT.ExitSelectorOpDuration(s)
	}
}




func (p *SolomonSelectorParser) SelectorOpDuration() (localctx ISelectorOpDurationContext) {
	localctx = NewSelectorOpDurationContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 10, SolomonSelectorParserRULE_selectorOpDuration)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(60)
		_la = p.GetTokenStream().LA(1)

		if !(((int64(_la) & ^0x3f) == 0 && ((int64(1) << _la) & 7864320) != 0)) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}



errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	if false { goto errorExit }
	return localctx
}


// ISelectorLeftOperandContext is an interface to support dynamic dispatch.
type ISelectorLeftOperandContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENT() antlr.TerminalNode
	IDENT_WITH_DOTS() antlr.TerminalNode
	STRING() antlr.TerminalNode

	// IsSelectorLeftOperandContext differentiates from other interfaces.
	IsSelectorLeftOperandContext()
}

type SelectorLeftOperandContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptySelectorLeftOperandContext() *SelectorLeftOperandContext {
	var p = new(SelectorLeftOperandContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonSelectorParserRULE_selectorLeftOperand
	return p
}

func InitEmptySelectorLeftOperandContext(p *SelectorLeftOperandContext)  {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonSelectorParserRULE_selectorLeftOperand
}

func (*SelectorLeftOperandContext) IsSelectorLeftOperandContext() {}

func NewSelectorLeftOperandContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SelectorLeftOperandContext {
	var p = new(SelectorLeftOperandContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonSelectorParserRULE_selectorLeftOperand

	return p
}

func (s *SelectorLeftOperandContext) GetParser() antlr.Parser { return s.parser }

func (s *SelectorLeftOperandContext) IDENT() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserIDENT, 0)
}

func (s *SelectorLeftOperandContext) IDENT_WITH_DOTS() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserIDENT_WITH_DOTS, 0)
}

func (s *SelectorLeftOperandContext) STRING() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserSTRING, 0)
}

func (s *SelectorLeftOperandContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SelectorLeftOperandContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}


func (s *SelectorLeftOperandContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonSelectorParserListener); ok {
		listenerT.EnterSelectorLeftOperand(s)
	}
}

func (s *SelectorLeftOperandContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonSelectorParserListener); ok {
		listenerT.ExitSelectorLeftOperand(s)
	}
}




func (p *SolomonSelectorParser) SelectorLeftOperand() (localctx ISelectorLeftOperandContext) {
	localctx = NewSelectorLeftOperandContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 12, SolomonSelectorParserRULE_selectorLeftOperand)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(62)
		_la = p.GetTokenStream().LA(1)

		if !(((int64(_la) & ^0x3f) == 0 && ((int64(1) << _la) & 81604378624) != 0)) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}



errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	if false { goto errorExit }
	return localctx
}


// INumberUnaryContext is an interface to support dynamic dispatch.
type INumberUnaryContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	NUMBER() antlr.TerminalNode
	PLUS() antlr.TerminalNode
	MINUS() antlr.TerminalNode

	// IsNumberUnaryContext differentiates from other interfaces.
	IsNumberUnaryContext()
}

type NumberUnaryContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyNumberUnaryContext() *NumberUnaryContext {
	var p = new(NumberUnaryContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonSelectorParserRULE_numberUnary
	return p
}

func InitEmptyNumberUnaryContext(p *NumberUnaryContext)  {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonSelectorParserRULE_numberUnary
}

func (*NumberUnaryContext) IsNumberUnaryContext() {}

func NewNumberUnaryContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *NumberUnaryContext {
	var p = new(NumberUnaryContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonSelectorParserRULE_numberUnary

	return p
}

func (s *NumberUnaryContext) GetParser() antlr.Parser { return s.parser }

func (s *NumberUnaryContext) NUMBER() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserNUMBER, 0)
}

func (s *NumberUnaryContext) PLUS() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserPLUS, 0)
}

func (s *NumberUnaryContext) MINUS() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserMINUS, 0)
}

func (s *NumberUnaryContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *NumberUnaryContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}


func (s *NumberUnaryContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonSelectorParserListener); ok {
		listenerT.EnterNumberUnary(s)
	}
}

func (s *NumberUnaryContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonSelectorParserListener); ok {
		listenerT.ExitNumberUnary(s)
	}
}




func (p *SolomonSelectorParser) NumberUnary() (localctx INumberUnaryContext) {
	localctx = NewNumberUnaryContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 14, SolomonSelectorParserRULE_numberUnary)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(65)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)


	if _la == SolomonSelectorParserPLUS || _la == SolomonSelectorParserMINUS {
		{
			p.SetState(64)
			_la = p.GetTokenStream().LA(1)

			if !(_la == SolomonSelectorParserPLUS || _la == SolomonSelectorParserMINUS) {
				p.GetErrorHandler().RecoverInline(p)
			} else {
				p.GetErrorHandler().ReportMatch(p)
				p.Consume()
			}
		}

	}
	{
		p.SetState(67)
		p.Match(SolomonSelectorParserNUMBER)
		if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
		}
	}



errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	if false { goto errorExit }
	return localctx
}


// ILabelAbsentContext is an interface to support dynamic dispatch.
type ILabelAbsentContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	MINUS() antlr.TerminalNode

	// IsLabelAbsentContext differentiates from other interfaces.
	IsLabelAbsentContext()
}

type LabelAbsentContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyLabelAbsentContext() *LabelAbsentContext {
	var p = new(LabelAbsentContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonSelectorParserRULE_labelAbsent
	return p
}

func InitEmptyLabelAbsentContext(p *LabelAbsentContext)  {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonSelectorParserRULE_labelAbsent
}

func (*LabelAbsentContext) IsLabelAbsentContext() {}

func NewLabelAbsentContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *LabelAbsentContext {
	var p = new(LabelAbsentContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonSelectorParserRULE_labelAbsent

	return p
}

func (s *LabelAbsentContext) GetParser() antlr.Parser { return s.parser }

func (s *LabelAbsentContext) MINUS() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserMINUS, 0)
}

func (s *LabelAbsentContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *LabelAbsentContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}


func (s *LabelAbsentContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonSelectorParserListener); ok {
		listenerT.EnterLabelAbsent(s)
	}
}

func (s *LabelAbsentContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonSelectorParserListener); ok {
		listenerT.ExitLabelAbsent(s)
	}
}




func (p *SolomonSelectorParser) LabelAbsent() (localctx ILabelAbsentContext) {
	localctx = NewLabelAbsentContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 16, SolomonSelectorParserRULE_labelAbsent)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(69)
		p.Match(SolomonSelectorParserMINUS)
		if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
		}
	}



errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	if false { goto errorExit }
	return localctx
}


// IIdentOrStringContext is an interface to support dynamic dispatch.
type IIdentOrStringContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENT() antlr.TerminalNode
	STRING() antlr.TerminalNode

	// IsIdentOrStringContext differentiates from other interfaces.
	IsIdentOrStringContext()
}

type IdentOrStringContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyIdentOrStringContext() *IdentOrStringContext {
	var p = new(IdentOrStringContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonSelectorParserRULE_identOrString
	return p
}

func InitEmptyIdentOrStringContext(p *IdentOrStringContext)  {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonSelectorParserRULE_identOrString
}

func (*IdentOrStringContext) IsIdentOrStringContext() {}

func NewIdentOrStringContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *IdentOrStringContext {
	var p = new(IdentOrStringContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonSelectorParserRULE_identOrString

	return p
}

func (s *IdentOrStringContext) GetParser() antlr.Parser { return s.parser }

func (s *IdentOrStringContext) IDENT() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserIDENT, 0)
}

func (s *IdentOrStringContext) STRING() antlr.TerminalNode {
	return s.GetToken(SolomonSelectorParserSTRING, 0)
}

func (s *IdentOrStringContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *IdentOrStringContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}


func (s *IdentOrStringContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonSelectorParserListener); ok {
		listenerT.EnterIdentOrString(s)
	}
}

func (s *IdentOrStringContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonSelectorParserListener); ok {
		listenerT.ExitIdentOrString(s)
	}
}




func (p *SolomonSelectorParser) IdentOrString() (localctx IIdentOrStringContext) {
	localctx = NewIdentOrStringContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 18, SolomonSelectorParserRULE_identOrString)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(71)
		_la = p.GetTokenStream().LA(1)

		if !(_la == SolomonSelectorParserIDENT || _la == SolomonSelectorParserSTRING) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}



errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	if false { goto errorExit }
	return localctx
}


