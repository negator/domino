package domino

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	eq  = "="
	neq = "<>"
	lt  = "<"
	lte = "<="
	gt  = ">"
	gte = ">="
)

var nonalpha *regexp.Regexp = regexp.MustCompile("[^a-zA-Z_0-9]")

func generatePlaceholder(a interface{}, counter int) string {
	r := fmt.Sprintf("%v_%v", a, counter)
	return ":" + nonalpha.ReplaceAllString(r, "_")
}

type Expression interface {
	construct(int) (string, map[string]interface{}, int)
}

type UpdateExpression struct {
	set    *func(counter int) (string, map[string]interface{}, int)
	add    *func(counter int) (string, map[string]interface{}, int)
	remove *func(counter int) (string, map[string]interface{}, int)
}

func (field *dynamoField) SetField(a interface{}, onlyIfEmpty bool) *UpdateExpression {
	f := func(c int) (string, map[string]interface{}, int) {
		ph := generatePlaceholder(a, c)
		r := ph
		if onlyIfEmpty {
			r = fmt.Sprintf("if_not_exists(%v,%v)", field.Name, ph)
		}
		s := field.name + " = " + r
		m := map[string]interface{}{
			ph: a,
		}
		c++
		return s, m, c
	}
	return &UpdateExpression{set: &f}
}

func (field *dynamoFieldNumeric) Add(amount float64) *UpdateExpression {
	f := func(c int) (string, map[string]interface{}, int) {
		ph := generatePlaceholder(amount, c)
		s := field.name + " " + ph
		m := map[string]interface{}{ph: amount}
		c++
		return s, m, c
	}
	return &UpdateExpression{add: &f}
}
func (field *dynamoCollectionField) Remove(s string) *UpdateExpression {
	f := func(c int) (string, map[string]interface{}, int) {
		c++
		m := make(map[string]interface{})
		return s, m, c
	}
	return &UpdateExpression{add: &f}
}

func (field *dynamoFieldNumeric) Increment(by int) *UpdateExpression {
	return field.Add(float64(by))
}

func (field *dynamoFieldNumeric) Decrement(by int) *UpdateExpression {
	return field.Add(float64(-by))
}

/*********************************************************************************/
/******************************** ExpressionGroups *******************************/
/*********************************************************************************/
/*Groups expression by AND and OR operators, i.e. <expr> OR <expr>*/

type expressionGroup struct {
	expressions []Expression
	op          string
}

func (e expressionGroup) construct(counter int) (string, map[string]interface{}, int) {
	a := e.expressions
	m := make(map[string]interface{})
	r := "("

	for i := 0; i < len(a); i++ {
		if i > 0 {
			r += " " + e.op + " "
		}
		substring, placeholders, newCounter := a[i].construct(counter)
		r += substring
		for k, v := range placeholders {
			m[k] = v
		}

		counter = newCounter
	}

	r += ")"
	return r, m, counter
}

func Or(c ...Expression) expressionGroup {
	return expressionGroup{
		c,
		"OR",
	}
}
func And(c ...Expression) expressionGroup {
	return expressionGroup{
		c,
		"AND",
	}
}

func (c expressionGroup) String() string {
	s, _, _ := c.construct(0)
	return s
}

/*********************************************************************************/
/******************************** Negation Expression ****************************/
/*********************************************************************************/
type negation struct {
	expression Expression
}

func (n negation) construct(counter int) (string, map[string]interface{}, int) {
	s, m, c := n.expression.construct(counter)
	r := "(NOT " + s + ")"
	return r, m, c
}

func (c negation) String() string {
	s, _, _ := c.construct(0)
	return s
}

func Not(c Expression) negation {
	return negation{c}
}

/*********************************************************************************/
/******************************** Conditions *************************************/
/*********************************************************************************/
/*Conditions that only apply to keys*/

type condition struct {
	exprF func([]string) string
	args  []interface{}
}

func (c condition) construct(counter int) (string, map[string]interface{}, int) {
	a := make([]string, len(c.args))
	m := make(map[string]interface{})
	for i, b := range c.args {
		a[i] = generatePlaceholder(b, counter)
		m[a[i]] = b
		counter++
	}
	s := c.exprF(a)
	return s, m, counter

}
func (c condition) String() string {
	s, _, _ := c.construct(0)
	return s
}

func (p *dynamoField) In(elems ...interface{}) condition {
	return condition{
		exprF: func(placeholders []string) string {
			return p.name + " in (" + strings.Join(placeholders, ",") + ")"
		},
		args: elems,
	}

}

func (p *dynamoField) Exists() condition {
	return condition{
		exprF: func(placeholders []string) string {
			return "attribute_exists(" + p.name + ")"
		},
	}
}

func (p *dynamoField) NotExists() condition {
	return condition{
		exprF: func(placeholders []string) string {
			return "attribute_not_exists(" + p.name + ")"
		},
	}
}

func (p *dynamoField) Contains(a interface{}) condition {
	return condition{
		exprF: func(placeholders []string) string {
			return fmt.Sprintf("contains("+p.name+",%v)", placeholders[0])
		},
		args: []interface{}{a},
	}
}

func (p *dynamoField) Size(op string, a interface{}) condition {
	return condition{
		exprF: func(placeholders []string) string {
			return fmt.Sprintf("size("+p.name+") "+op+"%v", placeholders[0])
		},
		args: []interface{}{a},
	}
}

/*********************************************************************************/
/******************************** Key Conditions *********************************/
/*********************************************************************************/
type KeyCondition struct {
	condition
}

func (p *dynamoField) operation(op string, a interface{}) KeyCondition {
	return KeyCondition{
		condition{
			exprF: func(placeholders []string) string {
				return fmt.Sprintf("("+p.name+" "+op+" %v)", placeholders[0])
			},
			args: []interface{}{a},
		},
	}
}
func (p *dynamoField) Equals(a interface{}) KeyCondition {
	return p.operation(eq, a)
}
func (p *dynamoField) NotEquals(a interface{}) KeyCondition {
	return p.operation(neq, a)
}
func (p *dynamoField) LessThan(a interface{}) KeyCondition {
	return p.operation(lt, a)
}
func (p *dynamoField) LessThanOrEq(a interface{}) KeyCondition {
	return p.operation(lte, a)
}
func (p *dynamoField) GreaterThan(a interface{}) KeyCondition {
	return p.operation(gt, a)
}
func (p *dynamoField) GreaterThanOrEq(a interface{}) KeyCondition {
	return p.operation(gte, a)
}

func (p *dynamoField) BeginsWith(a interface{}) KeyCondition {
	return KeyCondition{
		condition{
			exprF: func(placeholders []string) string {
				return fmt.Sprintf("begins_with("+p.name+",%v)", placeholders[0])
			},
			args: []interface{}{a},
		},
	}
}

func (p *dynamoField) Between(a interface{}, b interface{}) KeyCondition {
	return KeyCondition{
		condition{
			exprF: func(placeholders []string) string {
				return fmt.Sprintf("("+p.name+" between(%v, %v))", placeholders[0], placeholders[1])
			},
			args: []interface{}{a, b},
		},
	}
}
