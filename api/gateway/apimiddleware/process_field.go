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
			switch {
			case kind == reflect.Ptr && sliceElem.Elem().Kind() == reflect.Struct:
				for j := 0; j < v.Field(i).Len(); j++ {
					if err := processField(v.Field(i).Index(j).Interface(), processors); err != nil {
						return errors.Wrapf(err, "could not process field '%s'", t.Field(i).Name)
					}
				}
			// Process each string in string slices.
			case kind == reflect.String:
				for _, proc := range processors {
					_, hasTag := t.Field(i).Tag.Lookup(proc.tag)
					if !hasTag {
						continue
					}
					for j := 0; j < v.Field(i).Len(); j++ {
						if err := proc.f(v.Field(i).Index(j)); err != nil {
							return errors.Wrapf(err, "could not process field '%s'", t.Field(i).Name)
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
				if _, hasTag := field.Tag.Lookup(proc.tag); hasTag {
					if err := proc.f(v.Field(i)); err != nil {
						return errors.Wrapf(err, "could not process field '%s'", t.Field(i).Name)
					}
				}
			}
		}
	}
	return nil
}

func hexToBase64Processor(v reflect.Value) error {
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

func base64ToHexProcessor(v reflect.Value) error {
	if v.String() == "" {
		// Empty hex values are represented as "0x".
		v.SetString("0x")
		return nil
	}
	b, err := base64.StdEncoding.DecodeString(v.String())
	if err != nil {
		return err
	}
	v.SetString(hexutil.Encode(b))
	return nil
}

func base64ToUint256Processor(v reflect.Value) error {
	if v.String() == "" {
		return nil
	}
	littleEndian, err := base64.StdEncoding.DecodeString(v.String())
	if err != nil {
		return err
	}
	if len(littleEndian) != 32 {
		return errors.New("invalid length for Uint256")
	}

	// Integers are stored as little-endian, but
	// big.Int expects big-endian. So we need to reverse
	// the byte order before decoding.
	var bigEndian [32]byte
	for i := 0; i < len(littleEndian); i++ {
		bigEndian[i] = littleEndian[len(littleEndian)-1-i]
	}
	var uint256 big.Int
	uint256.SetBytes(bigEndian[:])
	v.SetString(uint256.String())
	return nil
}

func uint256ToBase64Processor(v reflect.Value) error {
	if v.String() == "" {
		return nil
	}
	uint256, ok := new(big.Int).SetString(v.String(), 10)
	if !ok {
		return fmt.Errorf("could not parse Uint256")
	}
	bigEndian := uint256.Bytes()
	if len(bigEndian) > 32 {
		return fmt.Errorf("number too big for Uint256")
	}

	// Integers are stored as little-endian, but
	// big.Int gives big-endian. So we need to reverse
	// the byte order before encoding.
	var littleEndian [32]byte
	for i := 0; i < len(bigEndian); i++ {
		littleEndian[i] = bigEndian[len(bigEndian)-1-i]
	}
	v.SetString(base64.StdEncoding.EncodeToString(littleEndian[:]))
	return nil
}

func enumToLowercaseProcessor(v reflect.Value) error {
	v.SetString(strings.ToLower(v.String()))
	return nil
}

func timeToUnixProcessor(v reflect.Value) error {
	t, err := time.Parse(time.RFC3339, v.String())
	if err != nil {
		return err
	}
	v.SetString(strconv.FormatUint(uint64(t.Unix()), 10))
	return nil
}
