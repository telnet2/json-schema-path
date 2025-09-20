package tree

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/telnet2/json-schema-path/spec"
)

// Transition types for pattern matching
type transition interface {
	matches(segment spec.PathSegment) bool
}

// node represents a state in the pattern matching trie
type node struct {
	properties        map[string]*node        // Direct property transitions
	arrayIndices      map[int]*node           // Direct array index transitions
	arrayWildcard     *node                   // [*] wildcard transition
	propertyWildcards map[string]*wildcardTransition // Property wildcard transitions
	regexTransitions  map[string]*regexTransition    // Regex pattern transitions
	epsilon           []*node                 // Epsilon (empty) transitions
	terminal          bool                    // Whether this is an accepting state
}

// wildcardTransition represents a property wildcard transition (e.g., [#*suffix])
type wildcardTransition struct {
	pattern string // Wildcard pattern like "*suffix" or "prefix*"
	target  *node  // Target node for this transition
}

// regexTransition represents a regex pattern transition (e.g., [~pattern])
type regexTransition struct {
	pattern string         // Original pattern string for debugging
	expr    *regexp.Regexp // Compiled regex expression
	target  *node          // Target node for this transition
}

// matches implements the transition interface for wildcardTransition
func (wt *wildcardTransition) matches(segment spec.PathSegment) bool {
	if segment.Type != spec.SegmentProperty {
		return false
	}
	
	// Handle different wildcard patterns
	switch {
	case strings.HasPrefix(wt.pattern, "*") && strings.HasSuffix(wt.pattern, "*"):
		// *contains* pattern
		contains := wt.pattern[1 : len(wt.pattern)-1]
		return strings.Contains(segment.Key, contains)
	case strings.HasPrefix(wt.pattern, "*"):
		// *suffix pattern
		suffix := wt.pattern[1:]
		return strings.HasSuffix(segment.Key, suffix)
	case strings.HasSuffix(wt.pattern, "*"):
		// prefix* pattern
		prefix := wt.pattern[:len(wt.pattern)-1]
		return strings.HasPrefix(segment.Key, prefix)
	default:
		// exact match (shouldn't happen with wildcards, but handle gracefully)
		return segment.Key == wt.pattern
	}
}

// matches implements the transition interface for regexTransition
func (rt *regexTransition) matches(segment spec.PathSegment) bool {
	if segment.Type != spec.SegmentProperty {
		return false
	}
	return rt.expr.MatchString(segment.Key)
}

// PatternTree stores compiled schema-path patterns as an epsilon-NFA backed trie.
type PatternTree struct {
	root *node
}

// NewPatternTree creates an empty pattern tree.
func NewPatternTree() *PatternTree {
	return &PatternTree{root: &node{}}
}

// PatternError represents an error during pattern compilation
type PatternError struct {
	Pattern string
	Message string
}

func (e PatternError) Error() string {
	if e.Pattern != "" {
		return fmt.Sprintf("pattern error in '%s': %s", e.Pattern, e.Message)
	}
	return fmt.Sprintf("pattern error: %s", e.Message)
}

// AddPattern compiles the given path expression into the trie.
func (t *PatternTree) AddPattern(expr *spec.PathExpression) error {
	if expr == nil {
		return PatternError{Message: "expression is nil"}
	}
	if expr.Root == nil {
		return PatternError{Message: "expression root is nil"}
	}
	
	endNodes, err := t.addSegments([]*node{t.root}, expr.Segments)
	if err != nil {
		return err
	}
	
	for _, n := range endNodes {
		n.terminal = true
	}
	return nil
}

func (t *PatternTree) addSegments(starts []*node, segments []spec.ASTNode) ([]*node, error) {
	current := epsilonClosure(starts)
	for _, segment := range segments {
		next, err := t.applySegment(current, segment)
		if err != nil {
			return nil, err
		}
		current = epsilonClosure(next)
	}
	return current, nil
}

func (t *PatternTree) applySegment(starts []*node, segment spec.ASTNode) ([]*node, error) {
	switch node := segment.(type) {
	case *spec.PropertyNode:
		return t.applyProperty(starts, node.Name), nil
	case *spec.BracketNode:
		return t.applyBracket(starts, node)
	case *spec.GroupNode:
		return t.applyGroup(starts, node)
	case *spec.RepetitionNode:
		return t.applyRepetition(starts, node)
	default:
		return nil, PatternError{
			Message: fmt.Sprintf("unsupported AST node type %T", segment),
		}
	}
}

func (t *PatternTree) applyProperty(starts []*node, name string) []*node {
	result := make([]*node, 0, len(starts))
	for _, start := range starts {
		child := start.ensureProperty(name)
		result = append(result, child)
	}
	return uniqueNodes(result)
}

func (t *PatternTree) applyBracket(starts []*node, bracket *spec.BracketNode) ([]*node, error) {
	result := make([]*node, 0, len(starts))
	for _, start := range starts {
		var child *node
		var err error
		switch bracket.Kind {
		case spec.BracketProperty:
			child = start.ensureProperty(bracket.Value)
		case spec.BracketPropertyWildcard:
			child, err = start.ensureWildcard(bracket.Value)
		case spec.BracketRegex:
			child, err = start.ensureRegex(bracket.Value)
		case spec.BracketArrayIndex:
			child = start.ensureArrayIndex(bracket.Index)
		case spec.BracketArrayWildcard:
			child = start.ensureArrayWildcard()
		default:
			err = fmt.Errorf("unknown bracket kind %v", bracket.Kind)
		}
		if err != nil {
			return nil, err
		}
		result = append(result, child)
	}
	return uniqueNodes(result), nil
}

func (t *PatternTree) applyGroup(starts []*node, group *spec.GroupNode) ([]*node, error) {
	if len(group.Alternatives) == 0 {
		return nil, fmt.Errorf("group must contain at least one alternative")
	}
	if group.Repetition {
		return t.applyRepeatingGroup(starts, group.Alternatives)
	}
	aggregate := make([]*node, 0)
	for _, alt := range group.Alternatives {
		altEnds, err := t.addSegments(starts, alt)
		if err != nil {
			return nil, err
		}
		aggregate = append(aggregate, altEnds...)
	}
	return uniqueNodes(aggregate), nil
}

func (t *PatternTree) applyRepeatingGroup(starts []*node, alternatives [][]spec.ASTNode) ([]*node, error) {
	for _, alt := range alternatives {
		if len(alt) == 0 {
			return nil, fmt.Errorf("repeating group alternative cannot be empty")
		}
	}
	result := make([]*node, 0, len(starts))
	result = append(result, starts...)
	for _, start := range starts {
		for _, alt := range alternatives {
			altEnds, err := t.addSegments([]*node{start}, alt)
			if err != nil {
				return nil, err
			}
			for _, end := range altEnds {
				end.addEpsilon(start)
			}
			result = append(result, altEnds...)
		}
	}
	return uniqueNodes(result), nil
}

func (t *PatternTree) applyRepetition(starts []*node, repetition *spec.RepetitionNode) ([]*node, error) {
	if len(repetition.Sequence) == 0 {
		return nil, fmt.Errorf("repetition sequence cannot be empty")
	}
	result := make([]*node, 0, len(starts))
	result = append(result, starts...)
	for _, start := range starts {
		seqEnds, err := t.addSegments([]*node{start}, repetition.Sequence)
		if err != nil {
			return nil, err
		}
		for _, end := range seqEnds {
			end.addEpsilon(start)
		}
		result = append(result, seqEnds...)
	}
	return uniqueNodes(result), nil
}

// MatchSegments checks whether the provided runtime path matches any pattern.
func (t *PatternTree) MatchSegments(segments []spec.PathSegment) bool {
	current := epsilonClosure([]*node{t.root})
	for _, segment := range segments {
		nextSet := make(map[*node]struct{})
		for _, state := range current {
			switch segment.Type {
			case spec.SegmentProperty:
				if child, ok := state.properties[segment.Key]; ok {
					nextSet[child] = struct{}{}
				}
				for _, trans := range state.propertyWildcards {
					if match, _ := path.Match(trans.pattern, segment.Key); match {
						nextSet[trans.target] = struct{}{}
					}
				}
				for _, trans := range state.regexTransitions {
					if trans.expr.MatchString(segment.Key) {
						nextSet[trans.target] = struct{}{}
					}
				}
			case spec.SegmentArrayIndex:
				if child, ok := state.arrayIndices[segment.Index]; ok {
					nextSet[child] = struct{}{}
				}
				if state.arrayWildcard != nil {
					nextSet[state.arrayWildcard] = struct{}{}
				}
			}
		}
		if len(nextSet) == 0 {
			return false
		}
		next := make([]*node, 0, len(nextSet))
		for n := range nextSet {
			next = append(next, n)
		}
		current = epsilonClosure(next)
		if len(current) == 0 {
			return false
		}
	}
	for _, state := range current {
		if state.terminal {
			return true
		}
	}
	return false
}

func (n *node) ensureProperty(name string) *node {
	if n.properties == nil {
		n.properties = make(map[string]*node)
	}
	child, ok := n.properties[name]
	if !ok {
		child = &node{}
		n.properties[name] = child
	}
	return child
}

func (n *node) ensureArrayIndex(index int) *node {
	if n.arrayIndices == nil {
		n.arrayIndices = make(map[int]*node)
	}
	child, ok := n.arrayIndices[index]
	if !ok {
		child = &node{}
		n.arrayIndices[index] = child
	}
	return child
}

func (n *node) ensureArrayWildcard() *node {
	if n.arrayWildcard == nil {
		n.arrayWildcard = &node{}
	}
	return n.arrayWildcard
}

func (n *node) ensureWildcard(pattern string) (*node, error) {
	if _, err := path.Match(pattern, pattern); err != nil {
		return nil, fmt.Errorf("invalid wildcard pattern %q: %w", pattern, err)
	}
	if n.propertyWildcards == nil {
		n.propertyWildcards = make(map[string]*wildcardTransition)
	}
	if trans, ok := n.propertyWildcards[pattern]; ok {
		return trans.target, nil
	}
	child := &node{}
	n.propertyWildcards[pattern] = &wildcardTransition{pattern: pattern, target: child}
	return child, nil
}

func (n *node) ensureRegex(pattern string) (*node, error) {
	if n.regexTransitions == nil {
		n.regexTransitions = make(map[string]*regexTransition)
	}
	if trans, ok := n.regexTransitions[pattern]; ok {
		return trans.target, nil
	}
	expr, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
	}
	child := &node{}
	n.regexTransitions[pattern] = &regexTransition{pattern: pattern, expr: expr, target: child}
	return child, nil
}

func (n *node) addEpsilon(target *node) {
	for _, existing := range n.epsilon {
		if existing == target {
			return
		}
	}
	n.epsilon = append(n.epsilon, target)
}

func epsilonClosure(nodes []*node) []*node {
	stack := make([]*node, len(nodes))
	copy(stack, nodes)
	visited := make(map[*node]struct{}, len(nodes))
	for _, n := range nodes {
		visited[n] = struct{}{}
	}
	result := make([]*node, 0, len(nodes))
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		result = append(result, n)
		for _, next := range n.epsilon {
			if _, seen := visited[next]; !seen {
				visited[next] = struct{}{}
				stack = append(stack, next)
			}
		}
	}
	return result
}

func uniqueNodes(nodes []*node) []*node {
	seen := make(map[*node]struct{}, len(nodes))
	result := make([]*node, 0, len(nodes))
	for _, n := range nodes {
		if _, ok := seen[n]; !ok {
			seen[n] = struct{}{}
			result = append(result, n)
		}
	}
	return result
}
