// Autogenerated file - DO NOT EDIT

name:       string
subservice: string
entityTypes?: [...#EntityType]
entities?: [...#Entity]
serviceMappings?: [...#ServiceMapping]
suscriptions?: [...#Suscription]
tables?: [...#Table]
services?: [...#Service]
devices?: [...#Device]
rules?: [...#Rule]
projects: [...#Project]

#EntityType: {
	entityID:   string
	entityType: string
	attrs: [...#Attribute]
}

#Attribute: {
	name:       string
	type:       string
	value?:     #Json
	metadatas?: #Json
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

#Suscription: {
	description:  string
	status?:      string
	notification: #Notification
	subject:      #Subject
	id?:          string @anonymous(SuscriptionStatus)
}

#Notification: {
	attrs?: [...string]
	exceptAttrs?: [...string]
	attrsFormat:        string
	http?:              #NotificationHTTP
	httpCustom?:        #NotificationCustom
	onlyChangedAttrs?:  bool
	lastFailure?:       string @anonymous(NotificationStatus)
	lastFailureReason?: string @anonymous(NotificationStatus)
	lastNotification?:  string @anonymous(NotificationStatus)
	lastSuccess?:       string @anonymous(NotificationStatus)
	lastSuccessCode?:   int    @anonymous(NotificationStatus)
	failsCounter?:      int    @anonymous(NotificationStatus)
	timesSent?:         int    @anonymous(NotificationStatus)
}

#NotificationHTTP: {
	url: string
}

#NotificationCustom: {
	url: string
	headers?: [string]: string
}

#Subject: {
	condition: #SubjectCondition
	entities: [...#SubjectEntity]
}

#SubjectCondition: {
	attrs?: [...string]
	expression?: #SubjectExpression
}

#SubjectExpression: {
	q?: string
}

#SubjectEntity: {
	idPattern?: string
	type:       string
}

#Table: {
	name: string
	columns: [...#TableColumn]
	primaryKey: [...string]
	indexes: [...#TableIndex]
	lastdata: bool
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
	expression?:  string
	entity_name?: string
	entity_type?: string
}

#DeviceCommand: {
	object_id?: string
	name?:      string
	type?:      string
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
	explicitAttrs?:      bool
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

#Json: _ // cuaquier cosa...