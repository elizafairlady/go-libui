package draw

import (
	"bytes"
	"strings"
	"testing"
)

// TestCompressionConstants tests the compression constants match the C defines.
func TestCompressionConstants(t *testing.T) {
	if NMATCH != 3 {
		t.Errorf("NMATCH = %d, want 3", NMATCH)
	}
	if NRUN != 15 {
		t.Errorf("NRUN = %d, want 15", NRUN)
	}
	if NDUMP != 128 {
		t.Errorf("NDUMP = %d, want 128", NDUMP)
	}
	if NMEM != 1024 {
		t.Errorf("NMEM = %d, want 1024", NMEM)
	}
	if NCBLOCK != 6000 {
		t.Errorf("NCBLOCK = %d, want 6000", NCBLOCK)
	}
}

// TestCompBlockSize tests the compressed block size calculation.
func TestCompBlockSize(t *testing.T) {
	r := Rect(0, 0, 100, 50)
	depth := 8
	bs := CompBlockSize(r, depth)
	bpl := bytesPerLine(r, depth)
	expected := bpl * 50
	if bs != expected {
		t.Errorf("CompBlockSize = %d, want %d", bs, expected)
	}
}

// TestWriteImageNil tests nil image safety.
func TestWriteImageNil(t *testing.T) {
	var img *Image
	var buf bytes.Buffer
	err := img.WriteImageWriter(&buf)
	if err == nil {
		t.Error("WriteImageWriter on nil should fail")
	}
}

// TestCwriteImageNil tests nil image safety.
func TestCwriteImageNil(t *testing.T) {
	var img *Image
	var buf bytes.Buffer
	err := img.CwriteImageWriter(&buf)
	if err == nil {
		t.Error("CwriteImageWriter on nil should fail")
	}
}

// TestWriteImageHeader tests writing just the header.
func TestWriteImageHeader(t *testing.T) {
	var buf bytes.Buffer
	r := Rect(0, 0, 640, 480)
	pix := XRGB32
	err := WriteImageHeader(&buf, pix, r)
	if err != nil {
		t.Fatal(err)
	}

	// Header should be 5*12 = 60 bytes
	if buf.Len() != 60 {
		t.Errorf("header length = %d, want 60", buf.Len())
	}

	// Parse it back
	hdr := buf.String()
	chanstr := strings.TrimSpace(hdr[0:11])
	if chanstr == "" {
		t.Error("empty channel string")
	}

	minx := atoi(hdr[12:23])
	miny := atoi(hdr[24:35])
	maxx := atoi(hdr[36:47])
	maxy := atoi(hdr[48:59])

	if minx != 0 || miny != 0 || maxx != 640 || maxy != 480 {
		t.Errorf("rect = (%d,%d)-(%d,%d), want (0,0)-(640,480)", minx, miny, maxx, maxy)
	}
}

// TestWriteImageHeaderRoundtrip tests header write-then-read roundtrip.
func TestWriteImageHeaderRoundtrip(t *testing.T) {
	tests := []struct {
		pix Pix
		r   Rectangle
	}{
		{GREY8, Rect(0, 0, 100, 100)},
		{XRGB32, Rect(10, 20, 30, 40)},
		{RGB24, Rect(-5, -5, 5, 5)},
	}

	for _, tt := range tests {
		var buf bytes.Buffer
		err := WriteImageHeader(&buf, tt.pix, tt.r)
		if err != nil {
			t.Errorf("WriteImageHeader(%v, %v): %v", tt.pix, tt.r, err)
			continue
		}

		hdr := buf.String()
		chanstr := strings.TrimSpace(hdr[0:11])
		gotPix := strtochan(chanstr)
		if gotPix != tt.pix {
			t.Errorf("pix roundtrip: got %v, want %v", gotPix, tt.pix)
		}

		minx := atoi(hdr[12:23])
		miny := atoi(hdr[24:35])
		maxx := atoi(hdr[36:47])
		maxy := atoi(hdr[48:59])
		gotR := Rect(minx, miny, maxx, maxy)
		if !gotR.Eq(tt.r) {
			t.Errorf("rect roundtrip: got %v, want %v", gotR, tt.r)
		}
	}
}

// TestWriteImageWriterNoDisplay tests writing an image with no display (header only).
func TestWriteImageWriterNoDisplay(t *testing.T) {
	img := &Image{
		Pix:   XRGB32,
		Depth: 32,
		R:     Rect(0, 0, 10, 10),
		Clipr: Rect(0, 0, 10, 10),
	}
	var buf bytes.Buffer
	err := img.WriteImageWriter(&buf)
	if err != nil {
		t.Fatal(err)
	}
	// Should have written the header (60 bytes) but no data since Display is nil
	if buf.Len() != 60 {
		t.Errorf("len = %d, want 60 (header only)", buf.Len())
	}
}

// TestCwriteImageWriterNoDisplay tests compressed write with no display.
func TestCwriteImageWriterNoDisplay(t *testing.T) {
	img := &Image{
		Pix:   GREY8,
		Depth: 8,
		R:     Rect(0, 0, 10, 10),
		Clipr: Rect(0, 0, 10, 10),
	}
	var buf bytes.Buffer
	err := img.CwriteImageWriter(&buf)
	if err != nil {
		t.Fatal(err)
	}
	// Should have "compressed\n" (11 bytes) + header (60 bytes)
	if buf.Len() != 71 {
		t.Errorf("len = %d, want 71", buf.Len())
	}
	if !strings.HasPrefix(buf.String(), "compressed\n") {
		t.Error("missing compressed marker")
	}
}
