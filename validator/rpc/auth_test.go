package rpc

import (
	"context"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	dbtest "github.com/prysmaticlabs/prysm/validator/db/testing"
)

func TestServer_Signup_PasswordAlreadyExists(t *testing.T) {
	valDB := dbtest.SetupDB(t, [][48]byte{})
	ctx := context.Background()
	ss := &Server{
		valDB: valDB,
	}

	// Save a hash password pre-emptively to the database.
	hashedPassword := []byte("2093402934902839489238492")
	require.NoError(t, valDB.SaveHashedPasswordForAPI(ctx, hashedPassword))

	// Attempt to signup despite already having a hashed password in the DB
	// which should immediately fail.
	strongPass := "29384283xasjasd32%%&*@*#*"
	_, err := ss.Signup(ctx, &pb.AuthRequest{
		Password: strongPass,
	})
	require.ErrorContains(t, "Validator already has a password set, cannot signup", err)
}

func TestServer_SignupAndLogin_RoundTrip(t *testing.T) {
	valDB := dbtest.SetupDB(t, [][48]byte{})
	ctx := context.Background()
	ss := &Server{
		valDB: valDB,
	}
	weakPass := "password"
	_, err := ss.Signup(ctx, &pb.AuthRequest{
		Password: weakPass,
	})
	require.ErrorContains(t, "Could not validate password input", err)

	// We assert we are able to signup with a strong password.
	strongPass := "29384283xasjasd32%%&*@*#*"
	_, err = ss.Signup(ctx, &pb.AuthRequest{
		Password: strongPass,
	})
	require.NoError(t, err)

	// Assert we stored the hashed password.
	hashedPass, err := valDB.HashedPasswordForAPI(ctx)
	require.NoError(t, err)
	assert.NotEqual(t, 0, len(hashedPass))

	// We assert we are able to login.
	_, err = ss.Login(ctx, &pb.AuthRequest{
		Password: strongPass,
	})
	require.NoError(t, err)
}
