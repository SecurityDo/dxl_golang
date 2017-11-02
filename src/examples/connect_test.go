package examples

import "testing"
import mqtt_client "mqtt"

func TestConnect(*testing.T) {
	config := &mqtt_client.MqttClientConfig{
		TLSEnable:      true,
		RootCAFile:     "/tmp/dxl_test/brokercerts.crt",
		ClientCertFile: "/tmp/dxl_test/client.crt",
		ClientKeyFile:  "/tmp/dxl_test/client.key",
		ServerURLs:     []string{"tls://192.168.2.104:8883"},
		SkipVerify:     true,
	}

	mqttClient := mqtt_client.NewMqttClient(config)
	mqttClient.Init()
}
