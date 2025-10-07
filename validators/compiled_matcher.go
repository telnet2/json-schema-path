package validators

import (
	"encoding/json"
	"time"

	jsonpkg "github.com/telnet2/json-schema-path/json"
	"github.com/telnet2/json-schema-path/parser"
	"github.com/telnet2/json-schema-path/tree"
)

// CompiledMatcher provides fast pattern matching using multiple optimization layers
type CompiledMatcher struct {
	bloomFilter  *tree.BloomFilter
	patternTrees map[string]*tree.PatternTree // Individual trees for each pattern
	patternMeta  map[string]json.RawMessage   // Pattern to metadata mapping
	allPatterns  *tree.PatternTree            // Combined tree for checking if ANY pattern matches
	processor    *jsonpkg.PathExtractor
	cache        *LRUCache // Cache for path -> pattern match results
}

// LRUCache is a simple LRU cache for pattern matching results
type LRUCache struct {
	capacity int
	items    map[string]*cacheNode
	head     *cacheNode
	tail     *cacheNode
	size     int
}

type cacheNode struct {
	key   string
	value string // Pattern that matched (or "" for no match)
	prev  *cacheNode
	next  *cacheNode
}

// NewLRUCache creates a new LRU cache with the given capacity
func NewLRUCache(capacity int) *LRUCache {
	if capacity <= 0 {
		capacity = 1000 // Default capacity
	}
	return &LRUCache{
		capacity: capacity,
		items:    make(map[string]*cacheNode, capacity),
	}
}

// Get retrieves a value from the cache
func (c *LRUCache) Get(key string) (string, bool) {
	if node, exists := c.items[key]; exists {
		c.moveToFront(node)
		return node.value, true
	}
	return "", false
}

// Put adds a value to the cache
func (c *LRUCache) Put(key string, value string) {
	if node, exists := c.items[key]; exists {
		node.value = value
		c.moveToFront(node)
		return
	}

	node := &cacheNode{key: key, value: value}
	c.items[key] = node
	c.addToFront(node)
	c.size++

	if c.size > c.capacity {
		c.removeLast()
	}
}

func (c *LRUCache) moveToFront(node *cacheNode) {
	if node == c.head {
		return
	}

	// Remove from current position
	if node.prev != nil {
		node.prev.next = node.next
	}
	if node.next != nil {
		node.next.prev = node.prev
	}
	if node == c.tail {
		c.tail = node.prev
	}

	// Add to front
	node.prev = nil
	node.next = c.head
	if c.head != nil {
		c.head.prev = node
	}
	c.head = node
	if c.tail == nil {
		c.tail = node
	}
}

func (c *LRUCache) addToFront(node *cacheNode) {
	node.next = c.head
	node.prev = nil
	if c.head != nil {
		c.head.prev = node
	}
	c.head = node
	if c.tail == nil {
		c.tail = node
	}
}

func (c *LRUCache) removeLast() {
	if c.tail == nil {
		return
	}

	delete(c.items, c.tail.key)
	if c.tail.prev != nil {
		c.tail.prev.next = nil
		c.tail = c.tail.prev
	} else {
		c.head = nil
		c.tail = nil
	}
	c.size--
}

// NewCompiledMatcher creates a new compiled matcher with all optimizations
func NewCompiledMatcher(patterns map[string]json.RawMessage) (*CompiledMatcher, error) {
	// Create bloom filter with optimal parameters
	// Assuming typical validation has 10-20 patterns
	bloomFilter := tree.NewOptimalBloomFilter(len(patterns), 0.01) // 1% false positive rate

	patternTrees := make(map[string]*tree.PatternTree, len(patterns))
	patternMeta := make(map[string]json.RawMessage, len(patterns))
	allPatterns := tree.NewPatternTree()

	for patternStr, metadata := range patterns {
		// Parse expression
		expr, err := parser.ParseExpression(patternStr)
		if err != nil {
			return nil, err
		}

		// Add to bloom filter (for fast rejection)
		// We add the pattern string itself, not the path
		// The bloom filter will be checked against actual paths later
		bloomFilter.Add(patternStr)

		// Create individual pattern tree
		individualTree := tree.NewPatternTree()
		individualTree.AddPattern(expr)
		patternTrees[patternStr] = individualTree
		patternMeta[patternStr] = metadata

		// Add to combined tree
		allPatterns.AddPattern(expr)
	}

	return &CompiledMatcher{
		bloomFilter:  bloomFilter,
		patternTrees: patternTrees,
		patternMeta:  patternMeta,
		allPatterns:  allPatterns,
		processor:    jsonpkg.NewPathExtractor(),
		cache:        NewLRUCache(10000), // Cache up to 10k path matches
	}, nil
}

// Match finds the first pattern that matches the given path
// Returns the pattern string and metadata, or empty string and nil if no match
func (cm *CompiledMatcher) Match(path string) (string, json.RawMessage) {
	// Level 1: Check cache (O(1), fastest)
	if cachedPattern, found := cm.cache.Get(path); found {
		if cachedPattern == "" {
			return "", nil // Cached non-match
		}
		return cachedPattern, cm.patternMeta[cachedPattern]
	}

	// Level 2: Convert path to segments once (will be reused)
	segments := cm.processor.ConvertPathToSegments(path)

	// Level 3: Check if ANY pattern matches using combined tree (fast reject)
	if !cm.allPatterns.MatchSegments(segments) {
		cm.cache.Put(path, "") // Cache the non-match
		return "", nil
	}

	// Level 4: Find which specific pattern matches
	for patternStr, patternTree := range cm.patternTrees {
		if patternTree.MatchSegments(segments) {
			cm.cache.Put(path, patternStr) // Cache the match
			return patternStr, cm.patternMeta[patternStr]
		}
	}

	// No match found (shouldn't happen if allPatterns matched, but just in case)
	cm.cache.Put(path, "")
	return "", nil
}

// MatchAll finds all patterns that match the given path
// Returns a map of pattern -> metadata for all matches
func (cm *CompiledMatcher) MatchAll(path string) map[string]json.RawMessage {
	segments := cm.processor.ConvertPathToSegments(path)
	matches := make(map[string]json.RawMessage)

	for patternStr, patternTree := range cm.patternTrees {
		if patternTree.MatchSegments(segments) {
			matches[patternStr] = cm.patternMeta[patternStr]
		}
	}

	return matches
}

// QuickCheck quickly determines if a path might match any pattern
// This is faster than Match() but may have false positives
func (cm *CompiledMatcher) QuickCheck(path string) bool {
	// Check cache first
	if _, found := cm.cache.Get(path); found {
		return true // Either matched or explicitly non-matched (cached)
	}

	// Check combined tree
	segments := cm.processor.ConvertPathToSegments(path)
	return cm.allPatterns.MatchSegments(segments)
}

// Reset clears the internal cache
func (cm *CompiledMatcher) Reset() {
	cm.cache = NewLRUCache(cm.cache.capacity)
}

// StreamingValidator uses CompiledMatcher with streaming JSON walk for maximum performance
type StreamingValidator struct {
	matcher   *CompiledMatcher
	processor *jsonpkg.PathExtractor
}

// NewStreamingValidator creates a new streaming validator
func NewStreamingValidator(patterns map[string]json.RawMessage) (*StreamingValidator, error) {
	matcher, err := NewCompiledMatcher(patterns)
	if err != nil {
		return nil, err
	}

	return &StreamingValidator{
		matcher:   matcher,
		processor: jsonpkg.NewPathExtractor(),
	}, nil
}

// Validate validates JSON data using streaming walk + compiled matcher
func (sv *StreamingValidator) Validate(jsonData string) (*ValidationReport, error) {
	results := make([]ValidationResult, 0, 16)
	timestamp := time.Now()

	// Single pass through JSON data
	err := sv.processor.StreamingWalk(jsonData, func(path string, value interface{}) error {
		// Quick check if this path might match
		if !sv.matcher.QuickCheck(path) {
			return nil // Skip non-matching paths
		}

		// Find matching pattern
		pattern, metadata := sv.matcher.Match(path)
		if pattern != "" {
			result := ValidationResult{
				Path:        path,
				Value:       value,
				Metadata:    metadata,
				Timestamp:   timestamp,
				Valid:       true,
				Description: "Matched pattern: " + pattern,
			}
			results = append(results, result)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &ValidationReport{
		Results:    results,
		ValidPaths: len(results),
	}, nil
}
