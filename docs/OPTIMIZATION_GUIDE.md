# Validator Optimization Guide

Based on benchmark results and code analysis, this document provides actionable optimizations to improve validator performance.

## Current Performance Bottlenecks

### 🔴 Critical Issues

#### 1. **Repeated PatternTree Creation** (Most Critical)
**Location**: `validators/optimized_generic.go:157-171` and `validators/complex_pattern.go:144-168`

**Problem**:
```go
// This code runs for EVERY matched path
func (v *OptimizedGenericValidator) getMetadataForPath(path string) json.RawMessage {
    for patternStr, metadata := range v.config.Paths {
        expr, err := parser.ParseExpression(patternStr)  // ❌ Parse same pattern repeatedly
        tempTree := tree.NewPatternTree()                // ❌ Create temp tree
        tempTree.AddPattern(expr)                        // ❌ Build tree again
        if tempTree.MatchSegments(segments) {
            return metadata
        }
    }
}
```

**Impact**: With 5 patterns and 100 matched paths, creates 500 PatternTree objects.

**Solution**:
```go
// Parse patterns once at initialization
type OptimizedGenericValidator struct {
    config          *GenericValidatorConfig
    patternTree     *tree.PatternTree
    precomputed     map[string]precomputedValidation
    processor       *jsonpkg.PathExtractor
    patternToMeta   map[string]json.RawMessage  // ✅ Add this
    parsedPatterns  map[string]*tree.PatternTree // ✅ Add this
}

func NewOptimizedGenericValidator(config *GenericValidatorConfig) (*OptimizedGenericValidator, error) {
    // ... existing code ...

    // Pre-parse all patterns
    parsedPatterns := make(map[string]*tree.PatternTree)
    patternToMeta := make(map[string]json.RawMessage)

    for patternStr, metadata := range config.Paths {
        expr, err := parser.ParseExpression(patternStr)
        if err != nil {
            return nil, err
        }

        patternTree := tree.NewPatternTree()
        patternTree.AddPattern(expr)
        parsedPatterns[patternStr] = patternTree
        patternToMeta[patternStr] = metadata
    }

    return &OptimizedGenericValidator{
        // ... existing fields ...
        parsedPatterns: parsedPatterns,
        patternToMeta:  patternToMeta,
    }, nil
}

// Optimized metadata lookup
func (v *OptimizedGenericValidator) getMetadataForPath(path string) json.RawMessage {
    // Try exact match first
    if metadata, exists := v.config.Paths[path]; exists {
        return metadata
    }

    // Use pre-parsed patterns
    segments := v.processor.ConvertPathToSegments(path)
    for patternStr, patternTree := range v.parsedPatterns {
        if patternTree.MatchSegments(segments) {
            return v.patternToMeta[patternStr]
        }
    }

    return nil
}
```

**Expected Impact**: 10-20x speedup for metadata lookup, reducing allocations by 90%.

---

#### 2. **Path String Construction Allocations**
**Location**: `json/processor.go:118-141`

**Problem**:
```go
func (pe *PathExtractor) extractPathsFromAST(node *ast.Node, currentPath string, paths *[]string) {
    // ...
    for key := range objMap {
        child := node.Get(key)
        newPath := currentPath + "." + key  // ❌ Creates new string allocation
        pe.extractPathsFromAST(child, newPath, paths)
    }
}
```

**Solution**:
```go
func (pe *PathExtractor) extractPathsFromAST(node *ast.Node, currentPath string, paths *[]string) {
    *paths = append(*paths, currentPath)

    switch node.Type() {
    case ast.V_OBJECT:
        if objMap, err := node.Map(); err == nil {
            var builder strings.Builder
            builder.Grow(len(currentPath) + 50) // Pre-allocate

            for key := range objMap {
                builder.Reset()
                builder.WriteString(currentPath)
                builder.WriteByte('.')
                builder.WriteString(key)

                child := node.Get(key)
                pe.extractPathsFromAST(child, builder.String(), paths)
            }
        }
    case ast.V_ARRAY:
        if arraySlice, err := node.Array(); err == nil {
            var builder strings.Builder
            builder.Grow(len(currentPath) + 20)

            for i := range arraySlice {
                builder.Reset()
                builder.WriteString(currentPath)
                builder.WriteByte('[')
                builder.WriteString(strconv.Itoa(i))
                builder.WriteByte(']')

                child := node.Index(i)
                pe.extractPathsFromAST(child, builder.String(), paths)
            }
        }
    }
}
```

**Expected Impact**: 30-40% reduction in memory allocations during path extraction.

---

#### 3. **Segment Conversion Overhead**
**Location**: `json/processor.go:172-242`

**Problem**: Called for every path, creates new slices every time.

**Solution**: Use sync.Pool for segment slice reuse:
```go
var segmentPool = sync.Pool{
    New: func() interface{} {
        segments := make([]spec.PathSegment, 0, 16)
        return &segments
    },
}

func (pe *PathExtractor) ConvertPathToSegments(path string) []spec.PathSegment {
    ptr := segmentPool.Get().(*[]spec.PathSegment)
    segments := (*ptr)[:0] // Reset length, keep capacity
    defer segmentPool.Put(ptr)

    // ... existing parsing logic ...

    // Return a copy since we're pooling
    result := make([]spec.PathSegment, len(segments))
    copy(result, segments)
    return result
}
```

**Expected Impact**: 20-30% reduction in allocations for path processing.

---

#### 4. **Pattern Matching Allocations**
**Location**: `tree/tree.go:244-291`

**Problem**:
```go
func (t *PatternTree) MatchSegments(segments []spec.PathSegment) bool {
    current := epsilonClosure([]*node{t.root})
    for _, segment := range segments {
        nextSet := make(map[*node]struct{})  // ❌ New map every iteration
        // ...
        next := make([]*node, 0, len(nextSet))  // ❌ New slice every iteration
    }
}
```

**Solution**:
```go
var nodeSetPool = sync.Pool{
    New: func() interface{} {
        m := make(map[*node]struct{}, 16)
        return &m
    },
}

var nodeSlicePool = sync.Pool{
    New: func() interface{} {
        s := make([]*node, 0, 16)
        return &s
    },
}

func (t *PatternTree) MatchSegments(segments []spec.PathSegment) bool {
    current := epsilonClosure([]*node{t.root})

    nextSetPtr := nodeSetPool.Get().(*map[*node]struct{})
    defer nodeSetPool.Put(nextSetPtr)
    nextSet := *nextSetPtr

    nextSlicePtr := nodeSlicePool.Get().(*[]*node)
    defer nodeSlicePool.Put(nextSlicePtr)

    for _, segment := range segments {
        // Clear the map
        for k := range nextSet {
            delete(nextSet, k)
        }

        // ... matching logic using nextSet ...

        // Reuse slice
        next := (*nextSlicePtr)[:0]
        for n := range nextSet {
            next = append(next, n)
        }
        current = epsilonClosure(next)

        if len(current) == 0 {
            return false
        }
    }
    // ... rest of function
}
```

**Expected Impact**: 40-50% reduction in allocations during pattern matching.

---

### 🟡 Medium Priority Issues

#### 5. **Redundant Time.Now() Calls**
**Location**: `validators/optimized_generic.go:67`

**Problem**: Called for every ValidationResult even though timestamp is barely used.

**Solution**:
```go
// Option 1: Single timestamp for entire report
func (v *OptimizedGenericValidator) Validate(jsonData string) (*ValidationReport, error) {
    start := time.Now()

    // ... validation logic ...

    timestamp := time.Now() // Single timestamp
    for _, precomp := range v.precomputed {
        // ... extract value ...
        validationResult := ValidationResult{
            Path:      precomp.path,
            Value:     value,
            Metadata:  precomp.metadata,
            Timestamp: timestamp,  // Reuse same timestamp
            Valid:     true,
        }
        results = append(results, validationResult)
    }
}

// Option 2: Make timestamp optional/lazy
type ValidationResult struct {
    Path        string
    Value       interface{}
    Metadata    json.RawMessage
    Valid       bool
    Description string
    Error       error
    // Remove Timestamp field or make it pointer for lazy init
}
```

**Expected Impact**: 5-10% improvement for high-volume validations.

---

#### 6. **Pre-allocate Result Slices**
**Location**: `validators/optimized_generic.go:57`

**Problem**: Slice grows dynamically, causing reallocations.

**Solution**:
```go
func (v *OptimizedGenericValidator) Validate(jsonData string) (*ValidationReport, error) {
    start := time.Now()

    if len(v.precomputed) == 0 {
        if err := v.precomputePaths(jsonData); err != nil {
            return nil, fmt.Errorf("failed to precompute paths: %w", err)
        }
    }

    // Pre-allocate with known size
    results := make([]ValidationResult, 0, len(v.precomputed))

    // ... rest of validation
}
```

**Expected Impact**: 10-15% improvement by avoiding slice reallocations.

---

### 🟢 Low Priority Optimizations

#### 7. **Lazy Value Extraction**
Only extract values when actually needed:

```go
type LazyValidationResult struct {
    Path        string
    Metadata    json.RawMessage
    Valid       bool
    extractor   *jsonpkg.PathExtractor
    jsonData    string
    cachedValue interface{}
}

func (r *LazyValidationResult) Value() interface{} {
    if r.cachedValue == nil {
        r.cachedValue, _ = r.extractor.ExtractValue(r.jsonData, r.Path)
    }
    return r.cachedValue
}
```

**Expected Impact**: 20-30% improvement when values are not always accessed.

---

#### 8. **Parallel Path Processing**
For large JSON documents:

```go
func (v *OptimizedGenericValidator) Validate(jsonData string) (*ValidationReport, error) {
    // ... precompute paths ...

    // Process paths in parallel
    var wg sync.WaitGroup
    resultChan := make(chan ValidationResult, len(v.precomputed))

    const workerCount = 4
    precomputedSlice := make([]precomputedValidation, 0, len(v.precomputed))
    for _, pc := range v.precomputed {
        precomputedSlice = append(precomputedSlice, pc)
    }

    chunkSize := (len(precomputedSlice) + workerCount - 1) / workerCount

    for i := 0; i < workerCount; i++ {
        start := i * chunkSize
        end := start + chunkSize
        if end > len(precomputedSlice) {
            end = len(precomputedSlice)
        }

        wg.Add(1)
        go func(chunk []precomputedValidation) {
            defer wg.Done()
            for _, precomp := range chunk {
                value, err := v.processor.ExtractValue(jsonData, precomp.path)
                if err == nil && value != nil {
                    resultChan <- ValidationResult{
                        Path:     precomp.path,
                        Value:    value,
                        Metadata: precomp.metadata,
                        Valid:    true,
                    }
                }
            }
        }(precomputedSlice[start:end])
    }

    go func() {
        wg.Wait()
        close(resultChan)
    }()

    results := make([]ValidationResult, 0, len(v.precomputed))
    for result := range resultChan {
        results = append(results, result)
    }

    // ... process results
}
```

**Expected Impact**: 2-3x speedup for large documents (1000+ paths).

---

## Performance Improvement Summary

| Optimization | Expected Speedup | Complexity | Priority |
|--------------|------------------|------------|----------|
| **Cache parsed patterns** | 10-20x | Low | 🔴 Critical |
| **String builder for paths** | 1.3-1.4x | Low | 🔴 Critical |
| **Object pooling (segments)** | 1.2-1.3x | Medium | 🔴 Critical |
| **Object pooling (tree matching)** | 1.4-1.5x | Medium | 🔴 Critical |
| **Remove redundant timestamps** | 1.05-1.1x | Low | 🟡 Medium |
| **Pre-allocate slices** | 1.1-1.15x | Low | 🟡 Medium |
| **Lazy value extraction** | 1.2-1.3x | Medium | 🟢 Low |
| **Parallel processing** | 2-3x | High | 🟢 Low |

### Combined Impact Estimate

Implementing **all critical optimizations** could achieve:
- **15-30x speedup** for `getMetadataForPath` operations
- **2-4x overall speedup** for full validation
- **60-80% reduction** in memory allocations
- **Sub-microsecond validation** for simple patterns

### Target Performance Goals

After optimization:
- **OptimizedGeneric**: < 1μs for `{*}` patterns (from 2.5μs)
- **ComplexPattern**: < 40μs for complex patterns (from 82.8μs)
- **Memory usage**: < 30KB per validation (from 48.5KB)
- **Allocations**: < 50 per operation (from 123)

---

## Implementation Roadmap

### Phase 1: Quick Wins (1-2 hours)
1. Cache parsed patterns in constructors
2. Pre-allocate result slices
3. Remove redundant Time.Now() calls

**Expected gain**: 10-15x improvement in hot paths

### Phase 2: Memory Optimizations (2-4 hours)
1. Implement sync.Pool for segments
2. Implement sync.Pool for tree matching
3. Use strings.Builder for path construction

**Expected gain**: Additional 1.5-2x improvement

### Phase 3: Advanced Optimizations (4-8 hours)
1. Lazy value extraction
2. Parallel path processing
3. String interning for common paths

**Expected gain**: Additional 2-3x for large documents

---

## Benchmarking Strategy

After each optimization:

```bash
# Run benchmarks with memory profiling
go test -bench=. -benchmem -cpuprofile=cpu.prof -memprofile=mem.prof ./validators

# Compare before/after
benchstat before.txt after.txt

# Profile allocations
go tool pprof -alloc_space mem.prof
```

Focus on:
- **Time per operation** (should decrease)
- **Allocations per op** (should decrease significantly)
- **Bytes per op** (should decrease)

---

## Validation

Test correctness after each optimization:

```bash
# Ensure all tests pass
go test ./... -v

# Run comprehensive validation tests
go test ./validators -run TestRecursiveNested -v

# Verify benchmark accuracy
go test -bench=BenchmarkRecursiveNestedSchema -benchtime=10s
```

---

## Conclusion

The **most critical optimization** is caching parsed patterns to eliminate the repeated `PatternTree` creation bottleneck. This single change could improve performance by **10-20x** in metadata lookup paths.

Combined with memory pooling and string optimizations, we can achieve:
- **Sub-microsecond validation** for simple patterns
- **2-4x overall speedup** across all validators
- **60-80% reduction** in memory allocations

These improvements would make the OptimizedGeneric validator truly optimal for production use at scale.
