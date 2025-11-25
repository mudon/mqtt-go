package main

import (
	"bytes"
	"log"
	"os"
	"os/signal"
	"syscall"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/mochi-mqtt/server/v2/packets"
)

// LogHook logs all MQTT events
type LogHook struct {
	mqtt.HookBase
}

func (h *LogHook) ID() string {
	return "log-hook"
}

func (h *LogHook) Provides(b byte) bool {
	return bytes.Contains([]byte{
		mqtt.OnConnect,
		mqtt.OnDisconnect,
		mqtt.OnSubscribed,
		mqtt.OnUnsubscribed,
		mqtt.OnPublished,
	}, []byte{b})
}

func (h *LogHook) OnConnect(cl *mqtt.Client, pk packets.Packet) error {
	log.Printf("[CONNECT] Client: %s connected from %s\n", cl.ID, cl.Net.Remote)
	return nil
}

func (h *LogHook) OnDisconnect(cl *mqtt.Client, err error, expire bool) {
	log.Printf("[DISCONNECT] Client: %s disconnected\n", cl.ID)
}

func (h *LogHook) OnSubscribed(cl *mqtt.Client, pk packets.Packet, reasonCodes []byte) {
	log.Printf("[SUBSCRIBE] Client: %s subscribed to topics\n", cl.ID)
	for _, sub := range pk.Filters {
		log.Printf("  - Topic: %s (QoS: %d)\n", sub.Filter, sub.Qos)
	}
}

func (h *LogHook) OnUnsubscribed(cl *mqtt.Client, pk packets.Packet) {
	log.Printf("[UNSUBSCRIBE] Client: %s unsubscribed from topics\n", cl.ID)
	for _, filter := range pk.Filters {
		log.Printf("  - Topic: %s\n", filter.Filter)
	}
}

func (h *LogHook) OnPublished(cl *mqtt.Client, pk packets.Packet) {
	log.Printf("[PUBLISH] Client: %s | Topic: %s | Payload: %s | QoS: %d\n",
		cl.ID, pk.TopicName, string(pk.Payload), pk.FixedHeader.Qos)
}

func main() {
	// Create the new MQTT Server
	server := mqtt.New(nil)

	// Allow all connections
	_ = server.AddHook(new(auth.AllowHook), nil)

	// Add our logging hook
	_ = server.AddHook(new(LogHook), nil)

	// Create a TCP listener on port 1883
	tcp := listeners.NewTCP(listeners.Config{
		ID:      "t1",
		Address: ":1883",
	})
	err := server.AddListener(tcp)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("MQTT server started on :1883")

	// Start server in goroutine
	go func() {
		err := server.Serve()
		if err != nil {
			log.Fatal(err)
		}
	}()

	// Wait for signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down server...")
	server.Close()
}