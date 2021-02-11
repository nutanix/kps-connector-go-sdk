package transport

import (
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const NatsTestPort = 8369

func runNatsServerOnPort(port int) *server.Server {
	opts := natsserver.DefaultTestOptions
	opts.Port = port
	return runServerWithOptions(&opts)
}

func runServerWithOptions(opts *server.Options) *server.Server {
	return natsserver.RunServer(opts)
}

func TestNewTransportClient(t *testing.T) {
	s := runNatsServerOnPort(NatsTestPort)
	defer s.Shutdown()

	brokerURL := fmt.Sprintf("nats://127.0.0.1:%d", NatsTestPort)
	originalBroker := transportCfg.NatsBroker
	transportCfg.NatsBroker = brokerURL
	defer func() {
		transportCfg.NatsBroker = originalBroker
	}()

	t.Run("constructor returns interface of expected underlying type", func(t *testing.T) {
		client, err := NewTransportClient()
		require.NoError(t, err)
		require.NotNil(t, client)
		assert.IsType(t, &natsClient{}, client)
	})

	t.Run("ensure client is same across constructor calls", func(t *testing.T) {
		c1, err := NewTransportClient()
		require.NoError(t, err)
		c2, err := NewTransportClient()
		require.NoError(t, err)
		assert.Same(t, c1, c2)
	})

	t.Run("publish subscribe unsubscribe lifecycle", func(t *testing.T) {
		channel := "testchannel"
		client, err := NewTransportClient()
		require.NoError(t, err)
		require.NotNil(t, client)
		msg := []byte("foo")

		called := make(chan bool)
		defer close(called)
		cb := func(m *Message) {
			called <- true
			assert.Equal(t, msg, m.Payload)
		}

		sub, err := client.Subscribe(channel, cb)
		require.NoError(t, err)
		assert.Equal(t, channel, sub.Channel())

		err = client.Publish(channel, Message{Payload: msg})
		require.NoError(t, err)

		time.Sleep(time.Second * 10)
		assert.True(t, <-called)

		err = sub.Unsubscribe()
		assert.NoError(t, err)
	})
}
