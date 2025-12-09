/*
Copyright 2023 eatmoreapple

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package juice

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-juicedev/juice/driver"
	"github.com/go-juicedev/juice/eval"
	"github.com/go-juicedev/juice/internal/reflectlite"
)

var (
	// paramRegex matches parameter placeholders in SQL queries using #{...} syntax.
	// Examples:
	//   - #{id}         -> matches
	//   - #{user.name}  -> matches
	//   - #{  age  }    -> matches (whitespace is ignored)
	//   - #{}           -> doesn't match (requires identifier)
	//   - #{123}        -> matches
	paramRegex = regexp.MustCompile(`#{\s*(\w+(?:\.\w+)*)\s*}`)

	// formatRegexp matches string interpolation placeholders using ${...} syntax.
	// Unlike paramRegex, these are replaced directly in the SQL string.
	// WARNING: Be careful with this as it can lead to SQL injection if not properly sanitized.
	// Examples:
	//   - ${tableName}  -> matches
	//   - ${db.schema}  -> matches
	//   - ${  field  }  -> matches (whitespace is ignored)
	//   - ${}           -> doesn't match (requires identifier)
	//   - ${123}        -> matches
	formatRegexp = regexp.MustCompile(`\${\s*(\w+(?:\.\w+)*)\s*}`)
)

// Node is the fundamental interface for all SQL generation components.
// It defines the contract for converting dynamic SQL structures into
// concrete SQL queries with their corresponding parameters.
//
// The Accept method follows the Visitor pattern, allowing different
// SQL dialects to be supported through the translator parameter.
//
// Parameters:
//   - translator: Handles dialect-specific SQL translations
//   - parameter: Contains parameter values for SQL generation
//
// Returns:
//   - query: The generated SQL fragment
//   - args: Slice of arguments for prepared statement
//   - err: Any error during SQL generation
//
// Implementing types include:
//   - SQLNode: Complete SQL statements
//   - WhereNode: WHERE clause handling
//   - SetNode: SET clause for updates
//   - IfNode: Conditional inclusion
//   - ChooseNode: Switch-like conditionals
//   - ForeachNode: Collection iteration
//   - TrimNode: String manipulation
//   - IncludeNode: SQL fragment reuse
//
// Example usage:
//
//	query, args, err := node.Accept(mysqlTranslator, params)
//	if err != nil {
//	  // handle error
//	}
//	// use query and args with database
//
// Note: Implementations should handle their specific SQL generation
// logic while maintaining consistency with the overall SQL structure.
type Node interface {
	// Accept processes the node with given translator and parameters
	// to produce a SQL fragment and its arguments.
	Accept(translator driver.Translator, p eval.Parameter) (query string, args []any, err error)
}

// NodeGroup wraps multiple nodes into a single node.
type NodeGroup []Node

// Accept processes all nodes in the group and combines their results.
// The method ensures proper spacing between node outputs and trims any extra whitespace.
// If the group is empty or no nodes produce output, it returns empty results.
func (g NodeGroup) Accept(translator driver.Translator, p eval.Parameter) (query string, args []any, err error) {
	// Return early if group is empty
	nodeLength := len(g)
	switch nodeLength {
	case 0:
		return "", nil, nil
	case 1:
		return g[0].Accept(translator, p)
	}

	var builder = getStringBuilder()
	defer putStringBuilder(builder)

	// Pre-allocate string builder capacity to minimize buffer reallocations
	estimatedCapacity := nodeLength*12 + nodeLength - 1
	builder.Grow(estimatedCapacity)

	// Pre-allocate args slice to avoid reallocations
	args = make([]any, 0, nodeLength)

	lastIdx := nodeLength - 1

	// Process each node in the group
	for i, node := range g {
		q, a, err := node.Accept(translator, p)
		if err != nil {
			return "", nil, err
		}
		if len(q) > 0 {
			builder.WriteString(q)

			// Add space between nodes, but not after the last one
			if i < lastIdx && !strings.HasSuffix(q, " ") {
				builder.WriteString(" ")
			}
		}
		if len(a) > 0 {
			args = append(args, a...)
		}
	}

	// Return empty results if no content was generated
	if builder.Len() == 0 {
		return "", nil, nil
	}

	return builder.String(), args, nil
}

// pureTextNode is a node of pure text.
var _ Node = (*pureTextNode)(nil)

// pureTextNode is a node of pure text.
// It is used to avoid unnecessary parameter replacement.
type pureTextNode string

func (p pureTextNode) Accept(_ driver.Translator, _ eval.Parameter) (query string, args []any, err error) {
	return string(p), nil, nil
}

var _ Node = (*TextNode)(nil)

// TextNode is a node of text.
// What is the difference between TextNode and pureTextNode?
// TextNode is used to replace parameters with placeholders.
// pureTextNode is used to avoid unnecessary parameter replacement.
type TextNode struct {
	value            string
	placeholder      [][]string // for example, #{id}
	textSubstitution [][]string // for example, ${id}
}

// Accept accepts parameters and returns query and arguments.
// Accept implements Node interface.
func (c *TextNode) Accept(translator driver.Translator, p eval.Parameter) (query string, args []any, err error) {
	// If there is no parameter, return the value as it is.
	if len(c.placeholder) == 0 && len(c.textSubstitution) == 0 {
		return c.value, nil, nil
	}
	// Otherwise, replace the parameter with a placeholder.
	query, args, err = c.replaceHolder(c.value, args, translator, p)
	if err != nil {
		return "", nil, err
	}
	query, err = c.replaceTextSubstitution(query, p)
	if err != nil {
		return "", nil, err
	}
	return query, args, nil
}

func (c *TextNode) replaceHolder(query string, args []any, translator driver.Translator, p eval.Parameter) (string, []any, error) {
	if len(c.placeholder) == 0 {
		return query, args, nil
	}

	builder := getStringBuilder()
	defer putStringBuilder(builder)
	builder.Grow(len(query))

	lastIndex := 0
	newArgs := make([]any, 0, len(args)+len(c.placeholder))
	newArgs = append(newArgs, args...)

	for _, param := range c.placeholder {
		if len(param) != 2 {
			return "", nil, fmt.Errorf("invalid parameter %v", param)
		}
		matched, name := param[0], param[1]

		value, exists := p.Get(name)
		if !exists {
			return "", nil, fmt.Errorf("parameter %s not found", name)
		}

		pos := strings.Index(query[lastIndex:], matched)
		if pos == -1 {
			continue
		}
		pos += lastIndex

		builder.WriteString(query[lastIndex:pos])
		builder.WriteString(translator.Translate(name))
		lastIndex = pos + len(matched)

		newArgs = append(newArgs, value.Interface())
	}

	builder.WriteString(query[lastIndex:])
	return builder.String(), newArgs, nil
}

// replaceTextSubstitution replaces text substitution.
func (c *TextNode) replaceTextSubstitution(query string, p eval.Parameter) (string, error) {
	if len(c.textSubstitution) == 0 {
		return query, nil
	}

	builder := getStringBuilder()
	defer putStringBuilder(builder)
	builder.Grow(len(query))

	lastIndex := 0
	for _, sub := range c.textSubstitution {
		if len(sub) != 2 {
			return "", fmt.Errorf("invalid text substitution %v", sub)
		}
		matched, name := sub[0], sub[1]

		value, exists := p.Get(name)
		if !exists {
			return "", fmt.Errorf("parameter %s not found", name)
		}

		pos := strings.Index(query[lastIndex:], matched)
		if pos == -1 {
			continue
		}
		pos += lastIndex

		builder.WriteString(query[lastIndex:pos])
		builder.WriteString(reflectValueToString(value))
		lastIndex = pos + len(matched)
	}

	builder.WriteString(query[lastIndex:])
	return builder.String(), nil
}

// NewTextNode creates a new text node based on the input string.
// It returns either a lightweight pureTextNode for static SQL,
// or a full TextNode for dynamic SQL with placeholders/substitutions.
func NewTextNode(str string) Node {
	placeholder := paramRegex.FindAllStringSubmatch(str, -1)

	textSubstitution := formatRegexp.FindAllStringSubmatch(str, -1)

	if len(placeholder) == 0 && len(textSubstitution) == 0 {
		return pureTextNode(str)
	}
	return &TextNode{value: str, placeholder: placeholder, textSubstitution: textSubstitution}
}

var ErrNilExpression = errors.New("juice: nil expression")

// ConditionNode represents a conditional SQL fragment with its evaluation expression and child nodes.
// It is used to conditionally include or exclude SQL fragments based on runtime parameters.
type ConditionNode struct {
	expr  eval.Expression
	Nodes NodeGroup
}

// Parse compiles the given expression string into an evaluable expression.
// The expression syntax supports various operations like:
//   - Comparison: ==, !=, >, <, >=, <=
//   - Logical: &&, ||, !
//   - Null checks: != null, == null
//   - Property access: user.age, order.status
//
// Examples:
//
//	"id != nil"              // Check for non-null
//	"age >= 18"               // Numeric comparison
//	"status == "ACTIVE""      // String comparison
//	"user.role == "ADMIN""    // Property access
func (c *ConditionNode) Parse(test string) (err error) {
	c.expr, err = eval.Compile(test)
	return err
}

// Accept accepts parameters and returns query and arguments.
// Accept implements Node interface.
func (c *ConditionNode) Accept(translator driver.Translator, p eval.Parameter) (query string, args []any, err error) {
	matched, err := c.Match(p)
	if err != nil {
		return "", nil, err
	}
	if !matched {
		return "", nil, nil
	}
	return c.Nodes.Accept(translator, p)
}

// Match evaluates if the condition is true based on the provided parameter.
// It handles different types of values and converts them to boolean results:
//   - Bool: returns the boolean value directly
//   - Integers (signed/unsigned): returns true if non-zero
//   - Floats: returns true if non-zero
//   - String: returns true if non-empty
func (c *ConditionNode) Match(p eval.Parameter) (bool, error) {
	if c.expr == nil {
		return false, ErrNilExpression
	}
	value, err := c.expr.Execute(p)
	if err != nil {
		return false, err
	}
	return !value.IsZero(), nil
}

var _ Node = (*ConditionNode)(nil)

// IfNode is an alias for ConditionNode, representing a conditional SQL fragment.
// It evaluates a condition and determines whether its content should be included in the final SQL.
//
// The condition can be based on various types:
//   - Boolean: direct condition
//   - Numbers: non-zero values are true
//   - Strings: non-empty strings are true
//
// Example usage:
//
//	<if test="id > 0">
//	    AND id = #{id}
//	</if>
//
// See ConditionNode for detailed behavior of condition evaluation.
type IfNode = ConditionNode

var _ Node = (*IfNode)(nil)

// WhereNode represents a SQL WHERE clause and its conditions.
// It manages a group of condition nodes that form the complete WHERE clause.
type WhereNode struct {
	Nodes NodeGroup
}

// Accept processes the WHERE clause and its conditions.
// It handles several special cases:
//  1. Removes leading "AND" or "OR" from the first condition
//  2. Ensures the clause starts with "WHERE" if not already present
//  3. Properly handles spacing between conditions
//
// Examples:
//
//	Input:  "AND id = ?"        -> Output: "WHERE id = ?"
//	Input:  "OR name = ?"       -> Output: "WHERE name = ?"
//	Input:  "WHERE age > ?"     -> Output: "WHERE age > ?"
//	Input:  "status = ?"        -> Output: "WHERE status = ?"
func (w WhereNode) Accept(translator driver.Translator, p eval.Parameter) (query string, args []any, err error) {
	query, args, err = w.Nodes.Accept(translator, p)
	if err != nil {
		return "", nil, err
	}

	if query == "" {
		return "", args, nil
	}
	// A space is required at the end; otherwise, it is meaningless.
	switch {
	case strings.HasPrefix(query, "and ") || strings.HasPrefix(query, "AND "):
		query = query[4:]
	case strings.HasPrefix(query, "or ") || strings.HasPrefix(query, "OR "):
		query = query[3:]
	}

	// A space is required at the end; otherwise, it is meaningless.
	if !strings.HasPrefix(query, "where ") && !strings.HasPrefix(query, "WHERE ") {
		query = "WHERE " + query
	}
	return
}

var _ Node = (*WhereNode)(nil)

// TrimNode handles SQL fragment cleanup by managing prefixes, suffixes, and their overrides.
// It's particularly useful for dynamically generated SQL where certain prefixes or suffixes
// might need to be added or removed based on the context.
//
// Fields:
//   - Nodes: Group of child nodes containing the SQL fragments
//   - Prefix: String to prepend to the result if content exists
//   - PrefixOverrides: Strings to remove if found at the start
//   - Suffix: String to append to the result if content exists
//   - SuffixOverrides: Strings to remove if found at the end
//
// Common use cases:
//  1. Removing leading AND/OR from WHERE clauses
//  2. Managing commas in clauses
//  3. Handling dynamic UPDATE SET statements
//
// Example XML:
//
//	<trim prefix="WHERE" prefixOverrides="AND|OR">
//	  <if test="id > 0">
//	    AND id = #{id}
//	  </if>
//	  <if test='name != ""'>
//	    AND name = #{name}
//	  </if>
//	</trim>
//
// Example Result:
//
//	Input:  "AND id = ? AND name = ?"
//	Output: "WHERE id = ? AND name = ?"
type TrimNode struct {
	Nodes           NodeGroup
	Prefix          string
	PrefixOverrides []string
	Suffix          string
	SuffixOverrides []string
}

// Accept accepts parameters and returns query and arguments.
func (t TrimNode) Accept(translator driver.Translator, p eval.Parameter) (query string, args []any, err error) {
	query, args, err = t.Nodes.Accept(translator, p)
	if err != nil {
		return "", nil, err
	}

	if len(query) == 0 {
		return "", nil, nil
	}

	// Handle prefix overrides before adding prefix
	if len(t.PrefixOverrides) > 0 {
		for _, prefix := range t.PrefixOverrides {
			if strings.HasPrefix(query, prefix) {
				query = query[len(prefix):]
				break
			}
		}
	}

	// Handle suffix overrides before adding suffix
	if len(t.SuffixOverrides) > 0 {
		for _, suffix := range t.SuffixOverrides {
			if strings.HasSuffix(query, suffix) {
				query = query[:len(query)-len(suffix)]
				break
			}
		}
	}

	// Build final query with prefix and suffix
	var builder = getStringBuilder()
	defer putStringBuilder(builder)

	builder.Grow(len(t.Prefix) + len(query) + len(t.Suffix))

	if t.Prefix != "" {
		builder.WriteString(t.Prefix)
	}
	builder.WriteString(query)
	if t.Suffix != "" {
		builder.WriteString(t.Suffix)
	}

	return builder.String(), args, nil
}

var _ Node = (*TrimNode)(nil)

// ForeachNode represents a dynamic SQL fragment that iterates over a collection.
// It's commonly used for IN clauses, batch inserts, or any scenario requiring
// iteration over a collection of values in SQL generation.
//
// Fields:
//   - Collection: Expression to get the collection to iterate over
//   - Nodes: SQL fragments to be repeated for each item
//   - Item: Variable name for the current item in iteration
//   - Index: Variable name for the current index (optional)
//   - Open: String to prepend before the iteration results
//   - Close: String to append after the iteration results
//   - Separator: String to insert between iterations
//
// Example XML:
//
//	<foreach collection="list" item="item" index="i" open="(" separator="," close=")">
//	  #{item}
//	</foreach>
//
// Usage scenarios:
//
//  1. IN clauses:
//     WHERE id IN (#{item})
//
//  2. Batch inserts:
//     INSERT INTO users VALUES
//     <foreach collection="users" item="user" separator=",">
//     (#{user.id}, #{user.name})
//     </foreach>
//
//  3. Multiple conditions:
//     <foreach collection="ids" item="id" separator="OR">
//     id = #{id}
//     </foreach>
//
// Example results:
//
//	Input collection: [1, 2, 3]
//	xmlConfiguration: open="(", separator=",", close=")"
//	Output: "(1,2,3)"
type ForeachNode struct {
	Collection string
	Nodes      []Node
	Item       string
	Index      string
	Open       string
	Close      string
	Separator  string
}

// Accept accepts parameters and returns query and arguments.
func (f ForeachNode) Accept(translator driver.Translator, p eval.Parameter) (query string, args []any, err error) {

	// if item already exists
	if _, exists := p.Get(f.Item); exists {
		return "", nil, fmt.Errorf("item %s already exists", f.Item)
	}

	// one collection from parameter
	value, exists := p.Get(f.Collection)
	if !exists {
		return "", nil, fmt.Errorf("collection %s not found", f.Collection)
	}

	// if valueItem can not be iterated
	if !value.CanInterface() {
		return "", nil, fmt.Errorf("collection %s can not be iterated", f.Collection)
	}

	// if valueItem is not a slice
	for value.Kind() == reflect.Interface {
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.Array, reflect.Slice:
		return f.acceptSlice(value, translator, p)
	case reflect.Map:
		return f.acceptMap(value, translator, p)
	default:
		return "", nil, fmt.Errorf("collection %s is not a slice or map", f.Collection)
	}
}

func (f ForeachNode) acceptSlice(value reflect.Value, translator driver.Translator, p eval.Parameter) (query string, args []any, err error) {
	sliceLength := value.Len()

	if sliceLength == 0 {
		return "", nil, nil
	}

	// Pre-allocate args slice capacity to avoid multiple growths
	// Estimate: number of slice elements * number of nodes
	estimatedArgsLen := sliceLength * len(f.Nodes)

	args = make([]any, 0, estimatedArgsLen)

	// Pre-allocate string builder capacity to minimize buffer reallocations
	// Capacity = open + items + separators + close
	estimatedBuilderCap := len(f.Open) + (2 * sliceLength) + (len(f.Separator) * (sliceLength - 1)) + len(f.Close)

	var builder = getStringBuilder()
	defer putStringBuilder(builder)

	builder.Grow(estimatedBuilderCap)

	builder.WriteString(f.Open)

	end := sliceLength - 1

	h := make(eval.H, 2)

	// Create and reuse GenericParameter outside the loop to avoid allocations per iteration
	genericParameter := &eval.GenericParameter{Value: reflect.ValueOf(h)}

	group := eval.ParamGroup{genericParameter, p}

	for i := 0; i < sliceLength; i++ {

		item := value.Index(i).Interface()

		h[f.Item] = item
		h[f.Index] = i

		for _, node := range f.Nodes {
			q, a, err := node.Accept(translator, group)
			if err != nil {
				return "", nil, err
			}
			if len(q) > 0 {
				builder.WriteString(q)
			}
			if len(a) > 0 {
				args = append(args, a...)
			}
		}

		if i < end {
			builder.WriteString(f.Separator)
		}
		genericParameter.Clear()
	}

	// if sliceLength is not zero, add close
	builder.WriteString(f.Close)

	return builder.String(), args, nil
}

func (f ForeachNode) acceptMap(value reflect.Value, translator driver.Translator, p eval.Parameter) (query string, args []any, err error) {
	keys := value.MapKeys()

	if len(keys) == 0 {
		return "", nil, nil
	}

	// Pre-allocate args slice capacity to avoid multiple growths
	// Estimate: number of slice elements * number of nodes
	estimatedArgsLen := len(keys) * len(f.Nodes)

	args = make([]any, 0, estimatedArgsLen)

	// Pre-allocate string builder capacity to minimize buffer reallocations
	// Capacity = open + items + separators + close
	estimatedBuilderCap := len(f.Open) + (2 * len(keys)) + (len(f.Separator) * (len(keys) - 1)) + len(f.Close)

	var builder = getStringBuilder()
	defer putStringBuilder(builder)

	builder.Grow(estimatedBuilderCap)

	builder.WriteString(f.Open)

	end := len(keys) - 1

	var index int

	h := make(eval.H, 2)

	// Create and reuse GenericParameter outside the loop to avoid allocations per iteration
	genericParameter := &eval.GenericParameter{Value: reflect.ValueOf(h)}

	group := eval.ParamGroup{genericParameter, p}

	for _, key := range keys {

		item := value.MapIndex(key).Interface()

		h[f.Item] = item
		h[f.Index] = key.Interface()

		for _, node := range f.Nodes {
			q, a, err := node.Accept(translator, group)
			if err != nil {
				return "", nil, err
			}
			if len(q) > 0 {
				builder.WriteString(q)
			}
			if len(a) > 0 {
				args = append(args, a...)
			}
		}

		if index < end {
			builder.WriteString(f.Separator)
		}

		genericParameter.Clear()

		index++
	}

	builder.WriteString(f.Close)

	return builder.String(), args, nil
}

var _ Node = (*ForeachNode)(nil)

// SetNode represents an SQL SET clause for UPDATE statements.
// It manages a group of assignment expressions and automatically handles
// the comma separators and SET prefix.
//
// Features:
//   - Automatically adds "SET" prefix
//   - Manages comma separators between assignments
//   - Handles dynamic assignments based on conditions
//
// Example XML:
//
//	<update id="updateUser">
//	  UPDATE users
//	  <set>
//	    <if test='name != ""'>
//	      name = #{name},
//	    </if>
//	    <if test="age > 0">
//	      age = #{age},
//	    </if>
//	    <if test="status != 0">
//	      status = #{status}
//	    </if>
//	  </set>
//	  WHERE id = #{id}
//	</update>
//
// Example results:
//
//	Case 1 (name and age set):
//	  UPDATE users SET name = ?, age = ? WHERE id = ?
//
//	Case 2 (only status set):
//	  UPDATE users SET status = ? WHERE id = ?
//
// Note: The node automatically handles trailing commas and ensures
// proper formatting of the SET clause regardless of which fields
// are included dynamically.
type SetNode struct {
	Nodes NodeGroup
}

// Accept accepts parameters and returns query and arguments.
func (s SetNode) Accept(translator driver.Translator, p eval.Parameter) (query string, args []any, err error) {
	query, args, err = s.Nodes.Accept(translator, p)
	if err != nil {
		return "", nil, err
	}
	if len(query) == 0 {
		return "", args, nil
	}
	// Remove trailing comma
	query = strings.TrimSuffix(query, ",")

	// Ensure SET prefix if not present
	if !strings.HasPrefix(query, "set ") && !strings.HasPrefix(query, "SET ") {
		query = "SET " + query
	}

	return query, args, nil
}

var _ Node = (*SetNode)(nil)

// SQLNode represents a complete SQL statement with its metadata and child nodes.
// It serves as the root node for a single SQL operation (SELECT, INSERT, UPDATE, DELETE)
// and manages the entire SQL generation process.
//
// Fields:
//   - id: Unique identifier for the SQL statement within the mapper
//   - nodes: Collection of child nodes that form the complete SQL
//   - mapper: Reference to the parent Mapper for context and configuration
//
// Example XML:
//
//	<select id="getUserById">
//	  SELECT *
//	  FROM users
//	  <where>
//	    <if test="id != 0">
//	      id = #{id}
//	    </if>
//	  </where>
//	</select>
//
// Usage scenarios:
//  1. SELECT statements with dynamic conditions
//  2. INSERT statements with optional fields
//  3. UPDATE statements with dynamic SET clauses
//  4. DELETE statements with complex WHERE conditions
//
// Features:
//   - Manages complete SQL statement generation
//   - Handles parameter binding
//   - Supports dynamic SQL through child nodes
//   - Maintains connection to mapper context
//   - Enables statement reuse through ID reference
//
// Note: The id must be unique within its mapper context to allow
// proper statement lookup and execution.
type SQLNode struct {
	id    string    // Unique identifier for the SQL statement
	nodes NodeGroup // Child nodes forming the SQL statement
}

// ID returns the id of the node.
func (s SQLNode) ID() string {
	return s.id
}

// Accept accepts parameters and returns query and arguments.
func (s SQLNode) Accept(translator driver.Translator, p eval.Parameter) (query string, args []any, err error) {
	return s.nodes.Accept(translator, p)
}

var _ Node = (*SQLNode)(nil)

// IncludeNode represents a reference to another SQL fragment, enabling SQL reuse.
// It allows common SQL fragments to be defined once and included in multiple places,
// promoting code reuse and maintainability.
//
// Fields:
//   - sqlNode: The referenced SQL fragment node
//   - mapper: Reference to the parent Mapper for context
//   - refId: ID of the SQL fragment to include
//
// Example XML:
//
//	<!-- Common WHERE clause -->
//	<sql id="userFields">
//	  id, name, age, status
//	</sql>
//
//	<!-- Using the include -->
//	<select id="getUsers">
//	  SELECT
//	  <include refid="userFields"/>
//	  FROM users
//	  WHERE status = #{status}
//	</select>
//
// Features:
//   - Enables SQL fragment reuse
//   - Supports cross-mapper references
//   - Maintains consistent SQL patterns
//   - Reduces code duplication
//
// Usage scenarios:
//  1. Common column lists
//  2. Shared WHERE conditions
//  3. Reusable JOIN clauses
//  4. Standard filtering conditions
//
// Note: The refId must reference an existing SQL fragment defined with
// the <sql> tag. The reference can be within the same mapper or from
// another mapper if properly configured.
type IncludeNode struct {
	sqlNode Node
	mapper  *Mapper
	refId   string
}

// Accept accepts parameters and returns query and arguments.
func (i *IncludeNode) Accept(translator driver.Translator, p eval.Parameter) (query string, args []any, err error) {
	if i.sqlNode == nil {
		// lazy loading
		// does it need to be thread safe?
		sqlNode, err := i.mapper.GetSQLNodeByID(i.refId)
		if err != nil {
			return "", nil, err
		}
		i.sqlNode = sqlNode
	}
	return i.sqlNode.Accept(translator, p)
}

var _ Node = (*IncludeNode)(nil)

// ChooseNode implements a switch-like conditional structure for SQL generation.
// It evaluates multiple conditions in order and executes the first matching case,
// with an optional default case (otherwise).
//
// Fields:
//   - WhenNodes: Ordered list of conditional branches to evaluate
//   - OtherwiseNode: Default branch if no when conditions match
//
// Example XML:
//
//	<choose>
//	  <when test="id != 0">
//	    AND id = #{id}
//	  </when>
//	  <when test='name != ""'>
//	    AND name LIKE CONCAT('%', #{name}, '%')
//	  </when>
//	  <otherwise>
//	    AND status = 'ACTIVE'
//	  </otherwise>
//	</choose>
//
// Behavior:
//  1. Evaluates each <when> condition in order
//  2. Executes SQL from first matching condition
//  3. If no conditions match, executes <otherwise> if present
//  4. If no conditions match and no otherwise, returns empty result
//
// Usage scenarios:
//  1. Complex conditional logic in WHERE clauses
//  2. Dynamic sorting options
//  3. Different JOIN conditions
//  4. Status-based queries
//
// Example results:
//
//	Case 1 (id present):
//	  AND id = ?
//	Case 2 (only name present):
//	  AND name LIKE ?
//	Case 3 (neither present):
//	  AND status = 'ACTIVE'
//
// Note: Similar to a switch statement in programming languages,
// only the first matching condition is executed.
type ChooseNode struct {
	WhenNodes     []Node
	OtherwiseNode Node
}

// Accept accepts parameters and returns query and arguments.
func (c ChooseNode) Accept(translator driver.Translator, p eval.Parameter) (query string, args []any, err error) {
	for _, node := range c.WhenNodes {
		q, a, err := node.Accept(translator, p)
		if err != nil {
			return "", nil, err
		}
		// if one of when nodes is true, return query and arguments
		if len(q) > 0 {
			return q, a, nil
		}
	}
	// if all when nodes are false, return otherwise node
	if c.OtherwiseNode != nil {
		return c.OtherwiseNode.Accept(translator, p)
	}
	return "", nil, nil
}

var _ Node = (*ChooseNode)(nil)

// WhenNode is an alias for ConditionNode, representing a conditional branch
// within a <choose> statement. It evaluates a condition and executes its
// content if the condition is true and it's the first matching condition
// in the choose block.
//
// Behavior:
//   - Evaluates condition using same rules as ConditionNode
//   - Only executes if it's the first true condition in choose
//   - Subsequent true conditions are ignored
//
// Example XML:
//
//	<choose>
//	  <when test='type == "PREMIUM"'>
//	    AND membership_level = 'PREMIUM'
//	  </when>
//	  <when test='type == "BASIC"'>
//	    AND membership_level IN ('BASIC', 'STANDARD')
//	  </when>
//	</choose>
//
// Supported conditions:
//   - Boolean expressions
//   - Numeric comparisons
//   - String comparisons
//   - Null checks
//   - Property access
//
// Note: Unlike a standalone ConditionNode, WhenNode's execution
// is controlled by its parent ChooseNode and follows choose-when
// semantics similar to switch-case statements.
//
// See ConditionNode for detailed condition evaluation rules.
type WhenNode = ConditionNode

var _ Node = (*WhenNode)(nil)

// OtherwiseNode represents the default branch in a <choose> statement,
// which executes when none of the <when> conditions are met.
// It's similar to the 'default' case in a switch statement.
//
// Fields:
//   - Nodes: Group of nodes containing the default SQL fragments
//
// Example XML:
//
//	<choose>
//	  <when test="status != nil">
//	    AND status = #{status}
//	  </when>
//	  <when test="type != nil">
//	    AND type = #{type}
//	  </when>
//	  <otherwise>
//	    AND is_deleted = 0
//	    AND status = 'ACTIVE'
//	  </otherwise>
//	</choose>
//
// Behavior:
//   - Executes only if all <when> conditions are false
//   - No condition evaluation needed
//   - Can contain multiple SQL fragments
//   - Optional within <choose> block
//
// Usage scenarios:
//  1. Default filtering conditions
//  2. Fallback sorting options
//  3. Default join conditions
//  4. Error prevention (ensuring non-empty WHERE clauses)
//
// Example results:
//
//	When no conditions match:
//	  AND is_deleted = 0 AND status = 'ACTIVE'
//
// Note: Unlike WhenNode, OtherwiseNode doesn't evaluate any conditions.
// It simply provides default SQL fragments when needed.
type OtherwiseNode struct {
	Nodes NodeGroup
}

// Accept accepts parameters and returns query and arguments.
func (o OtherwiseNode) Accept(translator driver.Translator, p eval.Parameter) (query string, args []any, err error) {
	return o.Nodes.Accept(translator, p)
}

var _ Node = (*OtherwiseNode)(nil)

// reflectValueToString converts reflect.Value to string
func reflectValueToString(v reflect.Value) string {
	v = reflectlite.Unwrap(v)
	switch t := v.Interface().(type) {
	case nil:
		return ""
	case string:
		return t
	case []byte:
		return string(v.Bytes())
	case fmt.Stringer:
		return t.String()
	case int, int8, int16, int32, int64:
		return strconv.FormatInt(v.Int(), 10)
	case uint, uint8, uint16, uint32, uint64:
		return strconv.FormatUint(v.Uint(), 10)
	case float32:
		return strconv.FormatFloat(v.Float(), 'g', -1, 32)
	case float64:
		return strconv.FormatFloat(v.Float(), 'g', -1, 64)
	case bool:
		return strconv.FormatBool(v.Bool())
	default:
		return fmt.Sprintf("%v", t)
	}
}

// BindNode represents a named bind variable backed by a compiled expression.
type BindNode struct {
	Name string
	expr eval.Expression
}

// Parse compiles the given expression string and stores the result.
func (b *BindNode) Parse(expression string) (err error) {
	b.expr, err = eval.Compile(expression)
	return err
}

// Execute evaluates the compiled expression against the provided Parameter
// and returns the resulting reflect.Value.
func (b *BindNode) Execute(p eval.Parameter) (reflect.Value, error) {
	value, err := b.expr.Execute(p)
	if err != nil {
		return reflect.Value{}, err
	}
	return value, nil
}

// ErrBindVariableNotFound is returned when a bind variable lookup fails.
var ErrBindVariableNotFound = errors.New("juice: bind variable not found")

// bindScope provides lookup and execution of bind variables within a scope.
type bindScope struct {
	nodes     []*BindNode
	parameter eval.Parameter
}

// Get finds a BindNode by name and executes it using the scope's parameter.
// Returns ErrBindVariableNotFound wrapped if no bind with the given name exists.
func (b bindScope) Get(name string) (reflect.Value, error) {
	for _, bind := range b.nodes {
		if bind.Name == name {
			return bind.Execute(b.parameter)
		}
	}
	return reflect.Value{}, fmt.Errorf("%w: %s", ErrBindVariableNotFound, name)
}
