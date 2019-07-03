package hobbits
//
//import (
//	"errors"
//	"fmt"
//	"reflect"
//	"strconv"
//	"testing"
//)
//
//func Test_processHobbitsMessage(t *testing.T) {
//	h := HobbitsNode{}
//
//	var test = []struct {
//		message HobbitsMessage
//		expected error
//	}{
//		{message: HobbitsMessage{
//			Version: "10.2.3",
//			Protocol: "RPC",
//			Header: []byte("doesn't matter"),
//			Body: []byte("also doesn't matter"),
//
//		}, expected: nil},
//		{message: HobbitsMessage{
//			Version: "10.2.5.8",
//			Protocol: "GOSSIP",
//			Header: []byte("test"),
//			Body: []byte("test"),
//		}, expected: nil},
//		{message: HobbitsMessage{
//			Version: "10.2",
//			Protocol: "NONE",
//			Header: []byte("unsuccessful test"),
//			Body: []byte(" "),
//		}, expected: errors.New("protocol unsupported")}, // TODO: this test is failing and i'm not sure wh
//	}
//
//	for i, tt := range test {
//		t.Run(strconv.Itoa(i), func(t *testing.T) {
//			fmt.Println(h.processHobbitsMessage(tt.message))
//			fmt.Println(tt.expected)
//			if !reflect.DeepEqual(h.processHobbitsMessage(tt.message), tt.expected) {
//				t.Error("return value of processHobbitsMessage does not match expected value")
//			}
//		})
//	}
//}
//
//func Test_processRPC(t *testing.T) {
//
//}
//
//func Test_processGossip(t *testing.T) {
//
//}
