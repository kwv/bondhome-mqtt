package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ssmall/bondhome-mqtt/bondhome"
	"github.com/ssmall/bondhome-mqtt/mqtt"
	"golang.org/x/sync/errgroup"

	paho "github.com/eclipse/paho.mqtt.golang"
)

var Version = "dev"

func main() {
	brokerAddress := flag.String("broker", "", "The broker to connect to; see https://godoc.org/github.com/eclipse/paho.mqtt.golang#ClientOptions.AddBroker")
	mqttUser := flag.String("mqtt-user", "", "The username for the MQTT broker")
	mqttPass := flag.String("mqtt-pass", "", "The password for the MQTT broker")
	mqttID := flag.String("mqtt-id", "", "The client ID for the MQTT broker (defaults to hostname)")
	bridgeAddress := flag.String("bridge", "", "The hostname or IP address of the Bond Home bridge")
	bridgeToken := flag.String("token", "", "The Bond Home bridge API token. See http://docs-local.appbond.com/#section/Getting-Started/Getting-the-Bond-Token")
	verbose := flag.Bool("v", false, "Enable verbose logging")
	flag.Parse()

	if *brokerAddress == "" {
		fmt.Fprintln(os.Stderr, "Must specify broker!")
		os.Exit(1)
	}
	if *bridgeAddress == "" {
		fmt.Fprintln(os.Stderr, "Must specify bridge!")
		os.Exit(1)
	}
	if *bridgeToken == "" {
		fmt.Fprintln(os.Stderr, "Must specify token!")
		os.Exit(1)
	}

	opts := &slog.HandlerOptions{}
	if *verbose {
		opts.Level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, opts))
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mqttClient, err := mqtt.NewClient(*brokerAddress, *mqttUser, *mqttPass, *mqttID)
	if err != nil {
		slog.Error("Unable to connect to MQTT broker", "error", err)
		os.Exit(1)
	}

	slog.Info("Connected to broker", "address", *brokerAddress)

	bridge := bondhome.NewBridge(*bridgeAddress, *bridgeToken)

	err = setupDeviceActionHandlers(ctx, bridge, mqttClient)
	if err != nil {
		slog.Error("Exiting due to error", "error", err)
		os.Exit(1)
	}

	pushClient, err := bondhome.NewClient(ctx, *bridgeAddress+":30007")
	if err != nil {
		slog.Error("Exiting due to error", "error", err)
		os.Exit(1)
	}

	err = setupDeviceStateHandlers(ctx, pushClient, mqttClient)
	if err != nil {
		slog.Error("Exiting due to error", "error", err)
		os.Exit(1)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	s := <-c
	slog.Warn("Got signal, exiting", "signal", s)
}

func setupDeviceStateHandlers(ctx context.Context, pushClient bondhome.PushClient, mqttClient paho.Client) error {
	err := pushClient.StartListening()
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				update, err := pushClient.Receive(10 * time.Second)
				if err != nil {
					if _, ok := err.(bondhome.Timeout); !ok {
						panic(fmt.Errorf("error receiving from Bond Bridge: %w", err))
					}
				}
				if update != nil && update.Topic != "" {
					topic := "bondhome/" + update.Topic
					body, err := update.Body.MarshalJSON()
					if err != nil {
						slog.Error("Unable to marshal update body to JSON", "error", err)
					}
					slog.Debug("Publishing update", "topic", topic, "body", string(body))
					token := mqttClient.Publish(topic, byte(0), false, string(body))
					if token.Wait() && token.Error() != nil {
						slog.Error("Unable to publish to topic", "topic", topic, "error", token.Error())
					}
				} else if update != nil && update.ErrorMsg != "" {
					slog.Error("Got error response from Bond Home bridge", "code", update.ErrorID, "msg", update.ErrorMsg)
				}
			}
		}
	}()

	return nil
}

func setupDeviceActionHandlers(ctx context.Context, bridge bondhome.Bridge, mqttClient paho.Client) error {
	devices, err := bridge.GetDeviceIDs()

	if err != nil {
		return fmt.Errorf("could not get devices from bridge: %w", err)
	}

	slog.Info("Got device IDs", "devices", devices)

	var g errgroup.Group

	for _, deviceID := range devices {
		localDeviceID := deviceID
		g.Go(func() error {
			d, err := bridge.GetDevice(localDeviceID)
			if err != nil {
				return err
			}
			slog.Info("Discovered device", "id", localDeviceID, "device", d)

			var hg errgroup.Group

			for _, actionID := range d.Actions {
				localActionID := actionID
				hg.Go(func() error {
					return actionHandler(mqttClient, bridge, localDeviceID, localActionID)
				})
			}

			return hg.Wait()
		})
	}

	err = g.Wait()
	if err != nil {
		return fmt.Errorf("error setting up listeners: %w", err)
	}
	return nil
}

func actionHandler(mqtt paho.Client, bridge bondhome.Bridge, deviceID string, actionID string) error {
	topic := fmt.Sprintf("bondhome/devices/%s/%s", deviceID, actionID)

	token := mqtt.Subscribe(topic, byte(0), func(c paho.Client, m paho.Message) {
		slog.Debug("Received message", "id", m.MessageID(), "payload", string(m.Payload()), "topic", m.Topic())

		payload := m.Payload()
		if err := json.Unmarshal(payload, &map[string]interface{}{}); err != nil {
			slog.Debug("Message payload is not an object, wrapping as object", "payload", string(payload), "error", err)
			payload = []byte(fmt.Sprintf("{\"body\": %s}", payload))
		}

		if err := bridge.ExecuteAction(deviceID, actionID, string(payload)); err != nil {
			slog.Error("Not acking message due to error executing action", "error", err)
		} else {
			m.Ack()
		}
	})

	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("unable to subscribe to topic %s: %w", topic, token.Error())
	}

	slog.Info("Subscribed to topic", "topic", topic)

	return nil
}
