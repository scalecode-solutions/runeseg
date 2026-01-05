package runeseg_test

import (
	"fmt"

	"github.com/scalecode-solutions/runeseg"
)

func ExampleGraphemeClusterCount() {
	n := runeseg.GraphemeClusterCount("🇩🇪🏳️‍🌈")
	fmt.Println(n)
	// Output: 2
}

func ExampleFirstGraphemeCluster() {
	b := []byte("🇩🇪🏳️‍🌈!")
	state := -1
	var c []byte
	for len(b) > 0 {
		var width int
		c, b, width, state = runeseg.FirstGraphemeCluster(b, state)
		fmt.Println(string(c), width)
	}
	// Output: 🇩🇪 2
	//🏳️‍🌈 2
	//! 1
}

func ExampleFirstGraphemeClusterInString() {
	str := "🇩🇪🏳️‍🌈!"
	state := -1
	var c string
	for len(str) > 0 {
		var width int
		c, str, width, state = runeseg.FirstGraphemeClusterInString(str, state)
		fmt.Println(c, width)
	}
	// Output: 🇩🇪 2
	//🏳️‍🌈 2
	//! 1
}

func ExampleFirstWord() {
	b := []byte("Hello, world!")
	state := -1
	var c []byte
	for len(b) > 0 {
		c, b, state = runeseg.FirstWord(b, state)
		fmt.Printf("(%s)\n", string(c))
	}
	// Output: (Hello)
	//(,)
	//( )
	//(world)
	//(!)
}

func ExampleFirstWordInString() {
	str := "Hello, world!"
	state := -1
	var c string
	for len(str) > 0 {
		c, str, state = runeseg.FirstWordInString(str, state)
		fmt.Printf("(%s)\n", c)
	}
	// Output: (Hello)
	//(,)
	//( )
	//(world)
	//(!)
}

func ExampleFirstSentence() {
	b := []byte("This is sentence 1.0. And this is sentence two.")
	state := -1
	var c []byte
	for len(b) > 0 {
		c, b, state = runeseg.FirstSentence(b, state)
		fmt.Printf("(%s)\n", string(c))
	}
	// Output: (This is sentence 1.0. )
	//(And this is sentence two.)
}

func ExampleFirstSentenceInString() {
	str := "This is sentence 1.0. And this is sentence two."
	state := -1
	var c string
	for len(str) > 0 {
		c, str, state = runeseg.FirstSentenceInString(str, state)
		fmt.Printf("(%s)\n", c)
	}
	// Output: (This is sentence 1.0. )
	//(And this is sentence two.)
}

func ExampleFirstLineSegment() {
	b := []byte("First line.\nSecond line.")
	state := -1
	var (
		c         []byte
		mustBreak bool
	)
	for len(b) > 0 {
		c, b, mustBreak, state = runeseg.FirstLineSegment(b, state)
		fmt.Printf("(%s)", string(c))
		if mustBreak {
			fmt.Print("!")
		}
	}
	// Output: (First )(line.
	//)!(Second )(line.)!
}

func ExampleFirstLineSegmentInString() {
	str := "First line.\nSecond line."
	state := -1
	var (
		c         string
		mustBreak bool
	)
	for len(str) > 0 {
		c, str, mustBreak, state = runeseg.FirstLineSegmentInString(str, state)
		fmt.Printf("(%s)", c)
		if mustBreak {
			fmt.Println(" < must break")
		} else {
			fmt.Println(" < may break")
		}
	}
	// Output: (First ) < may break
	//(line.
	//) < must break
	//(Second ) < may break
	//(line.) < must break
}

func ExampleStep_graphemes() {
	b := []byte("🇩🇪🏳️‍🌈!")
	state := -1
	var c []byte
	for len(b) > 0 {
		var boundaries int
		c, b, boundaries, state = runeseg.Step(b, state)
		fmt.Println(string(c), boundaries>>runeseg.ShiftWidth)
	}
	// Output: 🇩🇪 2
	//🏳️‍🌈 2
	//! 1
}

func ExampleStepString_graphemes() {
	str := "🇩🇪🏳️‍🌈!"
	state := -1
	var c string
	for len(str) > 0 {
		var boundaries int
		c, str, boundaries, state = runeseg.StepString(str, state)
		fmt.Println(c, boundaries>>runeseg.ShiftWidth)
	}
	// Output: 🇩🇪 2
	//🏳️‍🌈 2
	//! 1
}

func ExampleStep_word() {
	b := []byte("Hello, world!")
	state := -1
	var (
		c          []byte
		boundaries int
	)
	for len(b) > 0 {
		c, b, boundaries, state = runeseg.Step(b, state)
		fmt.Print(string(c))
		if boundaries&runeseg.MaskWord != 0 {
			fmt.Print("|")
		}
	}
	// Output: Hello|,| |world|!|
}

func ExampleStepString_word() {
	str := "Hello, world!"
	state := -1
	var (
		c          string
		boundaries int
	)
	for len(str) > 0 {
		c, str, boundaries, state = runeseg.StepString(str, state)
		fmt.Print(c)
		if boundaries&runeseg.MaskWord != 0 {
			fmt.Print("|")
		}
	}
	// Output: Hello|,| |world|!|
}

func ExampleStep_sentence() {
	b := []byte("This is sentence 1.0. And this is sentence two.")
	state := -1
	var (
		c          []byte
		boundaries int
	)
	for len(b) > 0 {
		c, b, boundaries, state = runeseg.Step(b, state)
		fmt.Print(string(c))
		if boundaries&runeseg.MaskSentence != 0 {
			fmt.Print("|")
		}
	}
	// Output: This is sentence 1.0. |And this is sentence two.|
}

func ExampleStepString_sentence() {
	str := "This is sentence 1.0. And this is sentence two."
	state := -1
	var (
		c          string
		boundaries int
	)
	for len(str) > 0 {
		c, str, boundaries, state = runeseg.StepString(str, state)
		fmt.Print(c)
		if boundaries&runeseg.MaskSentence != 0 {
			fmt.Print("|")
		}
	}
	// Output: This is sentence 1.0. |And this is sentence two.|
}

func ExampleStep_lineBreaking() {
	b := []byte("First line.\nSecond line.")
	state := -1
	var (
		c          []byte
		boundaries int
	)
	for len(b) > 0 {
		c, b, boundaries, state = runeseg.Step(b, state)
		fmt.Print(string(c))
		switch boundaries & runeseg.MaskLine {
		case runeseg.LineCanBreak:
			fmt.Print("|")
		case runeseg.LineMustBreak:
			fmt.Print("‖")
		}
	}
	// Output: First |line.
	//‖Second |line.‖
}

func ExampleStepString_lineBreaking() {
	str := "First line.\nSecond line."
	state := -1
	var (
		c          string
		boundaries int
	)
	for len(str) > 0 {
		c, str, boundaries, state = runeseg.StepString(str, state)
		fmt.Print(c)
		switch boundaries & runeseg.MaskLine {
		case runeseg.LineCanBreak:
			fmt.Print("|")
		case runeseg.LineMustBreak:
			fmt.Print("‖")
		}
	}
	// Output: First |line.
	//‖Second |line.‖
}

func ExampleGraphemes_graphemes() {
	g := runeseg.NewGraphemes("🇩🇪🏳️‍🌈")
	for g.Next() {
		fmt.Println(g.Str())
	}
	// Output: 🇩🇪
	//🏳️‍🌈
}

func ExampleGraphemes_word() {
	g := runeseg.NewGraphemes("Hello, world!")
	for g.Next() {
		fmt.Print(g.Str())
		if g.IsWordBoundary() {
			fmt.Print("|")
		}
	}
	// Output: Hello|,| |world|!|
}

func ExampleGraphemes_sentence() {
	g := runeseg.NewGraphemes("This is sentence 1.0. And this is sentence two.")
	for g.Next() {
		fmt.Print(g.Str())
		if g.IsSentenceBoundary() {
			fmt.Print("|")
		}
	}
	// Output: This is sentence 1.0. |And this is sentence two.|
}

func ExampleGraphemes_lineBreaking() {
	g := runeseg.NewGraphemes("First line.\nSecond line.")
	for g.Next() {
		fmt.Print(g.Str())
		if g.LineBreak() == runeseg.LineCanBreak {
			fmt.Print("|")
		} else if g.LineBreak() == runeseg.LineMustBreak {
			fmt.Print("‖")
		}
	}
	if g.LineBreak() == runeseg.LineMustBreak {
		fmt.Print("\nNo clusters left. LineMustBreak")
	}
	g.Reset()
	if g.LineBreak() == runeseg.LineDontBreak {
		fmt.Print("\nIterator has been reset. LineDontBreak")
	}
	// Output: First |line.
	//‖Second |line.‖
	//No clusters left. LineMustBreak
	//Iterator has been reset. LineDontBreak
}

func ExampleStringWidth() {
	fmt.Println(runeseg.StringWidth("Hello, 世界"))
	// Output: 11
}
