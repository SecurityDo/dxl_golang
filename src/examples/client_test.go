package examples

import "testing"
import "time"
import "fmt"
import mqtt_client "mqtt"
import dxl "dxl_client"

func TestClient(*testing.T) {
	config := &mqtt_client.MqttClientConfig{
		TLSEnable:      true,
		RootCAFile:     "/tmp/dxl_test/brokercerts.crt",
		ClientCertFile: "/tmp/dxl_test/client.crt",
		ClientKeyFile:  "/tmp/dxl_test/client.key",
		ServerURLs:     []string{"tls://192.168.2.104:8883"},
		SkipVerify:     true,
	}

	dxlClient := dxl.NewDXLClient(config, 1)

	dxlClient.Init()

	data, err := dxlClient.Call("/isecg/sample/service", "sample request id", 5)
	//p := "{\"serviceType\":\"/mycompany/myservice\",\"serviceGuid\":\"{9c4dc46c-42ed-49e6-4ea0-d0b96baa4167}\",\"ttlMins\":60,\"requestChannels\":[\"/isecg/sample/service\"],\"metaData\":{}}"
	//data, err := dxlClient.Call("/mcafee/service/dxl/svcregistry/register", p, 5)

	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("data:", data)
	}

	time.Sleep(10000 * time.Millisecond)
	fmt.Println("Close dxlClient")
	dxlClient.Stop()
	time.Sleep(2000 * time.Millisecond)
}
