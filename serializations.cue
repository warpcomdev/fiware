// Autogenerated file - DO NOT EDIT

name:       string
subservice: string
entityTypes?: [...#EntityType]
entities?: [...#Entity]
serviceMappings?: [...#ServiceMapping]
environment?: #Environment
subscriptions?: [string]: #Subscription
registrations?: [...#Registration]
tables?: [...#Table]
views?: [...#View]
services?: [...#Service]
devices?: [...#Device]
rules?: [string]: #Rule
projects?: [...#Project]
panels?: [string]:    #UrboPanel
verticals?: [string]: #UrboVertical

#EntityType: {
	entityID?:  string
	entityType: string
	attrs: [...#Attribute]
}

#Attribute: {
	name: string
	type: string
	description?: [...string]
	value?:        #Json
	metadatas?:    #Json
	singletonKey?: bool
	simulated?:    bool
	longterm?:     string
	longtermOptions?: [...string]
}

#Entity: {
	entityID:   string
	entityType: string
	attrs: [string]:      #Json
	metadatas?: [string]: #Json
}

#ServiceMapping: {
	originalService?: string
	newService?:      string
	servicePathMappings: [...#ServicePathMapping]
}

#ServicePathMapping: {
	originalServicePath?: string
	newServicePath?:      string
	entityMappings: [...#EntityMapping]
}

#EntityMapping: {
	originalEntityId?:   string
	newEntityId?:        string
	originalEntityType?: string
	newEntityType?:      string
	attributeMappings: [...#AttributeMapping]
}

#AttributeMapping: {
	originalAttributeName?: string
	originalAttributeType?: string
	newAttributeName?:      string
	newAttributeType?:      string
}

#Environment: {
	notificationEndpoints: [string]: string
}

#Subscription: {
	description:  string
	status?:      string
	expires?:     string
	notification: #Notification
	subject:      #Subject
	id?:          string @anonymous(SubscriptionStatus)
}

#Notification: {
	attrs?: [...string]
	exceptAttrs?: [...string]
	attrsFormat:        string
	http?:              #NotificationHTTP
	httpCustom?:        #NotificationCustom
	mqtt?:              #NotificationMQTT
	mqttCustom?:        #NotificationMQTTCustom
	onlyChangedAttrs?:  bool
	covered?:           bool
	lastFailure?:       string @anonymous(NotificationStatus)
	lastFailureReason?: string @anonymous(NotificationStatus)
	lastNotification?:  string @anonymous(NotificationStatus)
	lastSuccess?:       string @anonymous(NotificationStatus)
	lastSuccessCode?:   int    @anonymous(NotificationStatus)
	failsCounter?:      int    @anonymous(NotificationStatus)
	timesSent?:         int    @anonymous(NotificationStatus)
}

#NotificationHTTP: {
	url:      string
	timeout?: int
}

#NotificationCustom: {
	url:      string
	timeout?: int
	headers?: [string]: string
	qs?: [string]:      string
	method?:  string
	payload?: #Json
	json?:    #Json
	ngsi?:    #Json
}

#NotificationMQTT: {
	url:       string
	string:    string
	qos?:      string
	user?:     string
	password?: string
}

#NotificationMQTTCustom: {
	url:       string
	string:    string
	qos?:      string
	user?:     string
	password?: string
	payload?:  #Json
	json?:     #Json
	ngsi?:     #Json
}

#Subject: {
	condition: #SubjectCondition
	entities: [...#SubjectEntity]
}

#SubjectCondition: {
	attrs: [...string]
	expression?: #SubjectExpression
}

#SubjectExpression: {
	q?: string
}

#SubjectEntity: {
	idPattern?: string
	type:       string
}

#Registration: {
	id:            string
	description:   string
	dataProvided?: #Json
	provider?:     #Json
	status:        string @anonymous(RegistrationStatus)
}

#Table: {
	name: string
	columns: [...#TableColumn]
	primaryKey: [...string]
	indexes: [...#TableIndex]
	lastdata: bool
	singleton?: [...string]
}

#TableColumn: {
	name:     string
	type:     string
	notNull?: bool
	default?: string
}

#TableIndex: {
	name: string
	columns: [...string]
	geometry?: bool
}

#View: {
	materialized?: bool
	name:          string
	from:          string
	group: [...string]
	columns: [...#ViewColumn]
}

#ViewColumn: {
	name:       string
	expression: string
}

#Service: {
	resource:       string
	apikey:         string
	entity_type:    string
	description?:   string
	protocol:       string
	transport?:     string
	timestamp?:     bool
	explicitAttrs?: #Json
	internal_attributes?: [...#DeviceAttribute]
	attributes: [...#DeviceAttribute]
	lazy?: [...#DeviceAttribute]
	static_attributes?: [...#DeviceAttribute]
	commands?: [...#DeviceCommand]
	expressionLanguage?: string
	entityNameExp?:      string
	_id?:                string @anonymous(GroupStatus)
	iotagent?:           string @anonymous(GroupStatus)
	service_path?:       string @anonymous(GroupStatus)
	service?:            string @anonymous(GroupStatus)
	cbHost?:             string @anonymous(GroupStatus)
}

#DeviceAttribute: {
	object_id:    string
	name:         string
	type?:        string
	value?:       #Json
	expression?:  string
	entity_name?: string
	entity_type?: string
}

#DeviceCommand: {
	object_id?: string
	name?:      string
	type?:      string
	value?:     string
	mqtt?:      #Json
}

#Device: {
	device_id:    string
	apikey?:      string
	entity_name?: string
	entity_type:  string
	polling?:     bool
	transport:    string
	timestamp?:   bool
	endpoint?:    string
	attributes?: [...#DeviceAttribute]
	lazy?: [...#DeviceAttribute]
	commands?: [...#DeviceCommand]
	static_attributes?: [...#DeviceAttribute]
	protocol:            string
	expressionLanguage?: string
	explicitAttrs?:      #Json
	service?:            string @anonymous(DeviceStatus)
	service_path?:       string @anonymous(DeviceStatus)
}

#Rule: {
	name:         string
	description?: string
	misc?:        string
	text?:        string
	VR?:          string
	action?:      #Json
	nosignal?:    #Json
	subservice?:  string @anonymous(RuleStatus)
	service?:     string @anonymous(RuleStatus)
	_id?:         string @anonymous(RuleStatus)
}

#Project: {
	is_domain:    bool
	description?: string
	tags?:        #Json
	enabled:      bool
	id:           string
	parent_id?:   string
	domain_id?:   string
	name:         string
	links?:       #Json @anonymous(ProjectStatus)
}

#UrboPanel: {
	name:           string
	description?:   string
	slug:           string
	lowercaseSlug?: string
	widgetCount?:   int
	isShadow?:      bool
	section?:       string
}

#UrboVertical: {
	panels?: [...#UrboPanel]
	shadowPanels?: [...#UrboPanel]
	i18n?: #Json
	name:  string
	slug:  string
}

#Json: _ // cuaquier cosa...
