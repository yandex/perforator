package operator

type Operator int

const (
	Eq Operator = iota
	LT
	LTE
	GT
	GTE
	Regex
	Exists
	Glob
)

func (op Operator) IsOrderingOperator() bool {
	switch op {
	case LT, LTE, GT, GTE:
		return true
	}
	return false
}

func Repr(op Operator, inverse bool) string {
	switch op {
	case LT:
		return "<"
	case LTE:
		return "<="
	case GT:
		return ">"
	case GTE:
		return ">="
	case Regex:
		if inverse {
			return "!regex"
		}
		return "regex"
	case Glob:
		if inverse {
			return "!glob"
		}
		return "glob"
	case Exists:
		if inverse {
			return "!exists"
		}
		return "exists"
	case Eq:
		if inverse {
			return "!="
		}
		return "="
	default:
		return "unknown_operator"
	}
}

func (op Operator) String() string {
	switch op {
	case Eq:
		return "Equals"
	case LT:
		return "Less than"
	case LTE:
		return "Less than or equal"
	case GT:
		return "Greater than"
	case GTE:
		return "Greater than or equal"
	case Regex:
		return "Regex"
	case Exists:
		return "Exists"
	case Glob:
		return "Glob"
	default:
		return "Unknown"
	}
}
