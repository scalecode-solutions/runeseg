//go:build generate

// This program generates a property file for Indic_Conjunct_Break (InCB) from
// the Unicode Character Database DerivedCoreProperties.txt file.
//
//go:generate go run gen_incb.go

package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	incbURL = `https://www.unicode.org/Public/17.0.0/ucd/DerivedCoreProperties.txt`
)

// The regular expression for a line containing an InCB property.
var incbPattern = regexp.MustCompile(`^([0-9A-F]{4,6})(\.\.([0-9A-F]{4,6}))?\s*;\s*InCB\s*;\s*(\w+)\s*#\s*(.+)$`)

func main() {
	log.SetPrefix("gen_incb: ")
	log.SetFlags(0)

	src, err := parse()
	if err != nil {
		log.Fatal(err)
	}

	// Format the Go code.
	formatted, err := format.Source([]byte(src))
	if err != nil {
		log.Fatal("gofmt:", err)
	}

	// Save it to the target file.
	log.Print("Writing to incbproperties.go")
	if err := os.WriteFile("incbproperties.go", formatted, 0644); err != nil {
		log.Fatal(err)
	}
}

func parse() (string, error) {
	log.Printf("Parsing %s", incbURL)
	res, err := http.Get(incbURL)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	// Temporary buffer to hold properties.
	var properties [][4]string

	// Parse the file.
	scanner := bufio.NewScanner(res.Body)
	num := 0
	for scanner.Scan() {
		num++
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines.
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		// Check if this is an InCB line.
		if !strings.Contains(line, "; InCB;") {
			continue
		}

		// Parse the InCB property.
		from, to, value, comment, err := parseInCB(line)
		if err != nil {
			return "", fmt.Errorf("line %d: %v", num, err)
		}
		properties = append(properties, [4]string{from, to, value, comment})
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	// Avoid overflow during binary search.
	if len(properties) >= 1<<31 {
		return "", errors.New("too many properties")
	}

	// Sort properties.
	sort.Slice(properties, func(i, j int) bool {
		left, _ := strconv.ParseUint(properties[i][0], 16, 64)
		right, _ := strconv.ParseUint(properties[j][0], 16, 64)
		return left < right
	})

	// Header.
	var buf bytes.Buffer
	buf.WriteString(`// Code generated via go generate from gen_incb.go. DO NOT EDIT.

package runeseg

// incbCodePoints are taken from
// ` + incbURL + `
// on ` + time.Now().Format("January 2, 2006") + `. See https://www.unicode.org/license.html for the Unicode
// license agreement.
var incbCodePoints = [][3]int{
`)

	// Properties.
	for _, prop := range properties {
		value := translateInCBValue(prop[2])
		fmt.Fprintf(&buf, "\t{0x%s, 0x%s, %s}, // %s\n", prop[0], prop[1], value, prop[3])
	}

	// Tail.
	buf.WriteString("}\n")

	return buf.String(), nil
}

// parseInCB parses a line containing an InCB property.
func parseInCB(line string) (from, to, value, comment string, err error) {
	fields := incbPattern.FindStringSubmatch(line)
	if fields == nil {
		err = errors.New("no InCB property found")
		return
	}
	from = fields[1]
	to = fields[3]
	if to == "" {
		to = from
	}
	value = fields[4]
	comment = fields[5]
	return
}

// translateInCBValue translates an InCB value to a Go constant.
func translateInCBValue(value string) string {
	switch value {
	case "Linker":
		return "prInCBLinker"
	case "Consonant":
		return "prInCBConsonant"
	case "Extend":
		return "prInCBExtend"
	default:
		return "prInCBNone"
	}
}
