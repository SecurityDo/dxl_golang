package examples

import "testing"
import "time"
import "fmt"
import mqtt_client "mqtt"
import dxl "dxl_client"

func TestService(*testing.T) {
	//BrokerCertChain=/tmp/dxl_test/testDxlClient/brokercerts.crt
	//CertFile=/tmp/dxl_test/testDxlClient/client.crt
	//PrivateKey=/tmp/dxl_test/testDxlClient/client.key

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
	/*
		callbackFunc := func(input string) (output string, err error) {
			//log.Println(cli, message)
			fmt.Println("input: ", input)
			return "Echo from server:" + input, nil
		}

		dxlClient.RegisterTopics("/mycompany/myservice", []string{"/isecg/sample/service"}, 10, callbackFunc)
	*/

	callbackFunc := func(input string) (output string, err error) {
		//log.Println(cli, message)
		fmt.Println("input: ", input)
		return "Echo from server:" + input, nil
	}

	dxlClient.RegisterTopics("/mcafee/service/epo/remote", []string{"/mcafee/service/epo/remote/epo1"}, 10, callbackFunc)

	time.Sleep(10000 * time.Millisecond)
	fmt.Println("Close dxlClient")
	dxlClient.Stop()
	time.Sleep(5000 * time.Millisecond)

}
