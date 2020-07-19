package pubsubetcd

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	v3 "go.etcd.io/etcd/clientv3"
)

var heartbeatTimeoutSeconds = 5

type Message struct {
	Key    string
	Offset int64
	Value  string
}

type Subscription struct {
	Messages     chan Message
	Shutdown     chan bool
	ConsumerName string
	Topic        Topic
	Partition    int
}

func NewMessage(key string, offset int64, value string) Message {
	return Message{
		Key:    key,
		Offset: offset,
		Value:  value,
	}
}

func (t *Topic) AnnounceSubscription(consumerName string, partitionNumber int) error {
	path := fmt.Sprintf("%v/partition=%v/consumers/%v", t.GetName(), partitionNumber, consumerName)
	hb := path + "/heartbeat"
	re, err := t.etcd.Get(context.TODO(), hb)
	if err != nil {
		return err
	}

	var currentHb int64
	now := time.Now().Unix()
	for _, kv := range re.Kvs {
		currentHb, err = strconv.ParseInt(string(kv.Value), 10, 64)
		if err != nil {
			return err
		}
	}

	if currentHb < (now - int64(heartbeatTimeoutSeconds)) {
		_, err := t.etcd.Put(context.TODO(), hb, fmt.Sprint(now))
		if err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("partition %v is already subscribed to by %v", partitionNumber, consumerName)
}

func (t *Topic) SubscribeToPartition(consumerName string, partitionNumber int, offset int64) (Subscription, error) {
	if partitionNumber < 0 || partitionNumber >= t.Partitions {
		return Subscription{}, fmt.Errorf("partitionNumber %v out of topic partition range (0-%v)", partitionNumber, t.Partitions-1)
	}

	err := t.AnnounceSubscription(consumerName, partitionNumber)
	if err != nil {
		return Subscription{}, err
	}

	pref := fmt.Sprintf("%v/partition=%v/events", t.GetName().String(), partitionNumber)
	inc := t.etcd.Watch(context.TODO(), pref, v3.WithRev(offset))
	messages := make(chan Message)
	s := Subscription{}
	s.Messages = messages
	s.ConsumerName = consumerName
	s.Topic = *t
	s.Partition = partitionNumber
	s.Shutdown = s.KeepSubscriptionAlive()
	go func() {
		for {
			select {
			case <-s.Shutdown:
				break
			case msg := <-inc:
				for _, event := range msg.Events {
					// etcd sometimes sends invalid records when watching a channel, especially
					// when said channel is being closed. Let's ignore those records for now.
					// TODO: Figure out what's going on here really.
					if event.Kv.Version != 0 {
						message := NewMessage(string(event.Kv.Key), event.Kv.ModRevision, string(event.Kv.Value))
						messages <- message
					}
				}
			}
		}
	}()
	return s, nil
}

func (t *Topic) Subscribe(consumerName string) ([]Subscription, error) {
	subs := []Subscription{}
	subChan := make(chan Subscription)
	errChan := make(chan error)

	for partition := 0; partition < t.Partitions; partition++ {
		go func(consumerName string, partition int) {
			// TODO: Figure out how to toggle between different offset modes (first, last, specific)
			offset, err := t.GetConsumerOffset(consumerName, partition)
			if err != nil {
				errChan <- err
				return
			}
			sub, err := t.SubscribeToPartition(consumerName, partition, offset)
			if err != nil {
				errChan <- err
				return
			}

			subChan <- sub
		}(consumerName, partition)
	}

	for i := 0; i < t.Partitions; i++ {
		select {
		case sub := <-subChan:
			subs = append(subs, sub)
		case err := <-errChan:
			log.Printf("[ERROR] - %v", err)
		}
	}
	return subs, nil
}

func (s *Subscription) KeepSubscriptionAlive() chan bool {
	exit := make(chan bool)
	go func(s *Subscription, exit chan bool) {
		ticker := time.NewTicker(1 * time.Second)
		for {
			select {
			case <-exit:
				// We're unsubscribing / terminating the liveness of sub.
				p := fmt.Sprintf("%v/partition=%v/consumers/%v/heartbeat", s.Topic.GetName(), s.Partition, s.ConsumerName)
				s.Topic.etcd.Put(context.TODO(), p, fmt.Sprint(time.Now().Unix()-int64(heartbeatTimeoutSeconds)))
				break
			case <-ticker.C:
				p := fmt.Sprintf("%v/partition=%v/consumers/%v/heartbeat", s.Topic.GetName(), s.Partition, s.ConsumerName)
				_, err := s.Topic.etcd.Put(context.TODO(), p, fmt.Sprint(time.Now().Unix()))
				if err != nil {
					// TODO: Do something proper here, don't just die. Restart consumer? Announce partition is available?
					log.Printf("[ERROR] - %v failed to send heartbeat, comitting seppuku", s)
					s.Unsubscribe()
				}
			}
		}
	}(s, exit)
	return exit
}

func (s *Subscription) CommitOffset(offset int64) error {
	t := s.Topic
	co := fmt.Sprintf("%v/partition=%v/consumers/%v/offset", t.GetName().String(), s.Partition, s.ConsumerName)
	tx := t.etcd.Txn(context.TODO())
	_, err := tx.If().Then(
		v3.OpPut(co, fmt.Sprint(offset)),
	).Commit()

	if err != nil {
		return err
	}

	return nil
}

func (t *Topic) GetConsumerOffset(consumerName string, partitionNumber int) (int64, error) {
	co := fmt.Sprintf("%v/partition=%v/consumers/%v/offset", t.GetName().String(), partitionNumber, consumerName)
	re, err := t.etcd.Get(context.TODO(), co)
	if err != nil {
		return -1, err
	}

	var maxOffset int64
	for _, kv := range re.Kvs {
		s := string(kv.Value)
		in, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return -1, err
		}
		maxOffset = in
	}

	return maxOffset, nil
}

func (s *Subscription) Unsubscribe() {
	// Turn off both heartbeat and consumer channel
	s.Shutdown <- true
	s.Shutdown <- true
}
