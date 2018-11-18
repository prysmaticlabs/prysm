package p2p

import (
	"fmt"

	pb "github.com/prysmaticlabs/prysm/proto/testing"
)

// Feeds can be use to subscribe to any type of message.
func ExampleServer_Feed() {
	s, err := NewServer(&ServerConfig{})
	if err != nil {
		panic(err)
	}

	// Let's wait for a puzzle from our peers then try to solve it.
	feed := s.Feed(&pb.Puzzle{})

	ch := make(chan Message, 5) // Small buffer size. I don't expect many puzzles.
	sub := feed.Subscribe(ch)

	// Always close these resources.
	defer sub.Unsubscribe()
	defer close(ch)

	// Wait until we have a puzzle to solve.
	msg := <-ch
	puzzle, ok := msg.Data.(*pb.Puzzle)

	if !ok {
		panic("Received a message that wasn't a puzzle!")
	}

	fmt.Printf("Received puzzle %s from peer %v\n", puzzle, msg.Peer)

	if puzzle.Answer == "fourteen" {
		fmt.Println("I solved the puzzle!")
	} else {
		fmt.Println("The answer isn't \"fourteen\"... giving up")
	}
}
