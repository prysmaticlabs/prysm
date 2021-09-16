package bls

import (
	"bytes"
	"encoding/hex"
	"reflect"
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/testing/bls/utils"
	blst "github.com/supranational/blst/bindings/go"
)

func TestHashToG2(t *testing.T) {
	t.Run("blst", testHashToG2)
}

func testHashToG2(t *testing.T) {
	fNames, fContent := utils.RetrieveFiles("hash_to_G2", t)

	for i, file := range fNames {
		t.Run(file, func(t *testing.T) {
			test := &HashToG2Test{}
			require.NoError(t, yaml.Unmarshal(fContent[i], test))

			//t.Error(test.Input.Message)
			msgBytes := []byte(test.Input.Message)

			t.Errorf("%s", test.Output.X[2:])
			splitX := strings.Split(test.Output.X, ",")
			outputX, err := hex.DecodeString(splitX[0][2:])
			require.NoError(t, err)
			t.Errorf("%s", test.Output.Y[2:])
			splitY := strings.Split(test.Output.Y, ",")
			outputY, err := hex.DecodeString(splitY[0][2:])
			require.NoError(t, err)

			point := blst.HashToG2(msgBytes, nil)
			aff := point.ToAffine()
			returnedVal := reflect.ValueOf(aff).FieldByName("x").MethodByName("ToBEndian").Call(nil)
			if bytes.Equal(returnedVal[0].Bytes(), outputX) {
				t.Fatalf("Retrieved X value does not match output. "+
					"Expected %#v but received %#v for test case %d", outputX, returnedVal[0].Bytes(), i)
			}
			returnedVal = reflect.ValueOf(aff).FieldByName("y").MethodByName("ToBEndian").Call(nil)
			if bytes.Equal(returnedVal[0].Bytes(), outputY) {
				t.Fatalf("Retrieved Y value does not match output. "+
					"Expected %#v but received %#v for test case %d", outputY, returnedVal[0].Bytes(), i)
			}

			t.Log("Success")
		})
	}
}
