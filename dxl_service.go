package dxl

type ServiceRegistrationInfo struct {
	Service_type             string                 `json:"_service_type"`
	Service_id               string                 `json:"_service_id"`
	TTL                      int                    `json:"_ttl"`
	TTL_lower_limit          int                    `json:"_ttl_lower_limit"`
	Destination_tenant_guids []string               `json:"_destination_tenant_guids"`
	Metadata                 map[string]interface{} `json:"metadata"`
	Callbacks_by_topic       map[string]interface{} `json:"_callbacks_by_topic"`
}

func NewServiceRegistrationInfo(service_type string) *ServiceRegistrationInfo {

	sri := &ServiceRegistrationInfo{
		Service_type:    service_type,
		Service_id:      getUUID(),
		TTL:             60,
		TTL_lower_limit: 10,
	}

	sri.Metadata = make(map[string]interface{})
	sri.Callbacks_by_topic = make(map[string]interface{})
	sri.Destination_tenant_guids = []string{}

	return sri
}
