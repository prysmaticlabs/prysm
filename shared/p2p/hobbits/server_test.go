package hobbits

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"testing"
	"time"

	ttl "github.com/ReneKroon/ttlcache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/renaynay/go-hobbits/encoding"
	"github.com/renaynay/go-hobbits/tcp"
	"gopkg.in/mgo.v2/bson"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestHobbitsNode_Hello(t *testing.T) {
	fakeDB, err := db.NewDB("tmp/rene/")
	if err != nil {
		t.Errorf("could not generate new DB, %s", err.Error())
	}

	deposits, _ := testutil.SetupInitialDeposits(t, 5)
	beaconState, err := state.GenesisBeaconState(deposits, 0, nil)

	err = fakeDB.SaveState(context.Background(), beaconState)
	if err != nil {
		t.Errorf("error saving genesis state, %s", err.Error())
	}

	hobNode := Hobbits("127.0.0.1", 0, []string{}, fakeDB)

	cache := ttl.NewCache()
	cache.Set(string(make([]byte, 32)), true)

	hobNode.MessageStore = cache

	go func() {
		hobNode.Listen()
	}()

	for {
		if hobNode.Server.Addr() != nil {
			break
		}

		time.Sleep(1)
	}

	conn, err := net.Dial("tcp", hobNode.Server.Addr().String())
	if err != nil {
		t.Error("could not connect to TCP server: ", err)
	}

	responseBody := Hello{
		NodeID:               "12",
		LatestFinalizedRoot:  [32]byte{},
		LatestFinalizedEpoch: 0,
		BestRoot:             [32]byte{},
		BestSlot:             0,
	}

	marshBody, err := bson.Marshal(responseBody)
	if err != nil {
		t.Errorf("error bson marshaling response body")
	}

	responseHeader := RPCHeader{
		MethodID: 0x00,
	}

	marshHeader, err := bson.Marshal(responseHeader)
	if err != nil {
		fmt.Println("error bson marshaling response header")
	}

	msg := HobbitsMessage{
		Version:  CurrentHobbits,
		Protocol: encoding.RPC,
		Header:   marshHeader,
		Body:     marshBody,
	}

	toSend := encoding.Marshal(encoding.Message(msg))

	_, err = conn.Write(toSend)
	if err != nil {
		t.Error("could not write to the TCP server: ", err)
	}

	fmt.Println("writing...")

	select {}
}

//func TestHobbitsNode_GetBlock(t *testing.T) { //TODO finish this test
//	fakeDB, err := db.NewDB("tmp/rene/")
//	if err != nil {
//		t.Errorf("could not generate new DB, %s", err.Error())
//	}
//
//	cb1 := &v1alpha1.BeaconBlock{Slot: 999, ParentRoot: []byte{'A'}}
//	err = fakeDB.SaveBlock(cb1)
//	if err != nil {
//		t.Error("test block failed to save")
//	}
//
//	hobNode := Hobbits("127.0.0.1", 0, []string{}, fakeDB)
//
//	cache := ttl.NewCache()
//	cache.Set(string(make([]byte, 32)), true)
//
//	hobNode.MessageStore = cache
//
//	go func() {
//		hobNode.Listen()
//	}()
//
//	for {
//		if hobNode.Server.Addr() != nil {
//			break
//		}
//
//		time.Sleep(1)
//	}
//
//	conn, err := net.Dial("tcp", hobNode.Server.Addr().String())
//	if err != nil {
//		t.Error("could not connect to TCP server: ", err)
//	}
//
//	responseBody := Hello{
//		NodeID:               "12",
//		LatestFinalizedRoot:  [32]byte{},
//		LatestFinalizedEpoch: 0,
//		BestRoot:             [32]byte{},
//		BestSlot:             0,
//	}
//
//	marshBody, err := bson.Marshal(responseBody)
//	if err != nil {
//		t.Errorf("error bson marshaling response body")
//	}
//
//	responseHeader := RPCHeader{
//		MethodID: 0x00,
//	}
//
//	marshHeader, err := bson.Marshal(responseHeader)
//	if err != nil {
//		fmt.Println("error bson marshaling response header")
//	}
//
//	msg := HobbitsMessage{
//		Version:  CurrentHobbits,
//		Protocol: encoding.RPC,
//		Header:   marshHeader,
//		Body:     marshBody,
//	}
//
//	toSend := encoding.Marshal(encoding.Message(msg))
//
//	_, err = conn.Write(toSend)
//	if err != nil {
//		t.Error("could not write to the TCP server: ", err)
//	}
//
//	fmt.Println("writing...")
//
//	select {}
//}

func TestHobbitsNode_Broadcast(t *testing.T) {
	db, err := db.NewDB("go/src/renaynay/db")
	if err != nil {
		t.Errorf("can't construct new DB")
	}

	server := tcp.NewServer("127.0.0.1", 0)

	ch := make(chan HobbitsMessage)

	go func() {

		server.Listen(func(conn net.Conn, message encoding.Message) {
			fmt.Println("msg")

			ch <- HobbitsMessage(message)
		})
	}()

	for {
		if server.Addr() != nil {
			break
		}

		time.Sleep(1)
	}

	hobNode := Hobbits("127.0.0.1", 0, []string{server.Addr().String()}, db)

	hobNode.Start()

	for {
		if len(hobNode.PeerConns) == 0 {
			continue
		}

		break
	}

	header := GossipHeader{
		MethodID:    0,
		Topic:       "ATTESTATION",
		Timestamp:   uint64(time.Now().Unix()),
		MessageHash: [32]byte{240, 2, 6, 253, 232, 158},
		Hash:        [32]byte{},
	}

	attestation := &pb.AttestationAnnounce{
		Hash: header.Hash[:],
	}

	hobNode.Broadcast(context.WithValue(context.Background(), "message_hash", header.MessageHash), attestation)

	read := <-ch

	buf := new(GossipHeader)

	err = bson.Unmarshal(read.Header, buf)
	if err != nil {
		t.Errorf("error unmarshaling read header: %s", err.Error())
	}

	if !reflect.DeepEqual(header.Hash, buf.Hash) {
		t.Error("Broadcast did not propagate the expected message, unsuccessful")
	}
}
