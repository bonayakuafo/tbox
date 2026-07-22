package node

import (
	"fmt"
	"github.com/mattn/go-runewidth"
	"strings"
)

// RepeatChar returns a string of num copies of ch.
func RepeatChar(ch byte, num int) string {
	return strings.Repeat(string(ch), num)
}

// MaxWidth returns the maximum display width among the given strings.
func MaxWidth(str ...string) int {
	max := 0
	for _, s := range str {
		width := runewidth.StringWidth(s)
		if width > max {
			max = width
		}
	}
	return max
}

// ShowTopBottomSepLine prints the strings surrounded by top and bottom separator lines.
func ShowTopBottomSepLine(ch byte, str ...string) {
	width := MaxWidth(str...)
	fmt.Println(RepeatChar(ch, width))
	fmt.Println(strings.Join(str, "\n"))
	fmt.Println(RepeatChar(ch, width))
}
