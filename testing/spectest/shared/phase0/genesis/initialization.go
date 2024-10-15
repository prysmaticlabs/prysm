package genesis

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/snappy"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/transition"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/spectest/utils"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
	"github.com/wealdtech/go-bytesutil"
)

type Config struct {
	DepositsCount int `json:"deposits_count"`
}

type Eth1DataConfig struct {
	Eth1BlockHash string `json:"eth1_block_hash"`
}

func RunInitializationTest(t *testing.T, config string) {
	require.NoError(t, utils.SetConfig(t, config))

	testFolders, testsFolderPath := utils.TestFolders(t, config, "phase0", "genesis/initialization/pyspec_tests")
	if len(testFolders) == 0 {
		t.Fatalf("No test folders found for %s/%s/%s", config, "phase0", "genesis/initialization/pyspec_tests")
	}
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			metaFile, err := util.BazelFileBytes(testsFolderPath, folder.Name(), "meta.yaml")
			require.NoError(t, err)
			cfg := &Config{}
			require.NoError(t, utils.UnmarshalYaml(metaFile, cfg), "Failed to unmarshal metadata")
			deposits := make([]*ethpb.Deposit, cfg.DepositsCount)
			for i := 0; i < cfg.DepositsCount; i++ {
				depositFile, err := util.BazelFileBytes(testsFolderPath, folder.Name(), fmt.Sprintf("deposits_%d.ssz_snappy", i))
				require.NoError(t, err)
				depositSSZ, err := snappy.Decode(nil /* dst */, depositFile)
				require.NoError(t, err)
				d := &ethpb.Deposit{}
				require.NoError(t, d.UnmarshalSSZ(depositSSZ))
				deposits[i] = d
			}
			eth1DataFile, err := util.BazelFileBytes(testsFolderPath, folder.Name(), "eth1.yaml")
			require.NoError(t, err)
			eth1DataCfg := &Eth1DataConfig{}
			require.NoError(t, utils.UnmarshalYaml(eth1DataFile, eth1DataCfg))
			blockHash, err := bytesutil.FromHexString(eth1DataCfg.Eth1BlockHash)
			require.NoError(t, err)
			eth1Data := &ethpb.Eth1Data{
				DepositCount: uint64(cfg.DepositsCount),
				BlockHash:    blockHash,
			}
			st, err := transition.GenesisBeaconState(context.Background(), deposits, 0, eth1Data)
			require.NoError(t, err)
			expectedStateFile, err := util.BazelFileBytes(testsFolderPath, folder.Name(), "state.ssz_snappy")
			require.NoError(t, err)
			expectedStateSSZ, err := snappy.Decode(nil /* dst */, expectedStateFile)
			require.NoError(t, err)
			expectedSt := &ethpb.BeaconState{}
			require.NoError(t, expectedSt.UnmarshalSSZ(expectedStateSSZ))
			expectedRoot, err := expectedSt.HashTreeRoot()
			require.NoError(t, err)
			root, err := st.HashTreeRoot(context.Background())
			require.NoError(t, err)
			assert.DeepEqual(t, expectedRoot, root)
		})
	}
}
