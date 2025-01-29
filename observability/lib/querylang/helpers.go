package querylang

import (
	"github.com/yandex/perforator/observability/lib/querylang/operator"
)

func (c *Condition) IsStrictEq() bool {
	return c.Operator == operator.Eq && !c.Inverse
}

func (c *Condition) IsEqOrNotEqOrExists() bool {
	return c.Operator == operator.Eq || c.Operator == operator.Exists
}
