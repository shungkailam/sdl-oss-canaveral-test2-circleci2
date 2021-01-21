package filter

import (
	"cloudservices/common/errcode"
	"fmt"
	"strconv"
	"strings"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/repr"
)

// A simple parser based on participle for a simplified subset of
// SQL where clause conditions.
//

// reference

type Boolean bool

func (b *Boolean) Capture(values []string) error {
	*b = strings.ToUpper(values[0]) == "TRUE"
	return nil
}

// Expression root of filter expression AST using participle parser
type Expression struct {
	Or []*OrCondition `@@ { "OR" @@ }`
}

type OrCondition struct {
	And []*Condition `@@ { "AND" @@ }`
}

type Condition struct {
	Operand       *ConditionOperand `  @@`
	SubExpression *Expression       `| "(" @@ ")"`
}

type ConditionOperand struct {
	Operand      string        `@Ident`
	ConditionRHS *ConditionRHS `@@`
}

type ConditionRHS struct {
	Compare *Compare `  @@`
	Between *Between `|  @@`
	Like    *Like    `|  @@`
	In      *In      `|  @@`
}

type Compare struct {
	Operator string `@( "<>" | "<=" | ">=" | "=" | "<" | ">" | "!=" )`
	Value    *Value `@@`
}

type Like struct {
	Not   bool   `[ @"NOT" ]`
	Value *Value ` "LIKE" @@`
}

type Between struct {
	Not   bool   `[ @"NOT" ]`
	Start *Value `"BETWEEN" @@`
	End   *Value `"AND" @@`
}

type In struct {
	Not    bool     `[ @"NOT" ]`
	Values []*Value `"IN" "(" @@ { "," @@ } ")"`
}

type Value struct {
	Float   *float64 ` ( @Float`
	Integer *int64   ` | @Integer`
	String  *string  ` | @String`
	Boolean *Boolean ` | @("TRUE" | "FALSE") )`
}

var (
	filterLexer = lexer.Must(lexer.Regexp(`(\s+)` +
		`|(?P<Keyword>(?i)TRUE|FALSE|BETWEEN|NOT|AND|OR|LIKE)` +
		`|(?P<Ident>[a-zA-Z_][a-zA-Z0-9_]*)` +
		`|(?P<Float>[-+]?\d*\.\d+([eE][-+]?\d+)?)` +
		`|(?P<Integer>[-+]?\d+)` +
		`|(?P<String>'[^']*'|"[^"]*")` +
		`|(?P<Operators><>|!=|<=|>=|[()=<>,])`,
	))
	filterParser = participle.MustBuild(
		&Expression{},
		participle.Lexer(filterLexer),
		participle.Unquote("String"),
		participle.CaseInsensitive("Keyword"),
	)
)

// Parse method parses the input to get a filter expression
// This is mainly used to verify the input conforms to our grammar for filter expression.
func Parse(input string) (*Expression, error) {
	filter := &Expression{}
	err := filterParser.ParseString(input, filter)
	if err != nil {
		err = errcode.NewBadRequestExError("filter", fmt.Sprintf("Filter parsing error: %s", err.Error()))
	}
	return filter, err
}

func (expr Expression) String() string {
	return repr.String(expr, repr.Indent("  "), repr.OmitEmpty(true))
}

// ValidateFilter method validates filter to ensure all identifiers used in the filter are in the given key map
// prevent user filter on unsupported keys
func ValidateFilter(filterExpr *Expression, keyMap map[string]string) error {
	for _, oc := range filterExpr.Or {
		for _, c := range oc.And {
			err := validateConditionFilter(c, keyMap)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func validateConditionFilter(c *Condition, keyMap map[string]string) error {
	if c.Operand != nil {
		if _, ok := keyMap[strings.ToLower(c.Operand.Operand)]; !ok {
			return errcode.NewBadRequestExError("filter", fmt.Sprintf("Unsupported filter key: %s", c.Operand.Operand))
		}
	} else if c.SubExpression != nil {
		return ValidateFilter(c.SubExpression, keyMap)
	} else {
		return errcode.NewBadRequestExError("filter", fmt.Sprintf("Unsupported filter condition: %+v", *c))
	}
	return nil
}

// TransformFields transforms the filter fields into db fields and add table alias if any
func TransformFields(filterExp *Expression, keyMap map[string]string, defaultTableAlias string, tableAliasMapping map[string]string) string {
	orConditions := []string{}
	for _, oc := range filterExp.Or {
		conditions := []string{}
		for _, c := range oc.And {
			if c.Operand != nil {
				realOperand, ok := keyMap[c.Operand.Operand]
				if !ok {
					realOperand = c.Operand.Operand
				}
				alias := defaultTableAlias
				if tableAliasMapping != nil {
					if val, ok := tableAliasMapping[realOperand]; ok {
						alias = val
					}
				}
				conditionRHS := getConditionRHS(c.Operand.ConditionRHS)
				if len(alias) == 0 {
					conditions = append(conditions, fmt.Sprintf("%s %s", realOperand, conditionRHS))
				} else {
					conditions = append(conditions, fmt.Sprintf("%s.%s %s", alias, realOperand, conditionRHS))
				}

			} else if c.SubExpression != nil {
				// Nested with parenthesis
				conditions = append(conditions, fmt.Sprintf("(%s)", TransformFields(c.SubExpression, keyMap, defaultTableAlias, tableAliasMapping)))
			}
		}
		orConditions = append(orConditions, strings.Join(conditions, " AND "))
	}
	return strings.Join(orConditions, " OR ")
}

func getConditionRHS(condition *ConditionRHS) string {
	if condition == nil {
		return ""
	}
	if condition.Compare != nil {
		return fmt.Sprintf("%s %s", condition.Compare.Operator, getStringValue(condition.Compare.Value))
	}
	if condition.Between != nil {
		startValue := getStringValue(condition.Between.Start)
		endValue := getStringValue(condition.Between.End)
		if condition.Between.Not {
			return fmt.Sprintf("NOT BETWEEN %s AND %s", startValue, endValue)
		}
		return fmt.Sprintf("BETWEEN %s AND %s", startValue, endValue)
	}
	if condition.Like != nil {
		value := getStringValue(condition.Like.Value)
		if condition.Like.Not {
			return fmt.Sprintf("NOT LIKE %s", value)
		}
		return fmt.Sprintf("LIKE %s", value)
	}
	if condition.In != nil {
		values := []string{}
		for _, value := range condition.In.Values {
			values = append(values, getStringValue(value))
		}
		if condition.In.Not {
			return fmt.Sprintf("NOT IN (%s)", strings.Join(values, ", "))
		}
		return fmt.Sprintf("IN (%s)", strings.Join(values, ", "))
	}
	return ""
}

func getStringValue(value *Value) string {
	if value == nil {
		return ""
	}
	if value.Float != nil {
		return strconv.FormatFloat(*value.Float, 'f', 6, 64)
	}
	if value.Integer != nil {
		return strconv.FormatInt(*value.Integer, 10)
	}
	if value.String != nil {
		return fmt.Sprintf("'%s'", *value.String)
	}
	if value.Boolean != nil {
		fmt.Println(bool(*value.Boolean))
		return strconv.FormatBool(bool(*value.Boolean))
	}
	return ""
}
