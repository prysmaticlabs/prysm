package pagination

import (
	"strconv"

	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StartAndEndPage takes in the requested page token, wanted page size, total page size.
// It returns start, end page and the next page token.
func StartAndEndPage(pageToken string, pageSize int, totalSize int) (int, int, string, error) {
	if pageToken == "" {
		pageToken = "0"
	}
	if pageSize == 0 {
		pageSize = params.BeaconConfig().DefaultPageSize
	}

	token, err := strconv.Atoi(pageToken)
	if err != nil {
		return 0, 0, "", status.Errorf(codes.InvalidArgument, "could not convert page token: %v", err)
	}

	// Start page can not be greater than validator size.
	start := token * pageSize
	if start >= totalSize {
		return 0, 0, "", status.Errorf(codes.InvalidArgument, "page start %d >= validator list %d",
			start, totalSize)
	}

	// End page can not go out of bound.
	end := start + pageSize
	if end > totalSize {
		end = totalSize
	}

	nextPageToken := strconv.Itoa(token + 1)
	return start, end, nextPageToken, nil
}