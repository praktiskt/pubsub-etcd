package pubsubetcd

import (
	"fmt"
	"testing"
	"time"

	"go.etcd.io/etcd/clientv3"
)

var putTests = []struct {
	topicName string
	data      string
}{
	{GetUuid(), `{"a":"b"}`},
	{GetUuid(), `{"a":"b"}`},
}

func PutBatchesNPartitions(topicName string, topicPartitions, setSize int, b *testing.B) {
	// Connect to etcd
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: time.Duration(5) * time.Second,
	})

	if err != nil {
		b.Errorf("failed to connect to etcd: %w", err)
	}

	// Create topic
	top, err := CreateTopic(cli, topicName, topicPartitions)
	if err != nil {
		b.Errorf("%w", err)
	}

	// Create batch to send
	batch := []string{}
	for i := 0; i < setSize; i++ {
		batch = append(batch, fmt.Sprintf(`{"%v": "%v"}`, GetUuid(), GetUuid()))
	}

	// Send batch
	for i := 0; i < b.N; i++ {
		_, fail := top.PutBatch(batch)
		if len(fail) != 0 {
			b.Errorf("%v messages failed to send", len(fail))
		}
	}
}

func BenchmarkPutBatches1000Messages1Partition(b *testing.B) {
	PutBatchesNPartitions("topic-with-1-partition-"+GetUuid(), 1, 1000, b)
}
func BenchmarkPutBatches1000Messages10Partitions(b *testing.B) {
	PutBatchesNPartitions("topic-with-10-partitions-"+GetUuid(), 10, 1000, b)
}
func BenchmarkPutBatches1000Messages100Partitions(b *testing.B) {
	PutBatchesNPartitions("topic-with-100-partitions-"+GetUuid(), 100, 1000, b)
}
