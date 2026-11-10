package condition

type ConditionType int

const (
	Eq ConditionType = iota
	LT
	LTE
	GT
	GTE
	Regex
	Exists
	Glob
	ISubstring
)

func (c ConditionType) IsOperator() bool {
	switch c {
	case Eq, LT, LTE, GT, GTE, Regex, Exists, Glob, ISubstring:
		return true
	default:
		return false
	}
}

func (c ConditionType) IsOrderingOperator() bool {
	switch c {
	case LT, LTE, GT, GTE:
		return true
	}
	return false
}

func Repr(op ConditionType, inverse bool) string {
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
	case ISubstring:
		if inverse {
			return "!isubstring"
		}
		return "isubstring"
	default:
		return "unknown_operator"
	}
}

func (c ConditionType) String() string {
	switch c {
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
		return "*"
	case Glob:
		return "Glob"
	case ISubstring:
		return "ISubstring"
	default:
		return "Unknown"
	}
}
