# bondhome-mqtt
MQTT bridge for BondHome API. See http://docs-local.appbond.com

This is a modernized fork of the original `bondhome-mqtt` bridge, updated for Go 1.23 and aligned with standard CI/CD practices.

## Overview

This program does two things:
1. Relay commands received via MQTT to the Bond Bridge API
2. Update MQTT topics with device status (subscribed via BPUP<sup>[1]</sup>)

### Topics

On startup, the `bondhome-mqtt` program gets a list of all devices connected to the Bond Home Bridge and sets up the following MQTT topics for each device:

- `bondhome/devices/<device id>/<action>`: For triggering actions
- `bondhome/devices/<device id>/state`: For publishing device state

## Usage

For detailed information on building, testing, and contributing to this project, please see [CONTRIBUTING.md](CONTRIBUTING.md).

### Command line

```bash
./bondhome-mqtt -broker tcp://<host>:<port> -bridge <ip> -token <token> [options]
```

#### Options

*   `-broker`: The address of the MQTT broker (e.g., `tcp://localhost:1883`)
*   `-bridge`: The IP address of the Bond Home bridge
*   `-token`: The Bond API token
*   `-mqtt-user`: (Optional) Username for MQTT broker
*   `-mqtt-pass`: (Optional) Password for MQTT broker
*   `-mqtt-id`: (Optional) Custom Client ID for MQTT connection
*   `-v`: Enable verbose (debug) logging

### Docker Compose

You can also use Docker Compose for easy deployment:

1. Copy `.env.example` to `.env` and fill in your details.
2. Run the bridge:
```bash
docker-compose up -d
```

Example `docker-compose.yml`:
```yaml
services:
  bondhome-mqtt:
    image: kwv4/bondhome-mqtt:latest
    restart: unless-stopped
    environment:
      - BOND_BROKER=tcp://mqtt.home:1883
      - BOND_BRIDGE=
      - BOND_TOKEN=
```

[1]: http://docs-local.appbond.com/#section/Bond-Push-UDP-Protocol-(BPUP)
[2]: http://docs-local.appbond.com/#section/Getting-Started/Getting-the-Bond-Token