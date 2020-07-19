package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	pubsubetcd "github.com/magnusfurugard/pubsub-etcd"
	"go.etcd.io/etcd/clientv3"
)

func main() {

	endpoint := []string{}
	if len(os.Getenv("ETCD_ADVERTISE_CLIENT_URLS")) != 0 {
		endpoint = append(endpoint, os.Getenv("ETCD_ADVERTISE_CLIENT_URLS"))
	} else {
		endpoint = append(endpoint, "localhost:2379")
	}
	// Connect to running etcd instance(s).
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoint,
		DialTimeout: 5 * time.Second,
	})

	if err != nil {
		panic(err)
	}
	defer cli.Close()

	// Check if topic exists. If not, create it.
	partitions := 5
	tn := fmt.Sprintf("topic-with-%v-partitions", partitions)
	mt, err := pubsubetcd.CreateTopic(cli, tn, partitions)
	if err != nil {
		log.Printf("%v", err)
	}
	mt, err = pubsubetcd.GetTopic(cli, tn)
	if err != nil {
		panic(err)
	}

	uuid, _ := uuid.NewUUID()
	consumerName := "awesome-consumer-" + uuid.String()

	// Subscribe to all available partitions on the topic, and print a sample.
	subs, err := mt.Subscribe(consumerName)
	if err != nil {
		panic(err)
	}

	go PrintPeriodically(subs)
	go GenerateMockData(mt)

	// Wait here until user exists.
	termChan := make(chan os.Signal)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)
	<-termChan

	for _, sub := range subs {
		log.Printf("[INFO] - Unsubscribing to %v:%v", sub.ConsumerName, sub.Partition)
		sub.Unsubscribe()
	}
}

func PrintPeriodically(subscriptions []pubsubetcd.Subscription) {
	for _, subscription := range subscriptions {
		go func(subscription pubsubetcd.Subscription) {
			for {
				select {
				case msg := <-subscription.Messages:
					// Randomly print a sample if the incoming messages.
					if rand.Intn(100) == 0 {
						log.Printf("[INFO] - 1%% random sample on incomming messages: %v", msg)
					}

					// Every 100 messages, let etcd know where we are.
					if msg.Offset%100 == 0 {
						log.Printf("[INFO] - Comitting offset %v for %v:%v", msg.Offset, subscription.ConsumerName, subscription.Partition)
						if err := subscription.CommitOffset(msg.Offset); err != nil {
							log.Printf("[ERROR] - Failed to commit offset: %v", err)
						}
					}
				}
			}
		}(subscription)
	}
}

func GenerateMockData(mt pubsubetcd.Topic) {
	log.Print("Starting message production...")
	msgNum := 0
	for {
		time.Sleep(1 * time.Second)
		msgs := []string{}
		bs := rand.Intn(100)
		for {
			if bs == 0 {
				break
			} else {
				bs--
			}
			msgs = append(msgs, fmt.Sprintf(`{"message": "%v", "iteration": "%v"}`, msgNum, bs))
		}
		success, fail := mt.PutBatch(msgs)
		if len(fail) != 0 {
			log.Printf("[ERROR] - Failed to send %v messages", len(fail))
		} else {
			log.Printf("[INFO] - Sent %v messages", len(success))
			msgNum++
		}
	}
}
