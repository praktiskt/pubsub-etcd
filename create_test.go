package pubsubetcd

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"go.etcd.io/etcd/clientv3"
)

func GetUuid() string {
	u, _ := uuid.NewUUID()
	return u.String()
}

type TestCase struct {
	topic      string
	shouldFail bool
}

func NewTopicCase() TestCase {
	return TestCase{
		topic:      GetUuid(),
		shouldFail: false,
	}
}

var testCases = []struct {
	topic      string
	shouldFail bool
}{
	// Invalid, bad name
	{"abc$", true},

	// Valid, new topic is created
	{"abc", false},
	{"def", false},
	{GetUuid(), false},
}

func TestCreateGet(t *testing.T) {
	// Connect to etcd
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: time.Duration(5) * time.Second,
	})

	if err != nil {
		t.Errorf("%w", err)
	}

	// Create topics for tests that should succeed
	for _, tc := range testCases {
		if !tc.shouldFail {
			CreateTopic(cli, tc.topic, 10)
		}
	}

	// Get created topics. If the topic does not exist, an error is expected.
	for _, tc := range testCases {
		_, err = GetTopic(cli, tc.topic)
		if tc.shouldFail && err == nil {
			t.Errorf("%w", err)
		}
	}
}

func BenchmarkCreateGet(b *testing.B) {
	// Connect to etcd
	etcd, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: time.Duration(5) * time.Second,
	})

	if err != nil {
		b.Errorf("%w", err)
	}

	for i := 0; i < b.N; i++ {
		// Run and log tests
		tc := NewTopicCase()
		_, err = CreateTopic(etcd, tc.topic, 10)
		if err != nil {
			if !tc.shouldFail {
				b.Errorf("%w", err)
			}
		}
	}
}
