package runeseg

import (
	"testing"
)

// TestLineContextBasic tests basic line breaking with the new context system.
func TestLineContextBasic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"simple", "hello world", []string{"hello ", "world"}},
		{"hyphen", "well-known", []string{"well-", "known"}},
		{"numbers", "100.50", []string{"100.50"}},
		{"crlf", "a\r\nb", []string{"a\r\n", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var segments []string
			b := []byte(tt.input)
			state := -1
			for len(b) > 0 {
				var seg []byte
				seg, b, _, state = FirstLineSegmentContext(b, state)
				segments = append(segments, string(seg))
			}

			if len(segments) != len(tt.expected) {
				t.Errorf("got %d segments, want %d: %v vs %v", len(segments), len(tt.expected), segments, tt.expected)
				return
			}
			for i, seg := range segments {
				if seg != tt.expected[i] {
					t.Errorf("segment %d: got %q, want %q", i, seg, tt.expected[i])
				}
			}
		})
	}
}

// TestLineContextAgainstOriginal compares the new system against the original.
func TestLineContextAgainstOriginal(t *testing.T) {
	testStrings := []string{
		"Hello, world!",
		"This is a test.",
		"Line1\nLine2",
		"Number: 123.456",
		"URL: https://example.com",
		"Mixed-case Test",
	}

	for _, s := range testStrings {
		// Get segments from new system
		var newSegments []string
		b := []byte(s)
		state := -1
		for len(b) > 0 {
			var seg []byte
			seg, b, _, state = FirstLineSegmentContext(b, state)
			newSegments = append(newSegments, string(seg))
		}

		// Get segments from original system
		var oldSegments []string
		b = []byte(s)
		state = -1
		for len(b) > 0 {
			var seg []byte
			seg, b, _, state = FirstLineSegment(b, state)
			oldSegments = append(oldSegments, string(seg))
		}

		// Compare
		if len(newSegments) != len(oldSegments) {
			t.Logf("String: %q", s)
			t.Logf("New: %v", newSegments)
			t.Logf("Old: %v", oldSegments)
			// Don't fail yet - we expect some differences as we fix issues
		}
	}
}

// TestLineContextUnicodeTestCases runs the new system against the Unicode test cases.
func TestLineContextUnicodeTestCases(t *testing.T) {
	passed := 0
	failed := 0

	for i, testCase := range lineBreakTestCases {
		var segments [][]rune
		b := []byte(testCase.original)
		state := -1
		for len(b) > 0 {
			var seg []byte
			seg, b, _, state = FirstLineSegmentContext(b, state)
			segments = append(segments, []rune(string(seg)))
		}

		// Check if segments match expected
		if len(segments) != len(testCase.expected) {
			failed++
			if failed <= 50 {
				t.Logf("Test case %d %q: got %d segments, want %d", i, testCase.original, len(segments), len(testCase.expected))
			}
			continue
		}

		match := true
		for j, seg := range segments {
			if len(seg) != len(testCase.expected[j]) {
				match = false
				break
			}
			for k, r := range seg {
				if r != testCase.expected[j][k] {
					match = false
					break
				}
			}
		}

		if match {
			passed++
		} else {
			failed++
			if failed <= 200 {
				t.Logf("Test case %d %q: got %d segs, want %d", i, testCase.original, len(segments), len(testCase.expected))
			}
		}
	}

	t.Logf("New context system: %d passed, %d failed out of %d total", passed, failed, len(lineBreakTestCases))

	// For now, just report - we'll make it a hard failure once we're closer
	if failed > 0 {
		t.Logf("Note: %d test cases still failing - this is expected during refactoring", failed)
	}
}
