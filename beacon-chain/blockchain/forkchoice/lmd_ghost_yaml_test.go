package forkchoice

import (
	"bytes"
	"context"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"gopkg.in/yaml.v2"
)

type Config struct {
	TestCases []struct {
		Blocks []struct {
			Id     string `yaml:"id"`
			Parent string `yaml:"parent"`
		} `yaml:"blocks"`
		Weights map[string]int `yaml:"weights"`
		Head    string         `yaml:"head"`
	} `yaml:"test_cases"`
}

func TestGetHeadFromYaml(t *testing.T) {
	ctx := context.Background()
	filename, _ := filepath.Abs("./lmd_ghost_test.yaml")
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	var c *Config
	err = yaml.Unmarshal(yamlFile, &c)

	for _, test := range c.TestCases {
		db := internal.SetupDB(t)
		kv := db.(*kv.Store)
		defer internal.TeardownDB(t, kv)

		blksRoot := make(map[int][]byte)
		// Construct block tree from yaml
		for _, blk := range test.Blocks {
			// genesis block
			if blk.Id == blk.Parent {
				b := &ethpb.BeaconBlock{Slot: 0, ParentRoot: []byte{'g'}}
				if err := db.SaveBlock(ctx, b); err != nil {
					t.Fatal(err)
				}
				root, err := ssz.SigningRoot(b)
				if err != nil {
					t.Fatal(err)
				}
				blksRoot[0] = root[:]
			} else {
				slot, err := strconv.Atoi(blk.Id[1:])
				if err != nil {
					t.Fatal(err)
				}
				parentSlot, err := strconv.Atoi(blk.Parent[1:])
				if err != nil {
					t.Fatal(err)
				}
				b := &ethpb.BeaconBlock{Slot: uint64(slot), ParentRoot: blksRoot[parentSlot]}
				if err := db.SaveBlock(ctx, b); err != nil {
					t.Fatal(err)
				}
				root, err := ssz.SigningRoot(b)
				if err != nil {
					t.Fatal(err)
				}
				blksRoot[slot] = root[:]
			}
		}

		// Assign validator votes to the blocks as weights
		count := 0
		for blk, votes := range test.Weights {
			slot, err := strconv.Atoi(blk[1:])
			if err != nil {
				t.Fatal(err)
			}
			max := count + votes
			for i := count; i < max; i++ {
				if err := db.SaveValidatorLatestVote(ctx, uint64(i), &pb.ValidatorLatestVote{Root: blksRoot[slot]}); err != nil {
					t.Fatal(err)
				}
				count++
			}
		}

		store := NewForkChoiceService(ctx, kv)
		validators := make([]*ethpb.Validator, count)
		for i := 0; i < len(validators); i++ {
			validators[i] = &ethpb.Validator{ExitEpoch: 2, EffectiveBalance: 1e9}
		}

		s := &pb.BeaconState{Validators: validators}

		if err := store.GensisStore(s); err != nil {
			t.Fatal(err)
		}

		store.justifiedCheckpt.Root = blksRoot[0]
		store.checkptBlkRoot[store.justifiedCheckpt] = bytesutil.ToBytes32(blksRoot[0])

		head, err := store.Head()
		if err != nil {
			t.Fatal(err)
		}

		headSlot, err := strconv.Atoi(test.Head[1:])
		if err != nil {
			t.Fatal(err)
		}
		wantedHead := blksRoot[headSlot]

		if !bytes.Equal(head, wantedHead) {
			t.Errorf("wanted root %#x, got root %#x", wantedHead, head)
		}
	}
}
