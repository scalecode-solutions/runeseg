# Failing Line Break Tests Analysis

**Status:** 26 out of 19,338 tests failing (99.87% pass rate)

**Last Updated:** January 5, 2026

**Recent Fixes:**
- ✅ LB20a word-initial hyphen rule (4 tests fixed: 19318, 19319, 19330, 19332)

**System Status:** All line breaking (public API and internal) now uses the unified context-based system (`linecontext.go`). The legacy state machine (`linerules.go`) is deprecated.

This document catalogs all failing Unicode line break test cases and explains why they fail.

---

## Summary by Category

| Category | Count | Priority |
|----------|-------|----------|
| Quotation Mark Handling | 12 | High |
| ~~Hyphen + Letter Combinations~~ | ~~4~~ → 0 | ✅ Fixed |
| Numeric Sequences | 3 | Medium |
| Emoji with Modifiers | 2 | Medium |
| Parentheses/Braces with Operators | 4 | Medium |
| Southeast Asian Scripts | 1 | Medium |
| Hebrew/Akkadian Text | 2 | Medium |
| RTL/LTR Directional Marks | 2 | Low |

---

## Detailed Test Case Analysis

### 1. Quotation Mark Handling (12 cases)

#### Issue: Opening Quotation Marks Not Staying With Content

**Related Unicode Rules:** LB15, LB19

**Test Cases:**
- **19309:** `"vous me heurtez, vous dites : « Excusez-moi, » et vous croyez que cela suffit ?"`
  - Got: 15 segments | Expected: 13
  - Problem: French guillemet `«` breaking from following text
  
- **19310:** `"j'ai dit : « Excusez-moi. » Il me semble donc que c'est assez."`
  - Got: 13 segments | Expected: 11
  - Problem: Opening quote separating from text
  
- **19311:** `"Et vise au front mon père en criant : « Caramba ! »..."`
  - Got: 41 segments | Expected: 37
  - Problem: Multiple quotation marks not handled correctly

- **19313:** `"— Không ai hãm bao giờ mà bây giờ hãm, thế nó mới « mới »."`
  - Got: 16 segments | Expected: 14
  - Problem: Vietnamese text with French quotes

- **19314:** `"Pas une citation »Zitat« Pas une citation non plus"`
  - Got: 8 segments | Expected: 9
  - Problem: Reversed guillemet handling

- **19315:** `"« Citation »\u200bKein Zitat\u200b« Autre citation »"`
  - Got: 7 segments | Expected: 5
  - Problem: Zero-width space + quotation mark interaction

- **19320:** `"子曰："学而时习之，不亦说乎？有朋自远方来，不亦乐乎？人不知而不愠，不亦君子乎？""`
  - Got: 31 segments | Expected: 32
  - Problem: Chinese left double quotation mark (U+201C)

- **19321:** Long Chinese text with nested quotes
  - Got: 56 segments | Expected: 64
  - Problem: Complex quotation nesting

- **19322:** `"哪一所中国学校乃"为各省派往日本游学之首倡"？"`
  - Got: 19 segments | Expected: 20
  - Problem: Chinese right double quotation mark (U+201D)

- **19323:** `"哪个商标以人名为名，因特色小吃"五台杂烩汤"而入选"新疆老字号"？"`
  - Got: 24 segments | Expected: 27
  - Problem: Multiple Chinese quote pairs

- **19326:** `"Z-1"莱贝雷希特·马斯"号是德国国家海军暨战争海军于1930年代"`
  - Got: 24 segments | Expected: 25
  - Problem: German ship name with Chinese quotation marks (Latin-to-CJK transition)

- **19327:** `"Anmerkung: „White" bzw. ‚白人' – in der Amtlichen Statistik"`
  - Got: 8 segments | Expected: 10
  - Problem: German quotes + Chinese quotes

**Root Cause:**

The LB19 rule states: `× QU` and `QU ×` (don't break before or after quotation marks). However, the implementation needs to distinguish between:
- **Opening quotes (Pi)** - should stay with following content
- **Closing quotes (Pf)** - should stay with preceding content  
- **Ambiguous quotes** - context-dependent

The current implementation in `linecontext.go` (lines 359-375) treats all QU uniformly, but needs special handling for Pi/Pf general categories.

**Fix Strategy:**
1. Track general category (Pi vs Pf) for quotation marks
2. Implement LB15/LB15.1 properly for opening quotes
3. Handle space after opening quotes correctly

---

### 2. ~~Hyphen + Letter Combinations~~ ✅ FIXED

**Status:** All 4 tests now passing (fixed in commit f55122f)

**Fixed Tests:** 19318, 19319, 19330, 19332

**Solution:** Added `lbCtxWordStart` flag to track word-initial hyphen positions. The LB20a rule now correctly applies when HY/HH follows any of: sot, BK, CR, LF, NL, SP (including SP-derived states), ZW, CB, GL.

---

### 3. Numeric Sequences (3 cases)

#### Issue: Decimal Points Not Staying With Digits

**Related Unicode Rules:** LB25

**Test Cases:**
- **19096:** `"equals .35 cents"`
  - Got: 2 segments | Expected: 3
  - Problem: `.35` treated as one segment instead of breaking after "equals "
  
- **19316:** `"start .789 end"`
  - Got: 2 segments | Expected: 3
  - Problem: Leading decimal point numbers

- **19317:** `"$-5 -.3 £(123.456) 123.€ +.25 1/2"`
  - Got: 9 segments | Expected: 6
  - Problem: Complex numeric expressions with currency and parentheses

**Root Cause:**

LB25 handles numeric sequences: `(PR | PO) × ( OP | HY )? NU`, and `NU (NU | SY | IS)* × (NU | SY | IS | CL | CP)`. 

The issue is with leading decimal points (`.35`) where the period should be treated as part of the number if it precedes digits. The current implementation may not correctly handle:
1. Sentence-ending period vs. decimal point disambiguation
2. Space before decimal-point-number combinations

**Fix Strategy:**
1. Look ahead when encountering IS (period) followed by NU
2. Distinguish between sentence-final periods and numeric decimal points
3. Handle PR/PO + decimal combinations correctly

---

### 4. Emoji with Skin Tone Modifiers (2 cases)

#### Issue: Extended Emoji Not Recognized

**Related Unicode Rules:** LB30b, GB11

**Test Cases:**
- **16828:** `"\U0001f8ff🏻"`
  - Got: 2 segments | Expected: 1
  - Problem: U+1F8FF (unassigned in Emoji) + U+1F3FB (skin tone modifier)

- **16830:** `"\U0001f8ff̈🏻"`
  - Got: 2 segments | Expected: 1
  - Problem: Same emoji + combining diaeresis (U+0308) + skin tone modifier

**Root Cause:**

U+1F8FF is in the range reserved for emoji but may not be in the Extended_Pictographic property table. The skin tone modifier (U+1F3FB) should attach to the preceding character via GB11 (grapheme cluster rule), but the line break rules need to respect this.

LB30b: `EB × EM` (Extended Base × Emoji Modifier) should prevent breaks.

**Fix Strategy:**
1. Verify Extended_Pictographic property coverage for U+1F8FF range
2. Ensure EM (Emoji Modifier) property correctly identifies skin tones
3. Implement LB30b rule for EB × EM sequences

---

### 5. Parentheses/Braces with Operators (4 cases)

#### Issue: Mathematical Expressions Breaking Incorrectly

**Related Unicode Rules:** LB14, LB16, LB30

**Test Cases:**
- **19133:** `"ambigu(« ̈ »)(ë)"`
  - Got: 4 segments | Expected: 2
  - Problem: Parentheses with quotes and combining marks

- **19134:** `"ambigu« ( ̈ ) »(ë)"`
  - Got: 2 segments | Expected: 3
  - Problem: Quotes containing parentheses with combining marks

- **19138:** `"ambigu{« ̈ »}(ë)"`
  - Got: 4 segments | Expected: 2
  - Problem: Braces with nested quotes

- **19139:** `"ambigu« { ̈ } »(ë)"`
  - Got: 2 segments | Expected: 3
  - Problem: Quotes containing braces

- **19165:** `"(0,1)+(2,3)⊕(−4,5)⊖(6,7)"`
  - Got: 3 segments | Expected: 1
  - Problem: Mathematical operators (⊕ U+2295, ⊖ U+2296) causing breaks

- **19166:** `"{0,1}+{2,3}⊕{−4,5}⊖{6,7}"`
  - Got: 5 segments | Expected: 3
  - Problem: Braces in math expression

**Root Cause:**

Multiple interacting rules:
1. **LB14:** `OP SP* ×` - don't break after opening punctuation
2. **LB16:** `(CL | CP) SP* ×` - complex handling of closing punctuation
3. **LB30:** `(AL | HL | NU) × OP` - don't break before opening after letters/numbers

The mathematical operators (⊕, ⊖) may have AL (Alphabetic) properties, and the complex nesting of operators, parentheses, quotes, and combining marks creates edge cases.

**Fix Strategy:**
1. Verify line break properties for mathematical operators
2. Improve handling of nested OP/CL pairs
3. Correctly process combining marks (U+0308) within expressions

---

### 6. Southeast Asian Scripts (1 case)

#### Issue: Aksara Line Breaking Not Working

**Related Unicode Rules:** LB28a

**Test Cases:**
- **19301:** `"ᯗᯬᯒᯪᯉ᯳ᯂᯧᯉ᯳"` (Batak script)
  - Got: 5 segments | Expected: 3
  - Problem: Aksara clusters breaking incorrectly

**Root Cause:**

LB28a is a new rule in Unicode 17.0 for Aksara-based scripts:

> **LB28a:** `AP × (AK | AS)` and `(AK | AS) × (VF | VI)`

The properties involved:
- **AK** - Aksara (base characters)
- **AS** - Aksara Start
- **AP** - Aksara Prebase  
- **VF** - Virama Final
- **VI** - Virama

This handles complex ligature formations in scripts like Batak, Balinese, Javanese, etc. The README mentions this is implemented, but test 19301 shows it's not working correctly.

**Fix Strategy:**
1. Review Aksara property assignments in `lineproperties.go`
2. Verify LB28a implementation in `linecontext.go`
3. Check for correct Virama handling

---

### 7. Hebrew/Akkadian Text (3 cases - already covered)

See section 2 (Hyphen + Letter Combinations) for:
- Test 19329: Hebrew text with maqaf
- Test 19330: Akkadian suffix
- Test 19332: Hebrew with RTL mark

---

### 8. RTL/LTR Directional Marks (2 cases)

#### Issue: Directional Formatting Characters

**Related Unicode Rules:** LB1

**Test Cases:**
- **19328:** `" \u2067John ו-Michael\u2069;"`
  - Got: 4 segments | Expected: 3
  - Problem: RLI (U+2067 Right-to-Left Isolate) and PDI (U+2069 Pop Directional Isolate) with Hebrew conjunction vav (ו) and hyphen

- **19329:** `"וַֽיְהִי־כֵֽן׃"`
  - Got: 1 segments | Expected: 2
  - Problem: Hebrew text with maqaf (U+05BE, Hebrew hyphen) should allow break

**Root Cause:**

1. **U+2067 (RLI)** and **U+2069 (PDI)** are formatting characters that should be treated as CM (Combining Mark) by LB10
2. The Hebrew maqaf (U+05BE) has line break property BA (Break After), which should allow a break after it

The handling of directional formatting characters interacting with hyphens and Hebrew text needs refinement.

**Fix Strategy:**
1. Ensure RLI/PDI treated as CM and handled by LB9/LB10
2. Verify Hebrew-specific properties (maqaf, geresh, gershayim)
3. Handle combining marks in RTL contexts

---

## Additional Test Case

### 19305: Dotted Circle Sequences

**Test:** `"◌᭄◌᭄ᬬ"` (Dotted circles + Balinese adeg-adeg + Balinese letter)
- This case is not appearing in the failing tests output, but may be related to the Aksara handling

---

## Implementation Priority

### High Priority (Critical Rules)
1. **Quotation marks (LB15, LB19)** - 11 failures, affects all languages
2. **Hyphen-letter combos (LB20.1)** - 4 failures, common in many languages

### Medium Priority (Important)
3. **Numeric sequences (LB25)** - 3 failures, affects financial/technical text
4. **Parentheses/operators** - 4 failures, affects mathematical and technical content
5. **Aksara scripts (LB28a)** - 1 failure, but important for Unicode 17.0 compliance
6. **Emoji modifiers (LB30b)** - 2 failures, increasing importance

### Low Priority (Edge Cases)
7. **RTL marks** - 2 failures, complex but less common

---

## Testing Notes

- All grapheme, word, and sentence boundary tests pass ✅
- Step and width tests pass ✅  
- Only line breaking tests have failures
- All line breaking now uses the unified context-based system
- Legacy `transitionLineBreakState` in `linerules.go` is deprecated
- Each failing test has a corresponding GitHub issue (#1-#30)

---

## Next Steps

1. **Implement quotation mark tracking** - Add Pi/Pf detection and state tracking
2. **Fix LB20.1 hyphen rule** - Extend beyond SOT to word boundaries  
3. **Review numeric handling** - Improve decimal point disambiguation
4. **Verify Unicode 17.0 properties** - Ensure all new properties loaded correctly
5. **Add targeted unit tests** - Create focused tests for each rule before fixing

---

## References

- [Unicode Standard Annex #14 - Line Breaking](https://unicode.org/reports/tr14/)
- [Unicode 17.0 Line Break Test Cases](https://www.unicode.org/Public/17.0.0/ucd/auxiliary/LineBreakTest.txt)
- Original issue noted in `linecontext_test.go:138` - "30 test cases still failing - this is expected during refactoring"

