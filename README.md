# runeseg

[![Go Reference](https://pkg.go.dev/badge/github.com/scalecode-solutions/runeseg.svg)](https://pkg.go.dev/github.com/scalecode-solutions/runeseg)
[![Go Report Card](https://goreportcard.com/badge/github.com/scalecode-solutions/runeseg)](https://goreportcard.com/report/github.com/scalecode-solutions/runeseg)

A Go package for Unicode text segmentation, line breaking, and monospace width calculation. Implements [UAX #29](https://unicode.org/reports/tr29/) (Text Segmentation) and [UAX #14](https://unicode.org/reports/tr14/) (Line Breaking) for **Unicode 17.0**.

## Installation

```bash
go get github.com/scalecode-solutions/runeseg
```

## Why runeseg?

Go strings are byte slices, and `[]rune(str)` gives you code points—but neither represents what users actually see as "characters." A single emoji like 👨‍👩‍👧‍👦 (family) is **7 code points** but renders as **1 character**. This package handles that complexity for you.

| What you see | Bytes | Code Points | Grapheme Clusters |
|--------------|-------|-------------|-------------------|
| Héllo | 6 | 5 (`H é l l o`) | 5 |
| 👨‍👩‍👧‍👦 | 25 | 7 | 1 |
| 가 | 3 | 1 | 1 |
| क्षि | 12 | 4 | 1 |

## Quick Start

### Count User-Perceived Characters

```go
// Counts grapheme clusters, not bytes or runes
n := runeseg.GraphemeClusterCount("👨‍👩‍👧‍👦")
fmt.Println(n) // 1
```

### Calculate Display Width

```go
// For terminal/monospace font rendering
w := runeseg.StringWidth("Hello世界")
fmt.Println(w) // 9 (5 + 2 + 2)
```

### Iterate Over Graphemes

```go
gr := runeseg.NewGraphemes("नमस्ते")
for gr.Next() {
    fmt.Printf("%s ", gr.Str())
}
// न म स् ते
```

### Word Segmentation

```go
str := "Hello, 世界!"
state := -1
for len(str) > 0 {
    var word string
    word, str, state = runeseg.FirstWordInString(str, state)
    fmt.Printf("[%s] ", word)
}
// [Hello] [,] [ ] [世界] [!]
```

### Line Breaking

```go
// Find valid line break points for word wrapping
str := "The quick brown fox"
state := -1
for len(str) > 0 {
    var segment string
    var breaking int
    segment, str, breaking, state = runeseg.StepString(str, state)
    
    canBreak := breaking&runeseg.MaskLine == runeseg.LineCanBreak
    fmt.Printf("%q (break: %v)\n", segment, canBreak)
}
```

## Features

### Text Segmentation (UAX #29)
- **Grapheme Clusters** — User-perceived characters
- **Word Boundaries** — For search, selection, cursor movement
- **Sentence Boundaries** — For text processing and NLP

### Line Breaking (UAX #14)
- **Break Opportunities** — Where text can wrap
- **Mandatory Breaks** — Where text must break (newlines)
- **No-Break Rules** — Keep units together (numbers, URLs)

### Width Calculation
- **Monospace Width** — Terminal and fixed-width font rendering
- **East Asian Width** — Proper CJK character handling
- **Emoji Width** — Modern emoji sequences

## Unicode 17.0 Support

Full support for the latest Unicode standard including:

- **Indic Conjunct Break** — Devanagari, Bengali, Tamil, and other Indic scripts
- **Aksara Sequences** — Balinese, Javanese, and Southeast Asian scripts  
- **Extended Pictographic** — All emoji sequences and ZWJ combinations
- **Unambiguous Hyphen** — Improved line breaking around hyphens

## API Overview

### High-Level (Allocates)

```go
// Grapheme iterator
gr := runeseg.NewGraphemes(str)
for gr.Next() {
    gr.Str()      // Current grapheme as string
    gr.Runes()    // Current grapheme as []rune
    gr.Width()    // Display width
}

// Counting
runeseg.GraphemeClusterCount(str)
runeseg.StringWidth(str)

// Reversing (preserves graphemes)
runeseg.ReverseString(str)
```

### Low-Level (Zero Allocation)

```go
// All-in-one step function
cluster, rest, boundaries, state := runeseg.Step(b, state)
cluster, rest, boundaries, state := runeseg.StepString(str, state)

// Boundary info from Step:
boundaries & runeseg.MaskLine     // Line break type
boundaries & runeseg.MaskWord     // Word boundary?
boundaries & runeseg.MaskSentence // Sentence boundary?
boundaries >> runeseg.ShiftWidth  // Display width

// Individual segmentation
runeseg.FirstGraphemeCluster(b, state)
runeseg.FirstWord(b, state)
runeseg.FirstSentence(b, state)
runeseg.FirstLineSegment(b, state)
```

## Performance

This package is designed for high performance:

- **Zero allocations** in low-level APIs
- **Single-pass processing** for all boundary types
- **Efficient state machine** implementation
- **No external dependencies**

## Attribution

This package is based on [github.com/rivo/uniseg](https://github.com/rivo/uniseg) by Oliver Kuederle, updated to Unicode 17.0 with architectural improvements. The original work is licensed under MIT.

## License

MIT License - see [LICENSE.txt](LICENSE.txt)
