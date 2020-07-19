package pubsubetcd

import (
	"context"
	"fmt"
	"time"

	"go.etcd.io/etcd/clientv3"
)

func GetTopic(etcd *clientv3.Client, name string) (Topic, error) {
	n := TopicName(name)
	if !n.IsValid() {
		return Topic{}, fmt.Errorf(`Invalid topic name, must match regex [A-z0-9\-_]{3,}`)
	}
	top := Topic{}
	top.etcd = etcd
	err := top.GetTopicMetadata("/topic/" + n.String())
	if err != nil {
		return Topic{}, err
	}
	return top, nil
}

func CreateTopic(etcd *clientv3.Client, name string, partitions int) (Topic, error) {
	n := TopicName(name)
	if !n.IsValid() {
		return Topic{}, fmt.Errorf(`Invalid topic name, must match regex [A-z0-9\-_]{3,}`)
	}

	if partitions < 0 {
		return Topic{}, fmt.Errorf("partitions must be at least 1")
	}

	tn := "/topic/" + n.String()
	re, err := etcd.Get(context.TODO(), tn+"/taken")
	if err != nil {
		return Topic{}, err
	}

	if len(re.Kvs) != 0 {
		return Topic{}, fmt.Errorf("topic %v already exists", name)
	}

	// Topic is available, occupy it and set default values
	tx := etcd.Txn(context.TODO())
	_, err = tx.If().
		Then(
			clientv3.OpPut(tn+"/taken", "true"),
			clientv3.OpPut(tn+"/partitions", fmt.Sprint(partitions)),
			clientv3.OpPut(tn+"/created", fmt.Sprint(time.Now().Unix())),
		).Commit()

	if err != nil {
		return Topic{}, err
	}

	t := Topic{
		Name:       TopicName(tn),
		Partitions: partitions,
		etcd:       etcd,
	}

	return t, nil
}
