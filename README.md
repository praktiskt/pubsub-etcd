# pubsub-etcd
Abusing etcd by turning it into a pubsub system. Brokerless, only uses etcd to store consumer group state, offsets etc.

## Why?
In all honesty, it was a fun thing to build. 

## Contributing
The idea is to keep this package brokerless. That means that consumers should be responsible for reporting what kind of state they are in, and ultimately cleaning up the topic and partition they are consuming.

## Running
For pubsub-etcd to work, you need an etcd instance up and running. There's a convenience command for this in the `_examples` folder.
```
cd _examples && make standalone-etcd
```

We can now connect to the instance.
```	
etcd, _ := clientv3.New(clientv3.Config{
    Endpoints:   []string{"localhost:2379"},
    DialTimeout: 5 * time.Second,
})
```

Let's create a topic, which is the place where we will store messages. Partitions are there to spread out events for increased throughput (read and write).
```
name := "my-awesome-topic"
partitions := 3 // Max is 127
myTopic, err := pubsubetcd.CreateTopic(etcd, name, partitions)
if err != nil {
    log.Printf("%v", err)
}
```

Note that if the topic already exists, we will get an error trying to create it again. In such cases we can just get it instead.
```
myTopic, _ = pubsubetcd.GetTopic(etcd, name)
```

Subscribing to a topic means you'll start receiving messages from said topic. In this case, we'll subscribe to all partitions using `Subscribe`. _We are now a consumer of a topic and a set of partitions._
```
consumerName := "foobar"
subscriptions, err := myTopic.Subscribe(consumerName)
if err != nil {
    log.Printf("%v", err)
}
```

We get back one subscription for every partition we now are subscribed to. Each subscription gives us a `Messages` channel on which all events are received. Let's set up a listener for it to watch incoming messages.
```
for _, sub := range subscriptions {
    go func(sub pubsubetcd.Subscription) {
        for {
            select {
            case msg := <-sub.Messages:
                log.Printf("key: %v, value: %v, offset: %v", msg.Key, msg.Value, msg.Offset)
            }
        }
    }(sub)
}
```

We can send any slice of strings to the topic. The default behaviour is to randomly distribute incoming data across all available partitions. _We now produced messages to a topic._
```
success, fail := myTopic.PutBatch([]string{"hello", "darkness", "my", "old", "friend"})
log.Printf("successfully sent %v messages, failed to send %v messages", len(success), len(fail))
```

Since we've already set up a listener, lets wait for a second to receive everything we sent.
```
time.Sleep(1 * time.Second)
// Events should show up in stdout.
```

When you're done consuming messages make sure to unsubscribe. This will eventually let the topic and it's partitions be available for other consumers.
```
for _, sub := range subscriptions {
    log.Printf("Unsubscribing from %v, partition %v", sub.Topic.Name, sub.Partition)
    sub.Unsubscribe()
}ª
```

## More examples
Check the `_examples` folder.
```
cd _examples

# Start an etcd instance locally.
make standalone-etcd

# Run the example you want.
```


## Todo / Done
* [ ] Topics
    * [x] Create
    * [x] Get
    * [ ] Update
    * [ ] Remove
    * [ ] Message ttl
* [x] Partitions
* [x] Consumer groups
* [x] Offset tracking
* [x] Put / producing messages
    * [x] Put
    * [x] BatchPut
* [ ] Delete
    * [ ] Delete
    * [ ] BatchDelete
* [ ] Compacting specific topics (probably not possible)


## Tests
Tests require an etcd instance to be running. `cd _examples && make standalone-etcd` can start such an instance for you.