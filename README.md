# resource-pulsar

A Go resource wrapper for [Apache Pulsar](https://pulsar.apache.org/) built on top of the official [pulsar-client-go](https://github.com/apache/pulsar-client-go).

It manages the Pulsar client lifecycle (`Start`/`Stop`) and provides a simple consumer abstraction with message handler support.

## Installation

```bash
go get github.com/jamillosantos/resource-pulsar
```

## Usage

### Creating a resource

```go
rsc := rscpulsar.New(rscpulsar.PlatformConfig{
    URL: "pulsar://localhost:6650",
    Timeouts: rscpulsar.Timeouts{
        Connection: 10 * time.Second,
        Operation:  30 * time.Second,
    },
})

if err := rsc.Start(ctx); err != nil {
    log.Fatal(err)
}
defer rsc.Stop(ctx)
```

### Subscribing to a topic

```go
consumer, err := rsc.Subscribe(rscpulsar.SubscriptionPlatformConfig{
    SubscriptionName: "my-subscription",
    Topic:            "persistent://public/default/my-topic",
}, func(ctx context.Context, msg pulsar.Message) consume.MessageHandlerResult {
    fmt.Println(string(msg.Payload()))
    return consume.Ack()
})
if err != nil {
    log.Fatal(err)
}

if err := consumer.Listen(ctx); err != nil {
    log.Fatal(err)
}
defer consumer.Close(ctx)
```

### Message handler results

The handler function returns a `MessageHandlerResult` that controls acknowledgment:

- `consume.Ack()` -- acknowledge the message
- `consume.Nack()` -- negative acknowledge (redelivery)
- `consume.Later(delay)` -- reconsume after a delay

### Custom Pulsar options

Use `WithCustomPulsarConfig` to access the underlying `pulsar.ClientOptions`:

```go
rsc := rscpulsar.New(cfg, rscpulsar.WithCustomPulsarConfig(func(opts *pulsar.ClientOptions) {
    opts.Authentication = pulsar.NewAuthenticationToken("my-token")
}))
```
