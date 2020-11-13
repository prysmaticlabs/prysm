package interchangeformat

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/rand"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
)

var numValidators = 2

func TestStore_ImportInterchangeData(t *testing.T) {
	pk := createKeys(t)
	db := kv.setupDB(t, pk)
	att, pro := createProtectionData(t)
	ctx := context.Background()
	var pif PlainDataInterchangeFormat
	pif.Metadata.GenesisValidatorsRoot = hex.EncodeToString(bytesutil.PadTo([]byte{32}, 32))
	pif.Metadata.InterchangeFormatVersion = "5"
	for i := 0; i < numValidators; i++ {
		data := interchange.Data{
			Pubkey: hex.EncodeToString(pk[i][:]),
		}

		lew, err := att[i].GetLatestEpochWritten(ctx)
		require.NoError(t, err)
		for target := uint64(0); target <= lew; target++ {
			hd, err := att[i].GetTargetData(ctx, target)
			require.NoError(t, err)
			data.SignedAttestations = append(data.SignedAttestations, interchange.SignedAttestations{
				TargetEpoch: strconv.FormatUint(target, 10),
				SourceEpoch: strconv.FormatUint(hd.Source, 10),
				SigningRoot: hex.EncodeToString(hd.SigningRoot)},
			)
			sr := pro[i][target]
			data.SignedBlocks = append(data.SignedBlocks, interchange.SignedBlocks{
				Slot:        strconv.FormatUint(target, 10),
				SigningRoot: hex.EncodeToString(sr[:])},
			)

		}
		pif.Data = append(pif.Data, data)
	}
	blob, err := json.Marshal(pif)
	require.NoError(t, err)
	err = db.ImportInterchangeData(ctx, blob)
	require.NoError(t, err)
	resAtt, err := db.AttestationHistoryForPubKeysV2(ctx, pk)
	require.NoError(t, err)
	for i := 0; i < numValidators; i++ {
		require.DeepEqual(t, att[i], resAtt[pk[i]], "Imported attestations are different then the generated ones")
		for slot, root := range pro[i] {
			sr, err := db.ProposalHistoryForSlot(ctx, pk[i][:], slot)
			require.NoError(t, err)
			require.DeepEqual(t, root[:], sr, "Imported proposals are different then the generated ones")
		}
	}

}

func createKeys(t *testing.T) [][48]byte {
	pubKeys := make([][48]byte, numValidators)
	for i := 0; i < numValidators; i++ {
		randKey, err := bls.RandKey()
		require.NoError(t, err)
		copy(pubKeys[i][:], randKey.PublicKey().Marshal())
	}
	return pubKeys
}

func createProtectionData(t *testing.T) ([]*kv.EncHistoryData, []map[uint64][32]byte) {
	attData := make([]*kv.EncHistoryData, numValidators)
	proposalData := make([]map[uint64][32]byte, numValidators)
	gen := rand.NewGenerator()
	ctx := context.Background()
	for v := 0; v < numValidators; v++ {
		var err error
		proposalData[v] = make(map[uint64][32]byte)
		latestTarget := gen.Intn(int(params.BeaconConfig().WeakSubjectivityPeriod) / 100)
		hd := kv.NewAttestationHistoryArray(uint64(latestTarget))
		for i := 1; i < latestTarget; i += 2 {
			historyData := &kv.HistoryData{Source: uint64(gen.Intn(100000)), SigningRoot: bytesutil.PadTo([]byte{byte(i)}, 32)}
			hd, err = hd.SetTargetData(ctx, uint64(i), historyData)
			require.NoError(t, err)
			proposalData[v][uint64(i)] = [32]byte{byte(i)}
		}
		hd, err = hd.SetLatestEpochWritten(ctx, uint64(latestTarget))
		attData[v] = hd
	}
	return attData, proposalData
}
