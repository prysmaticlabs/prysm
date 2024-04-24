package filesystem

/*
func TestTryPruneDir_CachedNotExpired(t *testing.T) {
		fs := afero.NewMemMapFs()
		pr := newBlobPruner(0)
		epoch := pr.retentionPeriod
		slot, err := slots.EpochStart(epoch)
		require.NoError(t, err)
		_, sidecars := util.GenerateTestDenebBlockWithSidecar(t, [32]byte{}, slot, fieldparams.MaxBlobsPerBlock)
		sc, err := verification.BlobSidecarNoop(sidecars[0])
		require.NoError(t, err)
		ident := identForSidecar(sc)
		dir := ident.dir()
		// This slot is right on the edge of what would need to be pruned, so by adding it to the cache and
		// skipping any other test setup, we can be certain the hot cache path never touches the filesystem.
		require.NoError(t, pr.cache.ensure(sc.BlockRoot(), sc.Slot(), 0))
		pruned, err := pr.tryPruneDir(dir, pr.windowSize)
		require.NoError(t, err)
		require.Equal(t, 0, pruned)
	}

	func TestTryPruneDir_CachedExpired(t *testing.T) {
		t.Run("empty directory", func(t *testing.T) {
			fs := afero.NewMemMapFs()
			p := newBlobPruner(0)
			var slot primitives.Slot = 0
			_, sidecars := util.GenerateTestDenebBlockWithSidecar(t, [32]byte{}, slot, 1)
			sc, err := verification.BlobSidecarNoop(sidecars[0])
			require.NoError(t, err)
			ident := identForSidecar(sc)
			dir := ident.dir()
			require.NoError(t, fs.Mkdir(dir, directoryPermissions)) // make empty directory
			require.NoError(t, pr.cache.ensure(sc.BlockRoot(), sc.Slot(), 0))
			pruned, err := pr.tryPruneDir(dir, slot+1)
			require.NoError(t, err)
			require.Equal(t, 0, pruned)
		})
		t.Run("blobs to delete", func(t *testing.T) {
			fs, bs := NewEphemeralBlobStorageAndFs(t)
			var slot primitives.Slot = 0
			_, sidecars := util.GenerateTestDenebBlockWithSidecar(t, [32]byte{}, slot, 2)
			scs, err := verification.BlobSidecarSliceNoop(sidecars)
			require.NoError(t, err)

			require.NoError(t, bs.Save(scs[0]))
			require.NoError(t, bs.Save(scs[1]))

			// check that the root->slot is cached
			root := scs[0].BlockRoot()
			ident := identForSidecar(scs[0])
			dir := ident.dir()
			cs, cok := bs.pruner.cache.epoch(root)
			require.Equal(t, true, cok)
			require.Equal(t, slots.ToEpoch(slot), cs)

			// ensure that we see the saved files in the filesystem
			files, err := listDir(fs, dir)
			require.NoError(t, err)
			require.Equal(t, 2, len(files))

			pruned, err := bs.pruner.tryPruneDir(dir, slot+1)
			require.NoError(t, err)
			require.Equal(t, 2, pruned)
			files, err = listDir(fs, dir)
			require.ErrorIs(t, err, os.ErrNotExist)
			require.Equal(t, 0, len(files))
		})
	}
func TestTryPruneDir_SlotFromFile(t *testing.T) {
	t.Run("expired blobs deleted", func(t *testing.T) {
		fs, bs := NewEphemeralBlobStorageAndFs(t)
		var slot primitives.Slot = 0
		_, sidecars := util.GenerateTestDenebBlockWithSidecar(t, [32]byte{}, slot, 2)
		scs, err := verification.BlobSidecarSliceNoop(sidecars)
		require.NoError(t, err)

		require.NoError(t, bs.Save(scs[0]))
		require.NoError(t, bs.Save(scs[1]))

		// check that the root->slot is cached
		root := scs[0].BlockRoot()
		ident := identForSidecar(scs[0])
		dir := ident.dir()
		cs, ok := bs.pruner.cache.epoch(root)
		require.Equal(t, true, ok)
		require.Equal(t, slots.ToEpoch(slot), cs)
		// evict it from the cache so that we trigger the file read path
		bs.pruner.cache.evict(root)
		_, ok = bs.pruner.cache.epoch(root)
		require.Equal(t, false, ok)

		// ensure that we see the saved files in the filesystem

		files, err := listDir(fs, dir)
		require.NoError(t, err)
		require.Equal(t, 2, len(files))

		pruned, err := bs.pruner.tryPruneDir(dir, slot+1)
		require.NoError(t, err)
		require.Equal(t, 2, pruned)
		files, err = listDir(fs, dir)
		require.ErrorIs(t, err, os.ErrNotExist)
		require.Equal(t, 0, len(files))
	})
	t.Run("not expired, intact", func(t *testing.T) {
		fs, bs := NewEphemeralBlobStorageAndFs(t)
		// Set slot equal to the window size, so it should be retained.
		slot := bs.pruner.windowSize
		_, sidecars := util.GenerateTestDenebBlockWithSidecar(t, [32]byte{}, slot, 2)
		scs, err := verification.BlobSidecarSliceNoop(sidecars)
		require.NoError(t, err)

		require.NoError(t, bs.Save(scs[0]))
		require.NoError(t, bs.Save(scs[1]))

		// Evict slot mapping from the cache so that we trigger the file read path.
		root := scs[0].BlockRoot()
		ident := identForSidecar(scs[0])
		dir := ident.dir()
		bs.pruner.cache.evict(root)
		_, ok := bs.pruner.cache.epoch(root)
		require.Equal(t, false, ok)

		// Ensure that we see the saved files in the filesystem.
		files, err := listDir(fs, dir)
		require.NoError(t, err)
		require.Equal(t, 2, len(files))

		// This should use the slotFromFile code (simulating restart).
		// Setting pruneBefore == slot, so that the slot will be outside the window (at the boundary).
		pruned, err := bs.pruner.tryPruneDir(dir, slot)
		require.NoError(t, err)
		require.Equal(t, 0, pruned)

		// Ensure files are still present.
		files, err = listDir(fs, dir)
		require.NoError(t, err)
		require.Equal(t, 2, len(files))
	})
}
*/
