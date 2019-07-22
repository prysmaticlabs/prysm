package hobbits

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/bazel-prysm/external/go_sdk/src/context"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/renaynay/go-hobbits/encoding"
	"gopkg.in/mgo.v2/bson"
)

func TestHobbitsNode_Listen(t *testing.T) {
	fakeDB, err := db.NewDB("tmp/rene/")
	if err != nil {
		t.Errorf("could not generate new DB, %s", err.Error())
	}

	deposits, _ := testutil.SetupInitialDeposits(t, 5, false)
	beaconState, err := state.GenesisBeaconState(deposits, 0, nil)

	err = fakeDB.SaveState(context.Background(), beaconState)
	if err != nil {
		t.Errorf("error saving genesis state, %s", err.Error())
	}

	hobNode := Hobbits("127.0.0.1", 0, []string{}, fakeDB)

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
		t.Errorf("error bson marshaling response boyd")
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

	_, err = conn.Write([]byte(toSend))
	if err != nil {
		t.Error("could not write to the TCP server: ", err)
	}

	fmt.Println("writing...")

	select {}
}

//func TestHobbitsNode_Broadcast(t *testing.T) {
//	db, err := db.NewDB("go/src/renaynay/db")
//	if err != nil {
//		t.Errorf("can't construct new DB")baz
//	}
//
//	hobNode := Hobbits("127.0.0.1", 0, []string{}, db)
//
//	go func() {
//		hobNode.Listen()
//	}()
//
//	time.Sleep(3000000000)
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
//	header := GossipHeader{
//		MethodID: 0,
//		Topic: "ATTESTATION",
//		Timestamp: uint64(time.Now().Unix()),
//		MessageHash: [32]byte{},
//		Hash: [32]byte{},
//	}
//
//	marshHeader, := bson.Marshal(header)
//
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
//	_, err = conn.Write([]byte(toSend))
//	if err != nil {
//		t.Error("could not write to the TCP server: ", err)
//	}
//
//	fmt.Println("writing...")
//
//	select {}
//}
