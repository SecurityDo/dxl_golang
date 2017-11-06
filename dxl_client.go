package dxl

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	mqtt_client "github.com/SecurityDo/dxl_golang/mqtt"

	mq "github.com/eclipse/paho.mqtt.golang"
	uuid "github.com/nu7hatch/gouuid"
)

type DXLClient struct {
	rpcManager     *RpcManager
	mqttClient     *mqtt_client.MqttClient
	reply_to_topic string
	clientId       string
	qos            byte
	stopChan       chan struct{}
	replyChan      chan mq.Message

	// service related
	topicHandlerMap map[string]DXLServiceCallback
	serviceGuid     string
	serviceType     string
	serviceTopics   []string
	expireEpoch     int64
	serviceTTL      int64
	ticker          *time.Ticker
	serviceChan     chan mq.Message
}

type ServiceResponse struct {
	response *Response
	errRes   *ErrorResponse
	err      error
}

type DXLServiceCallback func(input string) (output string, err error)

const (
	//# The default "reply-to" prefix. self is typically used for setting up response
	//# channels for requests, etc.
	_REPLY_TO_PREFIX = "/mcafee/client/"

	DXL_SERVICE_UNREGISTER_REQUEST_CHANNEL = "/mcafee/service/dxl/svcregistry/unregister"
	//# The channel notified when services are registered
	DXL_SERVICE_REGISTER_CHANNEL = "/mcafee/event/dxl/svcregistry/register"
	//# The channel to publish a service registration request to
	DXL_SERVICE_REGISTER_REQUEST_CHANNEL = "/mcafee/service/dxl/svcregistry/register"
	//# The channel notified when services are unregistered
	DXL_SERVICE_UNREGISTER_CHANNEL = "/mcafee/event/dxl/svcregistry/unregister"
)

/*
   # The default wait time for a synchronous request, defaults to 1 hour
   _DEFAULT_WAIT = 60 * 60
   # The default wait for policy delay (in seconds)
   _DEFAULT_WAIT_FOR_POLICY_DELAY = 2
   # The default minimum amount of threads in a thread pool
   _DEFAULT_MIN_POOL_SIZE = 10
   # The default maximum amount of threads in a thread pool
   _DEFAULT_MAX_POOL_SIZE = 25
   # The default quality of server (QOS) for messages
   _DEFAULT_QOS = 0
   # The default connect wait
   _DEFAULT_CONNECT_WAIT = 10  # seconds
*/

func NewDXLClient(config *mqtt_client.MqttClientConfig, qos byte) *DXLClient {
	r := new(DXLClient)
	r.clientId = getUUID()
	r.reply_to_topic = _REPLY_TO_PREFIX + r.clientId
	//r.qos = qos
	r.qos = 0
	r.stopChan = make(chan struct{})
	r.topicHandlerMap = make(map[string]DXLServiceCallback)
	r.replyChan = make(chan mq.Message, 100)
	r.serviceChan = make(chan mq.Message, 100)

	r.mqttClient = mqtt_client.NewMqttClient(config)
	r.rpcManager = newRpcManager()

	return r
}

func getUUID() string {
	uuidStruct, _ := uuid.NewV4()
	return "{" + uuidStruct.String() + "}"

}

func (r *DXLClient) Stop() {
	close(r.stopChan)
}

func (r *DXLClient) cleanup() {
	r.unregisterService()
	r.mqttClient.Close()
}

func (r *DXLClient) sendErrResponse(request *Request, error_code int, error_message string) error {

	response := NewErrorResponse(request, error_code, error_message)
	response.Payload = ""
	b := response.to_bytes()
	fmt.Println("reply to: ", request.Reply_to_topic)
	err := r.mqttClient.PublishMessage(request.Reply_to_topic, b, r.qos, false)
	if err != nil {
		fmt.Println("DXLClient:Service -- failed to send error response to topic: " + request.Reply_to_topic)
		return err
	}
	return nil
}

func (r *DXLClient) serviceRequestThread(request *Request) {
	fmt.Println("service Id: ", request.Service_id)
	fmt.Println("reply_topic", request.Reply_to_topic)
	fmt.Println("dest_topic", request.Destination_topic)

	handler := r.topicHandlerMap[request.Destination_topic]
	if handler == nil {
		fmt.Println("unknown topic")
		r.sendErrResponse(request, 0x80000001, "unable to locate service for request")
		return
	}
	var err error
	response := NewResponse(request)
	//response.Payload = "Sample response from dxl client!"
	response.Payload, err = handler(request.Payload)
	if err != nil {
		fmt.Println("internal error: ", err.Error())
		r.sendErrResponse(request, 0x90000001, err.Error())
		return
	}
	b := response.to_bytes()

	fmt.Println("reply to: ", request.Reply_to_topic)
	err = r.mqttClient.PublishMessage(request.Reply_to_topic, b, r.qos, false)
	if err != nil {
		fmt.Println("DXLClient:Service -- failed to send response to topic: " + request.Reply_to_topic)
		return
	}
}

func (r *DXLClient) serviceRequestHandler(message mq.Message) {
	fmt.Println("got service request message:", message.Topic(), message.MessageID())
	request, err := UnpackRequest(message.Payload())
	request.Destination_topic = message.Topic()
	if err != nil {
		fmt.Println("failed to unpack request message: ", err.Error())
		return
	}
	go r.serviceRequestThread(request)
}

func (r *DXLClient) replyMessageHandle(message mq.Message) {
	fmt.Println("got subscribed message:", message.Topic(), message.MessageID())
	response, err, errResponse := UnpackResponse(message.Payload())
	if err != nil {
		fmt.Println("invalid response: ", err.Error())
	} else if response != nil {
		fmt.Println("request message id: ", response.Request_message_id)
		r.rpcManager.replyHandle(response.Request_message_id, &ServiceResponse{
			response: response,
		})
	} else if errResponse != nil {
		fmt.Println("ERROR: request message id: ", errResponse.Request_message_id)
		fmt.Println("Error Response: Code: ", errResponse.Error_code, " message: ", errResponse.Error_message)
		r.rpcManager.replyHandle(errResponse.Request_message_id, &ServiceResponse{
			errRes: errResponse,
		})
	} else {
		fmt.Println("unreachable!")
	}
}

func (r *DXLClient) heartbeat() {
	cur := time.Now().Unix()
	if r.expireEpoch > 0 && cur > r.expireEpoch {
		fmt.Println("Re-Register Service: ", time.Now())
		go r.registerService()
	}
}
func (r *DXLClient) listenToReply() {
	for {
		select {
		case <-r.stopChan:
			fmt.Println("received stop command!")
			r.cleanup()
			return
		case <-r.ticker.C:
			fmt.Println("heartbeat")
			r.heartbeat()
		case message := <-r.replyChan:
			r.replyMessageHandle(message)
		case serviceReqMessage := <-r.serviceChan:
			r.serviceRequestHandler(serviceReqMessage)
		}
	}
}

func (r *DXLClient) Init() (err error) {
	err = r.mqttClient.Init()
	if err != nil {
		fmt.Println("failed to connect to mqtt server:", err.Error())
		return err
	}

	err = r.mqttClient.Subscribe(r.reply_to_topic, r.qos, r.replyChan)
	if err != nil {
		fmt.Println("failed to connect to mqtt server:", err.Error())
		return err
	}
	r.ticker = time.NewTicker(time.Minute)
	go r.listenToReply()
	return nil
}

func (r *DXLClient) Call(serviceTopic string, payload string, timeoutSec int) (data string, err error) {
	request := NewRequest(serviceTopic)
	request.Payload = payload

	request.Reply_to_topic = r.reply_to_topic

	b := request.to_bytes()
	err = r.mqttClient.PublishMessage(serviceTopic, b, r.qos, false)
	if err != nil {
		fmt.Println("DXLClient:Call -- failed to publish to topic: " + serviceTopic)
		return "", err
	}
	// save to rpc_manager
	c := make(chan *ServiceResponse, 1)

	r.rpcManager.enqueueRPC(request.Message_id, c)

	timeout := time.After(time.Duration(timeoutSec) * time.Second)
	select {
	case result := <-c:
		if result.err != nil {
			return "", result.err
		}
		if result.errRes != nil {
			return "", errors.New(result.errRes.Error_message)
		}
		return result.response.Payload, nil
	case <-timeout:
		fmt.Println("DXLCall timeout: ", serviceTopic)
		return "", errors.New("DXLCall Timeout!")
	}
	return "", err
}

type ServiceRegistrationRequest struct {
	ServiceType     string                 `json:"serviceType"`
	ServiceGuid     string                 `json:"serviceGuid"`
	TTLMins         int64                  `json:"ttlMins"`
	RequestChannels []string               `json:"requestChannels"`
	MetaData        map[string]interface{} `json:"metaData"`
}

func (r *DXLClient) registerRequestPayload() string {
	regReq := &ServiceRegistrationRequest{
		ServiceType:     r.serviceType,
		ServiceGuid:     r.serviceGuid,
		TTLMins:         r.serviceTTL,
		RequestChannels: r.serviceTopics,
		MetaData:        make(map[string]interface{}),
	}

	b, _ := json.Marshal(regReq)
	fmt.Println("registration request payload: ", string(b))
	return string(b)
}

type ServiceUnRegistrationRequest struct {
	ServiceGuid string `json:"serviceGuid"`
}

func (r *DXLClient) unregisterRequestPayload() string {
	regReq := &ServiceUnRegistrationRequest{
		ServiceGuid: r.serviceGuid,
	}
	b, _ := json.Marshal(regReq)
	fmt.Println("unregistration request payload: ", string(b))
	return string(b)
}

func (r *DXLClient) unregisterService() (err error) {
	data, err := r.Call(DXL_SERVICE_UNREGISTER_REQUEST_CHANNEL, r.unregisterRequestPayload(), 10)
	if err != nil {
		fmt.Println("failed to unregister dxl service: ", err.Error())
		return err
	}
	fmt.Println("unregister request OK:", data)
	return nil
}
func (r *DXLClient) registerService() (err error) {

	data, err := r.Call(DXL_SERVICE_REGISTER_REQUEST_CHANNEL, r.registerRequestPayload(), 10)
	if err != nil {
		fmt.Println("failed to register dxl service: ", err.Error())
		return err
	}
	fmt.Println("Register request OK:", data)
	r.expireEpoch = time.Now().Unix() + (r.serviceTTL-5)*60
	return nil

}

// only call once
func (r *DXLClient) RegisterTopics(service_type string, serviceTopics []string, ttlMin int64, handler DXLServiceCallback) (err error) {

	if ttlMin < 10 {
		return errors.New("TTLMin value too low!")
	}
	if r.serviceType != "" {
		return errors.New("ServiceType already registered!")
	}

	var qosList []byte
	for _, topic := range serviceTopics {
		r.topicHandlerMap[topic] = handler
		qosList = append(qosList, 0)
	}
	r.serviceType = service_type
	r.serviceTopics = serviceTopics
	r.serviceGuid = getUUID()
	r.serviceTTL = ttlMin
	err = r.registerService()
	if err != nil {
		fmt.Println(err.Error())
		return err
	} else {
		r.mqttClient.SubscribeTopics(serviceTopics, qosList, r.serviceChan)
	}

	return err
}

func parseBrokerList(filename string) []string {
	var keylist []string
	file, e := os.Open(filename)
	defer file.Close()
	if e != nil {
		println("File error: " + e.Error())
		return keylist
	}

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		lineBytes := scanner.Bytes()

		//{be1f857e-2f7e-11e7-0d40-000c2988b1ca}={be1f857e-2f7e-11e7-0d40-000c2988b1ca};8883;localhost;192.168.2.104

		line := string(lineBytes)
		var sl []string
		sl = strings.Split(line, ";")
		if len(sl) != 4 {
			continue // reject line
		}
		//[]string{"tls://192.168.2.104:8883"}
		keylist = append(keylist, "tls://"+sl[3]+":"+sl[1])
		//println("tls://" + sl[3] + ":" + sl[1])
	}

	if e := scanner.Err(); e != nil {
		println("Scanner error: " + e.Error())
		return keylist
	}

	return keylist
}

func LoadDXLClientConfig(cDir string) (config *mqtt_client.MqttClientConfig, err error) {
	config = &mqtt_client.MqttClientConfig{
		SkipVerify: true,
		TLSEnable:  true,

		RootCAFile:     cDir + "/brokercerts.crt",
		ClientCertFile: cDir + "/client.crt",
		ClientKeyFile:  cDir + "/client.key",
	}
	config.ServerURLs = parseBrokerList(cDir + "/brokerlist.properties")
	return config, nil
}
