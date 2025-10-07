package tree

import (
	"hash/fnv"
)

// BloomFilter is a probabilistic data structure for fast membership testing
// False positives are possible, but false negatives are not
type BloomFilter struct {
	bits      []uint64
	size      int
	numHashes int
}

// NewBloomFilter creates a new bloom filter with the given size (in bits) and number of hash functions
// For optimal performance:
// - size should be ~10 bits per expected element
// - numHashes should be (size/n) * ln(2) where n is expected number of elements
func NewBloomFilter(size int, numHashes int) *BloomFilter {
	// Round size up to nearest multiple of 64 for efficiency
	numWords := (size + 63) / 64
	return &BloomFilter{
		bits:      make([]uint64, numWords),
		size:      numWords * 64,
		numHashes: numHashes,
	}
}

// NewOptimalBloomFilter creates a bloom filter with optimal parameters for the given number of expected elements
// targetFalsePositiveRate is typically 0.01 (1%) to 0.001 (0.1%)
func NewOptimalBloomFilter(expectedElements int, targetFalsePositiveRate float64) *BloomFilter {
	// Optimal size: m = -n * ln(p) / (ln(2)^2)
	// where n = number of elements, p = target false positive rate
	m := float64(expectedElements) * -1 * ln(targetFalsePositiveRate) / (ln2 * ln2)
	size := int(m + 0.5) // Round to nearest int

	// Optimal number of hash functions: k = (m/n) * ln(2)
	k := float64(size) / float64(expectedElements) * ln2
	numHashes := int(k + 0.5)
	if numHashes < 1 {
		numHashes = 1
	}

	return NewBloomFilter(size, numHashes)
}

// Constants for optimal bloom filter calculations
const (
	ln2 = 0.69314718056 // ln(2)
)

// ln computes natural logarithm
func ln(x float64) float64 {
	if x <= 0 {
		return -1000 // Return large negative for invalid input
	}
	// Simple approximation using Taylor series
	// For production, use math.Log
	// This is here to avoid importing math for just one function
	n := 0.0
	y := (x - 1) / (x + 1)
	y2 := y * y
	term := y
	sum := y
	for i := 1; i < 100; i++ {
		term *= y2
		sum += term / float64(2*i+1)
		if term < 0.0000001 {
			break
		}
	}
	return 2 * sum + n
}

// Add adds an element to the bloom filter
func (bf *BloomFilter) Add(data string) {
	hash1, hash2 := bf.hash(data)
	for i := 0; i < bf.numHashes; i++ {
		// Double hashing: hash_i = hash1 + i * hash2
		h := (hash1 + uint64(i)*hash2) % uint64(bf.size)
		wordIndex := h / 64
		bitIndex := h % 64
		bf.bits[wordIndex] |= (1 << bitIndex)
	}
}

// MightContain checks if an element might be in the set
// Returns true if the element MIGHT be in the set (possible false positive)
// Returns false if the element is DEFINITELY NOT in the set (no false negatives)
func (bf *BloomFilter) MightContain(data string) bool {
	hash1, hash2 := bf.hash(data)
	for i := 0; i < bf.numHashes; i++ {
		h := (hash1 + uint64(i)*hash2) % uint64(bf.size)
		wordIndex := h / 64
		bitIndex := h % 64
		if (bf.bits[wordIndex] & (1 << bitIndex)) == 0 {
			return false // Definitely not in set
		}
	}
	return true // Might be in set
}

// hash computes two independent hash values using FNV-1a
// These are used for double hashing to generate multiple hash functions
func (bf *BloomFilter) hash(data string) (uint64, uint64) {
	// First hash: standard FNV-1a
	h1 := fnv.New64a()
	h1.Write([]byte(data))
	hash1 := h1.Sum64()

	// Second hash: FNV-1a with a different initial value (salt)
	h2 := fnv.New64a()
	h2.Write([]byte(data))
	h2.Write([]byte{0x13, 0x37}) // Salt to make hash2 independent
	hash2 := h2.Sum64()

	// Ensure hash2 is odd (required for good double hashing)
	if hash2%2 == 0 {
		hash2++
	}

	return hash1, hash2
}

// Reset clears all bits in the bloom filter
func (bf *BloomFilter) Reset() {
	for i := range bf.bits {
		bf.bits[i] = 0
	}
}

// EstimateFalsePositiveRate estimates the current false positive rate based on the number of elements added
func (bf *BloomFilter) EstimateFalsePositiveRate(numElements int) float64 {
	// False positive rate: (1 - e^(-k*n/m))^k
	// where k = numHashes, n = numElements, m = size
	if numElements == 0 {
		return 0
	}

	// Approximation: p ≈ (1 - e^(-k*n/m))^k
	exponent := -float64(bf.numHashes*numElements) / float64(bf.size)
	// Use approximation: e^x ≈ 1 + x for small x
	exp := 1.0
	term := 1.0
	for i := 1; i <= 20; i++ {
		term *= exponent / float64(i)
		exp += term
		if term < 0.0000001 && term > -0.0000001 {
			break
		}
	}

	base := 1.0 - exp
	result := 1.0
	for i := 0; i < bf.numHashes; i++ {
		result *= base
	}

	return result
}
