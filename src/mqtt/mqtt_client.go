package mqtt_client

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/nu7hatch/gouuid"

	mq "github.com/eclipse/paho.mqtt.golang"
)

type MqttClientConfig struct {
	ServerURLs     []string // []string{"tls://192.168.2.104:8883"},
	RootCAFile     string
	ClientKeyFile  string
	ClientCertFile string
	SkipVerify     bool
	TLSEnable      bool
	ClientId       string
}

type MqttClient struct {
	mqClient mq.Client
	config   *MqttClientConfig
	//errorChan chan<- error
}

func NewMqttClient(config *MqttClientConfig) *MqttClient {
	client := new(MqttClient)

	options := &mq.ClientOptions{}
	for _, url := range config.ServerURLs {
		options = options.AddBroker(url) // You need to get your specific endpoint using the describe-endpoint aws cli command
	}
	if config.ClientId != "" {
		options.SetClientID(config.ClientId)
	} else {
		options.SetClientID(getUUID())
	}
	options.SetCleanSession(true)
	// SetCleanSession(true) is very important, as is QoS = 1, not 0.
	//options.SetDefaultPublishHandler(subcallback)
	//options.OnConnectionLost = client.onConnectionLost
	options.SetAutoReconnect(true)
	if config.TLSEnable {
		options.SetTLSConfig(NewTlsConfig(config))
	}
	//todo:
	client.mqClient = mq.NewClient(options)
	client.config = config
	return client
}

func (r *MqttClient) Init() error {
	if token := r.mqClient.Connect(); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
		return token.Error()
	}
	return nil
}

func (r *MqttClient) Close() {
	if r.mqClient.IsConnected() {
		r.mqClient.Disconnect(5)
		fmt.Println("mqtt client disconnected!")
	}
}
func getUUID() string {
	uuidStruct, _ := uuid.NewV4()
	return uuidStruct.String()
}

func NewTlsConfig(config *MqttClientConfig) *tls.Config {
	// Import trusted certificates from CAfile.pem.
	// Alternatively, manually add CA certificates to
	// default openssl CA bundle.
	certpool := x509.NewCertPool()
	pemCerts, err := ioutil.ReadFile(config.RootCAFile)
	if err == nil {
		certpool.AppendCertsFromPEM(pemCerts)
	}

	// Import client certificate/key pair
	cert, err := tls.LoadX509KeyPair(config.ClientCertFile, config.ClientKeyFile)

	if err != nil {
		log.Println(err)
		panic(err)
	}

	// Just to print out the client certificate..
	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		panic(err)
	}
	//log.Println(cert.Leaf)

	// Create tls.Config with desired tls properties
	return &tls.Config{
		// RootCAs = certs used to verify server cert.
		RootCAs: certpool,
		// ClientAuth = whether to request cert from server.
		// Since the server is set up for SSL, this happens
		// anyways.
		ClientAuth: tls.NoClientCert,
		// ClientCAs = certs used to validate client cert.
		ClientCAs: nil,
		// InsecureSkipVerify = verify that cert contents
		// match server. IP matches what is in cert etc.
		InsecureSkipVerify: config.SkipVerify,
		// Certificates = list of certs client sends to server.
		Certificates: []tls.Certificate{cert},
	}
}

func (r *MqttClient) PublishMessage(topic string, data []byte, qos byte, retained bool) (err error) {

	if token := r.mqClient.Publish(topic, qos, retained, data); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		return err
	}
	return nil

}

/*
func subcallback(cli mq.Client, message mq.Message) {
	log.Println(cli, message)
	fmt.Println("sub callback")
}*/

func (r *MqttClient) Subscribe(topic string, qos byte, c chan<- mq.Message) (err error) {
	callbackFunc := func(cli mq.Client, message mq.Message) {
		//log.Println(cli, message)
		fmt.Println("sub_topic callback")
		c <- message
	}

	if token := r.mqClient.Subscribe(topic, qos, callbackFunc); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		return err
	}
	return nil
}

func (r *MqttClient) SubscribeTopics(topics []string, qos []byte, c chan<- mq.Message) (err error) {
	if len(topics) != len(qos) {
		fmt.Println("invalid arguments!")
		return errors.New("invalid arguments!")
	}
	filters := make(map[string]byte)

	for i, topic := range topics {
		filters[topic] = qos[i]
	}

	callbackFunc := func(cli mq.Client, message mq.Message) {
		//log.Println(cli, message)
		fmt.Println("sub_topics callback")
		c <- message
	}

	if token := r.mqClient.SubscribeMultiple(filters, callbackFunc); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		return err
	}
	return nil

}
