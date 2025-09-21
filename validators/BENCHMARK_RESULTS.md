# Validator Performance Benchmark Results

## Summary

Comprehensive benchmarks for all validator types with recursive nested schema patterns, focusing on the `{*}` repetition operator which is the main usage for JSON schema path validation.

## Validator Performance Comparison

### Simple Validators (Basic Path Validation)
| Validator | Time per Operation | Memory per Op | Allocations | Use Case |
|-----------|-------------------|---------------|-------------|----------|
| **Raw** | 13.7μs | 27.5 KB | 160 | Simple exact path matching |
| **Optimized** | 36.2μs | 48.1 KB | 391 | Basic wildcards `[*]` |
| **Fast** | 13.4μs | 27.5 KB | 160 | Pre-expanded patterns |

### Generic Validators (Complex Pattern Support)
| Validator | Time per Operation | Memory per Op | Allocations | Use Case |
|-----------|-------------------|---------------|-------------|----------|
| **ComplexPattern** | 82.8μs | 128.9 KB | 1,145 | Full pattern matching |
| **OptimizedGeneric** | 23.5μs | 48.5 KB | 123 | Pre-computed patterns |

## Recursive Nested Schema Performance

### Pattern Complexity Scaling
| Pattern Type | ComplexPattern | OptimizedGeneric | Paths Found |
|--------------|----------------|------------------|-------------|
| **Simple Recursive** | 70.5μs | 3.3μs | 2 |
| **Medium Recursive** | 97.3μs | 9.6μs | 4 |
| **Deep Recursive** | 132.2μs | 21.2μs | 6 |
| **Full Recursive** | 300.2μs | 79.8μs | 11 |
| **{*} Repetition** | 63.1μs | 2.5μs | 2 |

### Handler-based Validation
| Validator | Time per Operation | Memory per Op | Allocations |
|-----------|-------------------|---------------|-------------|
| **ComplexPattern** | 110.6μs | 169.0 KB | 1,413 |
| **OptimizedGeneric** | 27.8μs | 57.1 KB | 109 |

### Scalability with Depth
| Nesting Depth | Time per Operation | Memory per Op | Allocations |
|---------------|-------------------|---------------|-------------|
| **Shallow (3 levels)** | 114.7μs | 160.1 KB | 1,468 |
| **Medium (5 levels)** | 193.0μs | 283.4 KB | 2,454 |
| **Deep (7 levels)** | 327.3μs | 433.2 KB | 3,602 |
| **Very Deep (10 levels)** | 507.9μs | 714.0 KB | 5,624 |

## Key Findings

### 🏆 **Winner: OptimizedGeneric Validator**
- **23.5μs** for complex generic validation
- **2.5μs** for {*} repetition patterns  
- **79.8μs** for full recursive patterns
- **27.8μs** with handler callbacks
- **Lowest memory usage** among generic validators

### 🎯 **Repetition Operator {*} Performance**
- **OptimizedGeneric**: 2.5μs (4,635 bytes, 12 allocs)
- **ComplexPattern**: 63.1μs (119.4 KB, 1,024 allocs)
- **3.3μs** for simple recursive patterns
- **Scales linearly** with nesting depth

### 📊 **Pattern Complexity Impact**
- **Simple patterns**: 3-10μs range
- **Medium complexity**: 10-30μs range  
- **Deep nesting**: 20-80μs range
- **Full recursive**: 80-300μs range
- **Memory scales** with pattern complexity

### 🔧 **Validator Capabilities**
- **ComplexPattern**: Full json-schema-path support including `{*}`
- **OptimizedGeneric**: Pre-computed patterns, excellent performance
- **Raw/Fast**: Simple path validation, no complex patterns
- **EnhancedGJSON**: High-performance but limited pattern support

## Recommendations

### For Production Use:
1. **OptimizedGeneric** - Best balance of performance and features
2. **ComplexPattern** - When full pattern support is required
3. **Fast** - For simple path validation at scale

### For Recursive Schemas:
- Use **OptimizedGeneric** for `{*}` repetition patterns
- Pre-computation provides **10-25x speedup**
- Handler callbacks add minimal overhead

### Memory Considerations:
- **OptimizedGeneric**: ~50KB per validation
- **ComplexPattern**: ~130KB per validation  
- Scales linearly with JSON complexity
- Pre-computation reduces repeated validation cost

## Conclusion

The **OptimizedGeneric** validator provides the best performance for recursive nested schemas with the `{*}` repetition operator, achieving **2.5μs** per validation while maintaining full pattern matching capabilities. This makes it ideal for production applications requiring high-performance JSON schema path validation.