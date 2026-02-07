// Package frame implements Plan 9 frames of editable text.
//
// A Frame manages the display of editable text in a single font
// on a raster display image, as used in sam(1), acme(1), and rio(1).
// Frames may hold any character except NUL (0). Long lines are
// folded and tabs are at fixed intervals.
//
// The text within frames is not directly addressable; frames are
// designed to work alongside another structure that holds the text.
// The typical application is to display a section of a longer
// document such as a text file or terminal session. The program
// keeps its own copy of the text and passes changes to the frame
// via Insert and Delete.
//
// Ported from 9front /sys/src/libframe.
package frame

import (
	"github.com/elizafairlady/go-libui/draw"
)

// Color indices for Frame.Cols.
const (
	ColBack  = iota // background
	ColHigh         // highlight (selection) background
	ColBord         // border
	ColText         // normal text
	ColHText        // highlighted text
	NCol            // number of color slots
)

// FRTICKW is the width of the typing cursor tick in pixels.
const FRTICKW = 3

// frbox is an internal box within a frame.
//
// If nrune >= 0 the box holds nrune runes of text stored in ptr
// (UTF-8 encoded, no trailing NUL). wid is the pixel width of the text.
//
// If nrune < 0 the box represents a single break character (tab or
// newline) stored in bc. minwid is the minimum display width; for
// tabs the actual width is computed based on position and tab stops.
type frbox struct {
	wid    int    // width in pixels
	nrune  int    // number of runes, or <0 for break char
	ptr    []byte // UTF-8 text (when nrune >= 0)
	bc     rune   // break character (when nrune < 0)
	minwid int    // minimum width (when nrune < 0)
}

// nRune returns the logical rune count for the box.
// Break-character boxes count as 1.
func (b *frbox) nRune() int {
	if b.nrune < 0 {
		return 1
	}
	return b.nrune
}

// nbyte returns the byte length of the box text.
func (b *frbox) nbyte() int {
	return len(b.ptr)
}

// Frame holds the state of one frame of editable text.
type Frame struct {
	Font    *draw.Font        // font used for characters
	Display *draw.Display     // display on which frame appears
	B       *draw.Image       // image on which frame is drawn
	Cols    [NCol]*draw.Image // text and background colors

	R      draw.Rectangle // rectangle in which text appears
	Entire draw.Rectangle // full frame rectangle

	Scroll func(f *Frame, dl int) // scroll callback provided by application

	box    []frbox // internal box array
	P0, P1 uint32  // selection range (character positions)

	nbox   int // number of active boxes
	nalloc int // allocated box slots

	Maxtab       int    // maximum tab width in pixels
	Nchars       uint32 // number of runes in the frame
	Nlines       int    // number of lines with text
	Maxlines     int    // total lines that fit in the frame
	Lastlinefull int    // whether the last line fills the frame
	Modified     int    // changed since last Select()

	tick     *draw.Image // typing cursor image
	tickback *draw.Image // saved image under cursor
	Ticked   int         // is cursor visible?
}
