package hobbits

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/renaynay/go-hobbits/encoding"
	"gopkg.in/mgo.v2/bson"
)

func TestHobbitsNode_Listen(t *testing.T) {
	db, err := db.NewDB("go/src/renaynay/db")
	if err != nil {
		t.Errorf("can't construct new DB")
	}

	hobNode := Hobbits("127.0.0.1", 0, []string{}, db)

	go func() {
		hobNode.Listen()
	}()

	time.Sleep(3000000000)

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
		Version:  "12.4", // TODO: hits an error when over 2 decimals
		Protocol: encoding.RPC,
		Header:   marshHeader,
		Body:     marshBody,
	}

	toSend, err := encoding.Marshal(encoding.Message(msg))
	if err != nil {
		t.Errorf("could not marshal message for writing to conn")
	}

	_, err = conn.Write([]byte(toSend))
	if err != nil {
		t.Error("could not write to the TCP server: ", err)
	}

	fmt.Println("writing...")

	select {}
}
