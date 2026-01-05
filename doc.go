/*
Package runeseg implements Unicode Text Segmentation, Unicode Line Breaking, and
string width calculation for monospace fonts.

This package conforms to:
  - Unicode Standard Annex #29 (https://unicode.org/reports/tr29/) for text segmentation
  - Unicode Standard Annex #14 (https://unicode.org/reports/tr14/) for line breaking
  - Unicode version 17.0

# Overview

Using this package, you can:
  - Split strings into grapheme clusters (user-perceived "characters")
  - Find word and sentence boundaries
  - Determine line break opportunities for word wrapping
  - Calculate display width for monospace fonts

This is essential for internationalized text handling, especially with emojis,
combining characters, and scripts like Arabic, Hebrew, Indic, and East Asian languages.

# Getting Started

For simple use cases:
  - [GraphemeClusterCount] - Count user-perceived characters
  - [StringWidth] - Get display width for monospace fonts

For iteration:
  - [Step] / [StepString] - Process text with all boundary info (recommended)
  - [Graphemes] - Convenient iterator class

For specific boundaries only:
  - [FirstGraphemeCluster] / [FirstGraphemeClusterInString]
  - [FirstWord] / [FirstWordInString]
  - [FirstSentence] / [FirstSentenceInString]
  - [FirstLineSegment] / [FirstLineSegmentInString]

# Grapheme Clusters

A grapheme cluster is what users perceive as a single "character." For example,
the family emoji 👨‍👩‍👧‍👦 appears as one character but contains 7 Unicode code points
(25 bytes in UTF-8). Standard Go functions report misleading values:

	len("👨‍👩‍👧‍👦")                    // 25 (bytes)
	len([]rune("👨‍👩‍👧‍👦"))             // 7 (code points)
	runeseg.GraphemeClusterCount("👨‍👩‍👧‍👦") // 1 (what users see)

The [Graphemes] class and related functions correctly handle these cases.

# Word Boundaries

Word boundaries are used for:
  - Double-click text selection
  - Cursor movement (Ctrl+Arrow)
  - "Whole word" search
  - Database proximity queries

Use [FirstWord], [FirstWordInString], or check [Graphemes.IsWordBoundary].

# Sentence Boundaries

Sentence boundaries are used for:
  - Triple-click text selection
  - Text-to-speech segmentation
  - NLP sentence tokenization

Use [FirstSentence], [FirstSentenceInString], or check [Graphemes.IsSentenceBoundary].

# Line Breaking

Line breaking (word wrapping) determines where text can be broken across lines.
The package distinguishes:
  - Must break (after newlines)
  - May break (between words)
  - Must not break (within words, numbers, URLs)

Use [FirstLineSegment], [FirstLineSegmentInString], or check [Graphemes.LineBreak].
The [Step] function is preferred as it respects grapheme cluster boundaries.

# Monospace Width

For terminal UIs and fixed-width font rendering, characters have varying widths:
  - Most characters: width 1
  - East Asian wide/fullwidth (CJK): width 2
  - Emojis: width 2 (unless text presentation)
  - Combining marks, ZWJ, control chars: width 0
  - Special dashes (U+2E3A, U+2E3B): width 3-4

Use [StringWidth] or [Graphemes.Width]. Configure ambiguous width handling
with [EastAsianAmbiguousWidth].

Note: Actual rendering depends on your terminal/font. These calculations
follow common conventions but may not match all environments.
*/
package runeseg
