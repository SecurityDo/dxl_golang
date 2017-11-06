package examples

import "testing"
import mqtt_client "github.com/SecurityDo/dxl_golang/mqtt"

func TestConnect(*testing.T) {
	config := &mqtt_client.MqttClientConfig{
		TLSEnable:      true,
		RootCAFile:     "config/brokercerts.crt",
		ClientCertFile: "config/client.crt",
		ClientKeyFile:  "config/client.key",
		ServerURLs:     []string{"tls://192.168.2.104:8883"},
		SkipVerify:     true,
	}

	mqttClient := mqtt_client.NewMqttClient(config)
	mqttClient.Init()
}
