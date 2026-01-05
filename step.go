package runeseg

import "unicode/utf8"

// Bit masks for extracting boundary information from [Step] return values.
//
// The boundaries return value is a bit-packed integer containing:
//   - Line break (bits 0-1): LineDontBreak, LineCanBreak, or LineMustBreak
//   - Word boundary (bit 2): 1 if word boundary, 0 otherwise
//   - Sentence boundary (bit 3): 1 if sentence boundary, 0 otherwise
//   - Character width (bits 4+): monospace display width
const (
	MaskLine     = 3 // bits 0-1: line break type
	MaskWord     = 4 // bit 2: word boundary flag
	MaskSentence = 8 // bit 3: sentence boundary flag
)

// ShiftWidth is the number of bits to right-shift boundaries to get character width.
const ShiftWidth = 4

// Internal bit positions for boundary flags in the boundaries return value.
const (
	shiftWord     = 2 // Word boundary flag position
	shiftSentence = 3 // Sentence boundary flag position
)

// Internal bit positions for packing multiple parser states into a single int.
// The state integer layout (64 bits total):
//   - Bits 0-11:  Grapheme state (12 bits, includes InCB tracking)
//   - Bits 12-16: Word state (5 bits)
//   - Bits 17-20: Sentence state (4 bits)
//   - Bits 21-36: Line state (16 bits, context-aware system)
//   - Bits 37+:   Cached property for next iteration
const (
	shiftWordState     = 12
	shiftSentenceState = 17
	shiftLineState     = 21
	shiftPropState     = 37
)

// Bit masks for extracting individual parser states (after shifting).
const (
	maskGraphemeState = 0xfff  // 12 bits: base state + InCB tracking
	maskWordState     = 0x1f   // 5 bits
	maskSentenceState = 0xf    // 4 bits
	maskLineState     = 0xffff // 16 bits: 8 for state + 8 for context flags
)

// Step returns the first grapheme cluster (user-perceived character) found in
// the given byte slice. It also returns information about the boundary between
// that grapheme cluster and the one following it as well as the monospace width
// of the grapheme cluster. There are three types of boundary information: word
// boundaries, sentence boundaries, and line breaks. This function is therefore
// a combination of [FirstGraphemeCluster], [FirstWord], [FirstSentence], and
// [FirstLineSegment].
//
// The "boundaries" return value can be evaluated as follows:
//
//   - boundaries&MaskWord != 0: The boundary is a word boundary.
//   - boundaries&MaskWord == 0: The boundary is not a word boundary.
//   - boundaries&MaskSentence != 0: The boundary is a sentence boundary.
//   - boundaries&MaskSentence == 0: The boundary is not a sentence boundary.
//   - boundaries&MaskLine == LineDontBreak: You must not break the line at the
//     boundary.
//   - boundaries&MaskLine == LineMustBreak: You must break the line at the
//     boundary.
//   - boundaries&MaskLine == LineCanBreak: You may or may not break the line at
//     the boundary.
//   - boundaries >> ShiftWidth: The width of the grapheme cluster for most
//     monospace fonts where a value of 1 represents one character cell.
//
// This function can be called continuously to extract all grapheme clusters
// from a byte slice, as illustrated in the examples below.
//
// If you don't know which state to pass, for example when calling the function
// for the first time, you must pass -1. For consecutive calls, pass the state
// and rest slice returned by the previous call.
//
// The "rest" slice is the sub-slice of the original byte slice "b" starting
// after the last byte of the identified grapheme cluster. If the length of the
// "rest" slice is 0, the entire byte slice "b" has been processed. The
// "cluster" byte slice is the sub-slice of the input slice containing the
// first identified grapheme cluster.
//
// Given an empty byte slice "b", the function returns nil values.
//
// While slightly less convenient than using the Graphemes class, this function
// has much better performance and makes no allocations. It lends itself well to
// large byte slices.
//
// Note that in accordance with [UAX #14 LB3], the final segment will end with
// a mandatory line break (boundaries&MaskLine == LineMustBreak). You can choose
// to ignore this by checking if the length of the "rest" slice is 0 and calling
// [HasTrailingLineBreak] or [HasTrailingLineBreakInString] on the last rune.
//
// [UAX #14 LB3]: https://www.unicode.org/reports/tr14/#Algorithm
func Step(b []byte, state int) (cluster, rest []byte, boundaries int, newState int) {
	// An empty byte slice returns nothing.
	if len(b) == 0 {
		return
	}

	// Extract the first rune.
	r, length := utf8.DecodeRune(b)
	if len(b) <= length { // If we're already past the end, there is nothing else to parse.
		var prop int
		if state < 0 {
			prop = propertyGraphemes(r)
		} else {
			prop = state >> shiftPropState
		}
		return b, nil, LineMustBreak | (1 << shiftWord) | (1 << shiftSentence) | (runeWidth(r, prop) << ShiftWidth), grAny | (wbAny << shiftWordState) | (sbAny << shiftSentenceState) | (lbAny << shiftLineState) | (prop << shiftPropState)
	}

	// If we don't know the state, determine it now.
	var graphemeState, wordState, sentenceState, lineState, firstProp int
	remainder := b[length:]
	if state < 0 {
		graphemeState, firstProp, _ = transitionGraphemeState(state, r)
		wordState, _ = transitionWordBreakState(state, r, remainder, "")
		sentenceState, _ = transitionSentenceBreakState(state, r, remainder, "")
		lineState, _ = transitionLineBreakStateContext(state, r, remainder, "")
	} else {
		graphemeState = state & maskGraphemeState
		wordState = (state >> shiftWordState) & maskWordState
		sentenceState = (state >> shiftSentenceState) & maskSentenceState
		lineState = (state >> shiftLineState) & maskLineState
		firstProp = state >> shiftPropState
	}

	// Transition until we find a grapheme cluster boundary.
	width := runeWidth(r, firstProp)
	for {
		var (
			graphemeBoundary, wordBoundary, sentenceBoundary bool
			lineBreak, prop                                  int
		)

		r, l := utf8.DecodeRune(remainder)
		remainder = b[length+l:]

		graphemeState, prop, graphemeBoundary = transitionGraphemeState(graphemeState, r)
		wordState, wordBoundary = transitionWordBreakState(wordState, r, remainder, "")
		sentenceState, sentenceBoundary = transitionSentenceBreakState(sentenceState, r, remainder, "")
		lineState, lineBreak = transitionLineBreakStateContext(lineState, r, remainder, "")

		if graphemeBoundary {
			boundary := lineBreak | (width << ShiftWidth)
			if wordBoundary {
				boundary |= 1 << shiftWord
			}
			if sentenceBoundary {
				boundary |= 1 << shiftSentence
			}
			return b[:length], b[length:], boundary, graphemeState | (wordState << shiftWordState) | (sentenceState << shiftSentenceState) | (lineState << shiftLineState) | (prop << shiftPropState)
		}

		if firstProp == prExtendedPictographic {
			switch r {
			case vs15:
				width = 1
			case vs16:
				width = 2
			}
		} else if firstProp != prRegionalIndicator && firstProp != prL {
			width += runeWidth(r, prop)
		}

		length += l
		if len(b) <= length {
			return b, nil, LineMustBreak | (1 << shiftWord) | (1 << shiftSentence) | (width << ShiftWidth), grAny | (wbAny << shiftWordState) | (sbAny << shiftSentenceState) | (lbAny << shiftLineState) | (prop << shiftPropState)
		}
	}
}

// StepString is like [Step] but its input and outputs are strings.
func StepString(str string, state int) (cluster, rest string, boundaries int, newState int) {
	// An empty byte slice returns nothing.
	if len(str) == 0 {
		return
	}

	// Extract the first rune.
	r, length := utf8.DecodeRuneInString(str)
	if len(str) <= length { // If we're already past the end, there is nothing else to parse.
		prop := propertyGraphemes(r)
		return str, "", LineMustBreak | (1 << shiftWord) | (1 << shiftSentence) | (runeWidth(r, prop) << ShiftWidth), grAny | (wbAny << shiftWordState) | (sbAny << shiftSentenceState) | (lbAny << shiftLineState)
	}

	// If we don't know the state, determine it now.
	var graphemeState, wordState, sentenceState, lineState, firstProp int
	remainder := str[length:]
	if state < 0 {
		graphemeState, firstProp, _ = transitionGraphemeState(state, r)
		wordState, _ = transitionWordBreakState(state, r, nil, remainder)
		sentenceState, _ = transitionSentenceBreakState(state, r, nil, remainder)
		lineState, _ = transitionLineBreakStateContext(state, r, nil, remainder)
	} else {
		graphemeState = state & maskGraphemeState
		wordState = (state >> shiftWordState) & maskWordState
		sentenceState = (state >> shiftSentenceState) & maskSentenceState
		lineState = (state >> shiftLineState) & maskLineState
		firstProp = state >> shiftPropState
	}

	// Transition until we find a grapheme cluster boundary.
	width := runeWidth(r, firstProp)
	for {
		var (
			graphemeBoundary, wordBoundary, sentenceBoundary bool
			lineBreak, prop                                  int
		)

		r, l := utf8.DecodeRuneInString(remainder)
		remainder = str[length+l:]

		graphemeState, prop, graphemeBoundary = transitionGraphemeState(graphemeState, r)
		wordState, wordBoundary = transitionWordBreakState(wordState, r, nil, remainder)
		sentenceState, sentenceBoundary = transitionSentenceBreakState(sentenceState, r, nil, remainder)
		lineState, lineBreak = transitionLineBreakStateContext(lineState, r, nil, remainder)

		if graphemeBoundary {
			boundary := lineBreak | (width << ShiftWidth)
			if wordBoundary {
				boundary |= 1 << shiftWord
			}
			if sentenceBoundary {
				boundary |= 1 << shiftSentence
			}
			return str[:length], str[length:], boundary, graphemeState | (wordState << shiftWordState) | (sentenceState << shiftSentenceState) | (lineState << shiftLineState) | (prop << shiftPropState)
		}

		if firstProp == prExtendedPictographic {
			switch r {
			case vs15:
				width = 1
			case vs16:
				width = 2
			}
		} else if firstProp != prRegionalIndicator && firstProp != prL {
			width += runeWidth(r, prop)
		}

		length += l
		if len(str) <= length {
			return str, "", LineMustBreak | (1 << shiftWord) | (1 << shiftSentence) | (width << ShiftWidth), grAny | (wbAny << shiftWordState) | (sbAny << shiftSentenceState) | (lbAny << shiftLineState) | (prop << shiftPropState)
		}
	}
}
