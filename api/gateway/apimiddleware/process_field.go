package apimiddleware

import (
	"encoding/base64"
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/wealdtech/go-bytesutil"
)

// processField calls each processor function on any field that has the matching tag set.
// It is a recursive function.
func processField(s interface{}, processors []fieldProcessor) error {
	kind := reflect.TypeOf(s).Kind()
	if kind != reflect.Ptr && kind != reflect.Slice && kind != reflect.Array {
		return fmt.Errorf("processing fields of kind '%v' is unsupported", kind)
	}

	t := reflect.TypeOf(s).Elem()
	v := reflect.Indirect(reflect.ValueOf(s))

	for i := 0; i < t.NumField(); i++ {
		switch v.Field(i).Kind() {
		case reflect.Slice:
			sliceElem := t.Field(i).Type.Elem()
			kind := sliceElem.Kind()
			// Recursively process slices to struct pointers.
			if kind == reflect.Ptr && sliceElem.Elem().Kind() == reflect.Struct {
				for j := 0; j < v.Field(i).Len(); j++ {
					if err := processField(v.Field(i).Index(j).Interface(), processors); err != nil {
						return errors.Wrapf(err, "could not process field '%s'", t.Field(i).Name)
					}
				}
			}
			// Process each string in string slices.
			if kind == reflect.String {
				for _, proc := range processors {
					tag, hasTag := t.Field(i).Tag.Lookup(proc.tag)
					if hasTag {
						for j := 0; j < v.Field(i).Len(); j++ {
							if err := proc.f(tag, v.Field(i).Index(j)); err != nil {
								return errors.Wrapf(err, "could not process field '%s'", t.Field(i).Name)
							}
						}
					}
				}

			}
		// Recursively process struct pointers.
		case reflect.Ptr:
			if v.Field(i).Elem().Kind() == reflect.Struct {
				if err := processField(v.Field(i).Interface(), processors); err != nil {
					return errors.Wrapf(err, "could not process field '%s'", t.Field(i).Name)
				}
			}
		default:
			field := t.Field(i)
			for _, proc := range processors {
				if tag, hasTag := field.Tag.Lookup(proc.tag); hasTag {
					if err := proc.f(tag, v.Field(i)); err != nil {
						return errors.Wrapf(err, "could not process field '%s'", t.Field(i).Name)
					}
				}
			}
		}
	}
	return nil
}

func hexToBase64Processor(tag string, v reflect.Value) error {
	if v.String() == "0x" {
		v.SetString("")
		return nil
	}
	b, err := bytesutil.FromHexString(v.String())
	if err != nil {
		return err
	}
	v.SetString(base64.StdEncoding.EncodeToString(b))
	return nil
}

func base64ToHexProcessor(tag string, v reflect.Value) error {
	var alwaysPrefix bool
	for _, option := range strings.Split(tag, ",") {
		switch option {
		case "alwaysprefix":
			alwaysPrefix = true
		}
	}

	if v.String() == "" {
		if alwaysPrefix {
			v.SetString("0x")
		}
		return nil
	}
	b, err := base64.StdEncoding.DecodeString(v.String())
	if err != nil {
		return err
	}
	v.SetString(hexutil.Encode(b))
	return nil
}

func base64ToBigIntProcessor(tag string, v reflect.Value) error {
	if v.String() == "" {
		return nil
	}
	b, err := base64.StdEncoding.DecodeString(v.String())
	if err != nil {
		return err
	}

	// Integers are stored as little-endian, but
	// big.Int expects big-endian. So we need to reverse
	// the byte order.
	bigEndianBytes := make([]byte, len(b))
	for i := 0; i < len(b); i++ {
		bigEndianBytes[i] = b[len(b)-1-i]
	}
	var n big.Int
	n.SetBytes(bigEndianBytes)
	v.SetString(n.String())
	return nil
}

func bigIntToBase64Processor(tag string, v reflect.Value) error {
	if v.String() == "" {
		return nil
	}
	n, ok := new(big.Int).SetString(v.String(), 10)
	if !ok {
		return fmt.Errorf("could not parse big int '%s'", v.String())
	}
	b := n.Bytes()
	littleEndianBytes := make([]byte, len(b))
	for i := 0; i < len(b); i++ {
		littleEndianBytes[i] = b[len(b)-1-i]
	}
	v.SetString(base64.StdEncoding.EncodeToString(littleEndianBytes))
	return nil
}

func enumToLowercaseProcessor(tag string, v reflect.Value) error {
	v.SetString(strings.ToLower(v.String()))
	return nil
}

func timeToUnixProcessor(tag string, v reflect.Value) error {
	t, err := time.Parse(time.RFC3339, v.String())
	if err != nil {
		return err
	}
	v.SetString(strconv.FormatUint(uint64(t.Unix()), 10))
	return nil
}
