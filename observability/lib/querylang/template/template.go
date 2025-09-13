package template

import "fmt"

// Simple template representation.
// Strings are just strings and Identifiers are references to labels that should be wrapped by Strings.
// Example: "Hello, {{name}}!" -> Template{strings: []string{"Hello, ", "!"}, identifiers: []string{"name"}}
// So there is one simple law: forall t in Template . len(t.strings) == len(t.identifiers) + 1
type Template struct {
	strings     []string
	identifiers []string
}

type templateParserState int

const (
	templateParserStateNormal templateParserState = iota
	templateParserStateTemplate
)

func (t *Template) Parse(repr string) error {
	var (
		strings     = make([]string, 0)
		identifiers = make([]string, 0)
		current     string
		prev        rune
		state       = templateParserStateNormal
	)

	for _, char := range repr {
		if state == templateParserStateNormal {
			if char == '{' && prev == '{' {
				state = templateParserStateTemplate
				strings = append(strings, current)
				current = ""
			} else if char == '{' {
				// suspend addition to current string until we see that there isn't a second '{'
			} else if prev == '{' {
				current += "{" + string(char)
			} else {
				current += string(char)
			}
		} else if state == templateParserStateTemplate {
			if char == '}' && prev == '}' {
				if current == "" {
					return fmt.Errorf("empty identifier")
				}
				state = templateParserStateNormal
				identifiers = append(identifiers, current)
				current = ""
			} else if char == '}' {
				// suspend error until we see that there isn't a second '}'
			} else if char == '{' {
				return fmt.Errorf("unexpected \"{\" in identifier")
			} else if prev == '}' {
				return fmt.Errorf("unexpected \"}\" in identifier")
			} else {
				current += string(char)
			}
		}
		prev = char
	}

	if state == templateParserStateTemplate {
		return fmt.Errorf("unexpected end of template")
	}

	if current != "" {
		strings = append(strings, current)
	} else if len(strings) < len(identifiers)+1 {
		strings = append(strings, "")
	}

	if prev == '{' {
		strings[len(strings)-1] += "{"
	}

	t.strings = strings
	t.identifiers = identifiers

	return nil
}

func (t *Template) Repr() string {
	repr := ""
	for i, str := range t.strings {
		repr += str
		if i < len(t.identifiers) {
			repr += "{{" + t.identifiers[i] + "}}"
		}
	}
	return repr
}

func (t *Template) Needs() []string {
	return t.identifiers
}

func (t *Template) Apply(labelValues map[string]string) (string, bool) {
	result := ""
	for i, str := range t.strings {
		result += str
		if i < len(t.identifiers) {
			value, ok := labelValues[t.identifiers[i]]
			if !ok {
				return "", false
			}
			result += value
		}
	}
	return result, true
}
