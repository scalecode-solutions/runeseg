# runeseg - Unicode Text Segmentation for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/scalecode-solutions/runeseg.svg)](https://pkg.go.dev/github.com/scalecode-solutions/runeseg)
[![Go Report](https://img.shields.io/badge/go%20report-A%2B-brightgreen.svg)](https://goreportcard.com/report/github.com/scalecode-solutions/runeseg)

This Go package implements Unicode Text Segmentation according to [Unicode Standard Annex #29](https://unicode.org/reports/tr29/), Unicode Line Breaking according to [Unicode Standard Annex #14](https://unicode.org/reports/tr14/) (Unicode version 17.0.0), and monospace font string width calculation similar to [wcwidth](https://man7.org/linux/man-pages/man3/wcwidth.3.html).

## Background

### Grapheme Clusters

In Go, [strings are read-only slices of bytes](https://go.dev/blog/strings). They can be turned into Unicode code points using the `for` loop or by casting: `[]rune(str)`. However, multiple code points may be combined into one user-perceived character or what the Unicode specification calls "grapheme cluster". Here are some examples:

|String|Bytes (UTF-8)|Code points (runes)|Grapheme clusters|
|-|-|-|-|
|Käse|6 bytes: `4b 61 cc 88 73 65`|5 code points: `4b 61 308 73 65`|4 clusters: `[4b],[61 308],[73],[65]`|
|🏳️‍🌈|14 bytes: `f0 9f 8f b3 ef b8 8f e2 80 8d f0 9f 8c 88`|4 code points: `1f3f3 fe0f 200d 1f308`|1 cluster: `[1f3f3 fe0f 200d 1f308]`|
|🇩🇪|8 bytes: `f0 9f 87 a9 f0 9f 87 aa`|2 code points: `1f1e9 1f1ea`|1 cluster: `[1f1e9 1f1ea]`|

This package provides tools to iterate over these grapheme clusters. This may be used to determine the number of user-perceived characters, to split strings in their intended places, or to extract individual characters which form a unit.

### Word Boundaries

Word boundaries are used in a number of different contexts. The most familiar ones are selection (double-click mouse selection), cursor movement ("move to next word" control-arrow keys), and the dialog option "Whole Word Search" for search and replace. They are also used in database queries, to determine whether elements are within a certain number of words of one another. Searching may also use word boundaries in determining matching items. This package provides tools to determine word boundaries within strings.

### Sentence Boundaries

Sentence boundaries are often used for triple-click or some other method of selecting or iterating through blocks of text that are larger than single words. They are also used to determine whether words occur within the same sentence in database queries. This package provides tools to determine sentence boundaries within strings.

### Line Breaking

Line breaking, also known as word wrapping, is the process of breaking a section of text into lines such that it will fit in the available width of a page, window or other display area. This package provides tools to determine where a string may or may not be broken and where it must be broken (for example after newline characters).

### Monospace Width

Most terminals or text displays / text editors using a monospace font (for example source code editors) use a fixed width for each character. Some characters such as emojis or characters found in Asian and other languages may take up more than one character cell. This package provides tools to determine the number of cells a string will take up when displayed in a monospace font. See [here](https://pkg.go.dev/github.com/scalecode-solutions/runeseg#hdr-Monospace_Width) for more information.

## Installation

```bash
go get github.com/scalecode-solutions/runeseg
```

## Examples

### Counting Characters in a String

```go
n := runeseg.GraphemeClusterCount("🇩🇪🏳️‍🌈")
fmt.Println(n)
// 2
```

### Calculating the Monospace String Width

```go
width := runeseg.StringWidth("🇩🇪🏳️‍🌈!")
fmt.Println(width)
// 5
```

### Using the [`Graphemes`](https://pkg.go.dev/github.com/scalecode-solutions/runeseg#Graphemes) Class

This is the most convenient method of iterating over grapheme clusters:

```go
gr := runeseg.NewGraphemes("👍🏼!")
for gr.Next() {
	fmt.Printf("%x ", gr.Runes())
}
// [1f44d 1f3fc] [21]
```

### Using the [`Step`](https://pkg.go.dev/github.com/scalecode-solutions/runeseg#Step) or [`StepString`](https://pkg.go.dev/github.com/scalecode-solutions/runeseg#StepString) Function

This avoids allocating a new `Graphemes` object but it requires the handling of states and boundaries:

```go
str := "🇩🇪🏳️‍🌈"
state := -1
var c string
for len(str) > 0 {
	c, str, _, state = runeseg.StepString(str, state)
	fmt.Printf("%x ", []rune(c))
}
// [1f1e9 1f1ea] [1f3f3 fe0f 200d 1f308]
```

### Advanced Examples

The [`Graphemes`](https://pkg.go.dev/github.com/scalecode-solutions/runeseg#Graphemes) class offers the most convenient way to access all functionality of this package. But in some cases, it may be better to use the specialized functions directly. For example, if you're only interested in word segmentation, use [`FirstWord`](https://pkg.go.dev/github.com/scalecode-solutions/runeseg#FirstWord) or [`FirstWordInString`](https://pkg.go.dev/github.com/scalecode-solutions/runeseg#FirstWordInString):

```go
str := "Hello, world!"
state := -1
var c string
for len(str) > 0 {
	c, str, state = runeseg.FirstWordInString(str, state)
	fmt.Printf("(%s)\n", c)
}
// (Hello)
// (,)
// ( )
// (world)
// (!)
```

Similarly, use

- [`FirstGraphemeCluster`](https://pkg.go.dev/github.com/scalecode-solutions/runeseg#FirstGraphemeCluster) or [`FirstGraphemeClusterInString`](https://pkg.go.dev/github.com/scalecode-solutions/runeseg#FirstGraphemeClusterInString) for grapheme cluster determination only,
- [`FirstSentence`](https://pkg.go.dev/github.com/scalecode-solutions/runeseg#FirstSentence) or [`FirstSentenceInString`](https://pkg.go.dev/github.com/scalecode-solutions/runeseg#FirstSentenceInString) for sentence segmentation only, and
- [`FirstLineSegment`](https://pkg.go.dev/github.com/scalecode-solutions/runeseg#FirstLineSegment) or [`FirstLineSegmentInString`](https://pkg.go.dev/github.com/scalecode-solutions/runeseg#FirstLineSegmentInString) for line breaking / word wrapping (although using [`Step`](https://pkg.go.dev/github.com/scalecode-solutions/runeseg#Step) or [`StepString`](https://pkg.go.dev/github.com/scalecode-solutions/runeseg#StepString) is preferred as it will observe grapheme cluster boundaries).

If you're only interested in the width of characters, use [`FirstGraphemeCluster`](https://pkg.go.dev/github.com/scalecode-solutions/runeseg#FirstGraphemeCluster) or [`FirstGraphemeClusterInString`](https://pkg.go.dev/github.com/scalecode-solutions/runeseg#FirstGraphemeClusterInString). It is much faster than using [`Step`](https://pkg.go.dev/github.com/scalecode-solutions/runeseg#Step), [`StepString`](https://pkg.go.dev/github.com/scalecode-solutions/runeseg#StepString), or the [`Graphemes`](https://pkg.go.dev/github.com/scalecode-solutions/runeseg#Graphemes) class because it does not include the logic for word / sentence / line boundaries.

Finally, if you need to reverse a string while preserving grapheme clusters, use [`ReverseString`](https://pkg.go.dev/github.com/scalecode-solutions/runeseg#ReverseString):

```go
fmt.Println(runeseg.ReverseString("🇩🇪🏳️‍🌈"))
// 🏳️‍🌈🇩🇪
```

## Unicode 17.0 Features

This package fully supports Unicode 17.0, including:

- **Indic Conjunct Break (InCB)** - Proper handling of Indic scripts like Devanagari
- **Aksara properties** - For Southeast Asian scripts (Balinese, Javanese, etc.)
- **Unambiguous Hyphen (HH)** - Better line breaking around hyphens
- **Extended Pictographic** - Comprehensive emoji support

## Documentation

Refer to https://pkg.go.dev/github.com/scalecode-solutions/runeseg for the package's documentation.

## Dependencies

This package does not depend on any packages outside the standard library.

## Sponsor this Project

[Become a Sponsor on GitHub](https://github.com/sponsors/rivo?metadata_source=runeseg_readme) to support this project!

## Your Feedback

Add your issue here on GitHub, preferably before submitting any PR's. Feel free to get in touch if you have any questions.
