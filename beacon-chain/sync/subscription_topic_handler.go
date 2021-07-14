package sync

import (
	"sync"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// This is a subscription topic handler that is used to handle basic
// CRUD operations on the topic map. All operations are thread safe
// so they can be called from multiple routines.
type subTopicHandler struct {
	sync.RWMutex
	subTopics map[string]*pubsub.Subscription
}

func newSubTopicHandler() *subTopicHandler {
	return &subTopicHandler{subTopics: map[string]*pubsub.Subscription{}}
}

func (s *subTopicHandler) addTopic(topic string, sub *pubsub.Subscription) {
	s.Lock()
	s.subTopics[topic] = sub
	s.Unlock()
}

func (s *subTopicHandler) topicExists(topic string) bool {
	s.RLock()
	_, ok := s.subTopics[topic]
	s.RUnlock()
	return ok
}

func (s *subTopicHandler) removeTopic(topic string) {
	s.Lock()
	delete(s.subTopics, topic)
	s.Unlock()
}

func (s *subTopicHandler) allTopics() []string {
	topics := []string{}
	s.RLock()
	for t := range s.subTopics {
		copiedTopic := t
		topics = append(topics, copiedTopic)
	}
	s.RUnlock()
	return topics
}
