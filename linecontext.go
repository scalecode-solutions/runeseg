package runeseg

// linecontext.go - Clean context-based line breaking for Unicode 17.0
//
// This replaces the complex state machine in linerules.go with a cleaner
// architecture that separates:
// 1. Base states (the core line break class of the previous character)
// 2. Context flags (additional information needed for context-sensitive rules)
//
// Unicode TR14 has several rules that depend on context beyond just the
// previous character:
// - LB15.1: sot (QU_Pi SP*)+ × OP (start of text context)
// - LB15.2: SP × QU_Pf eot (end of text context)
// - LB20a: ^(HY|HH) × (AL|HL) (word-initial hyphen)
// - LB28a: Aksara conjunct sequences
//
// By tracking context explicitly with flags, we keep the base state space
// small and make the rules readable and maintainable.

import "unicode/utf8"

// Context flags for line breaking.
// These track additional context needed for context-sensitive rules.
const (
	// Start of text - set at beginning, cleared after first break
	lbCtxSot = 1 << 0

	// After initial quotation mark (QU with gc=Pi)
	lbCtxAfterQUPi = 1 << 1

	// After space following QU_Pi (for LB15.1: sot (QU_Pi SP*)+ × OP)
	lbCtxQUPiSP = 1 << 2

	// After ZWJ (for emoji sequences)
	lbCtxAfterZWJ = 1 << 4

	// After CP with East Asian width F/W/H (for LB30)
	lbCtxCPeaFWH = 1 << 5

	// Saw Virama in Aksara sequence
	lbCtxAksaraVirama = 1 << 7
)

// Base line break states - the core class of the previous character.
// These are much simpler than the old states because context is tracked separately.
const (
	lbcAny = iota // Any/initial state
	lbcCR
	lbcLF
	lbcBK
	lbcNL
	lbcSP
	lbcZW
	lbcWJ
	lbcGL
	lbcBA
	lbcHY
	lbcCL
	lbcCP
	lbcEX
	lbcIS
	lbcSY
	lbcOP
	lbcQU
	lbcNS
	lbcAL
	lbcHL
	lbcNU
	lbcPR
	lbcPO
	lbcID
	lbcEB
	lbcEM
	lbcIN
	lbcCB
	lbcB2
	lbcRI
	lbcZWJ
	lbcCM   // Combining mark (when not absorbed)
	lbcAK   // Aksara (Unicode 17)
	lbcAP   // Aksara_Prebase
	lbcAS   // Aksara_Start
	lbcVI   // Virama
	lbcVF   // Virama_Final
	lbcAKVI // (AK|AS|DottedCircle) followed by VI (for LB28.13)
	lbcHH   // Unambiguous_Hyphen (Unicode 17)

	// Korean Jamo and syllables (for LB26-27)
	lbcJL // Jamo Leading
	lbcJV // Jamo Vowel
	lbcJT // Jamo Trailing
	lbcH2 // Hangul LV syllable
	lbcH3 // Hangul LVT syllable

	// Special states for multi-character sequences
	lbcRIOdd        // Odd number of RI
	lbcRIEven       // Even number of RI
	lbcB2SP         // B2 followed by SP
	lbcCLCP         // Close followed by SP (for LB13)
	lbcQUSP         // QU followed by SP
	lbcOPSP         // OP followed by SP (for LB14)
	lbcQUPi         // QU with Pi (initial quotation)
	lbcQUPiSP       // QU_Pi followed by SP
	lbcZWSP         // ZW followed by SP (for LB8)
	lbcHLHY         // HL followed by HY (for LB21a)
	lbcSotHY        // HY at start of text (for LB20a)
	lbcSotHH        // HH at start of text (for LB20a)
	lbcDottedCircle // Dotted Circle (U+25CC) - acts like AL but also like AK for Aksara
	lbcBB           // Break Before class

	// Extended pictographic sequences
	lbcExtPic    // Extended Pictographic
	lbcExtPicCn  // Extended Pictographic, unassigned (gcCn)
	lbcExtPicZWJ // ExtPic followed by ZWJ
)

// LineContext holds the complete line breaking state.
type LineContext struct {
	State int // Base state (lbc* constants)
	Flags int // Context flags (lbCtx* constants)
}

// packLineContext packs a LineContext into an int for storage in the Step state.
// Layout: [flags: 8 bits][state: 8 bits] = 16 bits total
func packLineContext(ctx LineContext) int {
	return (ctx.Flags << 8) | (ctx.State & 0xFF)
}

// unpackLineContext unpacks an int into a LineContext.
func unpackLineContext(packed int) LineContext {
	if packed < 0 {
		return LineContext{State: -1, Flags: lbCtxSot} // Initial state with sot flag
	}
	return LineContext{
		State: packed & 0xFF,
		Flags: (packed >> 8) & 0xFF,
	}
}

// transitionLineBreakStateContext is the Step-compatible wrapper around the new
// context-based line breaking system. It has the same signature as the old
// transitionLineBreakState function.
func transitionLineBreakStateContext(state int, r rune, b []byte, str string) (int, int) {
	ctx := unpackLineContext(state)
	newCtx, lineBreak := transitionLineBreakContext(ctx, r, b, str)
	return packLineContext(newCtx), lineBreak
}

// transitionLineBreakContext is the main entry point for line break decisions.
// It takes the current context, the next rune, and lookahead data, and returns
// the new context and break decision.
func transitionLineBreakContext(ctx LineContext, r rune, b []byte, str string) (LineContext, int) {
	// Get properties
	prop, genCat := propertyLineBreak(r)

	// LB1: Resolve ambiguous properties
	switch prop {
	case prAI, prSG, prXX:
		prop = prAL
	case prSA:
		if genCat == gcMn || genCat == gcMc {
			prop = prCM
		} else {
			prop = prAL
		}
	case prCJ:
		prop = prNS
	}

	// Handle initial state
	if ctx.State < 0 {
		ctx.State = lbcAny
		ctx.Flags = lbCtxSot
	}

	// === COMBINING MARKS (LB9-LB10) - Must come before mandatory breaks ===
	// This is critical: CM/ZWJ after mandatory break states should be treated as AL

	if prop == prCM || prop == prZWJ {
		// LB9: Don't break before CM/ZWJ (treat X CM* as X)
		// States where we can't attach CM (need LB10 instead)
		isSpaceLike := ctx.State == lbcSP || ctx.State == lbcB2SP || ctx.State == lbcQUSP || ctx.State == lbcCLCP || ctx.State == lbcZWSP
		isMandatoryBreak := ctx.State == lbcBK || ctx.State == lbcCR || ctx.State == lbcLF || ctx.State == lbcNL || ctx.State == lbcZW
		isInitial := ctx.State == lbcAny

		if !isSpaceLike && !isMandatoryBreak && !isInitial {
			// LB9: Keep current state, just update ZWJ flag if needed
			newCtx := ctx
			if prop == prZWJ {
				newCtx.Flags |= lbCtxAfterZWJ
			} else {
				// CM absorbs the ZWJ effect - clear the flag
				newCtx.Flags &^= lbCtxAfterZWJ
			}
			return newCtx, LineDontBreak
		}
		// LB10: Treat CM/ZWJ as AL if no base
		newCtx := nextContext(ctx, lbcAL, prAL, r, genCat)
		if prop == prZWJ {
			newCtx.Flags |= lbCtxAfterZWJ
		}
		if isMandatoryBreak {
			return newCtx, LineMustBreak
		}
		if isSpaceLike {
			return newCtx, LineCanBreak
		}
		// Initial state - CM becomes AL, no break
		return newCtx, LineDontBreak
	}

	// === MANDATORY BREAKS (LB4-LB6) ===

	// LB4: BK !
	if ctx.State == lbcBK {
		return applyBreak(ctx, prop, r, genCat, LineMustBreak)
	}

	// LB5: CR × LF, CR !, LF !, NL !
	if ctx.State == lbcCR {
		if prop == prLF {
			newCtx := nextContext(ctx, lbcLF, prop, r, genCat)
			return newCtx, LineDontBreak
		}
		return applyBreak(ctx, prop, r, genCat, LineMustBreak)
	}
	if ctx.State == lbcLF || ctx.State == lbcNL {
		return applyBreak(ctx, prop, r, genCat, LineMustBreak)
	}

	// LB6: × (BK | CR | LF | NL) - don't break before hard breaks
	if prop == prBK || prop == prCR || prop == prLF || prop == prNL {
		newCtx := nextContext(ctx, propToState(prop), prop, r, genCat)
		return newCtx, LineDontBreak
	}

	// === SPACES AND ZW (LB7-LB8a) ===

	// LB7: × SP, × ZW
	if prop == prSP || prop == prZW {
		newState := lbcSP
		if prop == prZW {
			newState = lbcZW
		} else if ctx.State == lbcZW || ctx.State == lbcZWSP {
			newState = lbcZWSP // Track ZW SP* for LB8
		} else if ctx.State == lbcB2 || ctx.State == lbcB2SP {
			newState = lbcB2SP // Track B2 SP* for LB17
		} else if ctx.State == lbcCL || ctx.State == lbcCP || ctx.State == lbcCLCP {
			newState = lbcCLCP // Track (CL|CP) SP* for LB16
		} else if ctx.State == lbcOP || ctx.State == lbcOPSP {
			newState = lbcOPSP // Track OP SP* for LB14
		} else if ctx.State == lbcQUPi || ctx.State == lbcQUPiSP {
			newState = lbcQUPiSP // Track QU_Pi SP* for LB15-like behavior
		}
		newCtx := nextContext(ctx, newState, prop, r, genCat)
		// Clear ZWJ flag after SP - ZWJ effect doesn't carry through spaces
		newCtx.Flags &^= lbCtxAfterZWJ
		return newCtx, LineDontBreak
	}

	// LB8: ZW SP* ÷
	if ctx.State == lbcZW || ctx.State == lbcZWSP {
		return applyBreak(ctx, prop, r, genCat, LineCanBreak)
	}

	// LB8a: ZWJ ×
	if ctx.Flags&lbCtxAfterZWJ != 0 {
		newCtx := nextContext(ctx, propToState(prop), prop, r, genCat)
		newCtx.Flags &^= lbCtxAfterZWJ
		return newCtx, LineDontBreak
	}

	// === WORD JOINER (LB11) ===

	// LB11: × WJ, WJ ×
	if prop == prWJ || ctx.State == lbcWJ {
		newCtx := nextContext(ctx, propToState(prop), prop, r, genCat)
		return newCtx, LineDontBreak
	}

	// === GLUE (LB12-LB12a) ===

	// LB12: GL ×
	if ctx.State == lbcGL {
		newCtx := nextContext(ctx, propToState(prop), prop, r, genCat)
		return newCtx, LineDontBreak
	}

	// LB12a: [^SP BA HY] × GL
	// B2SP, QUSP, CLCP, ZWSP, HH also allow breaks before GL
	if prop == prGL {
		isSpaceLike := ctx.State == lbcSP || ctx.State == lbcB2SP || ctx.State == lbcQUSP || ctx.State == lbcCLCP || ctx.State == lbcZWSP
		if !isSpaceLike && ctx.State != lbcBA && ctx.State != lbcHY && ctx.State != lbcHH && ctx.State != lbcHLHY {
			newCtx := nextContext(ctx, lbcGL, prop, r, genCat)
			return newCtx, LineDontBreak
		}
	}

	// === BREAKING BEFORE CLOSE (LB13) ===

	// LB13: × CL, × CP, × EX, × IS, × SY
	if prop == prCL || prop == prCP || prop == prEX || prop == prIS || prop == prSY {
		newCtx := nextContext(ctx, propToState(prop), prop, r, genCat)
		return newCtx, LineDontBreak
	}

	// === OPENING PUNCTUATION (LB14) ===

	// LB14: OP SP* ×
	if ctx.State == lbcOP || ctx.State == lbcOPSP {
		newCtx := nextContext(ctx, propToState(prop), prop, r, genCat)
		return newCtx, LineDontBreak
	}

	// === QUOTATION MARKS (LB15.1, LB15.2, LB19) ===

	// LB15.1: sot (QU_Pi SP*)+ × OP (handled via context flags)
	if ctx.Flags&lbCtxSot != 0 && ctx.Flags&lbCtxQUPiSP != 0 && prop == prOP {
		newCtx := nextContext(ctx, lbcOP, prop, r, genCat)
		return newCtx, LineDontBreak
	}

	// LB15.2: SP × QU_Pf eot
	isSpaceLike := ctx.State == lbcSP || ctx.State == lbcB2SP || ctx.State == lbcCLCP || ctx.State == lbcQUSP
	if isSpaceLike && prop == prQU && genCat == gcPf {
		// Check if this is end of text
		isEot := len(b) == 0 && str == ""
		if isEot {
			newCtx := nextContext(ctx, lbcQU, prop, r, genCat)
			return newCtx, LineDontBreak
		}
	}

	// === LB16: (CL|CP) SP* × NS ===
	if (ctx.State == lbcCL || ctx.State == lbcCP || ctx.State == lbcCLCP) && prop == prNS {
		newCtx := nextContext(ctx, lbcNS, prop, r, genCat)
		return newCtx, LineDontBreak
	}

	// === B2 (LB17) - must come before LB18 ===

	// LB17: B2 SP* × B2
	if ctx.State == lbcB2 && prop == prB2 {
		newCtx := nextContext(ctx, lbcB2, prop, r, genCat)
		return newCtx, LineDontBreak
	}
	if ctx.State == lbcB2SP && prop == prB2 {
		newCtx := nextContext(ctx, lbcB2, prop, r, genCat)
		return newCtx, LineDontBreak
	}

	// === AFTER SPACES (LB18) ===

	// LB18: SP ÷
	if ctx.State == lbcSP || ctx.State == lbcQUSP || ctx.State == lbcB2SP || ctx.State == lbcCLCP {
		return applyBreak(ctx, prop, r, genCat, LineCanBreak)
	}

	// === QUOTATIONS (LB19) ===

	// LB19: × QU, QU ×
	if prop == prQU {
		newState := lbcQU
		// Track if this is an initial quotation mark (like OP)
		if genCat == gcPi {
			newState = lbcQUPi
		}
		newCtx := nextContext(ctx, newState, prop, r, genCat)
		return newCtx, LineDontBreak
	}
	// QU × - don't break after QU (including QU_Pi and QU_Pi SP*)
	if ctx.State == lbcQU || ctx.State == lbcQUPi || ctx.State == lbcQUPiSP {
		newCtx := nextContext(ctx, propToState(prop), prop, r, genCat)
		return newCtx, LineDontBreak
	}

	// === WORD-INITIAL HYPHENS (LB20a) ===

	// LB20.1: (HY|HH) × (AL|HL) - don't break after hyphen before letters
	// HH × (AL|HL) always applies, but HY × (AL|HL) only at SOT
	if ctx.State == lbcHH {
		if prop == prAL || prop == prHL {
			newCtx := nextContext(ctx, propToState(prop), prop, r, genCat)
			return newCtx, LineDontBreak
		}
	}
	if ctx.Flags&lbCtxSot != 0 && ctx.State == lbcHY {
		if prop == prAL || prop == prHL {
			newCtx := nextContext(ctx, propToState(prop), prop, r, genCat)
			return newCtx, LineDontBreak
		}
	}

	// === CONTINGENT BREAK (LB20) ===

	// LB20: ÷ CB, CB ÷ (break around contingent break)
	if prop == prCB {
		newCtx := nextContext(ctx, lbcCB, prop, r, genCat)
		return newCtx, LineCanBreak // Break before CB
	}
	if ctx.State == lbcCB {
		return applyBreak(ctx, prop, r, genCat, LineCanBreak) // Break after CB
	}

	// === BREAKING AROUND HYPHENS (LB20, LB21) ===

	// LB21: × BA, × HY, × NS, BB ×
	// LB21.02 (Unicode 17.0): × HH (Unambiguous_Hyphen)
	if prop == prBA || prop == prHY || prop == prNS || prop == prHH {
		newCtx := nextContext(ctx, propToState(prop), prop, r, genCat)
		return newCtx, LineDontBreak
	}

	// LB21: BB × (don't break after BB)
	if ctx.State == lbcBB {
		newCtx := nextContext(ctx, propToState(prop), prop, r, genCat)
		return newCtx, LineDontBreak
	}

	// LB21a: HL (HY|BA) ×
	if ctx.State == lbcHLHY {
		newCtx := nextContext(ctx, propToState(prop), prop, r, genCat)
		return newCtx, LineDontBreak
	}

	// Track HL followed by HY/BA for LB21a
	if ctx.State == lbcHL && (prop == prHY || prop == prBA) {
		// Don't break, and remember for LB21a
		newCtx := LineContext{State: lbcHLHY, Flags: ctx.Flags}
		return newCtx, LineDontBreak
	}

	// LB21b: SY × HL
	if ctx.State == lbcSY && prop == prHL {
		newCtx := nextContext(ctx, lbcHL, prop, r, genCat)
		return newCtx, LineDontBreak
	}

	// === INFIXES (LB22) ===

	// LB22: × IN
	if prop == prIN {
		newCtx := nextContext(ctx, lbcIN, prop, r, genCat)
		return newCtx, LineDontBreak
	}

	// === NUMBERS (LB23-LB25) ===

	// LB23: (AL|HL) × NU, NU × (AL|HL)
	// Dotted Circle acts like AL
	if (ctx.State == lbcAL || ctx.State == lbcHL || ctx.State == lbcDottedCircle) && prop == prNU {
		newCtx := nextContext(ctx, lbcNU, prop, r, genCat)
		return newCtx, LineDontBreak
	}
	if ctx.State == lbcNU && (prop == prAL || prop == prHL) {
		newCtx := nextContext(ctx, propToState(prop), prop, r, genCat)
		return newCtx, LineDontBreak
	}

	// LB23a: PR × (ID|EB|EM), (ID|EB|EM) × PO
	if ctx.State == lbcPR && (prop == prID || prop == prEB || prop == prEM) {
		newCtx := nextContext(ctx, propToState(prop), prop, r, genCat)
		return newCtx, LineDontBreak
	}
	if (ctx.State == lbcID || ctx.State == lbcEB || ctx.State == lbcEM || ctx.State == lbcExtPic || ctx.State == lbcExtPicCn) && prop == prPO {
		newCtx := nextContext(ctx, lbcPO, prop, r, genCat)
		return newCtx, LineDontBreak
	}

	// LB24: (PR|PO) × (AL|HL), (AL|HL) × (PR|PO)
	// Dotted Circle acts like AL for these rules
	if (ctx.State == lbcPR || ctx.State == lbcPO) && (prop == prAL || prop == prHL) {
		newCtx := nextContext(ctx, propToState(prop), prop, r, genCat)
		return newCtx, LineDontBreak
	}
	if (ctx.State == lbcAL || ctx.State == lbcHL || ctx.State == lbcDottedCircle) && (prop == prPR || prop == prPO) {
		newCtx := nextContext(ctx, propToState(prop), prop, r, genCat)
		return newCtx, LineDontBreak
	}

	// LB25: Numeric sequences
	breakDecision := applyLB25(ctx, prop)
	if breakDecision != -1 {
		newCtx := nextContext(ctx, propToState(prop), prop, r, genCat)
		return newCtx, breakDecision
	}

	// === KOREAN (LB26-LB27) ===

	// LB26: JL × (JL|JV|H2|H3), (JV|H2) × (JV|JT), (JT|H3) × JT
	if ctx.State == lbcJL && (prop == prJL || prop == prJV || prop == prH2 || prop == prH3) {
		newCtx := nextContext(ctx, propToState(prop), prop, r, genCat)
		return newCtx, LineDontBreak
	}
	if (ctx.State == lbcJV || ctx.State == lbcH2) && (prop == prJV || prop == prJT) {
		newCtx := nextContext(ctx, propToState(prop), prop, r, genCat)
		return newCtx, LineDontBreak
	}
	if (ctx.State == lbcJT || ctx.State == lbcH3) && prop == prJT {
		newCtx := nextContext(ctx, lbcJT, prop, r, genCat)
		return newCtx, LineDontBreak
	}

	// LB27: (JL|JV|JT|H2|H3) × IN, (JL|JV|JT|H2|H3) × PO, PR × (JL|JV|JT|H2|H3)
	isKorean := ctx.State == lbcJL || ctx.State == lbcJV || ctx.State == lbcJT || ctx.State == lbcH2 || ctx.State == lbcH3
	if isKorean && (prop == prIN || prop == prPO) {
		newCtx := nextContext(ctx, propToState(prop), prop, r, genCat)
		return newCtx, LineDontBreak
	}
	if ctx.State == lbcPR && (prop == prJL || prop == prJV || prop == prJT || prop == prH2 || prop == prH3) {
		newCtx := nextContext(ctx, propToState(prop), prop, r, genCat)
		return newCtx, LineDontBreak
	}

	// === AKSARA (LB28a) - Unicode 17 ===

	// Dotted Circle (U+25CC) acts like Aksara for LB28 rules
	isDottedCircle := r == 0x25CC

	// Check if we should apply Aksara rules (Dotted Circle acts like AK for these)
	breakDecision = applyLB28a(ctx, prop, isDottedCircle)
	if breakDecision != -1 {
		newState := propToState(prop)
		if prop == prAK || prop == prAS {
			newState = lbcAK
		} else if isDottedCircle {
			newState = lbcDottedCircle
		} else if prop == prVI {
			// Check if we came from AK/AS/DottedCircle for LB28.13 tracking
			if ctx.State == lbcAK || ctx.State == lbcAS || ctx.State == lbcDottedCircle {
				newState = lbcAKVI
			} else {
				newState = lbcVI
			}
		}
		newCtx := nextContext(ctx, newState, prop, r, genCat)
		if prop == prVI || prop == prVF {
			newCtx.Flags |= lbCtxAksaraVirama
		} else if prop != prCM && prop != prZWJ {
			newCtx.Flags &^= lbCtxAksaraVirama
		}
		return newCtx, breakDecision
	}

	// Handle Dotted Circle specially - it acts like AL but transitions to lbcDottedCircle
	if isDottedCircle {
		newCtx := nextContext(ctx, lbcDottedCircle, prop, r, genCat)
		// Dotted Circle follows normal AL break rules (LB28, LB24, LB23, LB29, LB30)
		if ctx.State == lbcAL || ctx.State == lbcHL || ctx.State == lbcDottedCircle {
			return newCtx, LineDontBreak // LB28-like: AL × DottedCircle
		}
		if ctx.State == lbcPR || ctx.State == lbcPO {
			return newCtx, LineDontBreak // LB24: (PR|PO) × AL (DottedCircle acts as AL)
		}
		if ctx.State == lbcNU {
			return newCtx, LineDontBreak // LB23: NU × AL (DottedCircle acts as AL)
		}
		// LB29: IS × (AL|HL) - DottedCircle acts as AL
		if ctx.State == lbcIS {
			return newCtx, LineDontBreak
		}
		// LB30: CP × (AL|HL|NU) - DottedCircle acts as AL
		if ctx.State == lbcCP && ctx.Flags&lbCtxCPeaFWH == 0 {
			return newCtx, LineDontBreak
		}
		return newCtx, LineCanBreak
	}

	// Handle AK, AS, AP, VF, VI even if not matched by applyLB28a
	if prop == prAK || prop == prAS {
		newCtx := nextContext(ctx, lbcAK, prop, r, genCat)
		return newCtx, LineCanBreak
	}
	if prop == prAP {
		newCtx := nextContext(ctx, lbcAP, prop, r, genCat)
		return newCtx, LineCanBreak
	}
	if prop == prVF {
		newCtx := nextContext(ctx, lbcVF, prop, r, genCat)
		return newCtx, LineCanBreak
	}
	if prop == prVI {
		newCtx := nextContext(ctx, lbcVI, prop, r, genCat)
		return newCtx, LineCanBreak
	}

	// === ALPHABETICS (LB28-LB29) ===

	// LB28: (AL|HL) × (AL|HL)
	// Dotted Circle also acts like AL for LB28
	if (ctx.State == lbcAL || ctx.State == lbcHL || ctx.State == lbcDottedCircle) && (prop == prAL || prop == prHL) {
		newCtx := nextContext(ctx, propToState(prop), prop, r, genCat)
		return newCtx, LineDontBreak
	}

	// LB29: IS × (AL|HL)
	if ctx.State == lbcIS && (prop == prAL || prop == prHL) {
		newCtx := nextContext(ctx, propToState(prop), prop, r, genCat)
		return newCtx, LineDontBreak
	}

	// === OPENING/CLOSING (LB30) ===

	// LB30: (AL|HL|NU) × OP, CP × (AL|HL|NU)
	// Dotted Circle acts like AL
	if (ctx.State == lbcAL || ctx.State == lbcHL || ctx.State == lbcNU || ctx.State == lbcDottedCircle) && prop == prOP {
		// Check East Asian Width for LB30 exception
		eaw := propertyEastAsianWidth(r)
		if eaw != prF && eaw != prW && eaw != prH {
			newCtx := nextContext(ctx, lbcOP, prop, r, genCat)
			return newCtx, LineDontBreak
		}
	}
	if ctx.State == lbcCP && ctx.Flags&lbCtxCPeaFWH == 0 {
		if prop == prAL || prop == prHL || prop == prNU {
			newCtx := nextContext(ctx, propToState(prop), prop, r, genCat)
			return newCtx, LineDontBreak
		}
	}

	// === EMOJI (LB30a-LB30b) ===

	// LB30a: sot (RI RI)* RI × RI
	// First RI can be broken before, second RI cannot, then break before third, etc.
	if prop == prRI {
		switch ctx.State {
		case lbcRIOdd, lbcRI:
			// We have odd number of RI, this one makes it even - no break
			newCtx := nextContext(ctx, lbcRIEven, prop, r, genCat)
			return newCtx, LineDontBreak
		case lbcRIEven:
			// We have even number of RI, this one makes it odd - can break
			newCtx := nextContext(ctx, lbcRIOdd, prop, r, genCat)
			return newCtx, LineCanBreak
		default:
			// First RI - can break before it (LB31), then track as odd
			newCtx := nextContext(ctx, lbcRIOdd, prop, r, genCat)
			return newCtx, LineCanBreak
		}
	}

	// LB30b: EB × EM, ExtPic × EM
	// Also handle unassigned Extended_Pictographic (check grapheme property)
	if (ctx.State == lbcEB || ctx.State == lbcExtPic || ctx.State == lbcExtPicCn || ctx.State == lbcExtPicZWJ) && prop == prEM {
		newCtx := nextContext(ctx, lbcEM, prop, r, genCat)
		return newCtx, LineDontBreak
	}

	// Track Extended Pictographic (including unassigned codepoints)
	if prop == prExtendedPictographic {
		newCtx := nextContext(ctx, lbcExtPic, prop, r, genCat)
		return newCtx, LineCanBreak
	}
	// Check for unassigned Extended_Pictographic using grapheme property
	graphemeProp := propertyGraphemes(r)
	if graphemeProp == prExtendedPictographic && genCat == gcCn {
		newCtx := nextContext(ctx, lbcExtPicCn, prop, r, genCat)
		return newCtx, LineCanBreak
	}

	// === DEFAULT (LB31) ===

	// LB31: ALL ÷ ALL
	return applyBreak(ctx, prop, r, genCat, LineCanBreak)
}

// applyBreak applies a break and returns the new context.
func applyBreak(ctx LineContext, prop int, r rune, genCat int, breakType int) (LineContext, int) {
	newCtx := nextContext(ctx, propToState(prop), prop, r, genCat)
	// Clear sot flag after any break
	newCtx.Flags &^= lbCtxSot
	newCtx.Flags &^= lbCtxQUPiSP
	newCtx.Flags &^= lbCtxAfterQUPi
	return newCtx, breakType
}

// nextContext creates the next context after seeing a character.
func nextContext(ctx LineContext, newState, prop int, r rune, genCat int) LineContext {
	newCtx := LineContext{
		State: newState,
		Flags: ctx.Flags,
	}

	// Update context flags based on what we just saw
	if prop == prQU && genCat == gcPi {
		newCtx.Flags |= lbCtxAfterQUPi
	}

	// Track SP after QU_Pi for LB15.1
	if prop == prSP && ctx.Flags&lbCtxAfterQUPi != 0 {
		newCtx.Flags |= lbCtxQUPiSP
	}

	// Track CP East Asian Width for LB30
	if prop == prCP {
		eaw := propertyEastAsianWidth(r)
		if eaw == prF || eaw == prW || eaw == prH {
			newCtx.Flags |= lbCtxCPeaFWH
		} else {
			newCtx.Flags &^= lbCtxCPeaFWH
		}
	}

	return newCtx
}

// propToState converts a line break property to a base state.
func propToState(prop int) int {
	switch prop {
	case prBK:
		return lbcBK
	case prCR:
		return lbcCR
	case prLF:
		return lbcLF
	case prNL:
		return lbcNL
	case prSP:
		return lbcSP
	case prZW:
		return lbcZW
	case prWJ:
		return lbcWJ
	case prGL:
		return lbcGL
	case prBA:
		return lbcBA
	case prHY:
		return lbcHY
	case prCL:
		return lbcCL
	case prCP:
		return lbcCP
	case prEX:
		return lbcEX
	case prIS:
		return lbcIS
	case prSY:
		return lbcSY
	case prOP:
		return lbcOP
	case prQU:
		return lbcQU
	case prNS:
		return lbcNS
	case prAL:
		return lbcAL
	case prHL:
		return lbcHL
	case prNU:
		return lbcNU
	case prPR:
		return lbcPR
	case prPO:
		return lbcPO
	case prID:
		return lbcID
	case prEB:
		return lbcEB
	case prEM:
		return lbcEM
	case prIN:
		return lbcIN
	case prCB:
		return lbcCB
	case prRI:
		return lbcRI
	case prZWJ:
		return lbcZWJ
	case prCM:
		return lbcCM
	case prAK:
		return lbcAK
	case prAP:
		return lbcAP
	case prAS:
		return lbcAS
	case prVI:
		return lbcVI
	case prVF:
		return lbcVF
	case prHH:
		return lbcHH
	case prB2:
		return lbcB2
	case prBB:
		return lbcBB
	case prExtendedPictographic:
		return lbcExtPic
	case prH2:
		return lbcH2
	case prH3:
		return lbcH3
	case prJL:
		return lbcJL
	case prJV:
		return lbcJV
	case prJT:
		return lbcJT
	default:
		return lbcAL // Default to AL for unknown
	}
}

// applyLB25 applies LB25 numeric rules.
// Returns -1 if no rule applies, otherwise the break decision.
// Note: The full LB25 rule requires tracking numeric context. This is a simplified version
// that only applies the rules that don't require context.
func applyLB25(ctx LineContext, prop int) int {
	// LB25 simplified versions:
	// NU × (PO|PR) - don't break between number and postfix/prefix
	// (PO|PR|HY|IS|NU) × NU - don't break before number
	// Note: SY×NU requires numeric context (NU preceding SY) which we don't fully track
	switch {
	case ctx.State == lbcNU && (prop == prPO || prop == prPR):
		return LineDontBreak
	case (ctx.State == lbcPO || ctx.State == lbcPR || ctx.State == lbcHY || ctx.State == lbcIS || ctx.State == lbcNU) && prop == prNU:
		return LineDontBreak
	}
	return -1
}

// applyLB28a applies LB28a Aksara rules (Unicode 17).
// Returns -1 if no rule applies, otherwise the break decision.
// isDottedCircle indicates if the current rune is U+25CC (Dotted Circle)
func applyLB28a(ctx LineContext, prop int, isDottedCircle bool) int {
	// LB28.11: AP × (AK | DottedCircle | AS)
	if ctx.State == lbcAP && (prop == prAK || prop == prAS || isDottedCircle) {
		return LineDontBreak
	}

	// LB28.12: (AK | DottedCircle | AS) × (VF | VI)
	// Note: This transitions to lbcAKVI for VI to enable LB28.13
	if (ctx.State == lbcAK || ctx.State == lbcDottedCircle || ctx.State == lbcAS) && (prop == prVF || prop == prVI) {
		return LineDontBreak
	}

	// LB28.13: (AK | DottedCircle | AS) VI × (AK | DottedCircle | AS)
	// Only applies when we have the full (AK|AS|◌)VI sequence (tracked by lbcAKVI)
	if ctx.State == lbcAKVI && (prop == prAK || prop == prAS || isDottedCircle) {
		return LineDontBreak
	}

	// Note: There is NO VF × (AK...) rule in TR14 LB28a
	// VF followed by AK should break (fall through to default)

	return -1
}

// FirstLineSegmentContext returns the first line segment using the new context system.
// This is the equivalent of FirstLineSegment but uses the cleaner architecture.
func FirstLineSegmentContext(b []byte, state int) (segment, rest []byte, breakType int, newState int) {
	if len(b) == 0 {
		return nil, nil, LineDontBreak, -1
	}

	// Extract the first rune
	r, length := utf8.DecodeRune(b)
	if len(b) <= length {
		// Only one rune - LB3 mandates break at end
		return b, nil, LineMustBreak, packLineContext(LineContext{State: lbcAny})
	}

	// Process first rune to establish/update state
	ctx := unpackLineContext(state)
	ctx, _ = transitionLineBreakContext(ctx, r, b[length:], "")

	// Process subsequent runes to find break point
	for {
		r, l := utf8.DecodeRune(b[length:])
		remainder := b[length+l:]

		var newCtx LineContext
		newCtx, breakType = transitionLineBreakContext(ctx, r, remainder, "")

		if breakType != LineDontBreak {
			// Found a break point
			return b[:length], b[length:], breakType, packLineContext(ctx)
		}

		ctx = newCtx
		length += l

		if len(b) <= length {
			// End of input - LB3 mandates break at end
			return b, nil, LineMustBreak, packLineContext(ctx)
		}
	}
}
