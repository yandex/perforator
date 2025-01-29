// Package parser edits text proto files, applies standard formatting
// and preserves comments.
// See also: https://github.com/golang/protobuf/blob/master/proto/text_parser.go
//
// To disable a specific file from getting formatted, add '# txtpbfmt: disable'
// at the top of the file.
package parser

import (
	"bufio"
	"bytes"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/mitchellh/go-wordwrap"
	"github.com/protocolbuffers/txtpbfmt/ast"
	"github.com/protocolbuffers/txtpbfmt/unquote"
)

// Config can be used to pass additional config parameters to the formatter at
// the time of the API call.
type Config struct {
	// Do not apply any reformatting to this file.
	Disable bool

	// Expand all children irrespective of the initial state.
	ExpandAllChildren bool

	// Skip colons whenever possible.
	SkipAllColons bool

	// Allow unnamed nodes everywhere.
	// Default is to allow only top-level nodes to be unnamed.
	AllowUnnamedNodesEverywhere bool

	// Sort fields by field name.
	SortFieldsByFieldName bool

	// Sort adjacent scalar fields of the same field name by their contents.
	SortRepeatedFieldsByContent bool

	// Sort adjacent message fields of the given field name by the contents of the given subfield.
	// Format: either "field_name.subfield_name" or just "subfield_name" (applies to all field names).
	SortRepeatedFieldsBySubfield []string

	// Map from Node.Name to the order of all fields within that node. See AddFieldSortOrder().
	fieldSortOrder map[string][]string

	// RequireFieldSortOrderToMatchAllFieldsInNode will cause parsing to fail if a node was added via
	// AddFieldSortOrder() but 1+ fields under that node in the textproto aren't specified in the
	// field order. This won't fail for nodes that don't have a field order specified at all. Use this
	// to strictly enforce that your field order config always orders ALL the fields, and you're
	// willing for new fields in the textproto to break parsing in order to enforce it.
	RequireFieldSortOrderToMatchAllFieldsInNode bool

	// Remove lines that have the same field name and scalar value as another.
	RemoveDuplicateValuesForRepeatedFields bool

	// Permit usage of Python-style """ or ''' delimited strings.
	AllowTripleQuotedStrings bool

	// Max columns for string field values. If zero, no string wrapping will occur.
	// Strings that may contain HTML tags will never be wrapped.
	WrapStringsAtColumn int

	// Whether strings that appear to contain HTML tags should be wrapped
	// (requires WrapStringsAtColumn to be set).
	WrapHTMLStrings bool

	// Wrap string field values after each newline.
	// Should not be used with other Wrap* options.
	WrapStringsAfterNewlines bool

	// Whether angle brackets used instead of curly braces should be preserved
	// when outputting a formatted textproto.
	PreserveAngleBrackets bool

	// Use single quotes around strings that contain double but not single quotes.
	SmartQuotes bool

	// Logger enables logging when it is non-nil.
	// If the log messages aren't going to be useful, it's best to leave Logger
	// set to nil, as otherwise log messages will be constructed.
	Logger Logger
}

func (c *Config) infof(format string, args ...any) {
	if c.Logger != nil {
		c.Logger.Infof(format, args...)
	}
}
func (c *Config) infoLevel() bool {
	return c.Logger != nil
}

// Logger is a small glog-like interface.
type Logger interface {
	// Infof is used for informative messages, for testing or debugging.
	Infof(format string, args ...any)
}

// RootName contains a constant that can be used to identify the root of all Nodes.
const RootName = "__ROOT__"

// AddFieldSortOrder adds a config rule for the given Node.Name, so that all contained field names
// are output in the provided order. To specify an order for top-level Nodes, use RootName as the
// nodeName.
func (c *Config) AddFieldSortOrder(nodeName string, fieldOrder ...string) {
	if c.fieldSortOrder == nil {
		c.fieldSortOrder = make(map[string][]string)
	}
	c.fieldSortOrder[nodeName] = fieldOrder
}

// UnsortedFieldsError will be returned by ParseWithConfig if
// Config.RequireFieldSortOrderToMatchAllFieldsInNode is set, and an unrecognized field is found
// while parsing.
type UnsortedFieldsError struct {
	UnsortedFields []UnsortedField
}

// UnsortedField records details about a single unsorted field.
type UnsortedField struct {
	FieldName       string
	Line            int32
	ParentFieldName string
}

func (e *UnsortedFieldsError) Error() string {
	var errs []string
	for _, us := range e.UnsortedFields {
		errs = append(errs, fmt.Sprintf("  line: %d, parent field: %q, unsorted field: %q", us.Line, us.ParentFieldName, us.FieldName))
	}
	return fmt.Sprintf("fields parsed that were not specified in the parser.AddFieldSortOrder() call:\n%s", strings.Join(errs, "\n"))
}

type parser struct {
	in     []byte
	index  int
	length int
	// Maps the index of '{' characters on 'in' that have the matching '}' on
	// the same line to 'true'.
	bracketSameLine map[int]bool
	config          Config
	line, column    int // current position, 1-based.
}

var defConfig = Config{}
var tagRegex = regexp.MustCompile(`<.*>`)

const indentSpaces = "  "

// Format formats a text proto file preserving comments.
func Format(in []byte) ([]byte, error) {
	return FormatWithConfig(in, defConfig)
}

// FormatWithConfig functions similar to format, but allows the user to pass in
// additional configuration options.
func FormatWithConfig(in []byte, c Config) ([]byte, error) {
	if err := addMetaCommentsToConfig(in, &c); err != nil {
		return nil, err
	}
	if c.Disable {
		c.infof("Ignored file with 'disable' comment.")
		return in, nil
	}
	nodes, err := parseWithMetaCommentConfig(in, c)
	if err != nil {
		return nil, err
	}
	return PrettyBytes(nodes, 0), nil
}

// Return the byte-positions of each bracket which has the corresponding close on the
// same line as a set.
func sameLineBrackets(in []byte, allowTripleQuotedStrings bool) (map[int]bool, error) {
	line := 1
	type bracket struct {
		index int
		line  int
	}
	open := []bracket{} // Stack.
	res := map[int]bool{}
	insideComment := false
	insideString := false
	insideTemplate := false
	insideTripleQuotedString := false
	var stringDelimiter string
	isEscapedChar := false
	for i, c := range in {
		switch c {
		case '\n':
			line++
			insideComment = false
		case '{', '<':
			if insideComment || insideString || insideTemplate {
				continue
			}
			open = append(open, bracket{index: i, line: line})
		case '}', '>':
			if insideComment || insideString || insideTemplate {
				continue
			}
			if len(open) == 0 {
				return nil, fmt.Errorf("too many '}' or '>' at index %d", i)
			}
			last := len(open) - 1
			br := open[last]
			open = open[:last]
			if br.line == line {
				res[br.index] = true
			}
		case '#':
			if insideString {
				continue
			}
			insideComment = true
		case '%':
			if insideComment || insideString {
				continue
			}
			if insideTemplate {
				insideTemplate = false
			} else {
				insideTemplate = true
			}
		case '"', '\'':
			if insideComment {
				continue
			}
			delim := string(c)
			tripleQuoted := false
			if allowTripleQuotedStrings && i+3 <= len(in) {
				triple := string(in[i : i+3])
				if triple == `"""` || triple == `'''` {
					delim = triple
					tripleQuoted = true
				}
			}

			if insideString {
				if stringDelimiter == delim && (insideTripleQuotedString || !isEscapedChar) {
					insideString = false
					insideTripleQuotedString = false
				}
			} else {
				insideString = true
				if tripleQuoted {
					insideTripleQuotedString = true
				}
				stringDelimiter = delim
			}
		}

		if isEscapedChar {
			isEscapedChar = false
		} else if c == '\\' && insideString && !insideTripleQuotedString {
			isEscapedChar = true
		}
	}
	if insideString {
		return nil, fmt.Errorf("unterminated string literal")
	}
	return res, nil
}

func removeDeleted(nodes []*ast.Node) []*ast.Node {
	res := []*ast.Node{}
	// When removing a node which has an empty line before it, we should keep
	// the empty line before the next non-removed node to maintain the visual separation.
	// Consider the following:
	// foo: { name: "foo1" }
	// foo: { name: "foo2" }
	//
	// bar: { name: "bar1" }
	// bar: { name: "bar2" }
	//
	// If we decide to remove both foo2 and bar1, the result should still have one empty
	// line between foo1 and bar2.
	addEmptyLine := false
	for _, node := range nodes {
		if node.Deleted {
			if len(node.PreComments) > 0 && node.PreComments[0] == "" {
				addEmptyLine = true
			}
			continue
		}
		if len(node.Children) > 0 {
			node.Children = removeDeleted(node.Children)
		}
		if addEmptyLine && (len(node.PreComments) == 0 || node.PreComments[0] != "") {
			node.PreComments = append([]string{""}, node.PreComments...)
		}
		addEmptyLine = false
		res = append(res, node)
	}
	return res
}

var (
	spaceSeparators = []byte(" \t\n")
	valueSeparators = []byte(" \t\n{}:,[]<>;#")
)

// Parse returns a tree representation of a textproto file.
func Parse(in []byte) ([]*ast.Node, error) {
	return ParseWithConfig(in, defConfig)
}

// ParseWithConfig functions similar to Parse, but allows the user to pass in
// additional configuration options.
func ParseWithConfig(in []byte, c Config) ([]*ast.Node, error) {
	if err := addMetaCommentsToConfig(in, &c); err != nil {
		return nil, err
	}
	return parseWithMetaCommentConfig(in, c)
}

// Parses in textproto with MetaComments already added to configuration.
func parseWithMetaCommentConfig(in []byte, c Config) ([]*ast.Node, error) {
	p, err := newParser(in, c)
	if err != nil {
		return nil, err
	}
	if p.config.infoLevel() {
		p.config.infof("p.in: %q", string(p.in))
		p.config.infof("p.length: %v", p.length)
	}
	// Although unnamed nodes aren't strictly allowed, some formats represent a
	// list of protos as a list of unnamed top-level nodes.
	nodes, _, err := p.parse( /*isRoot=*/ true)
	if err != nil {
		return nil, err
	}
	if p.index < p.length {
		return nil, fmt.Errorf("parser didn't consume all input. Stopped at %s", p.errorContext())
	}
	if err := wrapStrings(nodes, 0, c); err != nil {
		return nil, err
	}
	if err := sortAndFilterNodes( /*parent=*/ nil, nodes, nodeSortFunction(c), nodeFilterFunction(c), valuesSortFunction(c)); err != nil {
		return nil, err
	}
	return nodes, nil
}

// There are two types of MetaComment, one in the format of <key>=<val> and the other one doesn't
// have the equal sign. Currently there are only two MetaComments that are in the former format:
//
//	"sort_repeated_fields_by_subfield": If this appears multiple times, then they will all be added
//	to the config and the order is perserved.
//	"wrap_strings_at_column": The <val> is expected to be an integer. If it is not, then it will be
//	ignored. If this appears multiple times, only the last one saved.
func addToConfig(metaComment string, c *Config) error {
	// Test if a MetaComment is in the format of <key>=<val>.
	key, val, hasEqualSign := strings.Cut(metaComment, "=")
	switch key {
	case "allow_triple_quoted_strings":
		c.AllowTripleQuotedStrings = true
	case "allow_unnamed_nodes_everywhere":
		c.AllowUnnamedNodesEverywhere = true
	case "disable":
		c.Disable = true
	case "expand_all_children":
		c.ExpandAllChildren = true
	case "preserve_angle_brackets":
		c.PreserveAngleBrackets = true
	case "remove_duplicate_values_for_repeated_fields":
		c.RemoveDuplicateValuesForRepeatedFields = true
	case "skip_all_colons":
		c.SkipAllColons = true
	case "smartquotes":
		c.SmartQuotes = true
	case "sort_fields_by_field_name":
		c.SortFieldsByFieldName = true
	case "sort_repeated_fields_by_content":
		c.SortRepeatedFieldsByContent = true
	case "sort_repeated_fields_by_subfield":
		// Take all the subfields and the subfields in order as tie breakers.
		if !hasEqualSign {
			return fmt.Errorf("format should be %s=<string>, got: %s", key, metaComment)
		}
		c.SortRepeatedFieldsBySubfield = append(c.SortRepeatedFieldsBySubfield, val)
	case "wrap_strings_at_column":
		// If multiple of this MetaComment exists in the file, take the last one.
		if !hasEqualSign {
			return fmt.Errorf("format should be %s=<int>, got: %s", key, metaComment)
		}
		i, err := strconv.Atoi(strings.TrimSpace(val))
		if err != nil {
			return fmt.Errorf("error parsing %s value %q (skipping): %v", key, val, err)
		}
		c.WrapStringsAtColumn = i
	case "wrap_html_strings":
		c.WrapHTMLStrings = true
	case "wrap_strings_after_newlines":
		c.WrapStringsAfterNewlines = true
	default:
		return fmt.Errorf("unrecognized MetaComment: %s", metaComment)
	}
	return nil
}

// Parses MetaComments and adds them to the configuration.
func addMetaCommentsToConfig(in []byte, c *Config) error {
	scanner := bufio.NewScanner(bytes.NewReader(in))
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}
		if line[0] != byte('#') {
			break // only process the leading comment block
		}

		// Look for comment lines in the format of "<key>:<value>", and process the lines with <key>
		// equals to "txtpbfmt". It's assumed that the MetaComments are given in the format of:
		// # txtpbfmt: <MetaComment 1>[, <MetaComment 2> ...]
		key, value, hasColon := strings.Cut(line[1:], ":") // Ignore the first '#'.
		if hasColon && strings.TrimSpace(key) == "txtpbfmt" {
			for _, s := range strings.Split(strings.TrimSpace(value), ",") {
				metaComment := strings.TrimSpace(s)
				if err := addToConfig(metaComment, c); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func newParser(in []byte, c Config) (*parser, error) {
	var bracketSameLine map[int]bool
	if c.ExpandAllChildren {
		bracketSameLine = map[int]bool{}
	} else {
		var err error
		if bracketSameLine, err = sameLineBrackets(in, c.AllowTripleQuotedStrings); err != nil {
			return nil, err
		}
	}
	if len(in) > 0 && in[len(in)-1] != '\n' {
		in = append(in, '\n')
	}
	parser := &parser{
		in:              in,
		index:           0,
		length:          len(in),
		bracketSameLine: bracketSameLine,
		config:          c,
		line:            1,
		column:          1,
	}
	return parser, nil
}

func (p *parser) nextInputIs(b byte) bool {
	return p.index < p.length && p.in[p.index] == b
}

func (p *parser) consume(b byte) bool {
	if !p.nextInputIs(b) {
		return false
	}
	p.index++
	p.column++
	if b == '\n' {
		p.line++
		p.column = 1
	}
	return true
}

// consumeString consumes the given string s, which should not have any newlines.
func (p *parser) consumeString(s string) bool {
	if p.index+len(s) > p.length {
		return false
	}
	if string(p.in[p.index:p.index+len(s)]) != s {
		return false
	}
	p.index += len(s)
	p.column += len(s)
	return true
}

// loopDetector detects if the parser is in an infinite loop (ie failing to
// make progress).
type loopDetector struct {
	lastIndex int
	count     int
	parser    *parser
}

func (p *parser) getLoopDetector() *loopDetector {
	return &loopDetector{lastIndex: p.index, parser: p}
}

func (l *loopDetector) iter() error {
	if l.parser.index == l.lastIndex {
		l.count++
		if l.count < 2 {
			return nil
		}
		return fmt.Errorf("parser failed to make progress at %s", l.parser.errorContext())
	}
	l.lastIndex = l.parser.index
	l.count = 0
	return nil
}

func (p parser) errorContext() string {
	index := p.index
	if index >= p.length {
		index = p.length - 1
	}
	// Provide the surrounding input as context.
	lastContentIndex := index + 20
	if lastContentIndex >= p.length {
		lastContentIndex = p.length - 1
	}
	previousContentIndex := index - 20
	if previousContentIndex < 0 {
		previousContentIndex = 0
	}
	before := string(p.in[previousContentIndex:index])
	after := string(p.in[index:lastContentIndex])
	return fmt.Sprintf("index %v\nposition %+v\nbefore: %q\nafter: %q\nbefore+after: %q", index, p.position(), before, after, before+after)
}

func (p *parser) position() ast.Position {
	return ast.Position{
		Byte:   uint32(p.index),
		Line:   int32(p.line),
		Column: int32(p.column),
	}
}

func (p *parser) consumeOptionalSeparator() error {
	if p.index > 0 && !p.isBlankSep(p.index-1) {
		// If an unnamed field immediately follows non-whitespace, we require a separator character first (key_one:,:value_two instead of key_one::value_two)
		if p.consume(':') {
			return fmt.Errorf("parser encountered unexpected : character (should be whitespace, or a ,; separator)")
		}
	}

	_ = p.consume(';') // Ignore optional ';'.
	_ = p.consume(',') // Ignore optional ','.

	return nil
}

// parse parses a text proto.
// It assumes the text to be either conformant with the standard text proto
// (i.e. passes proto.UnmarshalText() without error) or the alternative textproto
// format (sequence of messages, each of which passes proto.UnmarshalText()).
// endPos is the position of the first character on the first line
// after parsed nodes: that's the position to append more children.
func (p *parser) parse(isRoot bool) (result []*ast.Node, endPos ast.Position, err error) {
	res := []*ast.Node{}
	for ld := p.getLoopDetector(); p.index < p.length; {
		if err := ld.iter(); err != nil {
			return nil, ast.Position{}, err
		}

		// p.parse is often invoked with the index pointing at the newline character
		// after the previous item. We should still report that this item starts in
		// the next line.
		p.consume('\n')
		startPos := p.position()

		// Read PreComments.
		comments, blankLines := p.skipWhiteSpaceAndReadComments(true /* multiLine */)

		// Handle blank lines.
		if blankLines > 0 {
			if p.config.infoLevel() {
				p.config.infof("blankLines: %v", blankLines)
			}
			// Here we collapse the leading blank lines into one blank line.
			comments = append([]string{""}, comments...)
		}

		for p.nextInputIs('%') {
			comments = append(comments, p.readTemplate())
			c, _ := p.skipWhiteSpaceAndReadComments(false)
			comments = append(comments, c...)
		}

		if endPos := p.position(); p.consume('}') || p.consume('>') || p.consume(']') {
			// Handle comments after last child.

			if len(comments) > 0 {
				res = append(res, &ast.Node{Start: startPos, PreComments: comments})
			}

			// endPos points at the closing brace, but we should rather return the position
			// of the first character after the previous item. Therefore let's rewind a bit:
			for endPos.Byte > 0 && p.in[endPos.Byte-1] == ' ' {
				endPos.Byte--
				endPos.Column--
			}

			if err = p.consumeOptionalSeparator(); err != nil {
				return nil, ast.Position{}, err
			}

			// Done parsing children.
			return res, endPos, nil
		}

		nd := &ast.Node{
			Start:       startPos,
			PreComments: comments,
		}
		if p.config.infoLevel() {
			p.config.infof("PreComments: %q", strings.Join(nd.PreComments, "\n"))
		}

		// Skip white-space other than '\n', which is handled below.
		for p.consume(' ') || p.consume('\t') {
		}

		// Handle multiple comment blocks.
		// <example>
		// # comment block 1
		// # comment block 1
		//
		// # comment block 2
		// # comment block 2
		// </example>
		// Each block that ends on an empty line (instead of a field) gets its own
		// 'empty' node.
		if p.nextInputIs('\n') {
			res = append(res, nd)
			continue
		}

		// Handle end of file.
		if p.index >= p.length {
			nd.End = p.position()
			if len(nd.PreComments) > 0 {
				res = append(res, nd)
			}
			break
		}

		if p.consume('[') {
			// Read Name (of proto extension).
			nd.Name = fmt.Sprintf("[%s]", p.readExtension())
			_ = p.consume(']') // Ignore the ']'.
		} else {
			// Read Name.
			nd.Name = p.readFieldName()
			if nd.Name == "" && !isRoot && !p.config.AllowUnnamedNodesEverywhere {
				return nil, ast.Position{}, fmt.Errorf("Failed to find a FieldName at %s", p.errorContext())
			}
		}
		if p.config.infoLevel() {
			p.config.infof("name: %q", nd.Name)
		}
		// Skip separator.
		preCommentsBeforeColon, _ := p.skipWhiteSpaceAndReadComments(true /* multiLine */)
		nd.SkipColon = !p.consume(':')
		previousPos := p.position()
		preCommentsAfterColon, _ := p.skipWhiteSpaceAndReadComments(true /* multiLine */)

		if p.consume('{') || p.consume('<') {
			if p.config.SkipAllColons {
				nd.SkipColon = true
			}
			nd.ChildrenSameLine = p.bracketSameLine[p.index-1]
			nd.IsAngleBracket = p.config.PreserveAngleBrackets && p.in[p.index-1] == '<'
			// Recursive call to parse child nodes.
			nodes, lastPos, err := p.parse( /*isRoot=*/ false)
			if err != nil {
				return nil, ast.Position{}, err
			}
			nd.Children = nodes
			nd.End = lastPos

			nd.ClosingBraceComment = p.readInlineComment()
		} else if p.consume('[') {
			openBracketLine := p.line

			// Skip separator.
			preCommentsAfterListStart := p.readContinuousBlocksOfComments()

			var preComments []string
			preComments = append(preComments, preCommentsBeforeColon...)
			preComments = append(preComments, preCommentsAfterColon...)
			preComments = append(preComments, preCommentsAfterListStart...)

			if p.nextInputIs('{') {
				// Handle list of nodes.
				nd.ChildrenAsList = true

				nodes, lastPos, err := p.parse( /*isRoot=*/ true)
				if err != nil {
					return nil, ast.Position{}, err
				}
				if len(nodes) > 0 {
					nodes[0].PreComments = preComments
				}

				nd.Children = nodes
				nd.End = lastPos
				nd.ClosingBraceComment = p.readInlineComment()
				nd.ChildrenSameLine = openBracketLine == p.line
			} else {
				// Handle list of values.
				nd.ValuesAsList = true // We found values in list - keep it as list.

				for ld := p.getLoopDetector(); !p.consume(']') && p.index < p.length; {
					if err := ld.iter(); err != nil {
						return nil, ast.Position{}, err
					}

					// Read each value in the list.
					vals, err := p.readValues()
					if err != nil {
						return nil, ast.Position{}, err
					}
					if len(vals) != 1 {
						return nil, ast.Position{}, fmt.Errorf("multiple-string value not supported (%v). Please add comma explicitly, see http://b/162070952", vals)
					}
					if len(preComments) > 0 {
						// If we read preComments before readValues(), they should go first,
						// but avoid copy overhead if there are none.
						vals[0].PreComments = append(preComments, vals[0].PreComments...)
					}

					// Skip separator.
					_, _ = p.skipWhiteSpaceAndReadComments(false /* multiLine */)
					if p.consume(',') {
						vals[0].InlineComment = p.readInlineComment()
					}

					nd.Values = append(nd.Values, vals...)

					preComments, _ = p.skipWhiteSpaceAndReadComments(true /* multiLine */)
				}
				nd.ChildrenSameLine = openBracketLine == p.line

				res = append(res, nd)

				// Handle comments after last line (or for empty list)
				nd.PostValuesComments = preComments
				nd.ClosingBraceComment = p.readInlineComment()

				if err = p.consumeOptionalSeparator(); err != nil {
					return nil, ast.Position{}, err
				}

				continue
			}
		} else {
			// Rewind comments.
			p.index = int(previousPos.Byte)
			p.line = int(previousPos.Line)
			p.column = int(previousPos.Column)
			// Handle Values.
			nd.Values, err = p.readValues()
			if err != nil {
				return nil, ast.Position{}, err
			}
			if err = p.consumeOptionalSeparator(); err != nil {
				return nil, ast.Position{}, err
			}
		}
		if p.config.infoLevel() && p.index < p.length {
			p.config.infof("p.in[p.index]: %q", string(p.in[p.index]))
		}
		res = append(res, nd)
	}
	return res, p.position(), nil
}

func (p *parser) readFieldName() string {
	i := p.index
	for ; i < p.length && !p.isValueSep(i); i++ {
	}
	return p.advance(i)
}

func (p *parser) readExtension() string {
	i := p.index
	for ; i < p.length && (p.isBlankSep(i) || !p.isValueSep(i)); i++ {
	}
	return removeBlanks(p.advance(i))
}

func removeBlanks(in string) string {
	s := []byte(in)
	for _, b := range spaceSeparators {
		s = bytes.Replace(s, []byte{b}, nil, -1)
	}
	return string(s)
}

func (p *parser) readContinuousBlocksOfComments() []string {
	var preComments []string
	for {
		comments, blankLines := p.skipWhiteSpaceAndReadComments(true)
		if len(comments) == 0 {
			break
		}
		if blankLines > 0 && len(preComments) > 0 {
			comments = append([]string{""}, comments...)
		}
		preComments = append(preComments, comments...)
	}

	return preComments
}

// skipWhiteSpaceAndReadComments has multiple cases:
//   - (1) reading a block of comments followed by a blank line
//   - (2) reading a block of comments followed by non-blank content
//   - (3) reading the inline comments between the current char and the end of
//     the current line
//
// In both cases (1) and (2), there can also be blank lines before the comment
// starts.
//
// Lines of comments and number of blank lines before the comment will be
// returned. If there is no comment, the returned slice will be empty.
func (p *parser) skipWhiteSpaceAndReadComments(multiLine bool) ([]string, int) {
	i := p.index
	var foundComment, insideComment bool
	commentBegin := 0
	var comments []string
	// Number of blanks lines *before* the comment (if any) starts.
	blankLines := 0
	for ; i < p.length; i++ {
		if p.in[i] == '#' && !insideComment {
			insideComment = true
			foundComment = true
			commentBegin = i
		} else if p.in[i] == '\n' {
			if insideComment {
				comments = append(comments, string(p.in[commentBegin:i])) // Exclude the '\n'.
				insideComment = false
			} else if foundComment {
				i-- // Put back the last '\n' so the caller can detect that we're on case (1).
				break
			} else {
				blankLines++
			}
			if !multiLine {
				break
			}
		}
		if !insideComment && !p.isBlankSep(i) {
			break
		}
	}
	sep := p.advance(i)
	if p.config.infoLevel() {
		p.config.infof("sep: %q\np.index: %v", string(sep), p.index)
		if p.index < p.length {
			p.config.infof("p.in[p.index]: %q", string(p.in[p.index]))
		}
	}
	return comments, blankLines
}

func (p *parser) isBlankSep(i int) bool {
	return bytes.Contains(spaceSeparators, p.in[i:i+1])
}

func (p *parser) isValueSep(i int) bool {
	return bytes.Contains(valueSeparators, p.in[i:i+1])
}

func (p *parser) advance(i int) string {
	if i > p.length {
		i = p.length
	}
	res := p.in[p.index:i]
	p.index = i
	strRes := string(res)
	newlines := strings.Count(strRes, "\n")
	if newlines == 0 {
		p.column += len(strRes)
	} else {
		p.column = len(strRes) - strings.LastIndex(strRes, "\n")
		p.line += newlines
	}
	return string(res)
}

func (p *parser) readValues() ([]*ast.Value, error) {
	var values []*ast.Value
	var previousPos ast.Position
	preComments, _ := p.skipWhiteSpaceAndReadComments(true /* multiLine */)
	if p.nextInputIs('%') {
		values = append(values, p.populateValue(p.readTemplate(), nil))
		previousPos = p.position()
	}
	if p.config.AllowTripleQuotedStrings {
		v, err := p.readTripleQuotedString()
		if err != nil {
			return nil, err
		}
		if v != nil {
			values = append(values, v)
			previousPos = p.position()
		}
	}
	for p.consume('"') || p.consume('\'') {
		// Handle string value.
		stringBegin := p.index - 1 // Index of the quote.
		i := p.index
		for ; i < p.length; i++ {
			if p.in[i] == '\\' {
				i++ // Skip escaped char.
				continue
			}
			if p.in[i] == '\n' {
				p.index = i
				return nil, fmt.Errorf("found literal (unescaped) new line in string at %s", p.errorContext())
			}
			if p.in[i] == p.in[stringBegin] {
				var vl string
				if p.config.SmartQuotes {
					vl = smartQuotes(p.advance(i))
				} else {
					vl = fixQuotes(p.advance(i))
				}
				_ = p.advance(i + 1) // Skip the quote.
				values = append(values, p.populateValue(vl, preComments))

				previousPos = p.position()
				preComments, _ = p.skipWhiteSpaceAndReadComments(true /* multiLine */)
				break
			}
		}
		if i == p.length {
			p.index = i
			return nil, fmt.Errorf("unfinished string at %s", p.errorContext())
		}
	}
	if previousPos != (ast.Position{}) {
		// Rewind comments.
		p.index = int(previousPos.Byte)
		p.line = int(previousPos.Line)
		p.column = int(previousPos.Column)
	} else {
		i := p.index
		// Handle other values.
		for ; i < p.length; i++ {
			if p.isValueSep(i) {
				break
			}
		}
		vl := p.advance(i)
		values = append(values, p.populateValue(vl, preComments))
	}
	if p.config.infoLevel() {
		p.config.infof("values: %v", values)
	}
	return values, nil
}

func (p *parser) readTripleQuotedString() (*ast.Value, error) {
	start := p.index
	stringBegin := p.index
	delimiter := `"""`
	if !p.consumeString(delimiter) {
		delimiter = `'''`
		if !p.consumeString(delimiter) {
			return nil, nil
		}
	}

	for {
		if p.consumeString(delimiter) {
			break
		}
		if p.index == p.length {
			p.index = start
			return nil, fmt.Errorf("unfinished string at %s", p.errorContext())
		}
		p.index++
	}

	v := p.populateValue(string(p.in[stringBegin:p.index]), nil)

	return v, nil
}

func (p *parser) populateValue(vl string, preComments []string) *ast.Value {
	if p.config.infoLevel() {
		p.config.infof("value: %q", vl)
	}
	return &ast.Value{
		Value:         vl,
		InlineComment: p.readInlineComment(),
		PreComments:   preComments,
	}
}

func (p *parser) readInlineComment() string {
	inlineComment, _ := p.skipWhiteSpaceAndReadComments(false /* multiLine */)
	if p.config.infoLevel() {
		p.config.infof("inlineComment: %q", strings.Join(inlineComment, "\n"))
	}
	if len(inlineComment) > 0 {
		return inlineComment[0]
	}
	return ""
}

func (p *parser) readTemplate() string {
	if !p.nextInputIs('%') {
		return ""
	}
	i := p.index + 1
	for ; i < p.length; i++ {
		if p.in[i] == '"' || p.in[i] == '\'' {
			stringBegin := i // Index of quote.
			i++
			for ; i < p.length; i++ {
				if p.in[i] == '\\' {
					i++ // Skip escaped char.
					continue
				}
				if p.in[i] == p.in[stringBegin] {
					i++ // Skip end quote.
					break
				}
			}
		}
		if i < p.length && p.in[i] == '%' {
			i++
			break
		}
	}
	return p.advance(i)
}

// NodeSortFunction sorts the given nodes, using the parent node as context. parent can be nil.
type NodeSortFunction func(parent *ast.Node, nodes []*ast.Node) error

// NodeFilterFunction filters the given nodes.
type NodeFilterFunction func(nodes []*ast.Node)

// ValuesSortFunction sorts the given values.
type ValuesSortFunction func(values []*ast.Value)

func sortAndFilterNodes(parent *ast.Node, nodes []*ast.Node, sortFunction NodeSortFunction, filterFunction NodeFilterFunction, valuesSortFunction ValuesSortFunction) error {
	if len(nodes) == 0 {
		return nil
	}
	if filterFunction != nil {
		filterFunction(nodes)
	}
	for _, nd := range nodes {
		err := sortAndFilterNodes(nd, nd.Children, sortFunction, filterFunction, valuesSortFunction)
		if err != nil {
			return err
		}
		if valuesSortFunction != nil && nd.ValuesAsList {
			valuesSortFunction(nd.Values)
		}
	}
	if sortFunction != nil {
		return sortFunction(parent, nodes)
	}
	return nil
}

// RemoveDuplicates marks duplicate key:value pairs from nodes as Deleted.
func RemoveDuplicates(nodes []*ast.Node) {
	type nameAndValue struct {
		name, value string
	}
	seen := make(map[nameAndValue]bool)
	for _, nd := range nodes {
		if seen != nil && len(nd.Values) == 1 {
			key := nameAndValue{nd.Name, nd.Values[0].Value}
			if _, value := seen[key]; value {
				// Name-Value pair found in the same nesting level, deleting.
				nd.Deleted = true
			} else {
				seen[key] = true
			}
		}
	}
}

func wrapStrings(nodes []*ast.Node, depth int, c Config) error {
	if c.WrapStringsAtColumn == 0 && !c.WrapStringsAfterNewlines {
		return nil
	}
	for _, nd := range nodes {
		if nd.ChildrenSameLine {
			continue
		}
		if c.WrapStringsAtColumn > 0 && needsWrappingAtColumn(nd, depth, c) {
			if err := wrapLinesAtColumn(nd, depth, c); err != nil {
				return err
			}
		}
		if c.WrapStringsAfterNewlines && needsWrappingAfterNewlines(nd, c) {
			if err := wrapLinesAfterNewlines(nd, c); err != nil {
				return err
			}
		}
		if err := wrapStrings(nd.Children, depth+1, c); err != nil {
			return err
		}
	}
	return nil
}

func needsWrappingAtColumn(nd *ast.Node, depth int, c Config) bool {
	// Even at depth 0 we have a 2-space indent when the wrapped string is rendered on the line below
	// the field name.
	const lengthBuffer = 2
	maxLength := c.WrapStringsAtColumn - lengthBuffer - (depth * len(indentSpaces))

	if !c.WrapHTMLStrings {
		for _, v := range nd.Values {
			if tagRegex.Match([]byte(v.Value)) {
				return false
			}
		}
	}

	for _, v := range nd.Values {
		if len(v.Value) >= 3 && (strings.HasPrefix(v.Value, `'''`) || strings.HasPrefix(v.Value, `"""`)) {
			// Don't wrap triple-quoted strings
			return false
		}
		if len(v.Value) > 0 && v.Value[0] != '\'' && v.Value[0] != '"' {
			// Only wrap strings
			return false
		}
		if len(v.Value) > maxLength {
			return true
		}
	}
	return false
}

// If the Values of this Node constitute a string, and if Config.WrapStringsAtColumn > 0, then wrap
// the string so each line is within the specified columns. Wraps only the current Node (does not
// recurse into Children).
func wrapLinesAtColumn(nd *ast.Node, depth int, c Config) error {
	// This function looks at the unquoted ast.Value.Value string (i.e., with each Value's wrapping
	// quote chars removed). We need to remove these quotes, since otherwise they'll be re-flowed into
	// the body of the text.
	lengthBuffer := 4 // Even at depth 0 we have a 2-space indent and a pair of quotes
	maxLength := c.WrapStringsAtColumn - lengthBuffer - (depth * len(indentSpaces))

	str, err := unquote.Raw(nd)
	if err != nil {
		return fmt.Errorf("skipping string wrapping on node %q (error unquoting string): %v", nd.Name, err)
	}

	// Remove one from the max length since a trailing space may be added below.
	wrappedStr := wordwrap.WrapString(str, uint(maxLength)-1)
	lines := strings.Split(wrappedStr, "\n")
	newValues := make([]*ast.Value, 0, len(lines))
	// The Value objects have more than just the string in them. They also have any leading and
	// trailing comments. To maintain these comments we recycle the existing Value objects if
	// possible.
	var i int
	var line string
	for i, line = range lines {
		var v *ast.Value
		if i < len(nd.Values) {
			v = nd.Values[i]
		} else {
			v = &ast.Value{}
		}
		if i < len(lines)-1 {
			line = line + " "
		}
		v.Value = fmt.Sprintf(`"%s"`, line)
		newValues = append(newValues, v)
	}

	postWrapCollectComments(nd, i)

	nd.Values = newValues
	return nil
}

// N.b.: this will incorrectly match `\\\\x`, which hopefully is rare.
var byteEscapeRegex = regexp.MustCompile(`\\x`)

func needsWrappingAfterNewlines(nd *ast.Node, c Config) bool {
	for _, v := range nd.Values {
		if len(v.Value) >= 3 && (strings.HasPrefix(v.Value, `'''`) || strings.HasPrefix(v.Value, `"""`)) {
			// Don't wrap triple-quoted strings
			return false
		}
		if len(v.Value) > 0 && v.Value[0] != '\'' && v.Value[0] != '"' {
			// Only wrap strings
			return false
		}
		byteEscapeCount := len(byteEscapeRegex.FindAllStringIndex(v.Value, -1))
		if float64(byteEscapeCount) > float64(len(v.Value))*0.1 {
			// Only wrap UTF-8 looking strings (where less than ~10% of the characters are escaped).
			return false
		}
		// Check that there is at least one newline, *not* at the end of the string.
		if i := strings.Index(v.Value, `\n`); i >= 0 && i < len(v.Value)-3 {
			return true
		}
	}
	return false
}

// If the Values of this Node constitute a string, and if Config.WrapStringsAfterNewlines,
// then wrap the string so each line ends with a newline.
// Wraps only the current Node (does not recurse into Children).
func wrapLinesAfterNewlines(nd *ast.Node, c Config) error {
	str, err := unquote.Raw(nd)
	if err != nil {
		return fmt.Errorf("skipping string wrapping on node %q (error unquoting string): %v", nd.Name, err)
	}

	wrappedStr := strings.ReplaceAll(str, `\n`, `\n`+"\n")
	// Avoid empty string at end after splitting in case str ended with an (escaped) newline.
	wrappedStr = strings.TrimSuffix(wrappedStr, "\n")
	lines := strings.Split(wrappedStr, "\n")
	newValues := make([]*ast.Value, 0, len(lines))
	// The Value objects have more than just the string in them. They also have any leading and
	// trailing comments. To maintain these comments we recycle the existing Value objects if
	// possible.
	var i int
	var line string
	for i, line = range lines {
		var v *ast.Value
		if i < len(nd.Values) {
			v = nd.Values[i]
		} else {
			v = &ast.Value{}
		}
		v.Value = fmt.Sprintf(`"%s"`, line)
		newValues = append(newValues, v)
	}

	postWrapCollectComments(nd, i)

	nd.Values = newValues
	return nil
}

func postWrapCollectComments(nd *ast.Node, i int) {
	for i++; i < len(nd.Values); i++ {
		// If this executes, then the text was wrapped into less lines of text (less Values) than
		// previously. If any of these had comments on them, we collect them so they are not lost.
		v := nd.Values[i]
		nd.PostValuesComments = append(nd.PostValuesComments, v.PreComments...)
		if len(v.InlineComment) > 0 {
			nd.PostValuesComments = append(nd.PostValuesComments, v.InlineComment)
		}
	}
}

func fixQuotes(s string) string {
	res := make([]byte, 0, len(s))
	res = append(res, '"')
	for i := 0; i < len(s); i++ {
		if s[i] == '"' {
			res = append(res, '\\')
		} else if s[i] == '\\' {
			res = append(res, s[i])
			i++
		}
		res = append(res, s[i])
	}
	res = append(res, '"')
	return string(res)
}

func unescapeQuotes(s string) string {
	res := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		// If we hit an escape sequence...
		if s[i] == '\\' {
			// ... keep the backslash unless it's in front of a quote ...
			if i == len(s)-1 || (s[i+1] != '"' && s[i+1] != '\'') {
				res = append(res, '\\')
			}
			// ... then point at the escaped character so it is output verbatim below.
			// Doing this within the loop (without "continue") ensures correct handling
			// of escaped backslashes.
			i++
		}
		if i < len(s) {
			res = append(res, s[i])
		}
	}
	return string(res)
}

func smartQuotes(s string) string {
	s = unescapeQuotes(s)
	if strings.Contains(s, "\"") && !strings.Contains(s, "'") {
		// If we hit this branch, the string doesn't contain any single quotes, and
		// is being wrapped in single quotes, so no escaping is needed.
		return "'" + s + "'"
	}
	// fixQuotes will wrap the string in double quotes, but will escape any
	// double quotes that appear within the string.
	return fixQuotes(s)
}

// DebugFormat returns a textual representation of the specified nodes for
// consumption by humans when debugging (e.g. in test failures). No guarantees
// are made about the specific output.
func DebugFormat(nodes []*ast.Node, depth int) string {
	res := []string{""}
	prefix := strings.Repeat(".", depth)
	for _, nd := range nodes {
		var value string
		if nd.Deleted {
			res = append(res, "DELETED")
		}
		if nd.Children != nil { // Also for 0 children.
			value = fmt.Sprintf("children:%s", DebugFormat(nd.Children, depth+1))
		} else {
			value = fmt.Sprintf("values: %v\n", nd.Values)
		}
		res = append(res,
			fmt.Sprintf("name: %q", nd.Name),
			fmt.Sprintf("PreComments: %q (len %d)", strings.Join(nd.PreComments, "\n"), len(nd.PreComments)),
			value)
	}
	return strings.Join(res, fmt.Sprintf("\n%s ", prefix))
}

// Pretty formats the nodes at the given indentation depth (0 = top-level).
func Pretty(nodes []*ast.Node, depth int) string {
	var result strings.Builder
	formatter{&result}.writeNodes(removeDeleted(nodes), depth, false /* isSameLine */, false /* asListItems */)
	return result.String()
}

// PrettyBytes returns formatted nodes at the given indentation depth (0 = top-level) as bytes.
func PrettyBytes(nodes []*ast.Node, depth int) []byte {
	var result bytes.Buffer
	formatter{&result}.writeNodes(removeDeleted(nodes), depth, false /* isSameLine */, false /* asListItems */)
	return result.Bytes()
}

// UnsortedFieldCollector collects UnsortedFields during parsing.
type UnsortedFieldCollector struct {
	fields map[string]UnsortedField
}

func newUnsortedFieldCollector() *UnsortedFieldCollector {
	return &UnsortedFieldCollector{
		fields: make(map[string]UnsortedField),
	}
}

// UnsortedFieldCollectorFunc collects UnsortedFields during parsing.
type UnsortedFieldCollectorFunc func(name string, line int32, parent string)

func (ufc *UnsortedFieldCollector) collect(name string, line int32, parent string) {
	ufc.fields[name] = UnsortedField{name, line, parent}
}

func (ufc *UnsortedFieldCollector) asError() error {
	if len(ufc.fields) == 0 {
		return nil
	}
	var fields []UnsortedField
	for _, f := range ufc.fields {
		fields = append(fields, f)
	}
	return &UnsortedFieldsError{fields}
}

func nodeSortFunction(c Config) NodeSortFunction {
	var sorter ast.NodeLess = nil
	unsortedFieldCollector := newUnsortedFieldCollector()
	for name, fieldOrder := range c.fieldSortOrder {
		sorter = ast.ChainNodeLess(sorter, ByFieldOrder(name, fieldOrder, unsortedFieldCollector.collect))
	}
	if c.SortFieldsByFieldName {
		sorter = ast.ChainNodeLess(sorter, ast.ByFieldName)
	}
	if c.SortRepeatedFieldsByContent {
		sorter = ast.ChainNodeLess(sorter, ast.ByFieldValue)
	}
	for _, sf := range c.SortRepeatedFieldsBySubfield {
		field, subfield := parseSubfieldSpec(sf)
		if subfield != "" {
			sorter = ast.ChainNodeLess(sorter, ast.ByFieldSubfield(field, subfield))
		}
	}
	if sorter != nil {
		return func(parent *ast.Node, ns []*ast.Node) error {
			ast.SortNodes(parent, ns, sorter)
			if c.RequireFieldSortOrderToMatchAllFieldsInNode {
				return unsortedFieldCollector.asError()
			}
			return nil
		}
	}
	return nil
}

// Returns the field and subfield parts of spec "{field}.{subfield}".
// Spec without a dot is considered to be "{subfield}".
func parseSubfieldSpec(subfieldSpec string) (field string, subfield string) {
	parts := strings.SplitN(subfieldSpec, ".", 2)
	if len(parts) == 1 {
		return "", parts[0]
	}
	return parts[0], parts[1]
}

func nodeFilterFunction(c Config) NodeFilterFunction {
	if c.RemoveDuplicateValuesForRepeatedFields {
		return RemoveDuplicates
	}
	return nil
}

func valuesSortFunction(c Config) ValuesSortFunction {
	if c.SortRepeatedFieldsByContent {
		return ast.SortValues
	}
	return nil
}

func getNodePriorityForByFieldOrder(parent, node *ast.Node, name string, priorities map[string]int, unsortedCollector UnsortedFieldCollectorFunc) *int {
	if parent != nil && parent.Name != name {
		return nil
	}
	if parent == nil && name != RootName {
		return nil
	}
	// CommentOnly nodes don't set priority below, and default to MaxInt, which keeps them at the bottom
	prio := math.MaxInt

	// Unknown fields will get the int nil value of 0 from the order map, and bubble to the top.
	if !node.IsCommentOnly() {
		var ok bool
		prio, ok = priorities[node.Name]
		if !ok {
			unsortedCollector(node.Name, node.Start.Line, parent.Name)
		}
	}
	return &prio
}

// ByFieldOrder returns a NodeLess function that orders fields within a node named name
// by the order specified in fieldOrder. Nodes sorted but not specified by the field order
// are bubbled to the top and reported to unsortedCollector.
func ByFieldOrder(name string, fieldOrder []string, unsortedCollector UnsortedFieldCollectorFunc) ast.NodeLess {
	priorities := make(map[string]int)
	for i, fieldName := range fieldOrder {
		priorities[fieldName] = i + 1
	}
	return func(parent, ni, nj *ast.Node, isWholeSlice bool) bool {
		if !isWholeSlice {
			return false
		}
		vi := getNodePriorityForByFieldOrder(parent, ni, name, priorities, unsortedCollector)
		vj := getNodePriorityForByFieldOrder(parent, nj, name, priorities, unsortedCollector)
		if vi == nil {
			return vj != nil
		}
		if vj == nil {
			return false
		}
		return *vi < *vj
	}
}

// stringWriter abstracts over bytes.Buffer and strings.Builder
type stringWriter interface {
	WriteString(s string) (int, error)
}

// formatter accumulates pretty-printed textproto contents into a stringWriter.
type formatter struct {
	stringWriter
}

func (f formatter) writeNodes(nodes []*ast.Node, depth int, isSameLine, asListItems bool) {
	indent := " "
	if !isSameLine {
		indent = strings.Repeat(indentSpaces, depth)
	}

	lastNonCommentIndex := 0
	if asListItems {
		for i := len(nodes) - 1; i >= 0; i-- {
			if !nodes[i].IsCommentOnly() {
				lastNonCommentIndex = i
				break
			}
		}
	}

	for index, nd := range nodes {
		for _, comment := range nd.PreComments {
			if len(comment) == 0 {
				if !(depth == 0 && index == 0) {
					f.WriteString("\n")
				}
				continue
			}
			f.WriteString(indent)
			f.WriteString(comment)
			f.WriteString("\n")
		}

		if nd.IsCommentOnly() {
			// The comments have been printed already, no more work to do.
			continue
		}
		f.WriteString(indent)
		// Node name may be empty in alternative-style textproto files, because they
		// contain a sequence of proto messages of the same type:
		//   { name: "first_msg" }
		//   { name: "second_msg" }
		// In all other cases, nd.Name is not empty and should be printed.
		if nd.Name != "" {
			f.WriteString(nd.Name)
			if !nd.SkipColon {
				f.WriteString(":")
			}

			// The space after the name is required for one-liners and message fields:
			//   title: "there was a space here"
			//   metadata: { ... }
			// In other cases, there is a newline right after the colon, so no space required.
			if nd.Children != nil || (len(nd.Values) == 1 && len(nd.Values[0].PreComments) == 0) || nd.ValuesAsList {
				f.WriteString(" ")
			}
		}

		if nd.ValuesAsList { // For ValuesAsList option we will preserve even empty list  `field: []`
			f.writeValuesAsList(nd, nd.Values, indent+indentSpaces)
		} else if len(nd.Values) > 0 {
			f.writeValues(nd, nd.Values, indent+indentSpaces)
		}
		if nd.Children != nil { // Also for 0 Children.
			if nd.ChildrenAsList {
				f.writeChildrenAsListItems(nd.Children, depth+1, isSameLine || nd.ChildrenSameLine)
			} else {
				f.writeChildren(nd.Children, depth+1, isSameLine || nd.ChildrenSameLine, nd.IsAngleBracket)
			}
		}

		if asListItems && index < lastNonCommentIndex {
			f.WriteString(",")
		}

		if (nd.Children != nil || nd.ValuesAsList) && len(nd.ClosingBraceComment) > 0 {
			f.WriteString(indentSpaces)
			f.WriteString(nd.ClosingBraceComment)
		}

		if !isSameLine {
			f.WriteString("\n")
		}
	}
}

func (f formatter) writeValues(nd *ast.Node, vals []*ast.Value, indent string) {
	if len(vals) == 0 {
		// This should never happen: formatValues can be called only if there are some values.
		return
	}
	sep := "\n" + indent
	if len(vals) == 1 && len(vals[0].PreComments) == 0 {
		sep = ""
	}
	for _, v := range vals {
		f.WriteString(sep)
		for _, comment := range v.PreComments {
			f.WriteString(comment)
			f.WriteString(sep)
		}
		f.WriteString(v.Value)
		if len(v.InlineComment) > 0 {
			f.WriteString(indentSpaces)
			f.WriteString(v.InlineComment)
		}
	}
	for _, comment := range nd.PostValuesComments {
		f.WriteString(sep)
		f.WriteString(comment)
	}
}

func (f formatter) writeValuesAsList(nd *ast.Node, vals []*ast.Value, indent string) {
	// Checks if it's possible to put whole list in a single line.
	sameLine := nd.ChildrenSameLine && len(nd.PostValuesComments) == 0
	if sameLine {
		// Parser found all children on a same line, but we need to check again.
		// It's possible that AST was modified after parsing.
		for _, val := range vals {
			if len(val.PreComments) > 0 || len(vals[0].InlineComment) > 0 {
				sameLine = false
				break
			}
		}
	}
	sep := ""
	if !sameLine {
		sep = "\n" + indent
	}
	f.WriteString("[")

	for idx, v := range vals {
		for _, comment := range v.PreComments {
			f.WriteString(sep)
			f.WriteString(comment)
		}
		f.WriteString(sep)
		f.WriteString(v.Value)
		if idx < len(vals)-1 { // Don't put trailing comma that fails Python parser.
			f.WriteString(",")
			if sameLine {
				f.WriteString(" ")
			}
		}
		if len(v.InlineComment) > 0 {
			f.WriteString(indentSpaces)
			f.WriteString(v.InlineComment)
		}
	}
	for _, comment := range nd.PostValuesComments {
		f.WriteString(sep)
		f.WriteString(comment)
	}
	f.WriteString(strings.Replace(sep, indentSpaces, "", 1))
	f.WriteString("]")
}

// writeChildren writes the child nodes. The result always ends with a closing brace.
func (f formatter) writeChildren(children []*ast.Node, depth int, sameLine, isAngleBracket bool) {
	openBrace := "{"
	closeBrace := "}"
	if isAngleBracket {
		openBrace = "<"
		closeBrace = ">"
	}
	switch {
	case sameLine && len(children) == 0:
		f.WriteString(openBrace + closeBrace)
	case sameLine:
		f.WriteString(openBrace)
		f.writeNodes(children, depth, sameLine, false /* asListItems */)
		f.WriteString(" " + closeBrace)
	default:
		f.WriteString(openBrace + "\n")
		f.writeNodes(children, depth, sameLine, false /* asListItems */)
		f.WriteString(strings.Repeat(indentSpaces, depth-1))
		f.WriteString(closeBrace)
	}
}

// writeChildrenAsListItems writes the child nodes as list items.
func (f formatter) writeChildrenAsListItems(children []*ast.Node, depth int, sameLine bool) {
	openBrace := "["
	closeBrace := "]"
	switch {
	case sameLine && len(children) == 0:
		f.WriteString(openBrace + closeBrace)
	case sameLine:
		f.WriteString(openBrace)
		f.writeNodes(children, depth, sameLine, true /* asListItems */)
		f.WriteString(" " + closeBrace)
	default:
		f.WriteString(openBrace + "\n")
		f.writeNodes(children, depth, sameLine, true /* asListItems */)
		f.WriteString(strings.Repeat(indentSpaces, depth-1))
		f.WriteString(closeBrace)
	}
}
