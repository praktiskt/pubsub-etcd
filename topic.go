package pubsubetcd

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"go.etcd.io/etcd/clientv3"
)

type TopicName string

func (tn TopicName) String() string { return string(tn) }
func (tn TopicName) IsValid() bool {
	matched, _ := regexp.MatchString(`^[A-z0-9\-_]{3,}$`, tn.String())
	return matched
}

type Topic struct {
	Name        TopicName
	Partitions  int
	MaxMessages int64
	TTL         time.Duration
	etcd        *clientv3.Client
}

func (t Topic) GetName() TopicName { return t.Name }
func (t Topic) GetPartitions() int { return t.Partitions }

func (t *Topic) GetTopicMetadata(topicName string) error {
	attrs := []string{"partitions"}
	for i, attr := range attrs {
		re, _ := t.etcd.Get(context.TODO(), topicName+"/"+attr)
		if len(re.Kvs) == 0 {
			return fmt.Errorf("%v does not exist", topicName+"/"+attr)
		}
		attrs[i] = string(re.Kvs[0].Value)
	}
	t.Name = TopicName(topicName)

	p, err := strconv.Atoi(attrs[0])
	if err != nil {
		return err
	}
	t.Partitions = p

	return nil
}
