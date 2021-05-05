// Copyright (c) 2021 Nutanix, Inc.
package transport

import (
	"fmt"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/nats-io/nats-server/v2/server"
	natsserver "github.com/nats-io/nats-server/v2/test"
	connectorpb "github.com/nutanix/kps-connector-go-sdk/connector/v1"
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

		time.Sleep(time.Second * 5)
		assert.True(t, <-called)

		err = sub.Unsubscribe()
		assert.NoError(t, err)
	})

	t.Run("transport calls callback multiple times for multiple messages in one nats payload", func(t *testing.T) {
		channel := "testchannel"
		client, err := NewTransportClient()
		require.NoError(t, err)
		require.NotNil(t, client)
		validateMap := map[string]bool{
			"foo": false,
			"bar": false,
		}

		cb := func(m *Message) {
			payload := string(m.Payload)
			assert.Contains(t, validateMap, payload)
			validateMap[payload] = true
		}

		sub, err := client.Subscribe(channel, cb)
		require.NoError(t, err)
		assert.Equal(t, channel, sub.Channel())

		nc, ok := client.(*natsClient)
		require.True(t, ok)
		require.NotNil(t, nc)
		tMsg := &connectorpb.TransportMessage{
			Timestamp: time.Now().UnixNano(),
			Payload:   [][]byte{[]byte("foo"), []byte("bar")},
		}
		data, err := proto.Marshal(tMsg)
		require.NoError(t, err)
		err = nc.conn.Publish(channel, data)
		require.NoError(t, err)

		time.Sleep(time.Second * 5)
		assert.True(t, validateMap["foo"])
		assert.True(t, validateMap["bar"])

		err = sub.Unsubscribe()
		assert.NoError(t, err)
	})
}
