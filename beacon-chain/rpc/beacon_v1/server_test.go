package beacon_v1

//func TestNodeServer_GetSyncStatus(t *testing.T) {
//	mSync := &mockSync.Sync{IsSyncing: false}
//	ns := &Server{
//		SyncChecker: mSync,
//	}
//	res, err := ns.GetSyncStatus(context.Background(), &ptypes.Empty{})
//	require.NoError(t, err)
//	assert.Equal(t, false, res.Syncing)
//	ns.SyncChecker = &mockSync.Sync{IsSyncing: true}
//	res, err = ns.GetSyncStatus(context.Background(), &ptypes.Empty{})
//	require.NoError(t, err)
//	assert.Equal(t, true, res.Syncing)
//}
