package testing

import (
	"testing"

	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func VerifyBeaconStateModifyCurrentParticipationField(t *testing.T, factory getState) {
	st, err := factory()
	require.NoError(t, err)
	assert.NoError(t, st.ModifyCurrentParticipationBits(func(val []byte) ([]byte, error) {
		length := len(val)
		mid := length / 2

		val[0] = uint8(100)
		val[len(val)-1] = uint8(200)
		val[mid] = 150
		return val, nil
	}))
	participation, err := st.CurrentEpochParticipation()
	assert.NoError(t, err)

	mid := len(participation) / 2
	assert.Equal(t, participation[0], uint8(100))
	assert.Equal(t, participation[mid], uint8(150))
	assert.Equal(t, participation[len(participation)-1], uint8(200))
}

func VerifyBeaconStateModifyCurrentParticipationField_NestedAction(t *testing.T, factory getState) {
	st, err := factory()
	require.NoError(t, err)
	assert.NoError(t, st.ModifyCurrentParticipationBits(func(val []byte) ([]byte, error) {

		length := len(val)
		mid := length / 2

		val[0] = uint8(100)
		v1, err := st.ValidatorAtIndex(0)
		if err != nil {
			return nil, err
		}
		v1.WithdrawableEpoch = 2
		err = st.UpdateValidatorAtIndex(0, v1)
		if err != nil {
			return nil, err
		}

		val[mid] = 150

		v2, err := st.ValidatorAtIndex(types.ValidatorIndex(mid))
		if err != nil {
			return nil, err
		}
		v2.WithdrawableEpoch = 10
		err = st.UpdateValidatorAtIndex(types.ValidatorIndex(mid), v2)
		if err != nil {
			return nil, err
		}
		val[len(val)-1] = uint8(200)
		v3, err := st.ValidatorAtIndex(types.ValidatorIndex(len(val) - 1))
		if err != nil {
			return nil, err
		}
		v3.WithdrawableEpoch = 50
		err = st.UpdateValidatorAtIndex(types.ValidatorIndex(len(val)-1), v3)
		if err != nil {
			return nil, err
		}
		return val, nil
	}))
	participation, err := st.CurrentEpochParticipation()
	assert.NoError(t, err)

	mid := len(participation) / 2
	assert.Equal(t, participation[0], uint8(100))
	val, err := st.ValidatorAtIndex(types.ValidatorIndex(0))
	assert.NoError(t, err)
	assert.Equal(t, types.Epoch(2), val.WithdrawableEpoch)
	assert.Equal(t, participation[mid], uint8(150))
	val, err = st.ValidatorAtIndex(types.ValidatorIndex(mid))
	assert.NoError(t, err)
	assert.Equal(t, types.Epoch(10), val.WithdrawableEpoch)
	assert.Equal(t, participation[len(participation)-1], uint8(200))
	val, err = st.ValidatorAtIndex(types.ValidatorIndex(len(participation) - 1))
	assert.NoError(t, err)
	assert.Equal(t, types.Epoch(50), val.WithdrawableEpoch)
}

func VerifyBeaconStateModifyPreviousParticipationField(t *testing.T, factory getState) {
	st, err := factory()
	require.NoError(t, err)
	assert.NoError(t, st.ModifyPreviousParticipationBits(func(val []byte) ([]byte, error) {
		length := len(val)
		mid := length / 2

		val[0] = uint8(100)
		val[len(val)-1] = uint8(200)
		val[mid] = 150
		return val, nil
	}))
	participation, err := st.PreviousEpochParticipation()
	assert.NoError(t, err)

	mid := len(participation) / 2
	assert.Equal(t, participation[0], uint8(100))
	assert.Equal(t, participation[mid], uint8(150))
	assert.Equal(t, participation[len(participation)-1], uint8(200))
}

func VerifyBeaconStateModifyPreviousParticipationField_NestedAction(t *testing.T, factory getState) {
	st, err := factory()
	require.NoError(t, err)
	assert.NoError(t, st.ModifyPreviousParticipationBits(func(val []byte) ([]byte, error) {

		length := len(val)
		mid := length / 2

		val[0] = uint8(100)
		v1, err := st.ValidatorAtIndex(0)
		if err != nil {
			return nil, err
		}
		v1.WithdrawableEpoch = 2
		err = st.UpdateValidatorAtIndex(0, v1)
		if err != nil {
			return nil, err
		}

		val[mid] = 150

		v2, err := st.ValidatorAtIndex(types.ValidatorIndex(mid))
		if err != nil {
			return nil, err
		}
		v2.WithdrawableEpoch = 10
		err = st.UpdateValidatorAtIndex(types.ValidatorIndex(mid), v2)
		if err != nil {
			return nil, err
		}
		val[len(val)-1] = uint8(200)
		v3, err := st.ValidatorAtIndex(types.ValidatorIndex(len(val) - 1))
		if err != nil {
			return nil, err
		}
		v3.WithdrawableEpoch = 50
		err = st.UpdateValidatorAtIndex(types.ValidatorIndex(len(val)-1), v3)
		if err != nil {
			return nil, err
		}
		return val, nil
	}))
	participation, err := st.PreviousEpochParticipation()
	assert.NoError(t, err)

	mid := len(participation) / 2
	assert.Equal(t, participation[0], uint8(100))
	val, err := st.ValidatorAtIndex(types.ValidatorIndex(0))
	assert.NoError(t, err)
	assert.Equal(t, types.Epoch(2), val.WithdrawableEpoch)
	assert.Equal(t, participation[mid], uint8(150))
	val, err = st.ValidatorAtIndex(types.ValidatorIndex(mid))
	assert.NoError(t, err)
	assert.Equal(t, types.Epoch(10), val.WithdrawableEpoch)
	assert.Equal(t, participation[len(participation)-1], uint8(200))
	val, err = st.ValidatorAtIndex(types.ValidatorIndex(len(participation) - 1))
	assert.NoError(t, err)
	assert.Equal(t, types.Epoch(50), val.WithdrawableEpoch)
}
