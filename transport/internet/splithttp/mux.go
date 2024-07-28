package splithttp

import (
	"context"
	"sync"
	"time"

	"github.com/xtls/xray-core/common/errors"
	"github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/transport/internet"
)

type MuxManager struct {
	sync.Mutex
	config        *Multiplexing
	dialerClients []DialerClient
	totalConns    int32
	connUsage     map[DialerClient]int32
	connCreatedAt map[DialerClient]time.Time
	connLifetime map[DialerClient]time.Duration
}

func NewMuxManager(config *Multiplexing) *MuxManager {
	return &MuxManager{
		config:        config,
		dialerClients: make([]DialerClient, 0),
		connUsage:     make(map[DialerClient]int32),
		connCreatedAt: make(map[DialerClient]time.Time),
		connLifetime: make(map[DialerClient]time.Duration),
	}
}

func (m *MuxManager) Dial(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig) (DialerClient, error) {
	m.Lock()
	defer m.Unlock()
	errors.LogDebug(ctx, "SplitHTTP MUX - Dial ", dest)
	if len(m.dialerClients) > 0 {
		m.RemoveExpiredConnections()
	}
	errors.LogDebug(ctx, "SplitHTTP MUX - Mode=", m.config.Mode)
	switch m.config.Mode {
	case Multiplexing_PREFRE_EXTISTING:
		return m.dialPreferExisting(ctx, dest, streamSettings)
	case Multiplexing_PREFRE_NEW:
		return m.dialPreferNew(ctx, dest, streamSettings)
	default:
		return getHTTPClient(ctx, dest, streamSettings), nil
	}
}

func (m *MuxManager) dialPreferExisting(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig) (DialerClient, error) {
	for {
		for _, client := range m.dialerClients {
			if m.canReuseConnection(client) {
				errors.LogDebug(ctx, "SplitHTTP MUX - Reuse")
				m.connUsage[client]++
				return client, nil
			}
		}
		errors.LogDebug(ctx, "SplitHTTP MUX - No client available")
		if m.totalConns >= m.config.MaxConnections && m.config.MaxConnections != 0 {
			if streamSettings.ProtocolSettings.(*Config).MaxUploadSize.From > 0 {
				time.Sleep(streamSettings.ProtocolSettings.(*Config).GetNormalizedUploadDelay())
			}
			continue
		}
		break
	}
	errors.LogDebug(ctx, "SplitHTTP MUX - No client, try to create one")
	return m.createNewConnection(ctx, dest, streamSettings)
}

func (m *MuxManager) dialPreferNew(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig) (DialerClient, error) {
	for {
		if m.totalConns < m.config.MaxConnections || m.config.MaxConnections == 0 {
			errors.LogDebug(ctx, "SplitHTTP MUX - connections not full, try to create one")
			return m.createNewConnection(ctx, dest, streamSettings)
		}

		for _, client := range m.dialerClients {
			if m.canReuseConnection(client) {
				errors.LogDebug(ctx, "SplitHTTP MUX - Reuse")
				m.connUsage[client]++
				return client, nil
			}
		}
		errors.LogDebug(ctx, "SplitHTTP MUX - No client available")
		if streamSettings.ProtocolSettings.(*Config).MaxUploadSize.From > 0 {
			time.Sleep(streamSettings.ProtocolSettings.(*Config).GetNormalizedUploadDelay())
		}
		continue
	}
}

func (m *MuxManager) createNewConnection(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig) (DialerClient, error) {
	errors.LogDebug(ctx, "SplitHTTP MUX - Create")
	client := getHTTPClient(ctx, dest, streamSettings)
	m.dialerClients = append(m.dialerClients, client)
	m.connUsage[client] = 1
	m.connCreatedAt[client] = time.Now()
	m.connLifetime[client] = streamSettings.ProtocolSettings.(*Config).GetNormalizedUploadDelay()
	m.totalConns++
	errors.LogDebug(ctx, "SplitHTTP MUX - Return new client")
	return client, nil
}

func (m *MuxManager) canReuseConnection(client DialerClient) bool {
	usage := m.connUsage[client]
	createdAt := m.connCreatedAt[client]
	Lifetime := m.connLifetime[client]
	if usage >= m.config.MaxConnectionConcurrency.To {
		return false
	}

	if time.Since(createdAt) > Lifetime {
		return false
	}

	return true
}

func (m *MuxManager) RemoveExpiredConnections() {
	m.Lock()
	defer m.Unlock()

	now := time.Now()
	for i := 0; i < len(m.dialerClients); i++ {
		client := m.dialerClients[i]
		if now.Sub(m.connCreatedAt[client]) > time.Duration(m.config.MaxConnectionLifetime.To)*time.Second {
			errors.LogDebug(context.Background(), "SplitHTTP MUX - Expired: ", client)
			if i < len(m.dialerClients)-1 {
				m.dialerClients = append(m.dialerClients[:i], m.dialerClients[i+1:]...)
			} else {
				m.dialerClients = m.dialerClients[:i]
			}
			delete(m.connUsage, client)
			delete(m.connCreatedAt, client)
			delete(m.connLifetime, client)
			m.totalConns--
			i--
		}
	}
}
