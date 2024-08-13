package kv_test

//
//func TestStore_LightclientUpdate_CanSaveRetrieve(t *testing.T) {
//	db := setupDB(t)
//
//	l := util.NewTestLightClient(t).SetupTest()
//
//	update, err := lightclient.NewLightClientOptimisticUpdateFromBeaconState(l.Ctx, l.State, l.Block, l.AttestedState)
//	require.NoError(t, err)
//	require.NotNil(t, update, "update is nil")
//
//	require.Equal(t, l.Block.Block().Slot(), update.SignatureSlot, "Signature slot is not equal")
//
//	period := uint64(1)
//	err = db.SaveLightClientUpdate(l.Ctx, period, &ethpbv2.LightClientUpdateWithVersion{
//		Version: 1,
//		Data:    update,
//	})
//	require.NoError(t, err)
//
//}
