package light

//
//type Server struct {
//	Database iface.LightClientDatabase
//}

//// BestUpdates GET /eth/v1alpha1/lightclient/best_update/:periods.
//func (s *Server) BestUpdates(ctx context.Context, req *ethpb.BestUpdatesRequest) (*ethpb.BestUpdatesResponse, error) {
//	updates := make([]*ethpb.LightClientUpdate, 0)
//	for _, period := range req.SyncCommitteePeriods {
//		update, err := s.Database.LightClientBestUpdateForPeriod(ctx, period)
//		if errors.Is(err, kv.ErrNotFound) {
//			continue
//		} else if err != nil {
//			return nil, status.Errorf(codes.Internal, "Could not retrieve best update for %d: %v", period, err)
//		}
//		updates = append(updates, update)
//	}
//	return &ethpb.BestUpdatesResponse{Updates: updates}, nil
//}
//
//// LatestUpdateFinalized GET /eth/v1alpha1/lightclient/latest_update_finalized/
//func (s *Server) LatestUpdateFinalized(ctx context.Context, _ *empty.Empty) (*ethpb.LightClientUpdate, error) {
//	update, err := s.Database.LightClientLatestFinalizedUpdate(ctx)
//	if errors.Is(err, kv.ErrNotFound) {
//		return nil, status.Error(codes.Internal, "No latest finalized update found")
//	} else if err != nil {
//		return nil, status.Errorf(codes.Internal, "Could not retrieve latest finalized update: %v", err)
//	}
//	return update, nil
//}
//
//// LatestUpdateNonFinalized /eth/v1alpha1/lightclient/latest_update_nonfinalized/
//func (s *Server) LatestUpdateNonFinalized(ctx context.Context, _ *empty.Empty) (*ethpb.LightClientUpdate, error) {
//	update, err := s.Database.LightClientLatestNonFinalizedUpdate(ctx)
//	if errors.Is(err, kv.ErrNotFound) {
//		return nil, status.Error(codes.Internal, "No latest non-finalized update found")
//	} else if err != nil {
//		return nil, status.Errorf(codes.Internal, "Could not retrieve latest non-finalized update: %v", err)
//	}
//	return update, nil
//}
