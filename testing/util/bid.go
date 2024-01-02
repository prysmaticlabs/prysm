package util

import (
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls/common"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

type FakeBid struct {
	*ethpb.BuilderBid
	sk common.SecretKey
}

type FakeBidCapella struct {
	*ethpb.BuilderBidCapella
	sk common.SecretKey
}

type FakeBidDeneb struct {
	*ethpb.BuilderBidDeneb
	sk common.SecretKey
}

func DefaultBid() (*FakeBid, error) {
	sk, err := bls.RandKey()
	if err != nil {
		return nil, err
	}
	return &FakeBid{
		BuilderBid: &ethpb.BuilderBid{
			Header: DefaultPayloadHeader(),
			Pubkey: sk.PublicKey().Marshal(),
			Value:  bytesutil.PadTo([]byte{defaultBidValue}, 32),
		},
		sk: sk,
	}, nil
}

func (b *FakeBid) Sign() (*ethpb.SignedBuilderBid, error) {
	d := params.BeaconConfig().DomainApplicationBuilder
	domain, err := signing.ComputeDomain(d, nil, nil)
	if err != nil {
		return nil, err
	}
	sr, err := signing.ComputeSigningRoot(b.BuilderBid, domain)
	if err != nil {
		return nil, err
	}
	return &ethpb.SignedBuilderBid{
		Message:   b.BuilderBid,
		Signature: b.sk.Sign(sr[:]).Marshal(),
	}, nil
}

func DefaultBidCapella() (*FakeBidCapella, error) {
	sk, err := bls.RandKey()
	if err != nil {
		return nil, err
	}
	return &FakeBidCapella{
		BuilderBidCapella: &ethpb.BuilderBidCapella{
			Header: DefaultPayloadHeaderCapella(),
			Pubkey: sk.PublicKey().Marshal(),
			Value:  bytesutil.PadTo([]byte{defaultBidValue}, 32),
		},
		sk: sk,
	}, nil
}

func (b *FakeBidCapella) Sign() (*ethpb.SignedBuilderBidCapella, error) {
	d := params.BeaconConfig().DomainApplicationBuilder
	domain, err := signing.ComputeDomain(d, nil, nil)
	if err != nil {
		return nil, err
	}
	sr, err := signing.ComputeSigningRoot(b.BuilderBidCapella, domain)
	if err != nil {
		return nil, err
	}
	return &ethpb.SignedBuilderBidCapella{
		Message:   b.BuilderBidCapella,
		Signature: b.sk.Sign(sr[:]).Marshal(),
	}, nil
}

func DefaultBidDeneb() (*FakeBidDeneb, error) {
	sk, err := bls.RandKey()
	if err != nil {
		return nil, err
	}
	return &FakeBidDeneb{
		BuilderBidDeneb: &ethpb.BuilderBidDeneb{
			Header: DefaultPayloadHeaderDeneb(),
			Pubkey: sk.PublicKey().Marshal(),
			Value:  bytesutil.PadTo([]byte{defaultBidValue}, 32),
		},
		sk: sk,
	}, nil
}

func (b *FakeBidDeneb) Sign() (*ethpb.SignedBuilderBidDeneb, error) {
	d := params.BeaconConfig().DomainApplicationBuilder
	domain, err := signing.ComputeDomain(d, nil, nil)
	if err != nil {
		return nil, err
	}
	sr, err := signing.ComputeSigningRoot(b.BuilderBidDeneb, domain)
	if err != nil {
		return nil, err
	}
	return &ethpb.SignedBuilderBidDeneb{
		Message:   b.BuilderBidDeneb,
		Signature: b.sk.Sign(sr[:]).Marshal(),
	}, nil
}
