// Code generated from ../../../observability/lib/querylang/parser/exp/grammar/SolomonParser.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // SolomonParser

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

type SolomonParser struct {
	*antlr.BaseParser
}

var SolomonParserParserStaticData struct {
	once                   sync.Once
	serializedATN          []int32
	LiteralNames           []string
	SymbolicNames          []string
	RuleNames              []string
	PredictionContextCache *antlr.PredictionContextCache
	atn                    *antlr.ATN
	decisionToDFA          []*antlr.DFA
}

func solomonparserParserInit() {
	staticData := &SolomonParserParserStaticData
	staticData.LiteralNames = []string{
		"", "'let'", "'by'", "'return'", "'{'", "'}'", "'('", "','", "')'",
		"'['", "']'", "'->'", "'+'", "'-'", "'/'", "'*'", "'!'", "'&&'", "'||'",
		"'<'", "'>'", "'<='", "'>='", "'=='", "'!='", "'!=='", "'=~'", "'!~'",
		"'=*'", "'!=*'", "'?'", "':'", "'='", "';'", "'|'",
	}
	staticData.SymbolicNames = []string{
		"", "KW_LET", "KW_BY", "KW_RETURN", "OPENING_BRACE", "CLOSING_BRACE",
		"OPENING_PAREN", "COMMA", "CLOSING_PAREN", "OPENING_BRACKET", "CLOSING_BRACKET",
		"ARROW", "PLUS", "MINUS", "DIV", "MUL", "NOT", "AND", "OR", "LT", "GT",
		"LE", "GE", "EQ", "NE", "NOT_EQUIV", "REGEX", "NOT_REGEX", "ISUBSTRING",
		"NOT_ISUBSTRING", "QUESTION", "COLON", "ASSIGNMENT", "SEMICOLON", "PIPE",
		"IDENT_WITH_DOTS", "IDENT", "DURATION", "NUMBER", "STRING", "WS", "COMMENTS",
	}
	staticData.RuleNames = []string{
		"program", "programWithReturn", "preamble", "block", "statement", "anonymous",
		"assignment", "use", "expression", "exprPipe", "exprTernary", "lambda",
		"arglist", "exprOr", "exprAnd", "exprNot", "exprComp", "exprArith",
		"exprTerm", "exprUnary", "atom", "callWithBy", "call", "arguments",
		"sequence", "solomonSelectors", "selectors", "selectorList", "selector",
		"selectorOpString", "selectorOpNumber", "selectorOpDuration", "selectorLeftOperand",
		"numberUnary", "labelAbsent", "identOrString",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 1, 41, 326, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2, 4, 7,
		4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2, 10, 7,
		10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 2, 14, 7, 14, 2, 15, 7, 15,
		2, 16, 7, 16, 2, 17, 7, 17, 2, 18, 7, 18, 2, 19, 7, 19, 2, 20, 7, 20, 2,
		21, 7, 21, 2, 22, 7, 22, 2, 23, 7, 23, 2, 24, 7, 24, 2, 25, 7, 25, 2, 26,
		7, 26, 2, 27, 7, 27, 2, 28, 7, 28, 2, 29, 7, 29, 2, 30, 7, 30, 2, 31, 7,
		31, 2, 32, 7, 32, 2, 33, 7, 33, 2, 34, 7, 34, 2, 35, 7, 35, 1, 0, 1, 0,
		1, 0, 1, 0, 1, 1, 1, 1, 1, 1, 3, 1, 80, 8, 1, 1, 1, 1, 1, 3, 1, 84, 8,
		1, 1, 1, 1, 1, 1, 2, 5, 2, 89, 8, 2, 10, 2, 12, 2, 92, 9, 2, 1, 3, 5, 3,
		95, 8, 3, 10, 3, 12, 3, 98, 9, 3, 1, 4, 1, 4, 3, 4, 102, 8, 4, 1, 5, 1,
		5, 3, 5, 106, 8, 5, 1, 6, 1, 6, 1, 6, 1, 6, 1, 6, 3, 6, 113, 8, 6, 1, 7,
		1, 7, 1, 7, 3, 7, 118, 8, 7, 1, 8, 1, 8, 3, 8, 122, 8, 8, 1, 9, 1, 9, 1,
		9, 5, 9, 127, 8, 9, 10, 9, 12, 9, 130, 9, 9, 1, 10, 1, 10, 1, 10, 1, 10,
		1, 10, 1, 10, 3, 10, 138, 8, 10, 1, 11, 1, 11, 1, 11, 1, 11, 1, 11, 1,
		11, 1, 11, 1, 11, 1, 11, 3, 11, 149, 8, 11, 1, 12, 1, 12, 1, 12, 5, 12,
		154, 8, 12, 10, 12, 12, 12, 157, 9, 12, 1, 13, 1, 13, 1, 13, 5, 13, 162,
		8, 13, 10, 13, 12, 13, 165, 9, 13, 1, 14, 1, 14, 1, 14, 5, 14, 170, 8,
		14, 10, 14, 12, 14, 173, 9, 14, 1, 15, 3, 15, 176, 8, 15, 1, 15, 1, 15,
		1, 16, 1, 16, 1, 16, 5, 16, 183, 8, 16, 10, 16, 12, 16, 186, 9, 16, 1,
		17, 1, 17, 1, 17, 5, 17, 191, 8, 17, 10, 17, 12, 17, 194, 9, 17, 1, 18,
		1, 18, 1, 18, 5, 18, 199, 8, 18, 10, 18, 12, 18, 202, 9, 18, 1, 19, 3,
		19, 205, 8, 19, 1, 19, 1, 19, 1, 20, 1, 20, 1, 20, 1, 20, 1, 20, 1, 20,
		1, 20, 1, 20, 1, 20, 1, 20, 1, 20, 1, 20, 1, 20, 1, 20, 1, 20, 3, 20, 224,
		8, 20, 1, 21, 1, 21, 1, 21, 1, 21, 1, 21, 1, 21, 1, 21, 1, 21, 1, 21, 1,
		21, 1, 21, 1, 21, 1, 21, 1, 21, 1, 21, 5, 21, 241, 8, 21, 10, 21, 12, 21,
		244, 9, 21, 1, 21, 1, 21, 3, 21, 248, 8, 21, 1, 22, 1, 22, 1, 22, 1, 22,
		1, 22, 1, 23, 1, 23, 3, 23, 257, 8, 23, 1, 24, 1, 24, 1, 24, 5, 24, 262,
		8, 24, 10, 24, 12, 24, 265, 9, 24, 1, 25, 1, 25, 3, 25, 269, 8, 25, 1,
		25, 1, 25, 1, 26, 1, 26, 1, 26, 1, 26, 1, 26, 1, 26, 3, 26, 279, 8, 26,
		1, 27, 1, 27, 1, 27, 5, 27, 284, 8, 27, 10, 27, 12, 27, 287, 9, 27, 1,
		28, 1, 28, 1, 28, 1, 28, 3, 28, 293, 8, 28, 1, 28, 1, 28, 1, 28, 1, 28,
		1, 28, 1, 28, 1, 28, 1, 28, 1, 28, 1, 28, 1, 28, 1, 28, 3, 28, 307, 8,
		28, 1, 29, 1, 29, 1, 30, 1, 30, 1, 31, 1, 31, 1, 32, 1, 32, 1, 33, 3, 33,
		318, 8, 33, 1, 33, 1, 33, 1, 34, 1, 34, 1, 35, 1, 35, 1, 35, 0, 0, 36,
		0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26, 28, 30, 32, 34, 36,
		38, 40, 42, 44, 46, 48, 50, 52, 54, 56, 58, 60, 62, 64, 66, 68, 70, 0,
		8, 1, 0, 19, 24, 1, 0, 12, 13, 1, 0, 14, 15, 2, 0, 19, 29, 32, 32, 2, 0,
		19, 25, 32, 32, 1, 0, 19, 22, 2, 0, 35, 36, 39, 39, 2, 0, 36, 36, 39, 39,
		332, 0, 72, 1, 0, 0, 0, 2, 76, 1, 0, 0, 0, 4, 90, 1, 0, 0, 0, 6, 96, 1,
		0, 0, 0, 8, 101, 1, 0, 0, 0, 10, 103, 1, 0, 0, 0, 12, 107, 1, 0, 0, 0,
		14, 114, 1, 0, 0, 0, 16, 121, 1, 0, 0, 0, 18, 123, 1, 0, 0, 0, 20, 131,
		1, 0, 0, 0, 22, 148, 1, 0, 0, 0, 24, 150, 1, 0, 0, 0, 26, 158, 1, 0, 0,
		0, 28, 166, 1, 0, 0, 0, 30, 175, 1, 0, 0, 0, 32, 179, 1, 0, 0, 0, 34, 187,
		1, 0, 0, 0, 36, 195, 1, 0, 0, 0, 38, 204, 1, 0, 0, 0, 40, 223, 1, 0, 0,
		0, 42, 247, 1, 0, 0, 0, 44, 249, 1, 0, 0, 0, 46, 256, 1, 0, 0, 0, 48, 258,
		1, 0, 0, 0, 50, 268, 1, 0, 0, 0, 52, 278, 1, 0, 0, 0, 54, 280, 1, 0, 0,
		0, 56, 306, 1, 0, 0, 0, 58, 308, 1, 0, 0, 0, 60, 310, 1, 0, 0, 0, 62, 312,
		1, 0, 0, 0, 64, 314, 1, 0, 0, 0, 66, 317, 1, 0, 0, 0, 68, 321, 1, 0, 0,
		0, 70, 323, 1, 0, 0, 0, 72, 73, 3, 4, 2, 0, 73, 74, 3, 6, 3, 0, 74, 75,
		5, 0, 0, 1, 75, 1, 1, 0, 0, 0, 76, 77, 3, 4, 2, 0, 77, 79, 3, 6, 3, 0,
		78, 80, 5, 3, 0, 0, 79, 78, 1, 0, 0, 0, 79, 80, 1, 0, 0, 0, 80, 81, 1,
		0, 0, 0, 81, 83, 3, 16, 8, 0, 82, 84, 5, 33, 0, 0, 83, 82, 1, 0, 0, 0,
		83, 84, 1, 0, 0, 0, 84, 85, 1, 0, 0, 0, 85, 86, 5, 0, 0, 1, 86, 3, 1, 0,
		0, 0, 87, 89, 3, 14, 7, 0, 88, 87, 1, 0, 0, 0, 89, 92, 1, 0, 0, 0, 90,
		88, 1, 0, 0, 0, 90, 91, 1, 0, 0, 0, 91, 5, 1, 0, 0, 0, 92, 90, 1, 0, 0,
		0, 93, 95, 3, 8, 4, 0, 94, 93, 1, 0, 0, 0, 95, 98, 1, 0, 0, 0, 96, 94,
		1, 0, 0, 0, 96, 97, 1, 0, 0, 0, 97, 7, 1, 0, 0, 0, 98, 96, 1, 0, 0, 0,
		99, 102, 3, 10, 5, 0, 100, 102, 3, 12, 6, 0, 101, 99, 1, 0, 0, 0, 101,
		100, 1, 0, 0, 0, 102, 9, 1, 0, 0, 0, 103, 105, 3, 16, 8, 0, 104, 106, 5,
		33, 0, 0, 105, 104, 1, 0, 0, 0, 105, 106, 1, 0, 0, 0, 106, 11, 1, 0, 0,
		0, 107, 108, 5, 1, 0, 0, 108, 109, 5, 36, 0, 0, 109, 110, 5, 32, 0, 0,
		110, 112, 3, 16, 8, 0, 111, 113, 5, 33, 0, 0, 112, 111, 1, 0, 0, 0, 112,
		113, 1, 0, 0, 0, 113, 13, 1, 0, 0, 0, 114, 115, 5, 36, 0, 0, 115, 117,
		3, 50, 25, 0, 116, 118, 5, 33, 0, 0, 117, 116, 1, 0, 0, 0, 117, 118, 1,
		0, 0, 0, 118, 15, 1, 0, 0, 0, 119, 122, 3, 22, 11, 0, 120, 122, 3, 18,
		9, 0, 121, 119, 1, 0, 0, 0, 121, 120, 1, 0, 0, 0, 122, 17, 1, 0, 0, 0,
		123, 128, 3, 20, 10, 0, 124, 125, 5, 34, 0, 0, 125, 127, 3, 42, 21, 0,
		126, 124, 1, 0, 0, 0, 127, 130, 1, 0, 0, 0, 128, 126, 1, 0, 0, 0, 128,
		129, 1, 0, 0, 0, 129, 19, 1, 0, 0, 0, 130, 128, 1, 0, 0, 0, 131, 137, 3,
		26, 13, 0, 132, 133, 5, 30, 0, 0, 133, 134, 3, 26, 13, 0, 134, 135, 5,
		31, 0, 0, 135, 136, 3, 26, 13, 0, 136, 138, 1, 0, 0, 0, 137, 132, 1, 0,
		0, 0, 137, 138, 1, 0, 0, 0, 138, 21, 1, 0, 0, 0, 139, 140, 5, 36, 0, 0,
		140, 141, 5, 11, 0, 0, 141, 149, 3, 16, 8, 0, 142, 143, 5, 6, 0, 0, 143,
		144, 3, 24, 12, 0, 144, 145, 5, 8, 0, 0, 145, 146, 5, 11, 0, 0, 146, 147,
		3, 16, 8, 0, 147, 149, 1, 0, 0, 0, 148, 139, 1, 0, 0, 0, 148, 142, 1, 0,
		0, 0, 149, 23, 1, 0, 0, 0, 150, 155, 5, 36, 0, 0, 151, 152, 5, 7, 0, 0,
		152, 154, 5, 36, 0, 0, 153, 151, 1, 0, 0, 0, 154, 157, 1, 0, 0, 0, 155,
		153, 1, 0, 0, 0, 155, 156, 1, 0, 0, 0, 156, 25, 1, 0, 0, 0, 157, 155, 1,
		0, 0, 0, 158, 163, 3, 28, 14, 0, 159, 160, 5, 18, 0, 0, 160, 162, 3, 28,
		14, 0, 161, 159, 1, 0, 0, 0, 162, 165, 1, 0, 0, 0, 163, 161, 1, 0, 0, 0,
		163, 164, 1, 0, 0, 0, 164, 27, 1, 0, 0, 0, 165, 163, 1, 0, 0, 0, 166, 171,
		3, 30, 15, 0, 167, 168, 5, 17, 0, 0, 168, 170, 3, 30, 15, 0, 169, 167,
		1, 0, 0, 0, 170, 173, 1, 0, 0, 0, 171, 169, 1, 0, 0, 0, 171, 172, 1, 0,
		0, 0, 172, 29, 1, 0, 0, 0, 173, 171, 1, 0, 0, 0, 174, 176, 5, 16, 0, 0,
		175, 174, 1, 0, 0, 0, 175, 176, 1, 0, 0, 0, 176, 177, 1, 0, 0, 0, 177,
		178, 3, 32, 16, 0, 178, 31, 1, 0, 0, 0, 179, 184, 3, 34, 17, 0, 180, 181,
		7, 0, 0, 0, 181, 183, 3, 34, 17, 0, 182, 180, 1, 0, 0, 0, 183, 186, 1,
		0, 0, 0, 184, 182, 1, 0, 0, 0, 184, 185, 1, 0, 0, 0, 185, 33, 1, 0, 0,
		0, 186, 184, 1, 0, 0, 0, 187, 192, 3, 36, 18, 0, 188, 189, 7, 1, 0, 0,
		189, 191, 3, 36, 18, 0, 190, 188, 1, 0, 0, 0, 191, 194, 1, 0, 0, 0, 192,
		190, 1, 0, 0, 0, 192, 193, 1, 0, 0, 0, 193, 35, 1, 0, 0, 0, 194, 192, 1,
		0, 0, 0, 195, 200, 3, 38, 19, 0, 196, 197, 7, 2, 0, 0, 197, 199, 3, 38,
		19, 0, 198, 196, 1, 0, 0, 0, 199, 202, 1, 0, 0, 0, 200, 198, 1, 0, 0, 0,
		200, 201, 1, 0, 0, 0, 201, 37, 1, 0, 0, 0, 202, 200, 1, 0, 0, 0, 203, 205,
		7, 1, 0, 0, 204, 203, 1, 0, 0, 0, 204, 205, 1, 0, 0, 0, 205, 206, 1, 0,
		0, 0, 206, 207, 3, 40, 20, 0, 207, 39, 1, 0, 0, 0, 208, 209, 5, 6, 0, 0,
		209, 210, 3, 16, 8, 0, 210, 211, 5, 8, 0, 0, 211, 224, 1, 0, 0, 0, 212,
		213, 5, 9, 0, 0, 213, 214, 3, 48, 24, 0, 214, 215, 5, 10, 0, 0, 215, 224,
		1, 0, 0, 0, 216, 224, 3, 50, 25, 0, 217, 224, 5, 37, 0, 0, 218, 224, 5,
		38, 0, 0, 219, 224, 5, 39, 0, 0, 220, 224, 3, 22, 11, 0, 221, 224, 5, 36,
		0, 0, 222, 224, 3, 42, 21, 0, 223, 208, 1, 0, 0, 0, 223, 212, 1, 0, 0,
		0, 223, 216, 1, 0, 0, 0, 223, 217, 1, 0, 0, 0, 223, 218, 1, 0, 0, 0, 223,
		219, 1, 0, 0, 0, 223, 220, 1, 0, 0, 0, 223, 221, 1, 0, 0, 0, 223, 222,
		1, 0, 0, 0, 224, 41, 1, 0, 0, 0, 225, 248, 3, 44, 22, 0, 226, 227, 3, 44,
		22, 0, 227, 228, 5, 2, 0, 0, 228, 229, 5, 37, 0, 0, 229, 248, 1, 0, 0,
		0, 230, 231, 3, 44, 22, 0, 231, 232, 5, 2, 0, 0, 232, 233, 3, 70, 35, 0,
		233, 248, 1, 0, 0, 0, 234, 235, 3, 44, 22, 0, 235, 236, 5, 2, 0, 0, 236,
		237, 5, 6, 0, 0, 237, 242, 3, 70, 35, 0, 238, 239, 5, 7, 0, 0, 239, 241,
		3, 70, 35, 0, 240, 238, 1, 0, 0, 0, 241, 244, 1, 0, 0, 0, 242, 240, 1,
		0, 0, 0, 242, 243, 1, 0, 0, 0, 243, 245, 1, 0, 0, 0, 244, 242, 1, 0, 0,
		0, 245, 246, 5, 8, 0, 0, 246, 248, 1, 0, 0, 0, 247, 225, 1, 0, 0, 0, 247,
		226, 1, 0, 0, 0, 247, 230, 1, 0, 0, 0, 247, 234, 1, 0, 0, 0, 248, 43, 1,
		0, 0, 0, 249, 250, 5, 36, 0, 0, 250, 251, 5, 6, 0, 0, 251, 252, 3, 46,
		23, 0, 252, 253, 5, 8, 0, 0, 253, 45, 1, 0, 0, 0, 254, 257, 3, 48, 24,
		0, 255, 257, 1, 0, 0, 0, 256, 254, 1, 0, 0, 0, 256, 255, 1, 0, 0, 0, 257,
		47, 1, 0, 0, 0, 258, 263, 3, 16, 8, 0, 259, 260, 5, 7, 0, 0, 260, 262,
		3, 16, 8, 0, 261, 259, 1, 0, 0, 0, 262, 265, 1, 0, 0, 0, 263, 261, 1, 0,
		0, 0, 263, 264, 1, 0, 0, 0, 264, 49, 1, 0, 0, 0, 265, 263, 1, 0, 0, 0,
		266, 269, 3, 70, 35, 0, 267, 269, 5, 35, 0, 0, 268, 266, 1, 0, 0, 0, 268,
		267, 1, 0, 0, 0, 268, 269, 1, 0, 0, 0, 269, 270, 1, 0, 0, 0, 270, 271,
		3, 52, 26, 0, 271, 51, 1, 0, 0, 0, 272, 273, 5, 4, 0, 0, 273, 274, 3, 54,
		27, 0, 274, 275, 5, 5, 0, 0, 275, 279, 1, 0, 0, 0, 276, 277, 5, 4, 0, 0,
		277, 279, 5, 5, 0, 0, 278, 272, 1, 0, 0, 0, 278, 276, 1, 0, 0, 0, 279,
		53, 1, 0, 0, 0, 280, 285, 3, 56, 28, 0, 281, 282, 5, 7, 0, 0, 282, 284,
		3, 56, 28, 0, 283, 281, 1, 0, 0, 0, 284, 287, 1, 0, 0, 0, 285, 283, 1,
		0, 0, 0, 285, 286, 1, 0, 0, 0, 286, 55, 1, 0, 0, 0, 287, 285, 1, 0, 0,
		0, 288, 289, 3, 64, 32, 0, 289, 292, 3, 58, 29, 0, 290, 293, 5, 35, 0,
		0, 291, 293, 3, 70, 35, 0, 292, 290, 1, 0, 0, 0, 292, 291, 1, 0, 0, 0,
		293, 307, 1, 0, 0, 0, 294, 295, 3, 64, 32, 0, 295, 296, 3, 60, 30, 0, 296,
		297, 3, 66, 33, 0, 297, 307, 1, 0, 0, 0, 298, 299, 3, 64, 32, 0, 299, 300,
		3, 62, 31, 0, 300, 301, 5, 37, 0, 0, 301, 307, 1, 0, 0, 0, 302, 303, 3,
		64, 32, 0, 303, 304, 5, 32, 0, 0, 304, 305, 3, 68, 34, 0, 305, 307, 1,
		0, 0, 0, 306, 288, 1, 0, 0, 0, 306, 294, 1, 0, 0, 0, 306, 298, 1, 0, 0,
		0, 306, 302, 1, 0, 0, 0, 307, 57, 1, 0, 0, 0, 308, 309, 7, 3, 0, 0, 309,
		59, 1, 0, 0, 0, 310, 311, 7, 4, 0, 0, 311, 61, 1, 0, 0, 0, 312, 313, 7,
		5, 0, 0, 313, 63, 1, 0, 0, 0, 314, 315, 7, 6, 0, 0, 315, 65, 1, 0, 0, 0,
		316, 318, 7, 1, 0, 0, 317, 316, 1, 0, 0, 0, 317, 318, 1, 0, 0, 0, 318,
		319, 1, 0, 0, 0, 319, 320, 5, 38, 0, 0, 320, 67, 1, 0, 0, 0, 321, 322,
		5, 13, 0, 0, 322, 69, 1, 0, 0, 0, 323, 324, 7, 7, 0, 0, 324, 71, 1, 0,
		0, 0, 31, 79, 83, 90, 96, 101, 105, 112, 117, 121, 128, 137, 148, 155,
		163, 171, 175, 184, 192, 200, 204, 223, 242, 247, 256, 263, 268, 278, 285,
		292, 306, 317,
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

// SolomonParserInit initializes any static state used to implement SolomonParser. By default the
// static state used to implement the parser is lazily initialized during the first call to
// NewSolomonParser(). You can call this function if you wish to initialize the static state ahead
// of time.
func SolomonParserInit() {
	staticData := &SolomonParserParserStaticData
	staticData.once.Do(solomonparserParserInit)
}

// NewSolomonParser produces a new parser instance for the optional input antlr.TokenStream.
func NewSolomonParser(input antlr.TokenStream) *SolomonParser {
	SolomonParserInit()
	this := new(SolomonParser)
	this.BaseParser = antlr.NewBaseParser(input)
	staticData := &SolomonParserParserStaticData
	this.Interpreter = antlr.NewParserATNSimulator(this, staticData.atn, staticData.decisionToDFA, staticData.PredictionContextCache)
	this.RuleNames = staticData.RuleNames
	this.LiteralNames = staticData.LiteralNames
	this.SymbolicNames = staticData.SymbolicNames
	this.GrammarFileName = "SolomonParser.g4"

	return this
}

// SolomonParser tokens.
const (
	SolomonParserEOF             = antlr.TokenEOF
	SolomonParserKW_LET          = 1
	SolomonParserKW_BY           = 2
	SolomonParserKW_RETURN       = 3
	SolomonParserOPENING_BRACE   = 4
	SolomonParserCLOSING_BRACE   = 5
	SolomonParserOPENING_PAREN   = 6
	SolomonParserCOMMA           = 7
	SolomonParserCLOSING_PAREN   = 8
	SolomonParserOPENING_BRACKET = 9
	SolomonParserCLOSING_BRACKET = 10
	SolomonParserARROW           = 11
	SolomonParserPLUS            = 12
	SolomonParserMINUS           = 13
	SolomonParserDIV             = 14
	SolomonParserMUL             = 15
	SolomonParserNOT             = 16
	SolomonParserAND             = 17
	SolomonParserOR              = 18
	SolomonParserLT              = 19
	SolomonParserGT              = 20
	SolomonParserLE              = 21
	SolomonParserGE              = 22
	SolomonParserEQ              = 23
	SolomonParserNE              = 24
	SolomonParserNOT_EQUIV       = 25
	SolomonParserREGEX           = 26
	SolomonParserNOT_REGEX       = 27
	SolomonParserISUBSTRING      = 28
	SolomonParserNOT_ISUBSTRING  = 29
	SolomonParserQUESTION        = 30
	SolomonParserCOLON           = 31
	SolomonParserASSIGNMENT      = 32
	SolomonParserSEMICOLON       = 33
	SolomonParserPIPE            = 34
	SolomonParserIDENT_WITH_DOTS = 35
	SolomonParserIDENT           = 36
	SolomonParserDURATION        = 37
	SolomonParserNUMBER          = 38
	SolomonParserSTRING          = 39
	SolomonParserWS              = 40
	SolomonParserCOMMENTS        = 41
)

// SolomonParser rules.
const (
	SolomonParserRULE_program             = 0
	SolomonParserRULE_programWithReturn   = 1
	SolomonParserRULE_preamble            = 2
	SolomonParserRULE_block               = 3
	SolomonParserRULE_statement           = 4
	SolomonParserRULE_anonymous           = 5
	SolomonParserRULE_assignment          = 6
	SolomonParserRULE_use                 = 7
	SolomonParserRULE_expression          = 8
	SolomonParserRULE_exprPipe            = 9
	SolomonParserRULE_exprTernary         = 10
	SolomonParserRULE_lambda              = 11
	SolomonParserRULE_arglist             = 12
	SolomonParserRULE_exprOr              = 13
	SolomonParserRULE_exprAnd             = 14
	SolomonParserRULE_exprNot             = 15
	SolomonParserRULE_exprComp            = 16
	SolomonParserRULE_exprArith           = 17
	SolomonParserRULE_exprTerm            = 18
	SolomonParserRULE_exprUnary           = 19
	SolomonParserRULE_atom                = 20
	SolomonParserRULE_callWithBy          = 21
	SolomonParserRULE_call                = 22
	SolomonParserRULE_arguments           = 23
	SolomonParserRULE_sequence            = 24
	SolomonParserRULE_solomonSelectors    = 25
	SolomonParserRULE_selectors           = 26
	SolomonParserRULE_selectorList        = 27
	SolomonParserRULE_selector            = 28
	SolomonParserRULE_selectorOpString    = 29
	SolomonParserRULE_selectorOpNumber    = 30
	SolomonParserRULE_selectorOpDuration  = 31
	SolomonParserRULE_selectorLeftOperand = 32
	SolomonParserRULE_numberUnary         = 33
	SolomonParserRULE_labelAbsent         = 34
	SolomonParserRULE_identOrString       = 35
)

// IProgramContext is an interface to support dynamic dispatch.
type IProgramContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Preamble() IPreambleContext
	Block() IBlockContext
	EOF() antlr.TerminalNode

	// IsProgramContext differentiates from other interfaces.
	IsProgramContext()
}

type ProgramContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyProgramContext() *ProgramContext {
	var p = new(ProgramContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_program
	return p
}

func InitEmptyProgramContext(p *ProgramContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_program
}

func (*ProgramContext) IsProgramContext() {}

func NewProgramContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ProgramContext {
	var p = new(ProgramContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_program

	return p
}

func (s *ProgramContext) GetParser() antlr.Parser { return s.parser }

func (s *ProgramContext) Preamble() IPreambleContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IPreambleContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IPreambleContext)
}

func (s *ProgramContext) Block() IBlockContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IBlockContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IBlockContext)
}

func (s *ProgramContext) EOF() antlr.TerminalNode {
	return s.GetToken(SolomonParserEOF, 0)
}

func (s *ProgramContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ProgramContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ProgramContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterProgram(s)
	}
}

func (s *ProgramContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitProgram(s)
	}
}

func (p *SolomonParser) Program() (localctx IProgramContext) {
	localctx = NewProgramContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 0, SolomonParserRULE_program)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(72)
		p.Preamble()
	}
	{
		p.SetState(73)
		p.Block()
	}
	{
		p.SetState(74)
		p.Match(SolomonParserEOF)
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

// IProgramWithReturnContext is an interface to support dynamic dispatch.
type IProgramWithReturnContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Preamble() IPreambleContext
	Block() IBlockContext
	Expression() IExpressionContext
	EOF() antlr.TerminalNode
	KW_RETURN() antlr.TerminalNode
	SEMICOLON() antlr.TerminalNode

	// IsProgramWithReturnContext differentiates from other interfaces.
	IsProgramWithReturnContext()
}

type ProgramWithReturnContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyProgramWithReturnContext() *ProgramWithReturnContext {
	var p = new(ProgramWithReturnContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_programWithReturn
	return p
}

func InitEmptyProgramWithReturnContext(p *ProgramWithReturnContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_programWithReturn
}

func (*ProgramWithReturnContext) IsProgramWithReturnContext() {}

func NewProgramWithReturnContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ProgramWithReturnContext {
	var p = new(ProgramWithReturnContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_programWithReturn

	return p
}

func (s *ProgramWithReturnContext) GetParser() antlr.Parser { return s.parser }

func (s *ProgramWithReturnContext) Preamble() IPreambleContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IPreambleContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IPreambleContext)
}

func (s *ProgramWithReturnContext) Block() IBlockContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IBlockContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IBlockContext)
}

func (s *ProgramWithReturnContext) Expression() IExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExpressionContext)
}

func (s *ProgramWithReturnContext) EOF() antlr.TerminalNode {
	return s.GetToken(SolomonParserEOF, 0)
}

func (s *ProgramWithReturnContext) KW_RETURN() antlr.TerminalNode {
	return s.GetToken(SolomonParserKW_RETURN, 0)
}

func (s *ProgramWithReturnContext) SEMICOLON() antlr.TerminalNode {
	return s.GetToken(SolomonParserSEMICOLON, 0)
}

func (s *ProgramWithReturnContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ProgramWithReturnContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ProgramWithReturnContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterProgramWithReturn(s)
	}
}

func (s *ProgramWithReturnContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitProgramWithReturn(s)
	}
}

func (p *SolomonParser) ProgramWithReturn() (localctx IProgramWithReturnContext) {
	localctx = NewProgramWithReturnContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 2, SolomonParserRULE_programWithReturn)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(76)
		p.Preamble()
	}
	{
		p.SetState(77)
		p.Block()
	}
	p.SetState(79)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == SolomonParserKW_RETURN {
		{
			p.SetState(78)
			p.Match(SolomonParserKW_RETURN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}
	{
		p.SetState(81)
		p.Expression()
	}
	p.SetState(83)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == SolomonParserSEMICOLON {
		{
			p.SetState(82)
			p.Match(SolomonParserSEMICOLON)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}
	{
		p.SetState(85)
		p.Match(SolomonParserEOF)
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

// IPreambleContext is an interface to support dynamic dispatch.
type IPreambleContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllUse() []IUseContext
	Use(i int) IUseContext

	// IsPreambleContext differentiates from other interfaces.
	IsPreambleContext()
}

type PreambleContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyPreambleContext() *PreambleContext {
	var p = new(PreambleContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_preamble
	return p
}

func InitEmptyPreambleContext(p *PreambleContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_preamble
}

func (*PreambleContext) IsPreambleContext() {}

func NewPreambleContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *PreambleContext {
	var p = new(PreambleContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_preamble

	return p
}

func (s *PreambleContext) GetParser() antlr.Parser { return s.parser }

func (s *PreambleContext) AllUse() []IUseContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IUseContext); ok {
			len++
		}
	}

	tst := make([]IUseContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IUseContext); ok {
			tst[i] = t.(IUseContext)
			i++
		}
	}

	return tst
}

func (s *PreambleContext) Use(i int) IUseContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IUseContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IUseContext)
}

func (s *PreambleContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *PreambleContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *PreambleContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterPreamble(s)
	}
}

func (s *PreambleContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitPreamble(s)
	}
}

func (p *SolomonParser) Preamble() (localctx IPreambleContext) {
	localctx = NewPreambleContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 4, SolomonParserRULE_preamble)
	var _alt int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(90)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 2, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(87)
				p.Use()
			}

		}
		p.SetState(92)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 2, p.GetParserRuleContext())
		if p.HasError() {
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

// IBlockContext is an interface to support dynamic dispatch.
type IBlockContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllStatement() []IStatementContext
	Statement(i int) IStatementContext

	// IsBlockContext differentiates from other interfaces.
	IsBlockContext()
}

type BlockContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyBlockContext() *BlockContext {
	var p = new(BlockContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_block
	return p
}

func InitEmptyBlockContext(p *BlockContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_block
}

func (*BlockContext) IsBlockContext() {}

func NewBlockContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *BlockContext {
	var p = new(BlockContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_block

	return p
}

func (s *BlockContext) GetParser() antlr.Parser { return s.parser }

func (s *BlockContext) AllStatement() []IStatementContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IStatementContext); ok {
			len++
		}
	}

	tst := make([]IStatementContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IStatementContext); ok {
			tst[i] = t.(IStatementContext)
			i++
		}
	}

	return tst
}

func (s *BlockContext) Statement(i int) IStatementContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IStatementContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IStatementContext)
}

func (s *BlockContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *BlockContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *BlockContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterBlock(s)
	}
}

func (s *BlockContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitBlock(s)
	}
}

func (p *SolomonParser) Block() (localctx IBlockContext) {
	localctx = NewBlockContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 6, SolomonParserRULE_block)
	var _alt int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(96)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 3, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(93)
				p.Statement()
			}

		}
		p.SetState(98)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 3, p.GetParserRuleContext())
		if p.HasError() {
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

// IStatementContext is an interface to support dynamic dispatch.
type IStatementContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Anonymous() IAnonymousContext
	Assignment() IAssignmentContext

	// IsStatementContext differentiates from other interfaces.
	IsStatementContext()
}

type StatementContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyStatementContext() *StatementContext {
	var p = new(StatementContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_statement
	return p
}

func InitEmptyStatementContext(p *StatementContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_statement
}

func (*StatementContext) IsStatementContext() {}

func NewStatementContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *StatementContext {
	var p = new(StatementContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_statement

	return p
}

func (s *StatementContext) GetParser() antlr.Parser { return s.parser }

func (s *StatementContext) Anonymous() IAnonymousContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IAnonymousContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IAnonymousContext)
}

func (s *StatementContext) Assignment() IAssignmentContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IAssignmentContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IAssignmentContext)
}

func (s *StatementContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *StatementContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *StatementContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterStatement(s)
	}
}

func (s *StatementContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitStatement(s)
	}
}

func (p *SolomonParser) Statement() (localctx IStatementContext) {
	localctx = NewStatementContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 8, SolomonParserRULE_statement)
	p.SetState(101)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case SolomonParserOPENING_BRACE, SolomonParserOPENING_PAREN, SolomonParserOPENING_BRACKET, SolomonParserPLUS, SolomonParserMINUS, SolomonParserNOT, SolomonParserIDENT_WITH_DOTS, SolomonParserIDENT, SolomonParserDURATION, SolomonParserNUMBER, SolomonParserSTRING:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(99)
			p.Anonymous()
		}

	case SolomonParserKW_LET:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(100)
			p.Assignment()
		}

	default:
		p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
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

// IAnonymousContext is an interface to support dynamic dispatch.
type IAnonymousContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Expression() IExpressionContext
	SEMICOLON() antlr.TerminalNode

	// IsAnonymousContext differentiates from other interfaces.
	IsAnonymousContext()
}

type AnonymousContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyAnonymousContext() *AnonymousContext {
	var p = new(AnonymousContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_anonymous
	return p
}

func InitEmptyAnonymousContext(p *AnonymousContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_anonymous
}

func (*AnonymousContext) IsAnonymousContext() {}

func NewAnonymousContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *AnonymousContext {
	var p = new(AnonymousContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_anonymous

	return p
}

func (s *AnonymousContext) GetParser() antlr.Parser { return s.parser }

func (s *AnonymousContext) Expression() IExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExpressionContext)
}

func (s *AnonymousContext) SEMICOLON() antlr.TerminalNode {
	return s.GetToken(SolomonParserSEMICOLON, 0)
}

func (s *AnonymousContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *AnonymousContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *AnonymousContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterAnonymous(s)
	}
}

func (s *AnonymousContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitAnonymous(s)
	}
}

func (p *SolomonParser) Anonymous() (localctx IAnonymousContext) {
	localctx = NewAnonymousContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 10, SolomonParserRULE_anonymous)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(103)
		p.Expression()
	}
	p.SetState(105)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == SolomonParserSEMICOLON {
		{
			p.SetState(104)
			p.Match(SolomonParserSEMICOLON)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
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

// IAssignmentContext is an interface to support dynamic dispatch.
type IAssignmentContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	KW_LET() antlr.TerminalNode
	IDENT() antlr.TerminalNode
	ASSIGNMENT() antlr.TerminalNode
	Expression() IExpressionContext
	SEMICOLON() antlr.TerminalNode

	// IsAssignmentContext differentiates from other interfaces.
	IsAssignmentContext()
}

type AssignmentContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyAssignmentContext() *AssignmentContext {
	var p = new(AssignmentContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_assignment
	return p
}

func InitEmptyAssignmentContext(p *AssignmentContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_assignment
}

func (*AssignmentContext) IsAssignmentContext() {}

func NewAssignmentContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *AssignmentContext {
	var p = new(AssignmentContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_assignment

	return p
}

func (s *AssignmentContext) GetParser() antlr.Parser { return s.parser }

func (s *AssignmentContext) KW_LET() antlr.TerminalNode {
	return s.GetToken(SolomonParserKW_LET, 0)
}

func (s *AssignmentContext) IDENT() antlr.TerminalNode {
	return s.GetToken(SolomonParserIDENT, 0)
}

func (s *AssignmentContext) ASSIGNMENT() antlr.TerminalNode {
	return s.GetToken(SolomonParserASSIGNMENT, 0)
}

func (s *AssignmentContext) Expression() IExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExpressionContext)
}

func (s *AssignmentContext) SEMICOLON() antlr.TerminalNode {
	return s.GetToken(SolomonParserSEMICOLON, 0)
}

func (s *AssignmentContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *AssignmentContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *AssignmentContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterAssignment(s)
	}
}

func (s *AssignmentContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitAssignment(s)
	}
}

func (p *SolomonParser) Assignment() (localctx IAssignmentContext) {
	localctx = NewAssignmentContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 12, SolomonParserRULE_assignment)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(107)
		p.Match(SolomonParserKW_LET)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(108)
		p.Match(SolomonParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(109)
		p.Match(SolomonParserASSIGNMENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(110)
		p.Expression()
	}
	p.SetState(112)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == SolomonParserSEMICOLON {
		{
			p.SetState(111)
			p.Match(SolomonParserSEMICOLON)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
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

// IUseContext is an interface to support dynamic dispatch.
type IUseContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENT() antlr.TerminalNode
	SolomonSelectors() ISolomonSelectorsContext
	SEMICOLON() antlr.TerminalNode

	// IsUseContext differentiates from other interfaces.
	IsUseContext()
}

type UseContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyUseContext() *UseContext {
	var p = new(UseContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_use
	return p
}

func InitEmptyUseContext(p *UseContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_use
}

func (*UseContext) IsUseContext() {}

func NewUseContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *UseContext {
	var p = new(UseContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_use

	return p
}

func (s *UseContext) GetParser() antlr.Parser { return s.parser }

func (s *UseContext) IDENT() antlr.TerminalNode {
	return s.GetToken(SolomonParserIDENT, 0)
}

func (s *UseContext) SolomonSelectors() ISolomonSelectorsContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISolomonSelectorsContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISolomonSelectorsContext)
}

func (s *UseContext) SEMICOLON() antlr.TerminalNode {
	return s.GetToken(SolomonParserSEMICOLON, 0)
}

func (s *UseContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *UseContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *UseContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterUse(s)
	}
}

func (s *UseContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitUse(s)
	}
}

func (p *SolomonParser) Use() (localctx IUseContext) {
	localctx = NewUseContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 14, SolomonParserRULE_use)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(114)
		p.Match(SolomonParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(115)
		p.SolomonSelectors()
	}
	p.SetState(117)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == SolomonParserSEMICOLON {
		{
			p.SetState(116)
			p.Match(SolomonParserSEMICOLON)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
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

// IExpressionContext is an interface to support dynamic dispatch.
type IExpressionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Lambda() ILambdaContext
	ExprPipe() IExprPipeContext

	// IsExpressionContext differentiates from other interfaces.
	IsExpressionContext()
}

type ExpressionContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyExpressionContext() *ExpressionContext {
	var p = new(ExpressionContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_expression
	return p
}

func InitEmptyExpressionContext(p *ExpressionContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_expression
}

func (*ExpressionContext) IsExpressionContext() {}

func NewExpressionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ExpressionContext {
	var p = new(ExpressionContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_expression

	return p
}

func (s *ExpressionContext) GetParser() antlr.Parser { return s.parser }

func (s *ExpressionContext) Lambda() ILambdaContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ILambdaContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ILambdaContext)
}

func (s *ExpressionContext) ExprPipe() IExprPipeContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExprPipeContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExprPipeContext)
}

func (s *ExpressionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExpressionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ExpressionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterExpression(s)
	}
}

func (s *ExpressionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitExpression(s)
	}
}

func (p *SolomonParser) Expression() (localctx IExpressionContext) {
	localctx = NewExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 16, SolomonParserRULE_expression)
	p.SetState(121)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 8, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(119)
			p.Lambda()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(120)
			p.ExprPipe()
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

// IExprPipeContext is an interface to support dynamic dispatch.
type IExprPipeContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	ExprTernary() IExprTernaryContext
	AllPIPE() []antlr.TerminalNode
	PIPE(i int) antlr.TerminalNode
	AllCallWithBy() []ICallWithByContext
	CallWithBy(i int) ICallWithByContext

	// IsExprPipeContext differentiates from other interfaces.
	IsExprPipeContext()
}

type ExprPipeContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyExprPipeContext() *ExprPipeContext {
	var p = new(ExprPipeContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_exprPipe
	return p
}

func InitEmptyExprPipeContext(p *ExprPipeContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_exprPipe
}

func (*ExprPipeContext) IsExprPipeContext() {}

func NewExprPipeContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ExprPipeContext {
	var p = new(ExprPipeContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_exprPipe

	return p
}

func (s *ExprPipeContext) GetParser() antlr.Parser { return s.parser }

func (s *ExprPipeContext) ExprTernary() IExprTernaryContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExprTernaryContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExprTernaryContext)
}

func (s *ExprPipeContext) AllPIPE() []antlr.TerminalNode {
	return s.GetTokens(SolomonParserPIPE)
}

func (s *ExprPipeContext) PIPE(i int) antlr.TerminalNode {
	return s.GetToken(SolomonParserPIPE, i)
}

func (s *ExprPipeContext) AllCallWithBy() []ICallWithByContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(ICallWithByContext); ok {
			len++
		}
	}

	tst := make([]ICallWithByContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(ICallWithByContext); ok {
			tst[i] = t.(ICallWithByContext)
			i++
		}
	}

	return tst
}

func (s *ExprPipeContext) CallWithBy(i int) ICallWithByContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ICallWithByContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(ICallWithByContext)
}

func (s *ExprPipeContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExprPipeContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ExprPipeContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterExprPipe(s)
	}
}

func (s *ExprPipeContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitExprPipe(s)
	}
}

func (p *SolomonParser) ExprPipe() (localctx IExprPipeContext) {
	localctx = NewExprPipeContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 18, SolomonParserRULE_exprPipe)
	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(123)
		p.ExprTernary()
	}
	p.SetState(128)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 9, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(124)
				p.Match(SolomonParserPIPE)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(125)
				p.CallWithBy()
			}

		}
		p.SetState(130)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 9, p.GetParserRuleContext())
		if p.HasError() {
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

// IExprTernaryContext is an interface to support dynamic dispatch.
type IExprTernaryContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllExprOr() []IExprOrContext
	ExprOr(i int) IExprOrContext
	QUESTION() antlr.TerminalNode
	COLON() antlr.TerminalNode

	// IsExprTernaryContext differentiates from other interfaces.
	IsExprTernaryContext()
}

type ExprTernaryContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyExprTernaryContext() *ExprTernaryContext {
	var p = new(ExprTernaryContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_exprTernary
	return p
}

func InitEmptyExprTernaryContext(p *ExprTernaryContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_exprTernary
}

func (*ExprTernaryContext) IsExprTernaryContext() {}

func NewExprTernaryContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ExprTernaryContext {
	var p = new(ExprTernaryContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_exprTernary

	return p
}

func (s *ExprTernaryContext) GetParser() antlr.Parser { return s.parser }

func (s *ExprTernaryContext) AllExprOr() []IExprOrContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IExprOrContext); ok {
			len++
		}
	}

	tst := make([]IExprOrContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IExprOrContext); ok {
			tst[i] = t.(IExprOrContext)
			i++
		}
	}

	return tst
}

func (s *ExprTernaryContext) ExprOr(i int) IExprOrContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExprOrContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExprOrContext)
}

func (s *ExprTernaryContext) QUESTION() antlr.TerminalNode {
	return s.GetToken(SolomonParserQUESTION, 0)
}

func (s *ExprTernaryContext) COLON() antlr.TerminalNode {
	return s.GetToken(SolomonParserCOLON, 0)
}

func (s *ExprTernaryContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExprTernaryContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ExprTernaryContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterExprTernary(s)
	}
}

func (s *ExprTernaryContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitExprTernary(s)
	}
}

func (p *SolomonParser) ExprTernary() (localctx IExprTernaryContext) {
	localctx = NewExprTernaryContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 20, SolomonParserRULE_exprTernary)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(131)
		p.ExprOr()
	}
	p.SetState(137)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 10, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(132)
			p.Match(SolomonParserQUESTION)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(133)
			p.ExprOr()
		}
		{
			p.SetState(134)
			p.Match(SolomonParserCOLON)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(135)
			p.ExprOr()
		}

	} else if p.HasError() { // JIM
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

// ILambdaContext is an interface to support dynamic dispatch.
type ILambdaContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENT() antlr.TerminalNode
	ARROW() antlr.TerminalNode
	Expression() IExpressionContext
	OPENING_PAREN() antlr.TerminalNode
	Arglist() IArglistContext
	CLOSING_PAREN() antlr.TerminalNode

	// IsLambdaContext differentiates from other interfaces.
	IsLambdaContext()
}

type LambdaContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyLambdaContext() *LambdaContext {
	var p = new(LambdaContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_lambda
	return p
}

func InitEmptyLambdaContext(p *LambdaContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_lambda
}

func (*LambdaContext) IsLambdaContext() {}

func NewLambdaContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *LambdaContext {
	var p = new(LambdaContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_lambda

	return p
}

func (s *LambdaContext) GetParser() antlr.Parser { return s.parser }

func (s *LambdaContext) IDENT() antlr.TerminalNode {
	return s.GetToken(SolomonParserIDENT, 0)
}

func (s *LambdaContext) ARROW() antlr.TerminalNode {
	return s.GetToken(SolomonParserARROW, 0)
}

func (s *LambdaContext) Expression() IExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExpressionContext)
}

func (s *LambdaContext) OPENING_PAREN() antlr.TerminalNode {
	return s.GetToken(SolomonParserOPENING_PAREN, 0)
}

func (s *LambdaContext) Arglist() IArglistContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IArglistContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IArglistContext)
}

func (s *LambdaContext) CLOSING_PAREN() antlr.TerminalNode {
	return s.GetToken(SolomonParserCLOSING_PAREN, 0)
}

func (s *LambdaContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *LambdaContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *LambdaContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterLambda(s)
	}
}

func (s *LambdaContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitLambda(s)
	}
}

func (p *SolomonParser) Lambda() (localctx ILambdaContext) {
	localctx = NewLambdaContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 22, SolomonParserRULE_lambda)
	p.SetState(148)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case SolomonParserIDENT:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(139)
			p.Match(SolomonParserIDENT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(140)
			p.Match(SolomonParserARROW)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(141)
			p.Expression()
		}

	case SolomonParserOPENING_PAREN:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(142)
			p.Match(SolomonParserOPENING_PAREN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(143)
			p.Arglist()
		}
		{
			p.SetState(144)
			p.Match(SolomonParserCLOSING_PAREN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(145)
			p.Match(SolomonParserARROW)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(146)
			p.Expression()
		}

	default:
		p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
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

// IArglistContext is an interface to support dynamic dispatch.
type IArglistContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllIDENT() []antlr.TerminalNode
	IDENT(i int) antlr.TerminalNode
	AllCOMMA() []antlr.TerminalNode
	COMMA(i int) antlr.TerminalNode

	// IsArglistContext differentiates from other interfaces.
	IsArglistContext()
}

type ArglistContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyArglistContext() *ArglistContext {
	var p = new(ArglistContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_arglist
	return p
}

func InitEmptyArglistContext(p *ArglistContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_arglist
}

func (*ArglistContext) IsArglistContext() {}

func NewArglistContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ArglistContext {
	var p = new(ArglistContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_arglist

	return p
}

func (s *ArglistContext) GetParser() antlr.Parser { return s.parser }

func (s *ArglistContext) AllIDENT() []antlr.TerminalNode {
	return s.GetTokens(SolomonParserIDENT)
}

func (s *ArglistContext) IDENT(i int) antlr.TerminalNode {
	return s.GetToken(SolomonParserIDENT, i)
}

func (s *ArglistContext) AllCOMMA() []antlr.TerminalNode {
	return s.GetTokens(SolomonParserCOMMA)
}

func (s *ArglistContext) COMMA(i int) antlr.TerminalNode {
	return s.GetToken(SolomonParserCOMMA, i)
}

func (s *ArglistContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ArglistContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ArglistContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterArglist(s)
	}
}

func (s *ArglistContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitArglist(s)
	}
}

func (p *SolomonParser) Arglist() (localctx IArglistContext) {
	localctx = NewArglistContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 24, SolomonParserRULE_arglist)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(150)
		p.Match(SolomonParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(155)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == SolomonParserCOMMA {
		{
			p.SetState(151)
			p.Match(SolomonParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(152)
			p.Match(SolomonParserIDENT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

		p.SetState(157)
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

// IExprOrContext is an interface to support dynamic dispatch.
type IExprOrContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllExprAnd() []IExprAndContext
	ExprAnd(i int) IExprAndContext
	AllOR() []antlr.TerminalNode
	OR(i int) antlr.TerminalNode

	// IsExprOrContext differentiates from other interfaces.
	IsExprOrContext()
}

type ExprOrContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyExprOrContext() *ExprOrContext {
	var p = new(ExprOrContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_exprOr
	return p
}

func InitEmptyExprOrContext(p *ExprOrContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_exprOr
}

func (*ExprOrContext) IsExprOrContext() {}

func NewExprOrContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ExprOrContext {
	var p = new(ExprOrContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_exprOr

	return p
}

func (s *ExprOrContext) GetParser() antlr.Parser { return s.parser }

func (s *ExprOrContext) AllExprAnd() []IExprAndContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IExprAndContext); ok {
			len++
		}
	}

	tst := make([]IExprAndContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IExprAndContext); ok {
			tst[i] = t.(IExprAndContext)
			i++
		}
	}

	return tst
}

func (s *ExprOrContext) ExprAnd(i int) IExprAndContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExprAndContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExprAndContext)
}

func (s *ExprOrContext) AllOR() []antlr.TerminalNode {
	return s.GetTokens(SolomonParserOR)
}

func (s *ExprOrContext) OR(i int) antlr.TerminalNode {
	return s.GetToken(SolomonParserOR, i)
}

func (s *ExprOrContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExprOrContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ExprOrContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterExprOr(s)
	}
}

func (s *ExprOrContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitExprOr(s)
	}
}

func (p *SolomonParser) ExprOr() (localctx IExprOrContext) {
	localctx = NewExprOrContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 26, SolomonParserRULE_exprOr)
	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(158)
		p.ExprAnd()
	}
	p.SetState(163)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 13, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(159)
				p.Match(SolomonParserOR)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(160)
				p.ExprAnd()
			}

		}
		p.SetState(165)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 13, p.GetParserRuleContext())
		if p.HasError() {
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

// IExprAndContext is an interface to support dynamic dispatch.
type IExprAndContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllExprNot() []IExprNotContext
	ExprNot(i int) IExprNotContext
	AllAND() []antlr.TerminalNode
	AND(i int) antlr.TerminalNode

	// IsExprAndContext differentiates from other interfaces.
	IsExprAndContext()
}

type ExprAndContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyExprAndContext() *ExprAndContext {
	var p = new(ExprAndContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_exprAnd
	return p
}

func InitEmptyExprAndContext(p *ExprAndContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_exprAnd
}

func (*ExprAndContext) IsExprAndContext() {}

func NewExprAndContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ExprAndContext {
	var p = new(ExprAndContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_exprAnd

	return p
}

func (s *ExprAndContext) GetParser() antlr.Parser { return s.parser }

func (s *ExprAndContext) AllExprNot() []IExprNotContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IExprNotContext); ok {
			len++
		}
	}

	tst := make([]IExprNotContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IExprNotContext); ok {
			tst[i] = t.(IExprNotContext)
			i++
		}
	}

	return tst
}

func (s *ExprAndContext) ExprNot(i int) IExprNotContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExprNotContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExprNotContext)
}

func (s *ExprAndContext) AllAND() []antlr.TerminalNode {
	return s.GetTokens(SolomonParserAND)
}

func (s *ExprAndContext) AND(i int) antlr.TerminalNode {
	return s.GetToken(SolomonParserAND, i)
}

func (s *ExprAndContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExprAndContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ExprAndContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterExprAnd(s)
	}
}

func (s *ExprAndContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitExprAnd(s)
	}
}

func (p *SolomonParser) ExprAnd() (localctx IExprAndContext) {
	localctx = NewExprAndContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 28, SolomonParserRULE_exprAnd)
	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(166)
		p.ExprNot()
	}
	p.SetState(171)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 14, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(167)
				p.Match(SolomonParserAND)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(168)
				p.ExprNot()
			}

		}
		p.SetState(173)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 14, p.GetParserRuleContext())
		if p.HasError() {
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

// IExprNotContext is an interface to support dynamic dispatch.
type IExprNotContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	ExprComp() IExprCompContext
	NOT() antlr.TerminalNode

	// IsExprNotContext differentiates from other interfaces.
	IsExprNotContext()
}

type ExprNotContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyExprNotContext() *ExprNotContext {
	var p = new(ExprNotContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_exprNot
	return p
}

func InitEmptyExprNotContext(p *ExprNotContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_exprNot
}

func (*ExprNotContext) IsExprNotContext() {}

func NewExprNotContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ExprNotContext {
	var p = new(ExprNotContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_exprNot

	return p
}

func (s *ExprNotContext) GetParser() antlr.Parser { return s.parser }

func (s *ExprNotContext) ExprComp() IExprCompContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExprCompContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExprCompContext)
}

func (s *ExprNotContext) NOT() antlr.TerminalNode {
	return s.GetToken(SolomonParserNOT, 0)
}

func (s *ExprNotContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExprNotContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ExprNotContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterExprNot(s)
	}
}

func (s *ExprNotContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitExprNot(s)
	}
}

func (p *SolomonParser) ExprNot() (localctx IExprNotContext) {
	localctx = NewExprNotContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 30, SolomonParserRULE_exprNot)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(175)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == SolomonParserNOT {
		{
			p.SetState(174)
			p.Match(SolomonParserNOT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}
	{
		p.SetState(177)
		p.ExprComp()
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

// IExprCompContext is an interface to support dynamic dispatch.
type IExprCompContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllExprArith() []IExprArithContext
	ExprArith(i int) IExprArithContext
	AllLT() []antlr.TerminalNode
	LT(i int) antlr.TerminalNode
	AllGT() []antlr.TerminalNode
	GT(i int) antlr.TerminalNode
	AllLE() []antlr.TerminalNode
	LE(i int) antlr.TerminalNode
	AllGE() []antlr.TerminalNode
	GE(i int) antlr.TerminalNode
	AllEQ() []antlr.TerminalNode
	EQ(i int) antlr.TerminalNode
	AllNE() []antlr.TerminalNode
	NE(i int) antlr.TerminalNode

	// IsExprCompContext differentiates from other interfaces.
	IsExprCompContext()
}

type ExprCompContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyExprCompContext() *ExprCompContext {
	var p = new(ExprCompContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_exprComp
	return p
}

func InitEmptyExprCompContext(p *ExprCompContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_exprComp
}

func (*ExprCompContext) IsExprCompContext() {}

func NewExprCompContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ExprCompContext {
	var p = new(ExprCompContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_exprComp

	return p
}

func (s *ExprCompContext) GetParser() antlr.Parser { return s.parser }

func (s *ExprCompContext) AllExprArith() []IExprArithContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IExprArithContext); ok {
			len++
		}
	}

	tst := make([]IExprArithContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IExprArithContext); ok {
			tst[i] = t.(IExprArithContext)
			i++
		}
	}

	return tst
}

func (s *ExprCompContext) ExprArith(i int) IExprArithContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExprArithContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExprArithContext)
}

func (s *ExprCompContext) AllLT() []antlr.TerminalNode {
	return s.GetTokens(SolomonParserLT)
}

func (s *ExprCompContext) LT(i int) antlr.TerminalNode {
	return s.GetToken(SolomonParserLT, i)
}

func (s *ExprCompContext) AllGT() []antlr.TerminalNode {
	return s.GetTokens(SolomonParserGT)
}

func (s *ExprCompContext) GT(i int) antlr.TerminalNode {
	return s.GetToken(SolomonParserGT, i)
}

func (s *ExprCompContext) AllLE() []antlr.TerminalNode {
	return s.GetTokens(SolomonParserLE)
}

func (s *ExprCompContext) LE(i int) antlr.TerminalNode {
	return s.GetToken(SolomonParserLE, i)
}

func (s *ExprCompContext) AllGE() []antlr.TerminalNode {
	return s.GetTokens(SolomonParserGE)
}

func (s *ExprCompContext) GE(i int) antlr.TerminalNode {
	return s.GetToken(SolomonParserGE, i)
}

func (s *ExprCompContext) AllEQ() []antlr.TerminalNode {
	return s.GetTokens(SolomonParserEQ)
}

func (s *ExprCompContext) EQ(i int) antlr.TerminalNode {
	return s.GetToken(SolomonParserEQ, i)
}

func (s *ExprCompContext) AllNE() []antlr.TerminalNode {
	return s.GetTokens(SolomonParserNE)
}

func (s *ExprCompContext) NE(i int) antlr.TerminalNode {
	return s.GetToken(SolomonParserNE, i)
}

func (s *ExprCompContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExprCompContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ExprCompContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterExprComp(s)
	}
}

func (s *ExprCompContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitExprComp(s)
	}
}

func (p *SolomonParser) ExprComp() (localctx IExprCompContext) {
	localctx = NewExprCompContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 32, SolomonParserRULE_exprComp)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(179)
		p.ExprArith()
	}
	p.SetState(184)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 16, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(180)
				_la = p.GetTokenStream().LA(1)

				if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&33030144) != 0) {
					p.GetErrorHandler().RecoverInline(p)
				} else {
					p.GetErrorHandler().ReportMatch(p)
					p.Consume()
				}
			}
			{
				p.SetState(181)
				p.ExprArith()
			}

		}
		p.SetState(186)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 16, p.GetParserRuleContext())
		if p.HasError() {
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

// IExprArithContext is an interface to support dynamic dispatch.
type IExprArithContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllExprTerm() []IExprTermContext
	ExprTerm(i int) IExprTermContext
	AllPLUS() []antlr.TerminalNode
	PLUS(i int) antlr.TerminalNode
	AllMINUS() []antlr.TerminalNode
	MINUS(i int) antlr.TerminalNode

	// IsExprArithContext differentiates from other interfaces.
	IsExprArithContext()
}

type ExprArithContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyExprArithContext() *ExprArithContext {
	var p = new(ExprArithContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_exprArith
	return p
}

func InitEmptyExprArithContext(p *ExprArithContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_exprArith
}

func (*ExprArithContext) IsExprArithContext() {}

func NewExprArithContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ExprArithContext {
	var p = new(ExprArithContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_exprArith

	return p
}

func (s *ExprArithContext) GetParser() antlr.Parser { return s.parser }

func (s *ExprArithContext) AllExprTerm() []IExprTermContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IExprTermContext); ok {
			len++
		}
	}

	tst := make([]IExprTermContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IExprTermContext); ok {
			tst[i] = t.(IExprTermContext)
			i++
		}
	}

	return tst
}

func (s *ExprArithContext) ExprTerm(i int) IExprTermContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExprTermContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExprTermContext)
}

func (s *ExprArithContext) AllPLUS() []antlr.TerminalNode {
	return s.GetTokens(SolomonParserPLUS)
}

func (s *ExprArithContext) PLUS(i int) antlr.TerminalNode {
	return s.GetToken(SolomonParserPLUS, i)
}

func (s *ExprArithContext) AllMINUS() []antlr.TerminalNode {
	return s.GetTokens(SolomonParserMINUS)
}

func (s *ExprArithContext) MINUS(i int) antlr.TerminalNode {
	return s.GetToken(SolomonParserMINUS, i)
}

func (s *ExprArithContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExprArithContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ExprArithContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterExprArith(s)
	}
}

func (s *ExprArithContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitExprArith(s)
	}
}

func (p *SolomonParser) ExprArith() (localctx IExprArithContext) {
	localctx = NewExprArithContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 34, SolomonParserRULE_exprArith)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(187)
		p.ExprTerm()
	}
	p.SetState(192)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 17, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(188)
				_la = p.GetTokenStream().LA(1)

				if !(_la == SolomonParserPLUS || _la == SolomonParserMINUS) {
					p.GetErrorHandler().RecoverInline(p)
				} else {
					p.GetErrorHandler().ReportMatch(p)
					p.Consume()
				}
			}
			{
				p.SetState(189)
				p.ExprTerm()
			}

		}
		p.SetState(194)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 17, p.GetParserRuleContext())
		if p.HasError() {
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

// IExprTermContext is an interface to support dynamic dispatch.
type IExprTermContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllExprUnary() []IExprUnaryContext
	ExprUnary(i int) IExprUnaryContext
	AllDIV() []antlr.TerminalNode
	DIV(i int) antlr.TerminalNode
	AllMUL() []antlr.TerminalNode
	MUL(i int) antlr.TerminalNode

	// IsExprTermContext differentiates from other interfaces.
	IsExprTermContext()
}

type ExprTermContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyExprTermContext() *ExprTermContext {
	var p = new(ExprTermContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_exprTerm
	return p
}

func InitEmptyExprTermContext(p *ExprTermContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_exprTerm
}

func (*ExprTermContext) IsExprTermContext() {}

func NewExprTermContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ExprTermContext {
	var p = new(ExprTermContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_exprTerm

	return p
}

func (s *ExprTermContext) GetParser() antlr.Parser { return s.parser }

func (s *ExprTermContext) AllExprUnary() []IExprUnaryContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IExprUnaryContext); ok {
			len++
		}
	}

	tst := make([]IExprUnaryContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IExprUnaryContext); ok {
			tst[i] = t.(IExprUnaryContext)
			i++
		}
	}

	return tst
}

func (s *ExprTermContext) ExprUnary(i int) IExprUnaryContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExprUnaryContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExprUnaryContext)
}

func (s *ExprTermContext) AllDIV() []antlr.TerminalNode {
	return s.GetTokens(SolomonParserDIV)
}

func (s *ExprTermContext) DIV(i int) antlr.TerminalNode {
	return s.GetToken(SolomonParserDIV, i)
}

func (s *ExprTermContext) AllMUL() []antlr.TerminalNode {
	return s.GetTokens(SolomonParserMUL)
}

func (s *ExprTermContext) MUL(i int) antlr.TerminalNode {
	return s.GetToken(SolomonParserMUL, i)
}

func (s *ExprTermContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExprTermContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ExprTermContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterExprTerm(s)
	}
}

func (s *ExprTermContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitExprTerm(s)
	}
}

func (p *SolomonParser) ExprTerm() (localctx IExprTermContext) {
	localctx = NewExprTermContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 36, SolomonParserRULE_exprTerm)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(195)
		p.ExprUnary()
	}
	p.SetState(200)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 18, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(196)
				_la = p.GetTokenStream().LA(1)

				if !(_la == SolomonParserDIV || _la == SolomonParserMUL) {
					p.GetErrorHandler().RecoverInline(p)
				} else {
					p.GetErrorHandler().ReportMatch(p)
					p.Consume()
				}
			}
			{
				p.SetState(197)
				p.ExprUnary()
			}

		}
		p.SetState(202)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 18, p.GetParserRuleContext())
		if p.HasError() {
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

// IExprUnaryContext is an interface to support dynamic dispatch.
type IExprUnaryContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Atom() IAtomContext
	PLUS() antlr.TerminalNode
	MINUS() antlr.TerminalNode

	// IsExprUnaryContext differentiates from other interfaces.
	IsExprUnaryContext()
}

type ExprUnaryContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyExprUnaryContext() *ExprUnaryContext {
	var p = new(ExprUnaryContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_exprUnary
	return p
}

func InitEmptyExprUnaryContext(p *ExprUnaryContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_exprUnary
}

func (*ExprUnaryContext) IsExprUnaryContext() {}

func NewExprUnaryContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ExprUnaryContext {
	var p = new(ExprUnaryContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_exprUnary

	return p
}

func (s *ExprUnaryContext) GetParser() antlr.Parser { return s.parser }

func (s *ExprUnaryContext) Atom() IAtomContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IAtomContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IAtomContext)
}

func (s *ExprUnaryContext) PLUS() antlr.TerminalNode {
	return s.GetToken(SolomonParserPLUS, 0)
}

func (s *ExprUnaryContext) MINUS() antlr.TerminalNode {
	return s.GetToken(SolomonParserMINUS, 0)
}

func (s *ExprUnaryContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExprUnaryContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ExprUnaryContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterExprUnary(s)
	}
}

func (s *ExprUnaryContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitExprUnary(s)
	}
}

func (p *SolomonParser) ExprUnary() (localctx IExprUnaryContext) {
	localctx = NewExprUnaryContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 38, SolomonParserRULE_exprUnary)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(204)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == SolomonParserPLUS || _la == SolomonParserMINUS {
		{
			p.SetState(203)
			_la = p.GetTokenStream().LA(1)

			if !(_la == SolomonParserPLUS || _la == SolomonParserMINUS) {
				p.GetErrorHandler().RecoverInline(p)
			} else {
				p.GetErrorHandler().ReportMatch(p)
				p.Consume()
			}
		}

	}
	{
		p.SetState(206)
		p.Atom()
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

// IAtomContext is an interface to support dynamic dispatch.
type IAtomContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser
	// IsAtomContext differentiates from other interfaces.
	IsAtomContext()
}

type AtomContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyAtomContext() *AtomContext {
	var p = new(AtomContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_atom
	return p
}

func InitEmptyAtomContext(p *AtomContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_atom
}

func (*AtomContext) IsAtomContext() {}

func NewAtomContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *AtomContext {
	var p = new(AtomContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_atom

	return p
}

func (s *AtomContext) GetParser() antlr.Parser { return s.parser }

func (s *AtomContext) CopyAll(ctx *AtomContext) {
	s.CopyFrom(&ctx.BaseParserRuleContext)
}

func (s *AtomContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *AtomContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

type AtomDurationContext struct {
	AtomContext
}

func NewAtomDurationContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *AtomDurationContext {
	var p = new(AtomDurationContext)

	InitEmptyAtomContext(&p.AtomContext)
	p.parser = parser
	p.CopyAll(ctx.(*AtomContext))

	return p
}

func (s *AtomDurationContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *AtomDurationContext) DURATION() antlr.TerminalNode {
	return s.GetToken(SolomonParserDURATION, 0)
}

func (s *AtomDurationContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterAtomDuration(s)
	}
}

func (s *AtomDurationContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitAtomDuration(s)
	}
}

type AtomExpressionInParenthesesContext struct {
	AtomContext
}

func NewAtomExpressionInParenthesesContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *AtomExpressionInParenthesesContext {
	var p = new(AtomExpressionInParenthesesContext)

	InitEmptyAtomContext(&p.AtomContext)
	p.parser = parser
	p.CopyAll(ctx.(*AtomContext))

	return p
}

func (s *AtomExpressionInParenthesesContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *AtomExpressionInParenthesesContext) OPENING_PAREN() antlr.TerminalNode {
	return s.GetToken(SolomonParserOPENING_PAREN, 0)
}

func (s *AtomExpressionInParenthesesContext) Expression() IExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExpressionContext)
}

func (s *AtomExpressionInParenthesesContext) CLOSING_PAREN() antlr.TerminalNode {
	return s.GetToken(SolomonParserCLOSING_PAREN, 0)
}

func (s *AtomExpressionInParenthesesContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterAtomExpressionInParentheses(s)
	}
}

func (s *AtomExpressionInParenthesesContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitAtomExpressionInParentheses(s)
	}
}

type AtomLambdaContext struct {
	AtomContext
}

func NewAtomLambdaContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *AtomLambdaContext {
	var p = new(AtomLambdaContext)

	InitEmptyAtomContext(&p.AtomContext)
	p.parser = parser
	p.CopyAll(ctx.(*AtomContext))

	return p
}

func (s *AtomLambdaContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *AtomLambdaContext) Lambda() ILambdaContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ILambdaContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ILambdaContext)
}

func (s *AtomLambdaContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterAtomLambda(s)
	}
}

func (s *AtomLambdaContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitAtomLambda(s)
	}
}

type AtomVectorContext struct {
	AtomContext
}

func NewAtomVectorContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *AtomVectorContext {
	var p = new(AtomVectorContext)

	InitEmptyAtomContext(&p.AtomContext)
	p.parser = parser
	p.CopyAll(ctx.(*AtomContext))

	return p
}

func (s *AtomVectorContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *AtomVectorContext) OPENING_BRACKET() antlr.TerminalNode {
	return s.GetToken(SolomonParserOPENING_BRACKET, 0)
}

func (s *AtomVectorContext) Sequence() ISequenceContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISequenceContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISequenceContext)
}

func (s *AtomVectorContext) CLOSING_BRACKET() antlr.TerminalNode {
	return s.GetToken(SolomonParserCLOSING_BRACKET, 0)
}

func (s *AtomVectorContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterAtomVector(s)
	}
}

func (s *AtomVectorContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitAtomVector(s)
	}
}

type AtomNumberContext struct {
	AtomContext
}

func NewAtomNumberContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *AtomNumberContext {
	var p = new(AtomNumberContext)

	InitEmptyAtomContext(&p.AtomContext)
	p.parser = parser
	p.CopyAll(ctx.(*AtomContext))

	return p
}

func (s *AtomNumberContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *AtomNumberContext) NUMBER() antlr.TerminalNode {
	return s.GetToken(SolomonParserNUMBER, 0)
}

func (s *AtomNumberContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterAtomNumber(s)
	}
}

func (s *AtomNumberContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitAtomNumber(s)
	}
}

type AtomCallWithByContext struct {
	AtomContext
}

func NewAtomCallWithByContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *AtomCallWithByContext {
	var p = new(AtomCallWithByContext)

	InitEmptyAtomContext(&p.AtomContext)
	p.parser = parser
	p.CopyAll(ctx.(*AtomContext))

	return p
}

func (s *AtomCallWithByContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *AtomCallWithByContext) CallWithBy() ICallWithByContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ICallWithByContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ICallWithByContext)
}

func (s *AtomCallWithByContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterAtomCallWithBy(s)
	}
}

func (s *AtomCallWithByContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitAtomCallWithBy(s)
	}
}

type AtomStringContext struct {
	AtomContext
}

func NewAtomStringContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *AtomStringContext {
	var p = new(AtomStringContext)

	InitEmptyAtomContext(&p.AtomContext)
	p.parser = parser
	p.CopyAll(ctx.(*AtomContext))

	return p
}

func (s *AtomStringContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *AtomStringContext) STRING() antlr.TerminalNode {
	return s.GetToken(SolomonParserSTRING, 0)
}

func (s *AtomStringContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterAtomString(s)
	}
}

func (s *AtomStringContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitAtomString(s)
	}
}

type AtomSelectorsContext struct {
	AtomContext
}

func NewAtomSelectorsContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *AtomSelectorsContext {
	var p = new(AtomSelectorsContext)

	InitEmptyAtomContext(&p.AtomContext)
	p.parser = parser
	p.CopyAll(ctx.(*AtomContext))

	return p
}

func (s *AtomSelectorsContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *AtomSelectorsContext) SolomonSelectors() ISolomonSelectorsContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISolomonSelectorsContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISolomonSelectorsContext)
}

func (s *AtomSelectorsContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterAtomSelectors(s)
	}
}

func (s *AtomSelectorsContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitAtomSelectors(s)
	}
}

type AtomIdentContext struct {
	AtomContext
}

func NewAtomIdentContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *AtomIdentContext {
	var p = new(AtomIdentContext)

	InitEmptyAtomContext(&p.AtomContext)
	p.parser = parser
	p.CopyAll(ctx.(*AtomContext))

	return p
}

func (s *AtomIdentContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *AtomIdentContext) IDENT() antlr.TerminalNode {
	return s.GetToken(SolomonParserIDENT, 0)
}

func (s *AtomIdentContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterAtomIdent(s)
	}
}

func (s *AtomIdentContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitAtomIdent(s)
	}
}

func (p *SolomonParser) Atom() (localctx IAtomContext) {
	localctx = NewAtomContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 40, SolomonParserRULE_atom)
	p.SetState(223)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 20, p.GetParserRuleContext()) {
	case 1:
		localctx = NewAtomExpressionInParenthesesContext(p, localctx)
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(208)
			p.Match(SolomonParserOPENING_PAREN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(209)
			p.Expression()
		}
		{
			p.SetState(210)
			p.Match(SolomonParserCLOSING_PAREN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 2:
		localctx = NewAtomVectorContext(p, localctx)
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(212)
			p.Match(SolomonParserOPENING_BRACKET)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(213)
			p.Sequence()
		}
		{
			p.SetState(214)
			p.Match(SolomonParserCLOSING_BRACKET)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 3:
		localctx = NewAtomSelectorsContext(p, localctx)
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(216)
			p.SolomonSelectors()
		}

	case 4:
		localctx = NewAtomDurationContext(p, localctx)
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(217)
			p.Match(SolomonParserDURATION)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 5:
		localctx = NewAtomNumberContext(p, localctx)
		p.EnterOuterAlt(localctx, 5)
		{
			p.SetState(218)
			p.Match(SolomonParserNUMBER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 6:
		localctx = NewAtomStringContext(p, localctx)
		p.EnterOuterAlt(localctx, 6)
		{
			p.SetState(219)
			p.Match(SolomonParserSTRING)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 7:
		localctx = NewAtomLambdaContext(p, localctx)
		p.EnterOuterAlt(localctx, 7)
		{
			p.SetState(220)
			p.Lambda()
		}

	case 8:
		localctx = NewAtomIdentContext(p, localctx)
		p.EnterOuterAlt(localctx, 8)
		{
			p.SetState(221)
			p.Match(SolomonParserIDENT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 9:
		localctx = NewAtomCallWithByContext(p, localctx)
		p.EnterOuterAlt(localctx, 9)
		{
			p.SetState(222)
			p.CallWithBy()
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

// ICallWithByContext is an interface to support dynamic dispatch.
type ICallWithByContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser
	// IsCallWithByContext differentiates from other interfaces.
	IsCallWithByContext()
}

type CallWithByContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyCallWithByContext() *CallWithByContext {
	var p = new(CallWithByContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_callWithBy
	return p
}

func InitEmptyCallWithByContext(p *CallWithByContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_callWithBy
}

func (*CallWithByContext) IsCallWithByContext() {}

func NewCallWithByContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *CallWithByContext {
	var p = new(CallWithByContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_callWithBy

	return p
}

func (s *CallWithByContext) GetParser() antlr.Parser { return s.parser }

func (s *CallWithByContext) CopyAll(ctx *CallWithByContext) {
	s.CopyFrom(&ctx.BaseParserRuleContext)
}

func (s *CallWithByContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *CallWithByContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

type AtomCallContext struct {
	CallWithByContext
}

func NewAtomCallContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *AtomCallContext {
	var p = new(AtomCallContext)

	InitEmptyCallWithByContext(&p.CallWithByContext)
	p.parser = parser
	p.CopyAll(ctx.(*CallWithByContext))

	return p
}

func (s *AtomCallContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *AtomCallContext) Call() ICallContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ICallContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ICallContext)
}

func (s *AtomCallContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterAtomCall(s)
	}
}

func (s *AtomCallContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitAtomCall(s)
	}
}

type AtomCallByDurationContext struct {
	CallWithByContext
}

func NewAtomCallByDurationContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *AtomCallByDurationContext {
	var p = new(AtomCallByDurationContext)

	InitEmptyCallWithByContext(&p.CallWithByContext)
	p.parser = parser
	p.CopyAll(ctx.(*CallWithByContext))

	return p
}

func (s *AtomCallByDurationContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *AtomCallByDurationContext) Call() ICallContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ICallContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ICallContext)
}

func (s *AtomCallByDurationContext) KW_BY() antlr.TerminalNode {
	return s.GetToken(SolomonParserKW_BY, 0)
}

func (s *AtomCallByDurationContext) DURATION() antlr.TerminalNode {
	return s.GetToken(SolomonParserDURATION, 0)
}

func (s *AtomCallByDurationContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterAtomCallByDuration(s)
	}
}

func (s *AtomCallByDurationContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitAtomCallByDuration(s)
	}
}

type AtomCallByLabelContext struct {
	CallWithByContext
}

func NewAtomCallByLabelContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *AtomCallByLabelContext {
	var p = new(AtomCallByLabelContext)

	InitEmptyCallWithByContext(&p.CallWithByContext)
	p.parser = parser
	p.CopyAll(ctx.(*CallWithByContext))

	return p
}

func (s *AtomCallByLabelContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *AtomCallByLabelContext) Call() ICallContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ICallContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ICallContext)
}

func (s *AtomCallByLabelContext) KW_BY() antlr.TerminalNode {
	return s.GetToken(SolomonParserKW_BY, 0)
}

func (s *AtomCallByLabelContext) IdentOrString() IIdentOrStringContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IIdentOrStringContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IIdentOrStringContext)
}

func (s *AtomCallByLabelContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterAtomCallByLabel(s)
	}
}

func (s *AtomCallByLabelContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitAtomCallByLabel(s)
	}
}

type AtomCallByLabelsContext struct {
	CallWithByContext
}

func NewAtomCallByLabelsContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *AtomCallByLabelsContext {
	var p = new(AtomCallByLabelsContext)

	InitEmptyCallWithByContext(&p.CallWithByContext)
	p.parser = parser
	p.CopyAll(ctx.(*CallWithByContext))

	return p
}

func (s *AtomCallByLabelsContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *AtomCallByLabelsContext) Call() ICallContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ICallContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ICallContext)
}

func (s *AtomCallByLabelsContext) KW_BY() antlr.TerminalNode {
	return s.GetToken(SolomonParserKW_BY, 0)
}

func (s *AtomCallByLabelsContext) OPENING_PAREN() antlr.TerminalNode {
	return s.GetToken(SolomonParserOPENING_PAREN, 0)
}

func (s *AtomCallByLabelsContext) AllIdentOrString() []IIdentOrStringContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IIdentOrStringContext); ok {
			len++
		}
	}

	tst := make([]IIdentOrStringContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IIdentOrStringContext); ok {
			tst[i] = t.(IIdentOrStringContext)
			i++
		}
	}

	return tst
}

func (s *AtomCallByLabelsContext) IdentOrString(i int) IIdentOrStringContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IIdentOrStringContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IIdentOrStringContext)
}

func (s *AtomCallByLabelsContext) CLOSING_PAREN() antlr.TerminalNode {
	return s.GetToken(SolomonParserCLOSING_PAREN, 0)
}

func (s *AtomCallByLabelsContext) AllCOMMA() []antlr.TerminalNode {
	return s.GetTokens(SolomonParserCOMMA)
}

func (s *AtomCallByLabelsContext) COMMA(i int) antlr.TerminalNode {
	return s.GetToken(SolomonParserCOMMA, i)
}

func (s *AtomCallByLabelsContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterAtomCallByLabels(s)
	}
}

func (s *AtomCallByLabelsContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitAtomCallByLabels(s)
	}
}

func (p *SolomonParser) CallWithBy() (localctx ICallWithByContext) {
	localctx = NewCallWithByContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 42, SolomonParserRULE_callWithBy)
	var _la int

	p.SetState(247)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 22, p.GetParserRuleContext()) {
	case 1:
		localctx = NewAtomCallContext(p, localctx)
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(225)
			p.Call()
		}

	case 2:
		localctx = NewAtomCallByDurationContext(p, localctx)
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(226)
			p.Call()
		}
		{
			p.SetState(227)
			p.Match(SolomonParserKW_BY)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(228)
			p.Match(SolomonParserDURATION)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 3:
		localctx = NewAtomCallByLabelContext(p, localctx)
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(230)
			p.Call()
		}
		{
			p.SetState(231)
			p.Match(SolomonParserKW_BY)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(232)
			p.IdentOrString()
		}

	case 4:
		localctx = NewAtomCallByLabelsContext(p, localctx)
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(234)
			p.Call()
		}
		{
			p.SetState(235)
			p.Match(SolomonParserKW_BY)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(236)
			p.Match(SolomonParserOPENING_PAREN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(237)
			p.IdentOrString()
		}
		p.SetState(242)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == SolomonParserCOMMA {
			{
				p.SetState(238)
				p.Match(SolomonParserCOMMA)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(239)
				p.IdentOrString()
			}

			p.SetState(244)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}
		{
			p.SetState(245)
			p.Match(SolomonParserCLOSING_PAREN)
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

// ICallContext is an interface to support dynamic dispatch.
type ICallContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENT() antlr.TerminalNode
	OPENING_PAREN() antlr.TerminalNode
	Arguments() IArgumentsContext
	CLOSING_PAREN() antlr.TerminalNode

	// IsCallContext differentiates from other interfaces.
	IsCallContext()
}

type CallContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyCallContext() *CallContext {
	var p = new(CallContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_call
	return p
}

func InitEmptyCallContext(p *CallContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_call
}

func (*CallContext) IsCallContext() {}

func NewCallContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *CallContext {
	var p = new(CallContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_call

	return p
}

func (s *CallContext) GetParser() antlr.Parser { return s.parser }

func (s *CallContext) IDENT() antlr.TerminalNode {
	return s.GetToken(SolomonParserIDENT, 0)
}

func (s *CallContext) OPENING_PAREN() antlr.TerminalNode {
	return s.GetToken(SolomonParserOPENING_PAREN, 0)
}

func (s *CallContext) Arguments() IArgumentsContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IArgumentsContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IArgumentsContext)
}

func (s *CallContext) CLOSING_PAREN() antlr.TerminalNode {
	return s.GetToken(SolomonParserCLOSING_PAREN, 0)
}

func (s *CallContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *CallContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *CallContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterCall(s)
	}
}

func (s *CallContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitCall(s)
	}
}

func (p *SolomonParser) Call() (localctx ICallContext) {
	localctx = NewCallContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 44, SolomonParserRULE_call)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(249)
		p.Match(SolomonParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(250)
		p.Match(SolomonParserOPENING_PAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(251)
		p.Arguments()
	}
	{
		p.SetState(252)
		p.Match(SolomonParserCLOSING_PAREN)
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

// IArgumentsContext is an interface to support dynamic dispatch.
type IArgumentsContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Sequence() ISequenceContext

	// IsArgumentsContext differentiates from other interfaces.
	IsArgumentsContext()
}

type ArgumentsContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyArgumentsContext() *ArgumentsContext {
	var p = new(ArgumentsContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_arguments
	return p
}

func InitEmptyArgumentsContext(p *ArgumentsContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_arguments
}

func (*ArgumentsContext) IsArgumentsContext() {}

func NewArgumentsContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ArgumentsContext {
	var p = new(ArgumentsContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_arguments

	return p
}

func (s *ArgumentsContext) GetParser() antlr.Parser { return s.parser }

func (s *ArgumentsContext) Sequence() ISequenceContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISequenceContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISequenceContext)
}

func (s *ArgumentsContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ArgumentsContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ArgumentsContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterArguments(s)
	}
}

func (s *ArgumentsContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitArguments(s)
	}
}

func (p *SolomonParser) Arguments() (localctx IArgumentsContext) {
	localctx = NewArgumentsContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 46, SolomonParserRULE_arguments)
	p.SetState(256)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case SolomonParserOPENING_BRACE, SolomonParserOPENING_PAREN, SolomonParserOPENING_BRACKET, SolomonParserPLUS, SolomonParserMINUS, SolomonParserNOT, SolomonParserIDENT_WITH_DOTS, SolomonParserIDENT, SolomonParserDURATION, SolomonParserNUMBER, SolomonParserSTRING:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(254)
			p.Sequence()
		}

	case SolomonParserCLOSING_PAREN:
		p.EnterOuterAlt(localctx, 2)

	default:
		p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
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

// ISequenceContext is an interface to support dynamic dispatch.
type ISequenceContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllExpression() []IExpressionContext
	Expression(i int) IExpressionContext
	AllCOMMA() []antlr.TerminalNode
	COMMA(i int) antlr.TerminalNode

	// IsSequenceContext differentiates from other interfaces.
	IsSequenceContext()
}

type SequenceContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptySequenceContext() *SequenceContext {
	var p = new(SequenceContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_sequence
	return p
}

func InitEmptySequenceContext(p *SequenceContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_sequence
}

func (*SequenceContext) IsSequenceContext() {}

func NewSequenceContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SequenceContext {
	var p = new(SequenceContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_sequence

	return p
}

func (s *SequenceContext) GetParser() antlr.Parser { return s.parser }

func (s *SequenceContext) AllExpression() []IExpressionContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IExpressionContext); ok {
			len++
		}
	}

	tst := make([]IExpressionContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IExpressionContext); ok {
			tst[i] = t.(IExpressionContext)
			i++
		}
	}

	return tst
}

func (s *SequenceContext) Expression(i int) IExpressionContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExpressionContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExpressionContext)
}

func (s *SequenceContext) AllCOMMA() []antlr.TerminalNode {
	return s.GetTokens(SolomonParserCOMMA)
}

func (s *SequenceContext) COMMA(i int) antlr.TerminalNode {
	return s.GetToken(SolomonParserCOMMA, i)
}

func (s *SequenceContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SequenceContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *SequenceContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterSequence(s)
	}
}

func (s *SequenceContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitSequence(s)
	}
}

func (p *SolomonParser) Sequence() (localctx ISequenceContext) {
	localctx = NewSequenceContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 48, SolomonParserRULE_sequence)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(258)
		p.Expression()
	}
	p.SetState(263)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == SolomonParserCOMMA {
		{
			p.SetState(259)
			p.Match(SolomonParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(260)
			p.Expression()
		}

		p.SetState(265)
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

// ISolomonSelectorsContext is an interface to support dynamic dispatch.
type ISolomonSelectorsContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Selectors() ISelectorsContext
	IdentOrString() IIdentOrStringContext
	IDENT_WITH_DOTS() antlr.TerminalNode

	// IsSolomonSelectorsContext differentiates from other interfaces.
	IsSolomonSelectorsContext()
}

type SolomonSelectorsContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptySolomonSelectorsContext() *SolomonSelectorsContext {
	var p = new(SolomonSelectorsContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_solomonSelectors
	return p
}

func InitEmptySolomonSelectorsContext(p *SolomonSelectorsContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_solomonSelectors
}

func (*SolomonSelectorsContext) IsSolomonSelectorsContext() {}

func NewSolomonSelectorsContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SolomonSelectorsContext {
	var p = new(SolomonSelectorsContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_solomonSelectors

	return p
}

func (s *SolomonSelectorsContext) GetParser() antlr.Parser { return s.parser }

func (s *SolomonSelectorsContext) Selectors() ISelectorsContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISelectorsContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISelectorsContext)
}

func (s *SolomonSelectorsContext) IdentOrString() IIdentOrStringContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IIdentOrStringContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IIdentOrStringContext)
}

func (s *SolomonSelectorsContext) IDENT_WITH_DOTS() antlr.TerminalNode {
	return s.GetToken(SolomonParserIDENT_WITH_DOTS, 0)
}

func (s *SolomonSelectorsContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SolomonSelectorsContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *SolomonSelectorsContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterSolomonSelectors(s)
	}
}

func (s *SolomonSelectorsContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitSolomonSelectors(s)
	}
}

func (p *SolomonParser) SolomonSelectors() (localctx ISolomonSelectorsContext) {
	localctx = NewSolomonSelectorsContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 50, SolomonParserRULE_solomonSelectors)
	p.EnterOuterAlt(localctx, 1)
	p.SetState(268)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	switch p.GetTokenStream().LA(1) {
	case SolomonParserIDENT, SolomonParserSTRING:
		{
			p.SetState(266)
			p.IdentOrString()
		}

	case SolomonParserIDENT_WITH_DOTS:
		{
			p.SetState(267)
			p.Match(SolomonParserIDENT_WITH_DOTS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case SolomonParserOPENING_BRACE:

	default:
	}
	{
		p.SetState(270)
		p.Selectors()
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
	p.RuleIndex = SolomonParserRULE_selectors
	return p
}

func InitEmptySelectorsContext(p *SelectorsContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_selectors
}

func (*SelectorsContext) IsSelectorsContext() {}

func NewSelectorsContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SelectorsContext {
	var p = new(SelectorsContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_selectors

	return p
}

func (s *SelectorsContext) GetParser() antlr.Parser { return s.parser }

func (s *SelectorsContext) OPENING_BRACE() antlr.TerminalNode {
	return s.GetToken(SolomonParserOPENING_BRACE, 0)
}

func (s *SelectorsContext) SelectorList() ISelectorListContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISelectorListContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISelectorListContext)
}

func (s *SelectorsContext) CLOSING_BRACE() antlr.TerminalNode {
	return s.GetToken(SolomonParserCLOSING_BRACE, 0)
}

func (s *SelectorsContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SelectorsContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *SelectorsContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterSelectors(s)
	}
}

func (s *SelectorsContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitSelectors(s)
	}
}

func (p *SolomonParser) Selectors() (localctx ISelectorsContext) {
	localctx = NewSelectorsContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 52, SolomonParserRULE_selectors)
	p.SetState(278)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 26, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(272)
			p.Match(SolomonParserOPENING_BRACE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(273)
			p.SelectorList()
		}
		{
			p.SetState(274)
			p.Match(SolomonParserCLOSING_BRACE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(276)
			p.Match(SolomonParserOPENING_BRACE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(277)
			p.Match(SolomonParserCLOSING_BRACE)
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
	p.RuleIndex = SolomonParserRULE_selectorList
	return p
}

func InitEmptySelectorListContext(p *SelectorListContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_selectorList
}

func (*SelectorListContext) IsSelectorListContext() {}

func NewSelectorListContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SelectorListContext {
	var p = new(SelectorListContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_selectorList

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
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISelectorContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
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
	return s.GetTokens(SolomonParserCOMMA)
}

func (s *SelectorListContext) COMMA(i int) antlr.TerminalNode {
	return s.GetToken(SolomonParserCOMMA, i)
}

func (s *SelectorListContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SelectorListContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *SelectorListContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterSelectorList(s)
	}
}

func (s *SelectorListContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitSelectorList(s)
	}
}

func (p *SolomonParser) SelectorList() (localctx ISelectorListContext) {
	localctx = NewSelectorListContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 54, SolomonParserRULE_selectorList)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(280)
		p.Selector()
	}
	p.SetState(285)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == SolomonParserCOMMA {
		{
			p.SetState(281)
			p.Match(SolomonParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(282)
			p.Selector()
		}

		p.SetState(287)
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
	p.RuleIndex = SolomonParserRULE_selector
	return p
}

func InitEmptySelectorContext(p *SelectorContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_selector
}

func (*SelectorContext) IsSelectorContext() {}

func NewSelectorContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SelectorContext {
	var p = new(SelectorContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_selector

	return p
}

func (s *SelectorContext) GetParser() antlr.Parser { return s.parser }

func (s *SelectorContext) SelectorLeftOperand() ISelectorLeftOperandContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISelectorLeftOperandContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISelectorLeftOperandContext)
}

func (s *SelectorContext) SelectorOpString() ISelectorOpStringContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISelectorOpStringContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISelectorOpStringContext)
}

func (s *SelectorContext) IDENT_WITH_DOTS() antlr.TerminalNode {
	return s.GetToken(SolomonParserIDENT_WITH_DOTS, 0)
}

func (s *SelectorContext) IdentOrString() IIdentOrStringContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IIdentOrStringContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IIdentOrStringContext)
}

func (s *SelectorContext) SelectorOpNumber() ISelectorOpNumberContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISelectorOpNumberContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISelectorOpNumberContext)
}

func (s *SelectorContext) NumberUnary() INumberUnaryContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INumberUnaryContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(INumberUnaryContext)
}

func (s *SelectorContext) SelectorOpDuration() ISelectorOpDurationContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISelectorOpDurationContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISelectorOpDurationContext)
}

func (s *SelectorContext) DURATION() antlr.TerminalNode {
	return s.GetToken(SolomonParserDURATION, 0)
}

func (s *SelectorContext) ASSIGNMENT() antlr.TerminalNode {
	return s.GetToken(SolomonParserASSIGNMENT, 0)
}

func (s *SelectorContext) LabelAbsent() ILabelAbsentContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ILabelAbsentContext); ok {
			t = ctx.(antlr.RuleContext)
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
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterSelector(s)
	}
}

func (s *SelectorContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitSelector(s)
	}
}

func (p *SolomonParser) Selector() (localctx ISelectorContext) {
	localctx = NewSelectorContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 56, SolomonParserRULE_selector)
	p.SetState(306)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 29, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(288)
			p.SelectorLeftOperand()
		}
		{
			p.SetState(289)
			p.SelectorOpString()
		}
		p.SetState(292)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}

		switch p.GetTokenStream().LA(1) {
		case SolomonParserIDENT_WITH_DOTS:
			{
				p.SetState(290)
				p.Match(SolomonParserIDENT_WITH_DOTS)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}

		case SolomonParserIDENT, SolomonParserSTRING:
			{
				p.SetState(291)
				p.IdentOrString()
			}

		default:
			p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
			goto errorExit
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(294)
			p.SelectorLeftOperand()
		}
		{
			p.SetState(295)
			p.SelectorOpNumber()
		}
		{
			p.SetState(296)
			p.NumberUnary()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(298)
			p.SelectorLeftOperand()
		}
		{
			p.SetState(299)
			p.SelectorOpDuration()
		}
		{
			p.SetState(300)
			p.Match(SolomonParserDURATION)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 4:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(302)
			p.SelectorLeftOperand()
		}
		{
			p.SetState(303)
			p.Match(SolomonParserASSIGNMENT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(304)
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
	ISUBSTRING() antlr.TerminalNode
	NOT_ISUBSTRING() antlr.TerminalNode
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
	p.RuleIndex = SolomonParserRULE_selectorOpString
	return p
}

func InitEmptySelectorOpStringContext(p *SelectorOpStringContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_selectorOpString
}

func (*SelectorOpStringContext) IsSelectorOpStringContext() {}

func NewSelectorOpStringContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SelectorOpStringContext {
	var p = new(SelectorOpStringContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_selectorOpString

	return p
}

func (s *SelectorOpStringContext) GetParser() antlr.Parser { return s.parser }

func (s *SelectorOpStringContext) ASSIGNMENT() antlr.TerminalNode {
	return s.GetToken(SolomonParserASSIGNMENT, 0)
}

func (s *SelectorOpStringContext) NE() antlr.TerminalNode {
	return s.GetToken(SolomonParserNE, 0)
}

func (s *SelectorOpStringContext) EQ() antlr.TerminalNode {
	return s.GetToken(SolomonParserEQ, 0)
}

func (s *SelectorOpStringContext) NOT_EQUIV() antlr.TerminalNode {
	return s.GetToken(SolomonParserNOT_EQUIV, 0)
}

func (s *SelectorOpStringContext) REGEX() antlr.TerminalNode {
	return s.GetToken(SolomonParserREGEX, 0)
}

func (s *SelectorOpStringContext) NOT_REGEX() antlr.TerminalNode {
	return s.GetToken(SolomonParserNOT_REGEX, 0)
}

func (s *SelectorOpStringContext) ISUBSTRING() antlr.TerminalNode {
	return s.GetToken(SolomonParserISUBSTRING, 0)
}

func (s *SelectorOpStringContext) NOT_ISUBSTRING() antlr.TerminalNode {
	return s.GetToken(SolomonParserNOT_ISUBSTRING, 0)
}

func (s *SelectorOpStringContext) GT() antlr.TerminalNode {
	return s.GetToken(SolomonParserGT, 0)
}

func (s *SelectorOpStringContext) LT() antlr.TerminalNode {
	return s.GetToken(SolomonParserLT, 0)
}

func (s *SelectorOpStringContext) GE() antlr.TerminalNode {
	return s.GetToken(SolomonParserGE, 0)
}

func (s *SelectorOpStringContext) LE() antlr.TerminalNode {
	return s.GetToken(SolomonParserLE, 0)
}

func (s *SelectorOpStringContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SelectorOpStringContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *SelectorOpStringContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterSelectorOpString(s)
	}
}

func (s *SelectorOpStringContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitSelectorOpString(s)
	}
}

func (p *SolomonParser) SelectorOpString() (localctx ISelectorOpStringContext) {
	localctx = NewSelectorOpStringContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 58, SolomonParserRULE_selectorOpString)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(308)
		_la = p.GetTokenStream().LA(1)

		if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&5368184832) != 0) {
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
	p.RuleIndex = SolomonParserRULE_selectorOpNumber
	return p
}

func InitEmptySelectorOpNumberContext(p *SelectorOpNumberContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_selectorOpNumber
}

func (*SelectorOpNumberContext) IsSelectorOpNumberContext() {}

func NewSelectorOpNumberContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SelectorOpNumberContext {
	var p = new(SelectorOpNumberContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_selectorOpNumber

	return p
}

func (s *SelectorOpNumberContext) GetParser() antlr.Parser { return s.parser }

func (s *SelectorOpNumberContext) ASSIGNMENT() antlr.TerminalNode {
	return s.GetToken(SolomonParserASSIGNMENT, 0)
}

func (s *SelectorOpNumberContext) NE() antlr.TerminalNode {
	return s.GetToken(SolomonParserNE, 0)
}

func (s *SelectorOpNumberContext) EQ() antlr.TerminalNode {
	return s.GetToken(SolomonParserEQ, 0)
}

func (s *SelectorOpNumberContext) NOT_EQUIV() antlr.TerminalNode {
	return s.GetToken(SolomonParserNOT_EQUIV, 0)
}

func (s *SelectorOpNumberContext) GT() antlr.TerminalNode {
	return s.GetToken(SolomonParserGT, 0)
}

func (s *SelectorOpNumberContext) LT() antlr.TerminalNode {
	return s.GetToken(SolomonParserLT, 0)
}

func (s *SelectorOpNumberContext) GE() antlr.TerminalNode {
	return s.GetToken(SolomonParserGE, 0)
}

func (s *SelectorOpNumberContext) LE() antlr.TerminalNode {
	return s.GetToken(SolomonParserLE, 0)
}

func (s *SelectorOpNumberContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SelectorOpNumberContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *SelectorOpNumberContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterSelectorOpNumber(s)
	}
}

func (s *SelectorOpNumberContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitSelectorOpNumber(s)
	}
}

func (p *SolomonParser) SelectorOpNumber() (localctx ISelectorOpNumberContext) {
	localctx = NewSelectorOpNumberContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 60, SolomonParserRULE_selectorOpNumber)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(310)
		_la = p.GetTokenStream().LA(1)

		if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&4361551872) != 0) {
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
	p.RuleIndex = SolomonParserRULE_selectorOpDuration
	return p
}

func InitEmptySelectorOpDurationContext(p *SelectorOpDurationContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_selectorOpDuration
}

func (*SelectorOpDurationContext) IsSelectorOpDurationContext() {}

func NewSelectorOpDurationContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SelectorOpDurationContext {
	var p = new(SelectorOpDurationContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_selectorOpDuration

	return p
}

func (s *SelectorOpDurationContext) GetParser() antlr.Parser { return s.parser }

func (s *SelectorOpDurationContext) GT() antlr.TerminalNode {
	return s.GetToken(SolomonParserGT, 0)
}

func (s *SelectorOpDurationContext) LT() antlr.TerminalNode {
	return s.GetToken(SolomonParserLT, 0)
}

func (s *SelectorOpDurationContext) GE() antlr.TerminalNode {
	return s.GetToken(SolomonParserGE, 0)
}

func (s *SelectorOpDurationContext) LE() antlr.TerminalNode {
	return s.GetToken(SolomonParserLE, 0)
}

func (s *SelectorOpDurationContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SelectorOpDurationContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *SelectorOpDurationContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterSelectorOpDuration(s)
	}
}

func (s *SelectorOpDurationContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitSelectorOpDuration(s)
	}
}

func (p *SolomonParser) SelectorOpDuration() (localctx ISelectorOpDurationContext) {
	localctx = NewSelectorOpDurationContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 62, SolomonParserRULE_selectorOpDuration)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(312)
		_la = p.GetTokenStream().LA(1)

		if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&7864320) != 0) {
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
	p.RuleIndex = SolomonParserRULE_selectorLeftOperand
	return p
}

func InitEmptySelectorLeftOperandContext(p *SelectorLeftOperandContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_selectorLeftOperand
}

func (*SelectorLeftOperandContext) IsSelectorLeftOperandContext() {}

func NewSelectorLeftOperandContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SelectorLeftOperandContext {
	var p = new(SelectorLeftOperandContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_selectorLeftOperand

	return p
}

func (s *SelectorLeftOperandContext) GetParser() antlr.Parser { return s.parser }

func (s *SelectorLeftOperandContext) IDENT() antlr.TerminalNode {
	return s.GetToken(SolomonParserIDENT, 0)
}

func (s *SelectorLeftOperandContext) IDENT_WITH_DOTS() antlr.TerminalNode {
	return s.GetToken(SolomonParserIDENT_WITH_DOTS, 0)
}

func (s *SelectorLeftOperandContext) STRING() antlr.TerminalNode {
	return s.GetToken(SolomonParserSTRING, 0)
}

func (s *SelectorLeftOperandContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SelectorLeftOperandContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *SelectorLeftOperandContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterSelectorLeftOperand(s)
	}
}

func (s *SelectorLeftOperandContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitSelectorLeftOperand(s)
	}
}

func (p *SolomonParser) SelectorLeftOperand() (localctx ISelectorLeftOperandContext) {
	localctx = NewSelectorLeftOperandContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 64, SolomonParserRULE_selectorLeftOperand)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(314)
		_la = p.GetTokenStream().LA(1)

		if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&652835028992) != 0) {
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
	p.RuleIndex = SolomonParserRULE_numberUnary
	return p
}

func InitEmptyNumberUnaryContext(p *NumberUnaryContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_numberUnary
}

func (*NumberUnaryContext) IsNumberUnaryContext() {}

func NewNumberUnaryContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *NumberUnaryContext {
	var p = new(NumberUnaryContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_numberUnary

	return p
}

func (s *NumberUnaryContext) GetParser() antlr.Parser { return s.parser }

func (s *NumberUnaryContext) NUMBER() antlr.TerminalNode {
	return s.GetToken(SolomonParserNUMBER, 0)
}

func (s *NumberUnaryContext) PLUS() antlr.TerminalNode {
	return s.GetToken(SolomonParserPLUS, 0)
}

func (s *NumberUnaryContext) MINUS() antlr.TerminalNode {
	return s.GetToken(SolomonParserMINUS, 0)
}

func (s *NumberUnaryContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *NumberUnaryContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *NumberUnaryContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterNumberUnary(s)
	}
}

func (s *NumberUnaryContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitNumberUnary(s)
	}
}

func (p *SolomonParser) NumberUnary() (localctx INumberUnaryContext) {
	localctx = NewNumberUnaryContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 66, SolomonParserRULE_numberUnary)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(317)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == SolomonParserPLUS || _la == SolomonParserMINUS {
		{
			p.SetState(316)
			_la = p.GetTokenStream().LA(1)

			if !(_la == SolomonParserPLUS || _la == SolomonParserMINUS) {
				p.GetErrorHandler().RecoverInline(p)
			} else {
				p.GetErrorHandler().ReportMatch(p)
				p.Consume()
			}
		}

	}
	{
		p.SetState(319)
		p.Match(SolomonParserNUMBER)
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
	p.RuleIndex = SolomonParserRULE_labelAbsent
	return p
}

func InitEmptyLabelAbsentContext(p *LabelAbsentContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_labelAbsent
}

func (*LabelAbsentContext) IsLabelAbsentContext() {}

func NewLabelAbsentContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *LabelAbsentContext {
	var p = new(LabelAbsentContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_labelAbsent

	return p
}

func (s *LabelAbsentContext) GetParser() antlr.Parser { return s.parser }

func (s *LabelAbsentContext) MINUS() antlr.TerminalNode {
	return s.GetToken(SolomonParserMINUS, 0)
}

func (s *LabelAbsentContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *LabelAbsentContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *LabelAbsentContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterLabelAbsent(s)
	}
}

func (s *LabelAbsentContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitLabelAbsent(s)
	}
}

func (p *SolomonParser) LabelAbsent() (localctx ILabelAbsentContext) {
	localctx = NewLabelAbsentContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 68, SolomonParserRULE_labelAbsent)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(321)
		p.Match(SolomonParserMINUS)
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
	p.RuleIndex = SolomonParserRULE_identOrString
	return p
}

func InitEmptyIdentOrStringContext(p *IdentOrStringContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = SolomonParserRULE_identOrString
}

func (*IdentOrStringContext) IsIdentOrStringContext() {}

func NewIdentOrStringContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *IdentOrStringContext {
	var p = new(IdentOrStringContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = SolomonParserRULE_identOrString

	return p
}

func (s *IdentOrStringContext) GetParser() antlr.Parser { return s.parser }

func (s *IdentOrStringContext) IDENT() antlr.TerminalNode {
	return s.GetToken(SolomonParserIDENT, 0)
}

func (s *IdentOrStringContext) STRING() antlr.TerminalNode {
	return s.GetToken(SolomonParserSTRING, 0)
}

func (s *IdentOrStringContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *IdentOrStringContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *IdentOrStringContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.EnterIdentOrString(s)
	}
}

func (s *IdentOrStringContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(SolomonParserListener); ok {
		listenerT.ExitIdentOrString(s)
	}
}

func (p *SolomonParser) IdentOrString() (localctx IIdentOrStringContext) {
	localctx = NewIdentOrStringContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 70, SolomonParserRULE_identOrString)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(323)
		_la = p.GetTokenStream().LA(1)

		if !(_la == SolomonParserIDENT || _la == SolomonParserSTRING) {
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
