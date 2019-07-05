package hobbits

//import (
//	"reflect"
//	"strconv"
//	"testing"
//)
//
//func TestNewHobbitsNode(t *testing.T) {
//	var test = []struct {
//		node HobbitsNode
//		host        string
//		port        int
//		staticPeers []string
//	}{
//		{
//			node: NewHobbitsNode("123.12.14", 3333,[]string{"192.0.2.1"}),
//			host: "123.12.14",
//			port: 3333,
//			staticPeers: []string{"192.0.2.1"},
//		},
//		{
//			node: NewHobbitsNode("192.0.2.1", 5555, []string{"65.93.214.134", "66.171.248.170"}),
//			host: "192.0.2.1",
//			port: 5555,
//			staticPeers: []string{"65.93.214.134", "66.171.248.170"},
//		},
//
//	}
//
//	for i, tt := range test {
//		t.Run(strconv.Itoa(i), func(t *testing.T) {
//			if !reflect.DeepEqual(tt.node.Host, tt.host) || !reflect.DeepEqual(tt.node.Port, tt.port) || !reflect.DeepEqual(tt.node.StaticPeers, tt.staticPeers){
//				t.Error("return value of NewHobbitsNode does not match expected value")
//			}
//		})
//	}
//}
