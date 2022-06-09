package ssz_static

import (
       "bytes"
       "encoding/hex"
       "os"
       "testing"

       "github.com/golang/snappy"
       ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"

       //"github.com/prysmaticlabs/prysm/shared/testutil"
       "github.com/prysmaticlabs/prysm/shared/testutil/require"
       "github.com/prysmaticlabs/prysm/spectest/utils"
)

func TestFailingHTR(t *testing.T) {
       fh, err := os.Open("testdata/serialized.ssz_snappy")
       require.NoError(t, err)
       defer fh.Close()
       buf := bytes.NewBuffer(nil)
       _, err = buf.ReadFrom(fh)
       sszBytes, err := snappy.Decode(nil, buf.Bytes())
       require.NoError(t, err)
       o := &ethpb.AggregateAttestationAndProof{}
       err = o.XXUnmarshalSSZ(sszBytes)
       err = o.UnmarshalSSZ(sszBytes)
       require.NoError(t, err, "Could not unmarshall serialized SSZ")

       fh, err = os.Open("testdata/roots.yaml")
       require.NoError(t, err)
       defer fh.Close()
       buf = bytes.NewBuffer(nil)
       buf.ReadFrom(fh)
       rootsYaml := &SSZRoots{}
       require.NoError(t, utils.UnmarshalYaml(buf.Bytes(), rootsYaml))

       root, err := o.HashTreeRoot()
       require.NoError(t, err)

       rootBytes, err := hex.DecodeString(rootsYaml.Root[2:])
       require.NoError(t, err)
       require.DeepEqual(t, rootBytes, root[:], "Did not receive expected hash tree root")
}
