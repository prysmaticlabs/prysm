package keymanager

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	pb "github.com/wealdtech/eth2-signer-api/pb/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	// maxMessageSize is the largest message that can be received over GRPC.  Set to 8MB, which handles ~128K keys.
	maxMessageSize = 8 * 1024 * 1024
)

// Remote is a key manager that accesses a remote wallet daemon.
type Remote struct {
	paths               []string
	conn                *grpc.ClientConn
	accounts            map[[48]byte]*accountInfo
	signClientInitiator func(*grpc.ClientConn)
}

type accountInfo struct {
	Name   string `json:"name"`
	PubKey []byte `json:"pubkey"`
}

type remoteOpts struct {
	Location     string                 `json:"location"`
	Accounts     []string               `json:"accounts"`
	Certificates *remoteCertificateOpts `json:"certificates"`
}

type remoteCertificateOpts struct {
	CACert     string `json:"ca_cert"`
	ClientCert string `json:"client_cert"`
	ClientKey  string `json:"client_key"`
}

var remoteOptsHelp = `The remote key manager connects to a walletd instance.  The options are:
  - location This is the location to look for wallets.  If not supplied it will
    use the standard (operating system-dependent) path.
  - accounts This is a list of account specifiers.  An account specifier is of
    the form <wallet name>/[account name],  where the account name can be a
    regular expression.  If the account specifier is just <wallet name> all
    accounts in that wallet will be used.  Multiple account specifiers can be
    supplied if required.
  - certificates This provides paths to certificates:
    - ca_cert This is the path to the server's certificate authority certificate file
    - client_cert This is the path to the client's certificate file
    - client_key This is the path to the client's key file

An sample keymanager options file (with annotations; these should be removed if
using this as a template) is:

  {
	"location":    "host.example.com:12345", // Connect to walletd at host.example.com on port 12345
    "accounts":    ["Validators/Account.*"]  // Use all accounts in the 'Validators' wallet starting with 'Account'
	"certificates": {
	  "ca_cert": "/home/eth2/certs/ca.crt"         // Certificate file for the CA that signed the server's certificate
	  "client_cert": "/home/eth2/certs/client.crt" // Certificate file for this client
	  "client_key": "/home/eth2/certs/client.key"  // Key file for this client
	}
  }`

// NewRemoteWallet creates a key manager populated with the keys from walletd.
func NewRemoteWallet(input string) (KeyManager, string, error) {
	opts := &remoteOpts{}
	err := json.Unmarshal([]byte(input), opts)
	if err != nil {
		return nil, remoteOptsHelp, err
	}

	if len(opts.Accounts) == 0 {
		return nil, remoteOptsHelp, errors.New("at least one account specifier is required")
	}

	// Load the client certificates.
	if opts.Certificates == nil {
		return nil, remoteOptsHelp, errors.New("certificates are required")
	}
	if opts.Certificates.ClientCert == "" {
		return nil, remoteOptsHelp, errors.New("client certificate is required")
	}
	if opts.Certificates.ClientKey == "" {
		return nil, remoteOptsHelp, errors.New("client key is required")
	}
	clientPair, err := tls.LoadX509KeyPair(opts.Certificates.ClientCert, opts.Certificates.ClientKey)
	if err != nil {
		return nil, remoteOptsHelp, errors.Wrap(err, "failed to obtain client's certificate and/or key")
	}

	// Load the CA for the server certificate if present.
	cp := x509.NewCertPool()
	if opts.Certificates.CACert != "" {
		serverCA, err := ioutil.ReadFile(opts.Certificates.CACert)
		if err != nil {
			return nil, remoteOptsHelp, errors.Wrap(err, "failed to obtain server's CA certificate")
		}
		if !cp.AppendCertsFromPEM(serverCA) {
			return nil, remoteOptsHelp, errors.Wrap(err, "failed to add server's CA certificate to pool")
		}
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{clientPair},
		RootCAs:      cp,
	}
	clientCreds := credentials.NewTLS(tlsCfg)

	grpcOpts := []grpc.DialOption{
		// Require TLS with client certificate.
		grpc.WithTransportCredentials(clientCreds),
		// Receive large messages without erroring.
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxMessageSize)),
	}

	conn, err := grpc.Dial(opts.Location, grpcOpts...)
	if err != nil {
		return nil, remoteOptsHelp, errors.New("failed to connect to remote wallet")
	}

	km := &Remote{
		conn:  conn,
		paths: opts.Accounts,
	}

	err = km.RefreshValidatingKeys()
	if err != nil {
		return nil, remoteOptsHelp, errors.Wrap(err, "failed to fetch accounts from remote wallet")
	}

	return km, remoteOptsHelp, nil
}

// FetchValidatingKeys fetches the list of public keys that should be used to validate with.
func (km *Remote) FetchValidatingKeys() ([][48]byte, error) {
	res := make([][48]byte, 0, len(km.accounts))
	for _, accountInfo := range km.accounts {
		res = append(res, bytesutil.ToBytes48(accountInfo.PubKey))
	}
	return res, nil
}

// Sign without protection is not supported by remote keymanagers.
func (km *Remote) Sign(pubKey [48]byte, root [32]byte) (bls.Signature, error) {
	return nil, errors.New("remote keymanager does not support unprotected signing")
}

// SignGeneric signs a generic message for the validator to broadcast.
func (km *Remote) SignGeneric(pubKey [48]byte, root [32]byte, domain [32]byte) (bls.Signature, error) {
	accountInfo, exists := km.accounts[pubKey]
	if !exists {
		return nil, ErrNoSuchKey
	}

	client := pb.NewSignerClient(km.conn)
	req := &pb.SignRequest{
		Id:     &pb.SignRequest_Account{Account: accountInfo.Name},
		Data:   root[:],
		Domain: domain[:],
	}
	resp, err := client.Sign(context.Background(), req)
	if err != nil {
		return nil, err
	}
	switch resp.State {
	case pb.ResponseState_DENIED:
		return nil, ErrDenied
	case pb.ResponseState_FAILED:
		return nil, ErrCannotSign
	}
	return bls.SignatureFromBytes(resp.Signature)
}

// SignProposal signs a block proposal for the validator to broadcast.
func (km *Remote) SignProposal(pubKey [48]byte, domain [32]byte, data *ethpb.BeaconBlockHeader) (bls.Signature, error) {
	accountInfo, exists := km.accounts[pubKey]
	if !exists {
		return nil, ErrNoSuchKey
	}

	client := pb.NewSignerClient(km.conn)
	req := &pb.SignBeaconProposalRequest{
		Id:     &pb.SignBeaconProposalRequest_Account{Account: accountInfo.Name},
		Domain: domain[:],
		Data: &pb.BeaconBlockHeader{
			Slot:          data.Slot,
			ProposerIndex: data.ProposerIndex,
			ParentRoot:    data.ParentRoot,
			StateRoot:     data.StateRoot,
			BodyRoot:      data.BodyRoot,
		},
	}
	resp, err := client.SignBeaconProposal(context.Background(), req)
	if err != nil {
		return nil, err
	}
	switch resp.State {
	case pb.ResponseState_DENIED:
		return nil, ErrDenied
	case pb.ResponseState_FAILED:
		return nil, ErrCannotSign
	}
	return bls.SignatureFromBytes(resp.Signature)
}

// SignAttestation signs an attestation for the validator to broadcast.
func (km *Remote) SignAttestation(pubKey [48]byte, domain [32]byte, data *ethpb.AttestationData) (bls.Signature, error) {
	accountInfo, exists := km.accounts[pubKey]
	if !exists {
		return nil, ErrNoSuchKey
	}

	client := pb.NewSignerClient(km.conn)
	req := &pb.SignBeaconAttestationRequest{
		Id:     &pb.SignBeaconAttestationRequest_Account{Account: accountInfo.Name},
		Domain: domain[:],
		Data: &pb.AttestationData{
			Slot:            data.Slot,
			CommitteeIndex:  data.CommitteeIndex,
			BeaconBlockRoot: data.BeaconBlockRoot,
			Source: &pb.Checkpoint{
				Epoch: data.Source.Epoch,
				Root:  data.Source.Root,
			},
			Target: &pb.Checkpoint{
				Epoch: data.Target.Epoch,
				Root:  data.Target.Root,
			},
		},
	}
	resp, err := client.SignBeaconAttestation(context.Background(), req)
	if err != nil {
		return nil, err
	}
	switch resp.State {
	case pb.ResponseState_DENIED:
		return nil, ErrDenied
	case pb.ResponseState_FAILED:
		return nil, ErrCannotSign
	}
	return bls.SignatureFromBytes(resp.Signature)
}

// RefreshValidatingKeys refreshes the list of validating keys from the remote signer.
func (km *Remote) RefreshValidatingKeys() error {
	listerClient := pb.NewListerClient(km.conn)
	listAccountsReq := &pb.ListAccountsRequest{
		Paths: km.paths,
	}
	resp, err := listerClient.ListAccounts(context.Background(), listAccountsReq)
	if err != nil {
		return err
	}
	if resp.State == pb.ResponseState_DENIED {
		return errors.New("attempt to fetch keys denied")
	}
	if resp.State == pb.ResponseState_FAILED {
		return errors.New("attempt to fetch keys failed")
	}

	verificationRegexes := pathsToVerificationRegexes(km.paths)
	accounts := make(map[[48]byte]*accountInfo, len(resp.Accounts))
	for _, account := range resp.Accounts {
		verified := false
		for _, verificationRegex := range verificationRegexes {
			if verificationRegex.Match([]byte(account.Name)) {
				verified = true
				break
			}
		}
		if !verified {
			log.WithField("path", account.Name).Warn("Received unwanted account from server; ignoring")
			continue
		}
		account := &accountInfo{
			Name:   account.Name,
			PubKey: account.PublicKey,
		}
		accounts[bytesutil.ToBytes48(account.PubKey)] = account
	}
	km.accounts = accounts
	return nil
}

// pathsToVerificationRegexes turns path specifiers in to regexes to ensure accounts we are given are good.
func pathsToVerificationRegexes(paths []string) []*regexp.Regexp {
	regexes := make([]*regexp.Regexp, 0, len(paths))
	for _, path := range paths {
		log := log.WithField("path", path)
		parts := strings.Split(path, "/")
		if len(parts) == 0 || len(parts[0]) == 0 {
			log.Debug("Invalid path")
			continue
		}
		if len(parts) == 1 {
			parts = append(parts, ".*")
		}
		if strings.HasPrefix(parts[1], "^") {
			parts[1] = parts[1][1:]
		}
		var specifier string
		if strings.HasSuffix(parts[1], "$") {
			specifier = fmt.Sprintf("^%s/%s", parts[0], parts[1])
		} else {
			specifier = fmt.Sprintf("^%s/%s$", parts[0], parts[1])
		}
		regex, err := regexp.Compile(specifier)
		if err != nil {
			log.WithField("specifier", specifier).WithError(err).Warn("Invalid path regex")
			continue
		}
		regexes = append(regexes, regex)
	}
	return regexes
}
