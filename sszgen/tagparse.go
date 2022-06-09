package sszgen

import (
	"fmt"
	"strconv"
	"strings"
	"text/scanner"
)

type tokenState int

const (
	tsBegin tokenState = iota
	tsLabel
	tsValue
	tsCloseTick
)

type TagParser struct {
	sc scanner.Scanner
	buffer string
}

func (tp *TagParser) Init(tag string) {
	sr := strings.NewReader(tag)
	tp.sc = scanner.Scanner{}
	tp.sc.Init(sr)
	tp.sc.Filename = "tag"
	tp.sc.Mode ^= scanner.ScanRawStrings
}

func (tp TagParser) GetSSZTags() map[string]string {
	var labelStr string
	var state tokenState
	tags := make(map[string]string)
	for tok := tp.sc.Scan(); tok != scanner.EOF; tok = tp.sc.Scan() {
		if state == tsCloseTick {
			panic("undefined beyhavior when scanning beyond the end of the tag")
		}
		txt := tp.sc.TokenText()
		switch txt {
		case "`":
			if state == tsLabel {
				state = tsCloseTick
				continue
			}
			if state == tsBegin {
				state = tsLabel
				continue
			}
		case ":":
			if state == tsLabel {
				state = tsValue
				continue
			}
		case "\"":
			continue
		default:
			if state == tsValue {
				tags[labelStr] = trimQuotes(string(txt))
				state = tsLabel
				labelStr = ""
				continue
			}
			if state == tsLabel {
				labelStr += string(txt)
				continue
			}
		}
	}
	return tags
}

// cannot compare untyped nil to typed nil
// this value gives us a nil with type of *int
// to compare to ssz-size = '?' values
var nilInt *int

func extractSSZDimensions(tag string) ([]*SSZDimension, error) {
	tp := &TagParser{}
	tp.Init(tag)
	tags := tp.GetSSZTags()
	sszSizes, sizeDefined := tags["ssz-size"]
	sszMax, maxDefined:= tags["ssz-max"]
	if !sizeDefined {
		if !maxDefined {
			return nil, fmt.Errorf("No ssz-size or ssz-max tags found for element.")
		}
		max, err := strconv.Atoi(sszMax)
		if err != nil {
			return nil, err
		}
		return []*SSZDimension{{ListLength: &max}}, nil
	}
	dims := make([]*SSZDimension, 0)
	for _, sz := range strings.Split(sszSizes, ",") {
		if sz == "?" {
			if sszMax != "" {
				max, err := strconv.Atoi(sszMax)
				if err != nil {
					return nil, err
				}
				dims = append(dims, &SSZDimension{ListLength: &max})
				sszMax = ""
			} else {
				return nil, fmt.Errorf("More than one wildcard in ssz-size, or ssz-max undefined in tag %s", tag)
			}
		} else {
			vsize, err := strconv.Atoi(sz)
			if err != nil {
				return nil, err
			}
			dims = append(dims, &SSZDimension{VectorLength: &vsize})
		}
	}

	return dims, nil
}

type SSZDimension struct {
	VectorLength *int
	ListLength *int
}

func (dim *SSZDimension) IsVector() bool {
	return dim.VectorLength != nilInt
}

func (dim *SSZDimension) IsList() bool {
	return dim.ListLength != nilInt
}

func (dim *SSZDimension) ListLen() int {
	return *dim.ListLength
}

func (dim *SSZDimension) VectorLen() int {
	return *dim.VectorLength
}

type SSZListBounds struct {
	SSZSize []*int
	SSZMax *int
}

func trimQuotes(s string) string {
	if len(s) > 0 && s[0] == '"' {
		s = s[1:]
	}
	if len(s) > 0 && s[len(s)-1] == '"' {
		s = s[:len(s)-1]
	}
	return s
}
