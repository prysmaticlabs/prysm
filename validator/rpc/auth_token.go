package rpc

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/golang-jwt/jwt"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/crypto/rand"
	"github.com/prysmaticlabs/prysm/io/file"
	pb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	authTokenFileName = "auth-token"
)

// Initialize returns metadata regarding whether the caller has authenticated and has a wallet.
func (s *Server) Initialize(_ context.Context, _ *emptypb.Empty) (*pb.InitializeAuthResponse, error) {
	walletExists, err := wallet.Exists(s.walletDir)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not check if wallet exists")
	}
	authTokenPath := filepath.Join(s.walletDir, authTokenFileName)
	return &pb.InitializeAuthResponse{
		HasSignedUp: file.FileExists(authTokenPath),
		HasWallet:   walletExists,
	}, nil
}

// CreateAuthToken generates a new jwt key, token and writes them
// to a file in the specified directory. Also, it logs out a prepared URL
// for the user to navigate to and authenticate with the Prysm web interface.
func CreateAuthToken(walletDirPath, validatorWebAddr string) error {
	jwtKey, err := createRandomJWTSecret()
	if err != nil {
		return err
	}
	token, err := createTokenString(jwtKey)
	if err != nil {
		return err
	}
	authTokenPath := filepath.Join(walletDirPath, authTokenFileName)
	log.Infof("Generating auth token and saving it to %s", authTokenPath)
	if err := saveAuthToken(walletDirPath, jwtKey, token); err != nil {
		return err
	}
	logValidatorWebAuth(validatorWebAddr, token)
	return nil
}

// Upon launch of the validator client, we initialize an auth token by either creating
// one from scratch or reading it from a file. This token can then be shown to the
// user via stdout and the validator client should then attempt to open the default
// browser. The web interface authenticates by looking for this token in the query parameters
// of the URL. This token is then used as the bearer token for jwt auth.
func (s *Server) initializeAuthToken(walletDir string) (string, error) {
	authTokenFile := filepath.Join(walletDir, authTokenFileName)
	if file.FileExists(authTokenFile) {
		// #nosec G304
		f, err := os.Open(authTokenFile)
		if err != nil {
			return "", err
		}
		defer func() {
			if err := f.Close(); err != nil {
				log.Error(err)
			}
		}()
		secret, token, err := readAuthTokenFile(f)
		if err != nil {
			return "", err
		}
		s.jwtSecret = secret
		return token, nil
	}
	jwtKey, err := createRandomJWTSecret()
	if err != nil {
		return "", err
	}
	s.jwtSecret = jwtKey
	token, err := createTokenString(s.jwtSecret)
	if err != nil {
		return "", err
	}
	if err := saveAuthToken(walletDir, jwtKey, token); err != nil {
		return "", err
	}
	return token, nil
}

func (s *Server) refreshAuthTokenFromFileChanges(ctx context.Context, authTokenPath string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.WithError(err).Error("Could not initialize file watcher")
		return
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			log.WithError(err).Error("Could not close file watcher")
		}
	}()
	if err := watcher.Add(authTokenPath); err != nil {
		log.WithError(err).Errorf("Could not add file %s to file watcher", authTokenPath)
		return
	}
	for {
		select {
		case <-watcher.Events:
			// If a file was modified, we attempt to read that file
			// and parse it into our accounts store.
			token, err := s.initializeAuthToken(s.walletDir)
			if err != nil {
				log.WithError(err).Errorf("Could not watch for file changes for: %s", authTokenPath)
				continue
			}
			validatorWebAddr := fmt.Sprintf("%s:%d", s.validatorGatewayHost, s.validatorGatewayPort)
			logValidatorWebAuth(validatorWebAddr, token)
		case err := <-watcher.Errors:
			log.WithError(err).Errorf("Could not watch for file changes for: %s", authTokenPath)
		case <-ctx.Done():
			return
		}
	}
}

func logValidatorWebAuth(validatorWebAddr, token string) {
	webAuthURLTemplate := "http://%s/initialize?token=%s"
	webAuthURL := fmt.Sprintf(
		webAuthURLTemplate,
		validatorWebAddr,
		url.QueryEscape(token),
	)
	log.Infof(
		"Once your validator process is runinng, navigate to the link below to authenticate with " +
			"the Prysm web interface",
	)
	log.Info(webAuthURL)
}

func saveAuthToken(walletDirPath string, jwtKey []byte, token string) error {
	hashFilePath := filepath.Join(walletDirPath, authTokenFileName)
	bytesBuf := new(bytes.Buffer)
	if _, err := bytesBuf.Write([]byte(fmt.Sprintf("%x", jwtKey))); err != nil {
		return err
	}
	if _, err := bytesBuf.Write([]byte("\n")); err != nil {
		return err
	}
	if _, err := bytesBuf.Write([]byte(token)); err != nil {
		return err
	}
	if _, err := bytesBuf.Write([]byte("\n")); err != nil {
		return err
	}
	return file.WriteFile(hashFilePath, bytesBuf.Bytes())
}

func readAuthTokenFile(r io.Reader) (secret []byte, token string, err error) {
	br := bufio.NewReader(r)
	var jwtKeyHex string
	jwtKeyHex, err = br.ReadString('\n')
	if err != nil {
		return
	}
	secret, err = hex.DecodeString(strings.TrimSpace(jwtKeyHex))
	if err != nil {
		return
	}
	tokenBytes, _, err := br.ReadLine()
	if err != nil {
		return
	}
	token = strings.TrimSpace(string(tokenBytes))
	return
}

// Creates a JWT token string using the JWT key.
func createTokenString(jwtKey []byte) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{})
	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func createRandomJWTSecret() ([]byte, error) {
	r := rand.NewGenerator()
	jwtKey := make([]byte, 32)
	n, err := r.Read(jwtKey)
	if err != nil {
		return nil, err
	}
	if n != len(jwtKey) {
		return nil, errors.New("could not create appropriately sized random JWT secret")
	}
	return jwtKey, nil
}
