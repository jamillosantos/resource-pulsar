package rscpulsar

import (
	"time"
)

type Timeouts struct {
	Connection time.Duration `config:"connection"`
	Operation  time.Duration `config:"operation"`
}

type PlatformConfig struct {
	URL      string   `config:"url,secret"`
	Timeouts Timeouts `config:"timeouts"`
}

type SubscriptionPlatformConfig struct {
	Name             string   `config:"name"`
	SubscriptionName string   `config:"subscription_name,required"`
	Topic            string   `config:"topic"`
	Topics           []string `config:"topics"`
	TopicsPattern    string   `config:"topics_pattern"`
}
