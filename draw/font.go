package draw

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// OpenFont opens a font file and returns a Font.
func (d *Display) OpenFont(name string) (*Font, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return d.buildFont(f, name)
}

// buildFont parses a font file.
func (d *Display) buildFont(r *os.File, name string) (*Font, error) {
	scanner := bufio.NewScanner(r)

	// First line: height ascent
	if !scanner.Scan() {
		return nil, fmt.Errorf("empty font file")
	}
	line := scanner.Text()
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return nil, fmt.Errorf("bad font header: %s", line)
	}
	height, err := strconv.Atoi(fields[0])
	if err != nil {
		return nil, fmt.Errorf("bad height: %v", err)
	}
	ascent, err := strconv.Atoi(fields[1])
	if err != nil {
		return nil, fmt.Errorf("bad ascent: %v", err)
	}

	font := &Font{
		Display: d,
		Name:    name,
		Height:  height,
		Ascent:  ascent,
	}

	// Read subfont specifications
	// Format: min max offset filename
	// or: min max filename (offset=0)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		min, err := strconv.ParseInt(fields[0], 0, 32)
		if err != nil {
			continue
		}
		max, err := strconv.ParseInt(fields[1], 0, 32)
		if err != nil {
			continue
		}

		var offset int64
		var filename string
		if len(fields) >= 4 {
			offset, _ = strconv.ParseInt(fields[2], 0, 32)
			filename = fields[3]
		} else {
			filename = fields[2]
		}

		cf := &Cachefont{
			Min:    int(min),
			Max:    int(max) + 1, // max is inclusive in font file
			Offset: int(offset),
			Name:   filename,
		}
		font.sub = append(font.sub, cf)
	}

	return font, nil
}

// Free releases the resources associated with a font.
func (f *Font) Free() {
	if f == nil {
		return
	}
	// Free cached subfonts
	for i := range f.subf {
		if f.subf[i].f != nil {
			f.subf[i].f.Free()
		}
	}
	// Free cache image
	if f.cacheimage != nil {
		f.cacheimage.Free()
	}
	f.cache = nil
	f.subf = nil
	f.sub = nil
}

// BuildFont builds a font from raw data.
func (d *Display) BuildFont(data []byte, name string) (*Font, error) {
	lines := strings.Split(string(data), "\n")
	if len(lines) < 1 {
		return nil, fmt.Errorf("empty font data")
	}

	fields := strings.Fields(lines[0])
	if len(fields) < 2 {
		return nil, fmt.Errorf("bad font header")
	}
	height, _ := strconv.Atoi(fields[0])
	ascent, _ := strconv.Atoi(fields[1])

	font := &Font{
		Display: d,
		Name:    name,
		Height:  height,
		Ascent:  ascent,
	}

	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		min, _ := strconv.ParseInt(fields[0], 0, 32)
		max, _ := strconv.ParseInt(fields[1], 0, 32)
		var offset int64
		var filename string
		if len(fields) >= 4 {
			offset, _ = strconv.ParseInt(fields[2], 0, 32)
			filename = fields[3]
		} else {
			filename = fields[2]
		}
		cf := &Cachefont{
			Min:    int(min),
			Max:    int(max) + 1,
			Offset: int(offset),
			Name:   filename,
		}
		font.sub = append(font.sub, cf)
	}

	return font, nil
}
