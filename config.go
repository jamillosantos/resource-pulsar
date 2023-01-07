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
