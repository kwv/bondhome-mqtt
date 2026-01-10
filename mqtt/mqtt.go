package mqtt

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

const (
	connectTimeout = 10 * time.Second
)

// NewClient creates a new MQTT client and tries to establish
// a connection to the specified broker
func NewClient(broker, username, password, clientID string) (paho.Client, error) {
	if clientID == "" {
		var err error
		clientID, err = os.Hostname()
		if err != nil {
			return nil, err
		}
	}

	slog.Info("Establishing connection to MQTT broker", "broker", broker, "client_id", clientID)

	opts := paho.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientID)

	if username != "" {
		opts.SetUsername(username)
	}
	if password != "" {
		opts.SetPassword(password)
	}

	client := paho.NewClient(opts)
	connectToken := client.Connect()
	if !connectToken.WaitTimeout(connectTimeout) {
		return nil, fmt.Errorf("timed out after %v", connectTimeout)
	}
	return client, nil
}
