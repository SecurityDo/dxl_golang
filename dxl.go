package dxl

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/nu7hatch/gouuid"
	"gopkg.in/vmihailenco/msgpack.v2"
)

/*
"""
The messages module contains the different types of messages that are transmitted
over the Data Exchange Layer (DXL) fabric.

- :class:`Event`
- :class:`Request`
- :class:`Response`
- :class:`ErrorResponse`

:class:`Event` messages are sent using the :func:`dxlclient.client.DxlClient.send_event` method of a client instance.
Event messages are sent by one publisher and received by one or more recipients that are currently
subscribed to the :attr:`Message.destination_topic` associated with the event (otherwise known as one-to-many).

:class:`Request` messages are sent using the :func:`dxlclient.client.DxlClient.sync_request` and
:func:`dxlclient.client.DxlClient.async_request` methods of a client instance. Request messages are used when
invoking a method on a remote service. This communication is one-to-one where a client sends a request to a
service instance and in turn receives a response.

:class:`Response` messages are sent by service instances upon receiving :class:`Request` messages. Response messages
are sent using the :func:`dxlclient.client.DxlClient.send_response` method of a client instance. Clients that are
invoking the service (sending a request) will receive the response as a return value of the
:func:`dxlclient.client.DxlClient.sync_request` method of a client instance or via the
:class:`dxlclient.callbacks.ResponseCallback` callback when invoking the asynchronous method,
:func:`dxlclient.client.DxlClient.async_request`.

:class:`ErrorResponse` messages are sent by the DXL fabric itself or service instances upon receiving :class:`Request`
messages. The error response may indicate the inability to locate a service to handle the request or an internal
error within the service itself. Error response messages are sent using the
:func:`dxlclient.client.DxlClient.send_response` method of a client instance.

**NOTE:** Some services may chose to not send a :class:`Response` message when receiving a :class:`Request`.
This typically occurs if the service is being used to simply collect information from remote clients. In this scenario,
the client should use the asynchronous form for sending requests,
:func:`dxlclient.client.DxlClient.async_request`
"""

*/
const (
	//The message version
	MESSAGE_VERSION = 2

	MESSAGE_TYPE_REQUEST = 0
	//The numeric type identifier for `Request` message type
	MESSAGE_TYPE_RESPONSE = 1
	//The numeric type identifier for the `Response` message type
	MESSAGE_TYPE_EVENT = 2
	//The numeric type identifier for the `Event` message type
	MESSAGE_TYPE_ERROR = 3
	//The numeric type identifier for the `ErrorResponse` message type
)

// The base class for the different Data Exchange Layer (DXL) message types
type Message struct {
	Version           int      `json:"_version"`
	Type              int      `json:"_type"`
	Message_id        string   `json:"_message_id"`
	Source_client_id  string   `json:"_source_client_id"`
	Source_broker_id  string   `json:"_source_broker_id"`
	Destination_topic string   `json:"_destination_topic"`
	Payload           string   `json:"_payload"`
	Broker_ids        []string `json:"_broker_ids"`
	Client_ids        []string `json:"_client_ids"`
}

func initMessage(m *Message, destination_topic string) {
	//# Version 0
	// The version of the message
	m.Version = MESSAGE_VERSION
	// The unique identifier for the message
	uuidStruct, _ := uuid.NewV4()
	m.Message_id = uuidStruct.String()
	//# The identifier for the client that is the source of the message
	m.Source_client_id = ""
	//# The GUID for the broker that is the source of the message
	m.Source_broker_id = ""
	//# The channel that the message is published on
	m.Destination_topic = destination_topic
	//# The payload to send with the message
	m.Payload = ""
	//# The set of broker GUIDs to deliver the message to
	m.Broker_ids = []string{}
	//# The set of client GUIDs to deliver the message to
	m.Client_ids = []string{}

	// Version 1
	//# Other fields: way to add fields to message types
	//m._other_fields = {}
	//# Version 2
	//# The GUID for the tenant that is the source of the message
	//m._source_tenant_guid = ""
	//# The set of tenant GUIDs to deliver the message to
	//m._destination_tenant_guids = []
}

/*
   :class:`Request` messages are sent using the :func:`dxlclient.client.DxlClient.sync_request` and
   :func:`dxlclient.client.DxlClient.async_request` methods of a client instance. Request messages are used when
   invoking a method on a remote service. This communication is one-to-one where a client sends a request to a
   service instance and in turn receives a response.
*/
type Request struct {
	Message
	Reply_to_topic string `json:"_reply_to_topic"`
	Service_id     string `json:"_service_id"`
}

func (m *Message) to_bytes(enc *msgpack.Encoder) {

	emptySlice := make([]string, 0)
	enc.EncodeInt(m.Version)
	enc.EncodeInt(m.Type)
	enc.EncodeString(m.Message_id)
	enc.EncodeString(m.Source_client_id)
	enc.EncodeString(m.Source_broker_id)
	enc.Encode(emptySlice)
	enc.Encode(emptySlice)
	enc.EncodeString(m.Payload)
}

func (m *Message) to_bytes_v1v2(enc *msgpack.Encoder) {

	emptySlice := make([]string, 0)
	if m.Version > 0 {
		enc.Encode(emptySlice) // other fields
	}
	if m.Version > 1 {
		enc.EncodeString("")   // _source_tenant_guid
		enc.Encode(emptySlice) // _destination_tenant_guids
	}

}

func (m *Message) unpack_stage2(dec *msgpack.Decoder) (err error) {
	m.Message_id, err = dec.DecodeString()
	if err != nil {
		fmt.Println("Failed to decode message_id: ", err.Error())
		return err
	}

	m.Source_client_id, err = dec.DecodeString()
	if err != nil {
		fmt.Println("Failed to decode source_client_id: ", err.Error())
		return err
	}

	m.Source_broker_id, err = dec.DecodeString()
	if err != nil {
		fmt.Println("Failed to decode source_broker_id: ", err.Error())
		return err
	}

	_, err = dec.DecodeSlice()
	if err != nil {
		fmt.Println("Failed to decode client_ids: ", err.Error())
		return err
	}

	_, err = dec.DecodeSlice()
	if err != nil {
		fmt.Println("Failed to decode broker_ids: ", err.Error())
		return err
	}

	m.Payload, err = dec.DecodeString()
	if err != nil {
		fmt.Println("Failed to decode payload: ", err.Error())
		return err
	}
	return nil

}
func (m *Message) unpack(dec *msgpack.Decoder) (err error) {

	m.Version, err = dec.DecodeInt()
	if err != nil {
		fmt.Println("Failed to decode version: ", err.Error())
		return err
	}
	m.Type, err = dec.DecodeInt()
	if err != nil {
		fmt.Println("Failed to decode type: ", err.Error())
		return err
	}

	return m.unpack_stage2(dec)
}

func (r *Request) to_bytes() []byte {
	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)

	(&r.Message).to_bytes(enc)
	enc.EncodeString(r.Reply_to_topic)
	enc.EncodeString(r.Service_id)
	(&r.Message).to_bytes_v1v2(enc)
	return buf.Bytes()
}

func NewRequest(destination_topic string) *Request {
	m := new(Request)
	initMessage(&m.Message, destination_topic)
	m.Type = MESSAGE_TYPE_REQUEST
	m.Reply_to_topic = ""
	m.Service_id = ""
	return m
}

/*
   :class:`Response` messages are sent by service instances upon receiving :class:`Request` messages. Response messages
   are sent using the :func:`dxlclient.client.DxlClient.send_response` method of a client instance. Clients that are
   invoking the service (sending a request) will receive the response as a return value of the
   :func:`dxlclient.client.DxlClient.sync_request` method of a client instance or via the
   :class:`dxlclient.callbacks.ResponseCallback` callback when invoking the asynchronous method,
   :func:`dxlclient.client.DxlClient.async_request`.

*/
type Response struct {
	Message
	// The identifier for the request message that this is a response for
	Request_message_id string   `json:"_request_message_id"`
	Service_id         string   `json:"_service_id"`
	request            *Request // The request (only available when sending the response)
}

func UnpackRequest(data []byte) (req *Request, err error) {
	buf := bytes.NewBuffer(data)
	dec := msgpack.NewDecoder(buf)

	version, err := dec.DecodeInt()
	if err != nil {
		fmt.Println("Failed to decode version: ", err.Error())
		return nil, err
	}
	mType, err := dec.DecodeInt()
	if err != nil {
		fmt.Println("Failed to decode type: ", err.Error())
		return nil, err
	}

	if mType != MESSAGE_TYPE_REQUEST {
		fmt.Println("invalid message type: ", mType)
		return nil, errors.New("Message type is not request!")
	}

	req = new(Request)
	req.Message.Version = version
	req.Message.Type = mType
	err = (&req.Message).unpack_stage2(dec)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	req.Reply_to_topic, err = dec.DecodeString()
	if err != nil {
		fmt.Println("Failed to decode Reply_to_topic: ", err.Error())
		return nil, err
	}
	req.Service_id, err = dec.DecodeString()
	if err != nil {
		fmt.Println("Failed to decode Service_id: ", err.Error())
		return nil, err
	}
	return req, nil

}

func (r *Response) to_bytes() []byte {
	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)

	(&r.Message).to_bytes(enc)
	enc.EncodeString(r.Request_message_id)
	enc.EncodeString(r.Service_id)
	(&r.Message).to_bytes_v1v2(enc)
	return buf.Bytes()
}

func UnpackResponse(data []byte) (res *Response, err error, errRes *ErrorResponse) {
	buf := bytes.NewBuffer(data)
	dec := msgpack.NewDecoder(buf)

	version, err := dec.DecodeInt()
	if err != nil {
		fmt.Println("Failed to decode version: ", err.Error())
		return nil, err, nil
	}
	mType, err := dec.DecodeInt()
	if err != nil {
		fmt.Println("Failed to decode type: ", err.Error())
		return nil, err, nil
	}

	if mType == MESSAGE_TYPE_RESPONSE {
		res = new(Response)
		res.Message.Version = version
		res.Message.Type = mType

		err = (&res.Message).unpack_stage2(dec)
		if err != nil {
			fmt.Println(err.Error())
			return nil, err, nil
		}
		res.Request_message_id, err = dec.DecodeString()
		if err != nil {
			fmt.Println("Failed to decode request_message_id: ", err.Error())
			return nil, err, nil
		}

		res.Service_id, err = dec.DecodeString()
		if err != nil {
			fmt.Println("Failed to decode service_id: ", err.Error())
			return nil, err, nil
		}
		return res, err, nil
	} else if mType == MESSAGE_TYPE_ERROR {
		errRes = new(ErrorResponse)
		errRes.Message.Version = version
		errRes.Message.Type = mType
		err = (&errRes.Message).unpack_stage2(dec)
		if err != nil {
			fmt.Println(err.Error())
			return nil, err, nil
		}

		errRes.Request_message_id, err = dec.DecodeString()
		if err != nil {
			fmt.Println("Failed to decode request_message_id: ", err.Error())
			return nil, err, nil
		}

		errRes.Service_id, err = dec.DecodeString()
		if err != nil {
			fmt.Println("Failed to decode service_id: ", err.Error())
			return nil, err, nil
		}

		errRes.Error_code, err = dec.DecodeInt()
		if err != nil {
			fmt.Println("Failed to decode error_code: ", err.Error())
			return nil, err, nil
		}
		errRes.Error_message, err = dec.DecodeString()
		if err != nil {
			fmt.Println("Failed to decode error_message: ", err.Error())
			return nil, err, nil
		}

		return nil, err, errRes
	}
	fmt.Println("unexpected message type: ", mType)
	return nil, errors.New("unexpected message type: "), nil
}

func initResponse(m *Response, request *Request) {

	m.request = request
	if request != nil {
		initMessage(&m.Message, request.Reply_to_topic)
		m.Request_message_id = request.Message_id
		m.Service_id = request.Service_id
		id := request.Source_client_id
		if id != "" {
			m.Client_ids = append(m.Client_ids, id)
		}
		id = request.Source_broker_id
		if id != "" {
			m.Broker_ids = append(m.Broker_ids, id)
		}
	} else {
		initMessage(&m.Message, "")
	}

}

func NewResponse(request *Request) *Response {
	m := new(Response)
	initResponse(m, request)
	m.Type = MESSAGE_TYPE_RESPONSE
	return m
}

/*
   :class:`Event` messages are sent using the :func:`dxlclient.client.DxlClient.send_event` method of a client instance.
   Event messages are sent by one publisher and received by one or more recipients that are currently
   subscribed to the :attr:`Message.destination_topic` associated with the event (otherwise known as one-to-many).
*/
type Event struct {
	Message
}

func NewEvent(destination_topic string) *Event {
	m := new(Event)
	m.Type = MESSAGE_TYPE_EVENT
	initMessage(&m.Message, destination_topic)
	return m
}

/*
   :class:`ErrorResponse` messages are sent by the DXL fabric itself or service instances upon receiving
   :class:`Request` messages. The error response may indicate the inability to locate a service to handle the
   request or an internal error within the service itself. Error response messages are sent using the
   :func:`dxlclient.client.DxlClient.send_response` method of a client instance.

*/

/*
   :param request: The :class:`Request` message that this is a response for
   :param error_code: The numeric error code
   :param error_message: The textual error message

*/
type ErrorResponse struct {
	Response
	Error_code    int
	Error_message string
}

func (r *ErrorResponse) to_bytes() []byte {
	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)

	(&r.Message).to_bytes(enc)
	enc.EncodeString(r.Request_message_id)
	enc.EncodeString(r.Service_id)
	enc.EncodeInt(r.Error_code)
	enc.EncodeString(r.Error_message)
	(&r.Message).to_bytes_v1v2(enc)
	return buf.Bytes()
}

func NewErrorResponse(request *Request, error_code int, error_message string) *ErrorResponse {
	m := new(ErrorResponse)
	initResponse(&m.Response, request)
	m.Type = MESSAGE_TYPE_ERROR
	m.Error_code = error_code
	m.Error_message = error_message
	return m
}
