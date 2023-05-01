package cache

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestRegistrationCache(t *testing.T) {
	hook := logTest.NewGlobal()
	pubkey, err := hexutil.Decode("0x93247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a")
	require.NoError(t, err)
	validatorIndex := primitives.ValidatorIndex(1)
	cache := NewRegistrationCache()
	m := make(map[primitives.ValidatorIndex]*ethpb.ValidatorRegistrationV1)

	m[validatorIndex] = &ethpb.ValidatorRegistrationV1{
		FeeRecipient: []byte{},
		GasLimit:     100,
		Timestamp:    uint64(time.Now().Unix()),
		Pubkey:       pubkey,
	}
	cache.UpdateIndexToRegisteredMap(context.Background(), m)
	reg, err := cache.RegistrationByIndex(validatorIndex)
	require.NoError(t, err)
	require.Equal(t, string(reg.Pubkey), string(pubkey))
	t.Run("Registration expired", func(t *testing.T) {
		validatorIndex2 := primitives.ValidatorIndex(2)
		overExpirationPadTime := time.Second * time.Duration(params.BeaconConfig().SecondsPerSlot*uint64(params.BeaconConfig().SlotsPerEpoch)*4) // 4 epochs
		m[validatorIndex2] = &ethpb.ValidatorRegistrationV1{
			FeeRecipient: []byte{},
			GasLimit:     100,
			Timestamp:    uint64(time.Now().Add(-1 * overExpirationPadTime).Unix()),
			Pubkey:       pubkey,
		}
		cache.UpdateIndexToRegisteredMap(context.Background(), m)
		_, err := cache.RegistrationByIndex(validatorIndex2)
		require.ErrorContains(t, "no validator registered", err)
		require.LogsContain(t, hook, "expired")
	})
	t.Run("Registration close to expiration still passes", func(t *testing.T) {
		pubkey, err := hexutil.Decode("0x88247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a")
		require.NoError(t, err)
		validatorIndex2 := primitives.ValidatorIndex(2)
		overExpirationPadTime := time.Second * time.Duration((params.BeaconConfig().SecondsPerSlot*uint64(params.BeaconConfig().SlotsPerEpoch)*3)-5) // 3 epochs - 5 seconds
		m[validatorIndex2] = &ethpb.ValidatorRegistrationV1{
			FeeRecipient: []byte{},
			GasLimit:     100,
			Timestamp:    uint64(time.Now().Add(-1 * overExpirationPadTime).Unix()),
			Pubkey:       pubkey,
		}
		cache.UpdateIndexToRegisteredMap(context.Background(), m)
		reg, err := cache.RegistrationByIndex(validatorIndex2)
		require.NoError(t, err)
		require.Equal(t, string(reg.Pubkey), string(pubkey))
	})
}

func Test_RegistrationTimeStampExpired(t *testing.T) {
	// expiration set at 3 epochs
	t.Run("expired registration", func(t *testing.T) {
		overExpirationPadTime := time.Second * time.Duration(params.BeaconConfig().SecondsPerSlot*uint64(params.BeaconConfig().SlotsPerEpoch)*4) // 4 epochs
		ts := uint64(time.Now().Add(-1 * overExpirationPadTime).Unix())
		isExpired, err := RegistrationTimeStampExpired(ts)
		require.NoError(t, err)
		require.Equal(t, true, isExpired)
	})
	t.Run("is not expired registration", func(t *testing.T) {
		overExpirationPadTime := time.Second * time.Duration((params.BeaconConfig().SecondsPerSlot*uint64(params.BeaconConfig().SlotsPerEpoch)*3)-5) // 3 epochs -5 seconds
		ts := uint64(time.Now().Add(-1 * overExpirationPadTime).Unix())
		isExpired, err := RegistrationTimeStampExpired(ts)
		require.NoError(t, err)
		require.Equal(t, false, isExpired)
	})
}
