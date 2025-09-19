package tree

import (
        "fmt"
        "strings"

        "jsonpath-sdk/internal/spec"
)

// NodeType represents different types of tree nodes
type NodeType int

const (
        NodeRoot NodeType = iota
        NodeProperty
        NodeBracket
        NodeGroup
        NodeRepetition
)

// TreeNode represents a node in the trie/radix tree
type TreeNode struct {
        Type         NodeType
        Value        string  // property name or bracket content
        Quoted       bool    // for bracket nodes, whether the content was quoted
        Children     []*TreeNode
        Alternatives []*TreeNode // for group nodes, represents |alternatives
        IsRepeating  bool        // for repetition nodes, marks {*} patterns
        IsEnd        bool        // marks valid end of a path expression
        Parent       *TreeNode   // reference to parent node for cycle handling
}

// PatternTree represents the complete trie/radix tree for path patterns
type PatternTree struct {
        Root *TreeNode
}

// NewPatternTree creates a new pattern tree
func NewPatternTree() *PatternTree {
        return &PatternTree{
                Root: &TreeNode{
                        Type:     NodeRoot,
                        Value:    "$",
                        Children: make([]*TreeNode, 0),
                },
        }
}

// AddPattern adds a parsed path expression to the tree
func (t *PatternTree) AddPattern(expr *spec.PathExpression) error {
        current := t.Root
        
        // Process each segment in the expression
        for _, segment := range expr.Segments {
                var err error
                current, err = t.addSegment(current, segment)
                if err != nil {
                        return err
                }
        }
        
        // Mark this as a valid endpoint
        // Special handling: if the last segment was a group, mark the last group as end
        if len(expr.Segments) > 0 {
                lastSegment := expr.Segments[len(expr.Segments)-1]
                if _, ok := lastSegment.(*spec.GroupNode); ok {
                        // Find the last group node in current's children and mark it as end
                        for i := len(current.Children) - 1; i >= 0; i-- {
                                child := current.Children[i]
                                if child.Type == NodeGroup {
                                        child.IsEnd = true
                                        break
                                }
                        }
                }
        }
        current.IsEnd = true
        return nil
}

// addSegment adds a single segment to the tree from the current node
func (t *PatternTree) addSegment(current *TreeNode, segment spec.ASTNode) (*TreeNode, error) {
        switch node := segment.(type) {
        case *spec.PropertyNode:
                return t.addPropertyNode(current, node)
                
        case *spec.BracketNode:
                return t.addBracketNode(current, node)
                
        case *spec.GroupNode:
                return t.addGroupNode(current, node)
                
        default:
                return nil, fmt.Errorf("unsupported AST node type: %T", segment)
        }
}

// addPropertyNode adds a property node to the tree
func (t *PatternTree) addPropertyNode(current *TreeNode, prop *spec.PropertyNode) (*TreeNode, error) {
        // Look for existing property child with same name
        for _, child := range current.Children {
                if child.Type == NodeProperty && child.Value == prop.Name {
                        return child, nil
                }
        }
        
        // Create new property node
        propNode := &TreeNode{
                Type:     NodeProperty,
                Value:    prop.Name,
                Children: make([]*TreeNode, 0),
                Parent:   current,
        }
        
        current.Children = append(current.Children, propNode)
        return propNode, nil
}

// addBracketNode adds a bracket notation node to the tree
func (t *PatternTree) addBracketNode(current *TreeNode, bracket *spec.BracketNode) (*TreeNode, error) {
        // Look for existing bracket child with same content and quoting
        for _, child := range current.Children {
                if child.Type == NodeBracket && child.Value == bracket.Content && child.Quoted == bracket.Quoted {
                        return child, nil
                }
        }
        
        // Create new bracket node
        bracketNode := &TreeNode{
                Type:     NodeBracket,
                Value:    bracket.Content,
                Quoted:   bracket.Quoted,
                Children: make([]*TreeNode, 0),
                Parent:   current,
        }
        
        current.Children = append(current.Children, bracketNode)
        return bracketNode, nil
}

// addGroupNode adds a group node with alternatives and optional repetition
func (t *PatternTree) addGroupNode(current *TreeNode, group *spec.GroupNode) (*TreeNode, error) {
        // Create group node
        groupNode := &TreeNode{
                Type:         NodeGroup,
                IsRepeating:  group.Repetition,
                Alternatives: make([]*TreeNode, 0),
                Children:     make([]*TreeNode, 0),
                Parent:       current,
        }
        
        // Process each alternative in the group
        for _, alternative := range group.Alternatives {
                altRoot := &TreeNode{
                        Type:     NodeRoot, // temporary root for alternative
                        Children: make([]*TreeNode, 0),
                        Parent:   groupNode,
                }
                
                // Build tree for this alternative
                currentAlt := altRoot
                for _, altSegment := range alternative {
                        var err error
                        currentAlt, err = t.addSegment(currentAlt, altSegment)
                        if err != nil {
                                return nil, err
                        }
                }
                currentAlt.IsEnd = true
                
                // Add the alternative tree to the group
                groupNode.Alternatives = append(groupNode.Alternatives, altRoot)
        }
        
        current.Children = append(current.Children, groupNode)
        
        // Return the parent so subsequent segments become siblings of the group
        return current, nil
}

// MatchPath tests if a given JSON path matches any pattern in the tree
func (t *PatternTree) MatchPath(jsonPath []string) bool {
        return t.matchFromNode(t.Root, jsonPath, 0)
}

// matchFromNode recursively matches path segments from a given node
func (t *PatternTree) matchFromNode(node *TreeNode, path []string, pathIndex int) bool {
        // If we've consumed all path segments, check if we're at a valid end
        if pathIndex >= len(path) {
                return node.IsEnd
        }
        
        currentSegment := path[pathIndex]
        
        // Check all children of current node
        for _, child := range node.Children {
                switch child.Type {
                case NodeProperty:
                        if child.Value == currentSegment {
                                if t.matchFromNode(child, path, pathIndex+1) {
                                        return true
                                }
                        }
                        
                case NodeBracket:
                        // For bracket nodes, we match against the segment directly
                        // In real JSON path evaluation, this would be more complex
                        if t.matchBracketSegment(child, currentSegment) {
                                if t.matchFromNode(child, path, pathIndex+1) {
                                        return true
                                }
                        }
                        
                case NodeGroup:
                        if t.matchGroupNode(child, path, pathIndex) {
                                return true
                        }
                        
                }
        }
        
        return false
}

// matchBracketSegment matches a bracket pattern against a path segment
func (t *PatternTree) matchBracketSegment(node *TreeNode, segment string) bool {
        // For simplicity, we do literal string matching
        // In a full implementation, this would handle property name resolution
        if node.Quoted {
                // Quoted bracket content should match exactly
                return node.Value == segment
        } else {
                // Unquoted bracket content matches as property name
                return node.Value == segment
        }
}

// matchGroupNode matches a group node with alternatives
func (t *PatternTree) matchGroupNode(groupNode *TreeNode, path []string, pathIndex int) bool {
        if groupNode.IsRepeating {
                // For repeating groups, try 0 or more iterations
                return t.matchRepeatingGroup(groupNode, path, pathIndex)
        } else {
                // For non-repeating groups, match one alternative then continue with siblings
                return t.matchNonRepeatingGroup(groupNode, path, pathIndex)
        }
}

// matchNonRepeatingGroup matches a non-repeating group and continues with siblings
func (t *PatternTree) matchNonRepeatingGroup(groupNode *TreeNode, path []string, pathIndex int) bool {
        // Try each alternative
        for _, alternative := range groupNode.Alternatives {
                currentIndex := pathIndex
                if t.matchAlternativeSegments(alternative, path, &currentIndex) {
                        // After matching alternative, continue with next sibling of the group
                        return t.continueAfterGroup(groupNode, path, currentIndex)
                }
        }
        return false
}

// matchRepeatingGroup handles repetition logic
func (t *PatternTree) matchRepeatingGroup(groupNode *TreeNode, path []string, pathIndex int) bool {
        // First, try zero iterations (go directly to what comes after the group)
        if t.continueAfterGroup(groupNode, path, pathIndex) {
                return true
        }
        
        // Try one or more iterations
        for _, alternative := range groupNode.Alternatives {
                currentIndex := pathIndex
                if t.matchAlternativeSegments(alternative, path, &currentIndex) {
                        // After one iteration, recursively try more iterations or continue after group
                        if t.matchRepeatingGroup(groupNode, path, currentIndex) {
                                return true
                        }
                }
        }
        
        return false
}

// continueAfterGroup continues matching after a group node (siblings)
func (t *PatternTree) continueAfterGroup(groupNode *TreeNode, path []string, pathIndex int) bool {
        // First check if path is exhausted
        if pathIndex >= len(path) {
                return groupNode.IsEnd || (groupNode.Parent != nil && groupNode.Parent.IsEnd)
        }
        
        if groupNode.Parent == nil {
                return false
        }
        
        parent := groupNode.Parent
        currentSegment := path[pathIndex]
        
        // Find the group's position and try all siblings after it
        for i, child := range parent.Children {
                if child == groupNode {
                        // Try all siblings after the group
                        for j := i + 1; j < len(parent.Children); j++ {
                                sibling := parent.Children[j]
                                
                                switch sibling.Type {
                                case NodeProperty:
                                        if sibling.Value == currentSegment {
                                                if t.matchFromNode(sibling, path, pathIndex+1) {
                                                        return true
                                                }
                                        }
                                case NodeBracket:
                                        if t.matchBracketSegment(sibling, currentSegment) {
                                                if t.matchFromNode(sibling, path, pathIndex+1) {
                                                        return true
                                                }
                                        }
                                case NodeGroup:
                                        if t.matchGroupNode(sibling, path, pathIndex) {
                                                return true
                                        }
                                }
                        }
                        break
                }
        }
        
        return false
}

// matchAlternativeSegments matches the segments within an alternative using backtracking
func (t *PatternTree) matchAlternativeSegments(node *TreeNode, path []string, pathIndex *int) bool {
        if node.Type == NodeRoot {
                // For root nodes, try matching from this position with DFS
                return t.matchAlternativeDFS(node, path, *pathIndex, pathIndex)
        }
        
        // Match single segment
        if *pathIndex >= len(path) {
                return false
        }
        
        segment := path[*pathIndex]
        matched := false
        
        switch node.Type {
        case NodeProperty:
                matched = (node.Value == segment)
        case NodeBracket:
                matched = t.matchBracketSegment(node, segment)
        }
        
        if matched {
                (*pathIndex)++
                // Continue with children using DFS
                return t.matchAlternativeDFS(node, path, *pathIndex, pathIndex)
        }
        
        return false
}

// matchAlternativeDFS performs depth-first search on alternative tree with backtracking
func (t *PatternTree) matchAlternativeDFS(node *TreeNode, path []string, currentIndex int, pathIndex *int) bool {
        // If no children, we've successfully matched this branch
        if len(node.Children) == 0 {
                *pathIndex = currentIndex
                return true
        }
        
        // Try each child path independently with backtracking
        for _, child := range node.Children {
                // Save current position for backtracking
                savedIndex := currentIndex
                
                // Try matching this child path
                tempIndex := savedIndex
                if t.matchAlternativeSegments(child, path, &tempIndex) {
                        *pathIndex = tempIndex
                        return true
                }
                
                // Backtrack - tempIndex is automatically reset by the recursive call failure
        }
        
        return false
}



// String returns a string representation of the tree for debugging
func (t *PatternTree) String() string {
        var sb strings.Builder
        t.printNode(&sb, t.Root, 0)
        return sb.String()
}

// printNode recursively prints tree structure
func (t *PatternTree) printNode(sb *strings.Builder, node *TreeNode, depth int) {
        indent := strings.Repeat("  ", depth)
        
        switch node.Type {
        case NodeRoot:
                sb.WriteString(fmt.Sprintf("%s$ (ROOT)\n", indent))
        case NodeProperty:
                sb.WriteString(fmt.Sprintf("%s.%s\n", indent, node.Value))
        case NodeBracket:
                if node.Quoted {
                        sb.WriteString(fmt.Sprintf("%s[\"%s\"]\n", indent, node.Value))
                } else {
                        sb.WriteString(fmt.Sprintf("%s[%s]\n", indent, node.Value))
                }
        case NodeGroup:
                sb.WriteString(fmt.Sprintf("%s(GROUP", indent))
                if node.IsRepeating {
                        sb.WriteString(" {*}")
                }
                sb.WriteString(")\n")
                
                for i, alt := range node.Alternatives {
                        sb.WriteString(fmt.Sprintf("%s  ALT%d:\n", indent, i))
                        t.printNode(sb, alt, depth+2)
                }
        case NodeRepetition:
                sb.WriteString(fmt.Sprintf("%s{*} REPETITION\n", indent))
        }
        
        if node.IsEnd {
                sb.WriteString(fmt.Sprintf("%s  (END)\n", indent))
        }
        
        for _, child := range node.Children {
                t.printNode(sb, child, depth+1)
        }
}