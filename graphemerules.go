package runeseg

// States for the grapheme cluster parser.
// These track the parser's position within potential grapheme clusters.
const (
	grAny                     = iota // Default/initial state
	grCR                             // After carriage return (for GB3)
	grControlLF                      // After control/LF character
	grL                              // After Hangul L (leading consonant)
	grLVV                            // After Hangul LV or V
	grLVTT                           // After Hangul LVT or T
	grPrepend                        // After prepend character (GB9b)
	grExtendedPictographic           // After emoji/pictographic
	grExtendedPictographicZWJ        // After emoji + ZWJ (for GB11)
	grRIOdd                          // After odd number of Regional Indicators
	grRIEven                         // After even number of Regional Indicators
)

// GB9c Indic Conjunct Break (InCB) state tracking.
// Stored in upper bits of state to track Indic scripts like Devanagari.
//
// GB9c prevents breaks within conjunct clusters:
//   InCB=Consonant [InCB=Extend|Linker]* InCB=Linker [InCB=Extend|Linker]* × InCB=Consonant
//
// Example: क्षि (kṣi) should be kept together as one grapheme cluster.
//
// Values: 0x0000 = None, 0x0100 = Consonant, 0x0200 = Extend, 0x0300 = Linker
const (
	grInCBConsonant = 0x0100 // After InCB=Consonant (e.g., क)
	grInCBExtend    = 0x0200 // After Consonant + Extend* (no Linker yet)
	grInCBLinker    = 0x0300 // After Consonant + [Extend|Linker]*Linker (e.g., क्)
	grInCBMask      = 0x0F00 // Bit mask to extract InCB state
)

// The grapheme cluster parser's breaking instructions.
const (
	grNoBoundary = iota
	grBoundary
)

// grTransitions implements grapheme cluster boundary rules from UAX #29.
//
// Given current state and next character's property, returns:
//   - newState: the next parser state
//   - newProp: breaking instruction (grBoundary or grNoBoundary)
//   - rule: the rule number that matched (for conflict resolution)
//
// Transition resolution order:
//  1. Exact match (specific state + specific property)
//  2. State wildcard (specific state + any property)
//  3. Property wildcard (any state + specific property)
//  4. If both wildcards match, use lower rule number to decide break
//  5. Default: grAny state with boundary
//
// Unicode Standard Annex #29 (https://unicode.org/reports/tr29/)
// Unicode version 17.0.0.
func grTransitions(state, prop int) (newState int, newProp int, boundary int) {
	// It turns out that using a big switch statement is much faster than using
	// a map.

	switch uint64(state) | uint64(prop)<<32 {
	// GB5
	case grAny | prCR<<32:
		return grCR, grBoundary, 50
	case grAny | prLF<<32:
		return grControlLF, grBoundary, 50
	case grAny | prControl<<32:
		return grControlLF, grBoundary, 50

	// GB4
	case grCR | prAny<<32:
		return grAny, grBoundary, 40
	case grControlLF | prAny<<32:
		return grAny, grBoundary, 40

	// GB3
	case grCR | prLF<<32:
		return grControlLF, grNoBoundary, 30

	// GB6
	case grAny | prL<<32:
		return grL, grBoundary, 9990
	case grL | prL<<32:
		return grL, grNoBoundary, 60
	case grL | prV<<32:
		return grLVV, grNoBoundary, 60
	case grL | prLV<<32:
		return grLVV, grNoBoundary, 60
	case grL | prLVT<<32:
		return grLVTT, grNoBoundary, 60

	// GB7
	case grAny | prLV<<32:
		return grLVV, grBoundary, 9990
	case grAny | prV<<32:
		return grLVV, grBoundary, 9990
	case grLVV | prV<<32:
		return grLVV, grNoBoundary, 70
	case grLVV | prT<<32:
		return grLVTT, grNoBoundary, 70

	// GB8
	case grAny | prLVT<<32:
		return grLVTT, grBoundary, 9990
	case grAny | prT<<32:
		return grLVTT, grBoundary, 9990
	case grLVTT | prT<<32:
		return grLVTT, grNoBoundary, 80

	// GB9
	case grAny | prExtend<<32:
		return grAny, grNoBoundary, 90
	case grAny | prZWJ<<32:
		return grAny, grNoBoundary, 90

	// GB9a
	case grAny | prSpacingMark<<32:
		return grAny, grNoBoundary, 91

	// GB9b
	case grAny | prPrepend<<32:
		return grPrepend, grBoundary, 9990
	case grPrepend | prAny<<32:
		return grAny, grNoBoundary, 92

	// GB11
	case grAny | prExtendedPictographic<<32:
		return grExtendedPictographic, grBoundary, 9990
	case grExtendedPictographic | prExtend<<32:
		return grExtendedPictographic, grNoBoundary, 110
	case grExtendedPictographic | prZWJ<<32:
		return grExtendedPictographicZWJ, grNoBoundary, 110
	case grExtendedPictographicZWJ | prExtendedPictographic<<32:
		return grExtendedPictographic, grNoBoundary, 110

	// GB12 / GB13
	case grAny | prRegionalIndicator<<32:
		return grRIOdd, grBoundary, 9990
	case grRIOdd | prRegionalIndicator<<32:
		return grRIEven, grNoBoundary, 120
	case grRIEven | prRegionalIndicator<<32:
		return grRIOdd, grBoundary, 120
	default:
		return -1, -1, -1
	}
}

// transitionGraphemeState determines the new state of the grapheme cluster
// parser given the current state and the next code point. It also returns the
// code point's grapheme property (the value mapped by the [graphemeCodePoints]
// table) and whether a cluster boundary was detected.
func transitionGraphemeState(state int, r rune) (newState, prop int, boundary bool) {
	// Determine the property of the next character.
	prop = propertyGraphemes(r)

	// Extract the InCB state tracking if present.
	incbState := state & grInCBMask
	state = state & 0xFF

	// Get the InCB property of the current rune for GB9c.
	incbProp := propertyInCB(r)

	// Find the applicable transition.
	nextState, nextProp, _ := grTransitions(state, prop)
	if nextState >= 0 {
		// We have a specific transition. We'll use it.
		newState, boundary = nextState, nextProp == grBoundary
	} else {
		// No specific transition found. Try the less specific ones.
		anyPropState, anyPropProp, anyPropRule := grTransitions(state, prAny)
		anyStateState, anyStateProp, anyStateRule := grTransitions(grAny, prop)
		if anyPropState >= 0 && anyStateState >= 0 {
			// Both apply. We'll use a mix (see comments for grTransitions).
			newState = anyStateState
			boundary = anyStateProp == grBoundary
			if anyPropRule < anyStateRule {
				boundary = anyPropProp == grBoundary
			}
		} else if anyPropState >= 0 {
			// We only have a specific state.
			newState, boundary = anyPropState, anyPropProp == grBoundary
		} else if anyStateState >= 0 {
			// We only have a specific property.
			newState, boundary = anyStateState, anyStateProp == grBoundary
		} else {
			// No known transition. GB999: Any ÷ Any.
			newState, boundary = grAny, true
		}
	}

	// GB9c: Handle Indic conjunct clusters.
	// \p{InCB=Consonant} {[\p{InCB=Extend}\p{InCB=Linker}]* \p{InCB=Linker} [\p{InCB=Extend}\p{InCB=Linker}]*} × \p{InCB=Consonant}
	//
	// We track the InCB state separately and override the boundary decision
	// when we have a valid conjunct sequence.
	switch {
	case incbProp == prInCBConsonant:
		// We're seeing a consonant.
		if incbState == grInCBLinker {
			// GB9c applies: Consonant + [Extend|Linker]*Linker[Extend|Linker]* × Consonant
			// Don't break before this consonant.
			boundary = false
		}
		// Start tracking a new potential conjunct sequence.
		newState = newState | grInCBConsonant

	case incbProp == prInCBLinker:
		// We're seeing a linker.
		if incbState == grInCBConsonant || incbState == grInCBExtend || incbState == grInCBLinker {
			// We've seen a consonant and now we have a linker.
			// Mark that we have a valid conjunct sequence.
			newState = newState | grInCBLinker
		}

	case incbProp == prInCBExtend:
		// We're seeing an extend character.
		switch incbState {
		case grInCBConsonant:
			// Still in extend phase after consonant (no linker yet).
			newState = newState | grInCBExtend
		case grInCBExtend:
			// Continue in extend phase.
			newState = newState | grInCBExtend
		case grInCBLinker:
			// After linker, extends are allowed.
			newState = newState | grInCBLinker
		}

	default:
		// InCB=None - don't carry over InCB state (but keep any ZWJ handling).
		// The InCB state resets to 0 by default since we don't add any bits.
	}

	return
}
