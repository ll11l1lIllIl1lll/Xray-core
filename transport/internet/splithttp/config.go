package splithttp

import (
	"crypto/rand"
	"math/big"
	"net/http"
	"time"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/transport/internet"
)

func (c *Config) GetNormalizedPath() string {
	path := c.Path
	if path == "" {
		path = "/"
	}
	if path[0] != '/' {
		path = "/" + path
	}
	if path[len(path)-1] != '/' {
		path = path + "/"
	}
	return path
}

func (c *Config) GetRequestHeader() http.Header {
	header := http.Header{}
	for k, v := range c.Header {
		header.Add(k, v)
	}
	return header
}

func (c *Config) GetNormalizedMaxConcurrentUploads() int32 {
	if c.MaxConcurrentUploads == 0 {
		return 10
	}

	return c.MaxConcurrentUploads
}

func (c *Config) GetNormalizedMaxUploadSize() int32 {
	if c.MaxUploadSize.From == 0{
		return 1000000
	}
	return c.MaxUploadSize.roll()
}

func (c *Config) GetNormalizedUploadDelay() time.Duration {
	if c.MinUploadDelay.From == 0{
		return 0 * time.Millisecond
	}
	return time.Duration(c.MinUploadDelay.roll()) * time.Millisecond
}

func (c *Config) GetNormalizedMux() *Multiplexing {
    if c.Mux == nil {
        return &Multiplexing{
            Mode:                     Multiplexing_PREFRE_EXTISTING,
            MaxConnectionConcurrency: &RandRangeConfig{From: 40, To: 40},
            MaxConnectionLifetime:    &RandRangeConfig{From: 300, To: 600},
            MaxConnections:           1,
        }
    }
    return c.Mux
}

func (c *Config) GetNormalizedMinUploadInterval() RandRangeConfig {
	r := c.MinUploadDelay

	if r == nil {
		r = &RandRangeConfig{
			From: 30,
			To:   30,
		}
	}

	return *r
}

func init() {
	common.Must(internet.RegisterProtocolConfigCreator(protocolName, func() interface{} {
		return new(Config)
	}))
}

func (c RandRangeConfig) roll() int32 {
	if c.From == c.To {
		return c.From
	}
	bigInt, _ := rand.Int(rand.Reader, big.NewInt(int64(c.To-c.From)))
	return c.From + int32(bigInt.Int64())
}
