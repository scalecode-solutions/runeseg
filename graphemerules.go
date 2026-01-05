package runeseg

// The states of the grapheme cluster parser.
const (
	grAny = iota
	grCR
	grControlLF
	grL
	grLVV
	grLVTT
	grPrepend
	grExtendedPictographic
	grExtendedPictographicZWJ
	grRIOdd
	grRIEven

)

// GB9c InCB state tracking constants (stored in upper bits of state).
// These track the Indic conjunct break state for the GB9c rule.
const (
	grInCBNone      = 0x0000 // No InCB tracking / InCB=None
	grInCBConsonant = 0x0100 // Seen InCB=Consonant
	grInCBExtend    = 0x0200 // Seen InCB=Consonant + [Extend]* (no Linker yet)
	grInCBLinker    = 0x0300 // Seen InCB=Consonant + [Extend|Linker]*Linker[Extend|Linker]*
	grInCBMask      = 0x0F00 // Mask for extracting InCB state
)

// The grapheme cluster parser's breaking instructions.
const (
	grNoBoundary = iota
	grBoundary
)

// grTransitions implements the grapheme cluster parser's state transitions.
// Maps state and property to a new state, a breaking instruction, and rule
// number. The breaking instruction always refers to the boundary between the
// last and next code point. Returns negative values if no transition is found.
//
// This function is used as follows:
//
//  1. Find specific state + specific property. Stop if found.
//  2. Find specific state + any property.
//  3. Find any state + specific property.
//  4. If only (2) or (3) (but not both) was found, stop.
//  5. If both (2) and (3) were found, use state from (3) and breaking instruction
//     from the transition with the lower rule number, prefer (3) if rule numbers
//     are equal. Stop.
//  6. Assume grAny and grBoundary.
//
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
		if incbState == grInCBConsonant {
			// Still in extend phase after consonant (no linker yet).
			newState = newState | grInCBExtend
		} else if incbState == grInCBExtend {
			// Continue in extend phase.
			newState = newState | grInCBExtend
		} else if incbState == grInCBLinker {
			// After linker, extends are allowed.
			newState = newState | grInCBLinker
		}

	default:
		// InCB=None - don't carry over InCB state (but keep any ZWJ handling).
		// The InCB state resets to 0 by default since we don't add any bits.
	}

	return
}
