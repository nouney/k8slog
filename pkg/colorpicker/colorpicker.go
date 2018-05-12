package colorpicker

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/fatih/color"
)

type key struct {
	fg    color.Attribute
	style color.Attribute
}

type ColorPicker struct {
	fgColors []color.Attribute
	styles   []color.Attribute
	cache    map[key]struct{}
	colors   map[string]*color.Color
	src      rand.Source
}

func New() *ColorPicker {
	return &ColorPicker{
		fgColors: []color.Attribute{
			color.FgRed,
			color.FgGreen,
			color.FgYellow,
			color.FgBlue,
			color.FgMagenta,
			color.FgCyan,
			color.FgWhite,
			color.FgHiRed,
			color.FgHiGreen,
			color.FgHiYellow,
			color.FgHiBlue,
			color.FgHiMagenta,
			color.FgHiCyan,
			color.FgHiWhite,
		},
		styles: []color.Attribute{
			color.Bold,
			color.Italic,
			color.Underline,
			color.ReverseVideo,
		},
		cache:  make(map[key]struct{}),
		colors: make(map[string]*color.Color),
		src:    rand.NewSource(time.Now().UnixNano()),
	}
}

func (cp *ColorPicker) Pick(id string) *color.Color {
	clr := cp.colors[id]
	if clr != nil {
		return clr
	}
	var fg, style color.Attribute
	for {
		fg = cp.randomFgColor()
		style = cp.randomStyle()
		_, ok := cp.cache[key{fg, style}]
		if !ok {
			break
		}
	}
	clr = color.New(fg, style)
	cp.colors[id] = clr
	cp.cache[key{fg, style}] = struct{}{}
	return clr
}

func (cp ColorPicker) Palette() {
	palette(cp.fgColors, cp.styles)
}

func (cp ColorPicker) randomFgColor() color.Attribute {
	i := cp.src.Int63() % int64(len(cp.fgColors))
	return cp.fgColors[i]
}

func (cp ColorPicker) randomStyle() color.Attribute {
	i := cp.src.Int63() % int64(len(cp.styles))
	return cp.styles[i]
}

func palette(fgColors, styles []color.Attribute) {
	for _, style := range styles {
		for _, fgColor := range fgColors {
			color := color.New(fgColor, style)
			color.Print(" ", style, fgColor, " ")
		}
		fmt.Println()
	}
}
