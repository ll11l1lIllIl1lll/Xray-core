package splithttp

import (
	"crypto/rand"
	"math/big"
	"net/http"
	"strings"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/transport/internet"
)

func (c *Config) GetNormalizedPath() string {
	pathAndQuery := strings.SplitN(c.Path, "?", 2)
	path := pathAndQuery[0]

	if path == "" || path[0] != '/' {
		path = "/" + path
	}

	if path[len(path)-1] != '/' {
		path = path + "/"
	}

	return path
}

func (c *Config) GetNormalizedQuery() string {
	pathAndQuery := strings.SplitN(c.Path, "?", 2)
	query := ""

	if len(pathAndQuery) > 1 {
		query = pathAndQuery[1]
	}

	if query != "" {
		query += "&"
	}

	paddingLen := c.GetNormalizedXPaddingBytes().roll()
	if paddingLen > 0 {
		query += "x_padding=" + strings.Repeat("0", int(paddingLen))
	}

	return query
}

func (c *Config) GetRequestHeader() http.Header {
	header := http.Header{}
	for k, v := range c.Header {
		header.Add(k, v)
	}

	return header
}
func (c *Config) WriteResponseHeader(writer http.ResponseWriter) {
	paddingLen := c.GetNormalizedXPaddingBytes().roll()
	if paddingLen > 0 {
		writer.Header().Set("X-Padding", strings.Repeat("0", int(paddingLen)))
	}
}

func (c *Config) GetNormalizedScMaxConcurrentPosts() RandRangeConfig {
	if c.ScMaxConcurrentPosts == nil || c.ScMaxConcurrentPosts.To == 0 {
		return RandRangeConfig{
			From: 100,
			To:   100,
		}
	}

	return *c.ScMaxConcurrentPosts
}

func (m *Multiplexing) GetNormalizedMaxConnectionConcurrency() RandRangeConfig {
	if m.MaxConnectionConcurrency == nil || m.MaxConnectionConcurrency.To == 0 {
		return RandRangeConfig{
			From: 1,
			To:   3,
		}
	}

	return *m.MaxConnectionConcurrency
}

func (c *Multiplexing) GetNormalizedConnectionLifetime() RandRangeConfig {
	if c.MaxConnectionLifetime == nil || c.MaxConnectionLifetime.To == 0 {
		return RandRangeConfig{
			From: 60000,
			To:   90000,
		}
	}
	return *c.MaxConnectionLifetime
}

func (c *Config) GetNormalizedScMaxEachPostBytes() RandRangeConfig {
	if c.ScMaxEachPostBytes == nil || c.ScMaxEachPostBytes.To == 0 {
		return RandRangeConfig{
			From: 1000000,
			To:   1000000,
		}
	}
	return *c.ScMaxEachPostBytes
}
func (c *Config) GetNormalizedScMinPostsIntervalMs() RandRangeConfig {
	if c.ScMinPostsIntervalMs == nil || c.ScMinPostsIntervalMs.To == 0 {
		return RandRangeConfig{
			From: 30,
			To:   30,
		}
	}

	return *c.ScMinPostsIntervalMs
}

func (c *Config) GetNormalizedXPaddingBytes() RandRangeConfig {
	if c.XPaddingBytes == nil || c.XPaddingBytes.To == 0 {
		return RandRangeConfig{
			From: 100,
			To:   1000,
		}
	}

	return *c.XPaddingBytes
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
