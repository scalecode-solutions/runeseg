package runeseg

// Unicode properties used by the text segmentation parsers.
// Properties from UAX #29 (grapheme/word/sentence) and UAX #14 (line break).
//
// Note: Grapheme properties come first to minimize bits in state vectors.
const (
	prXX  = 0    // Unknown/unassigned (same as prAny)
	prAny = iota // Default/any property (must be 0)

	// Grapheme Cluster Break properties (UAX #29)
	prPrepend              // Characters that don't break before following char
	prCR                   // Carriage return
	prLF                   // Line feed
	prControl              // Control characters
	prExtend               // Extending characters (combining marks)
	prRegionalIndicator    // Flag emoji components (paired)
	prSpacingMark          // Spacing combining marks
	prL                    // Hangul leading consonant (Jamo L)
	prV                    // Hangul vowel (Jamo V)
	prT                    // Hangul trailing consonant (Jamo T)
	prLV                   // Hangul syllable LV
	prLVT                  // Hangul syllable LVT
	prZWJ                  // Zero Width Joiner
	prExtendedPictographic // Emoji and pictographic characters

	// Word Break properties (UAX #29)
	prNewline      // Newline characters
	prWSegSpace    // Whitespace for WB3d
	prDoubleQuote  // Double quotation mark
	prSingleQuote  // Single quotation mark (apostrophe)
	prMidNumLet    // Mid-word/number (e.g., period, colon)
	prNumeric      // Numeric digits
	prMidLetter    // Mid-letter (e.g., middle dot)
	prMidNum       // Mid-number (e.g., comma in numbers)
	prExtendNumLet // Underscore and similar
	prALetter      // Alphabetic letters
	prFormat       // Format characters
	prHebrewLetter // Hebrew letters
	prKatakana     // Japanese Katakana

	// Sentence Break properties (UAX #29)
	prSp        // Space
	prSTerm     // Sentence terminal (! ?)
	prClose     // Close punctuation
	prSContinue // Sentence continue
	prATerm     // Ambiguous terminal (.)
	prUpper     // Uppercase letters
	prLower     // Lowercase letters
	prSep       // Paragraph separator
	prOLetter   // Other letters

	// Line Break properties (UAX #14)
	prCM // Combining Mark
	prBA // Break After
	prBK // Mandatory Break
	prSP // Space
	prEX // Exclamation
	prQU // Quotation
	prAL // Ordinary Alphabetic
	prPR // Prefix Numeric
	prPO // Postfix Numeric
	prOP // Open Punctuation
	prCP // Close Parenthesis
	prIS // Infix Separator
	prHY // Hyphen
	prSY // Break Symbols
	prNU // Numeric
	prCL // Close Punctuation
	prNL // Next Line
	prGL // Non-breaking (Glue)
	prAI // Ambiguous (treated as AL)
	prBB // Break Before
	prHL // Hebrew Letter
	prSA // Complex Context (South Asian)
	prJL // Hangul L Jamo
	prJV // Hangul V Jamo
	prJT // Hangul T Jamo
	prNS // Nonstarter
	prZW // Zero Width Space
	prB2 // Break Opportunity Before and After
	prIN // Inseparable
	prWJ // Word Joiner
	prID // Ideographic
	prEB // Emoji Base
	prCJ // Conditional Japanese Starter
	prH2 // Hangul LV Syllable
	prH3 // Hangul LVT Syllable
	prSG // Surrogate (treat as AL)
	prCB // Contingent Break
	prRI // Regional Indicator
	prEM // Emoji Modifier

	// East Asian Width properties (UAX #11)
	prN                 // Neutral
	prNa                // Narrow
	prA                 // Ambiguous
	prW                 // Wide
	prH                 // Halfwidth
	prF                 // Fullwidth
	prEmojiPresentation // Has emoji presentation by default

	// Combined property for WB3.3: ALetter that is also Extended_Pictographic
	prALetterExtPict

	// Unicode 17.0 Line Break properties for Brahmic/Aksara scripts (LB28a)
	prAK // Aksara - letters that form conjuncts
	prAP // Aksara_Prebase - prebase combining characters
	prAS // Aksara_Start - consonants that start syllables
	prVF // Virama_Final - final/spacing viramas
	prVI // Virama - combining viramas
	prHH // Unambiguous_Hyphen - non-breaking hyphen (Unicode 17.0, LB20.1)

	// Indic_Conjunct_Break property values for grapheme rule GB9c
	prInCBNone      // Default - not part of conjunct
	prInCBLinker    // Virama - links consonants in conjuncts
	prInCBConsonant // Consonant - can form conjuncts
	prInCBExtend    // Extend - extends within conjuncts
)

// Unicode General Categories needed for segmentation decisions.
// Used primarily for distinguishing quotation mark types (Pi/Pf) in line breaking.
const (
	gcNone = iota // Unknown/default (must be 0)
	gcCc          // Control
	gcZs          // Space Separator
	gcPo          // Other Punctuation
	gcSc          // Currency Symbol
	gcPs          // Open Punctuation
	gcPe          // Close Punctuation
	gcSm          // Math Symbol
	gcPd          // Dash Punctuation
	gcNd          // Decimal Number
	gcLu          // Uppercase Letter
	gcSk          // Modifier Symbol
	gcPc          // Connector Punctuation
	gcLl          // Lowercase Letter
	gcSo          // Other Symbol
	gcLo          // Other Letter
	gcPi          // Initial Punctuation (opening quotes like «)
	gcCf          // Format
	gcNo          // Other Number
	gcPf          // Final Punctuation (closing quotes like »)
	gcLC          // Cased Letter
	gcLm          // Modifier Letter
	gcMn          // Nonspacing Mark
	gcMe          // Enclosing Mark
	gcMc          // Spacing Mark
	gcNl          // Letter Number
	gcZl          // Line Separator
	gcZp          // Paragraph Separator
	gcCn          // Unassigned
	gcCs          // Surrogate
	gcCo          // Private Use
)

// Variation Selectors for emoji presentation control.
const (
	vs15 = 0xfe0e // Variation Selector-15: force text presentation (width 1)
	vs16 = 0xfe0f // Variation Selector-16: force emoji presentation (width 2)
)

// propertySearch performs a binary search on a sorted property table.
// Each entry is [startCodePoint, endCodePoint, property, ...].
// Returns the matching entry, or zero-initialized entry if not found.
func propertySearch[E interface{ [3]int | [4]int }](dictionary []E, r rune) (result E) {
	// Run a binary search.
	from := 0
	to := len(dictionary)
	for to > from {
		middle := (from + to) / 2
		cpRange := dictionary[middle]
		if int(r) < cpRange[0] {
			to = middle
			continue
		}
		if int(r) > cpRange[1] {
			from = middle + 1
			continue
		}
		return cpRange
	}
	return
}

// property returns the Unicode property value (see constants above) of the
// given code point.
func property(dictionary [][3]int, r rune) int {
	return propertySearch(dictionary, r)[2]
}

// propertyLineBreak returns the Unicode property value and General Category
// (see constants above) of the given code point, as listed in the line break
// code points table, while fast tracking ASCII digits and letters.
func propertyLineBreak(r rune) (property, generalCategory int) {
	if r >= 'a' && r <= 'z' {
		return prAL, gcLl
	}
	if r >= 'A' && r <= 'Z' {
		return prAL, gcLu
	}
	if r >= '0' && r <= '9' {
		return prNU, gcNd
	}
	entry := propertySearch(lineBreakCodePoints, r)
	return entry[2], entry[3]
}

// propertyGraphemes returns the Unicode grapheme cluster property value of the
// given code point while fast tracking ASCII characters.
func propertyGraphemes(r rune) int {
	if r >= 0x20 && r <= 0x7e {
		return prAny
	}
	if r == 0x0a {
		return prLF
	}
	if r == 0x0d {
		return prCR
	}
	if r >= 0 && r <= 0x1f || r == 0x7f {
		return prControl
	}
	return property(graphemeCodePoints, r)
}

// propertyEastAsianWidth returns the Unicode East Asian Width property value of
// the given code point while fast tracking ASCII characters.
func propertyEastAsianWidth(r rune) int {
	if r >= 0x20 && r <= 0x7e {
		return prNa
	}
	if r >= 0 && r <= 0x1f || r == 0x7f {
		return prN
	}
	return property(eastAsianWidth, r)
}

// propertyInCB returns the Indic_Conjunct_Break property value for the given
// code point. This is used for the GB9c grapheme cluster boundary rule.
func propertyInCB(r rune) int {
	// Fast track ASCII - no InCB properties
	if r < 0x0300 {
		return prInCBNone
	}
	return property(incbCodePoints, r)
}
