// El paquete deployer define los modelos de datos básicos de urbo-deployer.
// El principal modelo es el de Vertical, que es el punto de entrada a
// partir del cual se accede al conjunto de datos de la vertical. El resto
// de tipos corresponden a sub-atributos dentro de la vertical.
package fiware

import (
	"encoding/json"
)

// Tipos de datos que se usan para relacionarse con la vertical

// Vertical representa una vertical
type Vertical struct {
	Name       string `json:"name"`       // `tourism`, `wifi`, `watermeter`, etc
	Subservice string `json:"subservice"` // `turismo`, `wifi`, `contadores`, etc.
	// Tipos de entidad definidos en la vertical.
	// El ID y los valores de los atributos son opcionales.
	EntityTypes []Entity `json:"entityTypes,omitempty"`
	// ServiceMappings para cygnus
	ServiceMappings []ServiceMapping `json:"serviceMappings,omitempty"`
	// Suscripciones al context broker
	Suscriptions []Suscription `json:"suscriptions,omitempty"`
	// Tablas *sencillas* relacionadas con entidades
	Tables []Table `json:"tables,omitempty"`
	// Grupos de dispositivos
	Services []Service `json:"services,omitempty"`
	Devices  []Device  `json:"devices,omitempty"`
	// CEP rules
	Rules []Rule `json:"rules,omitempty"`
}

// Entity representa una entidad
type Entity struct {
	ID   string `json:"entityID"`
	Type string `json:"entityType"`
	// Usamos una lista en vez de un map para poder
	// establecer un orden específico, por si nos interesa
	// conservar el orden de atributos para algo.
	Attrs []Attribute `json:"attrs"`
}

// Attribute representa un atributo de una entidad
type Attribute struct {
	Name  string          `json:"name"`
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value,omitempty"`
}

// ServiceMapping es cada uno de los serviceMappings de cygnus
type ServiceMapping struct {
	OriginalService     string               `json:"originalService,omitempty"`
	NewService          string               `json:"newService,omitempty"`
	ServicePathMappings []ServicePathMapping `json:"servicePathMappings"`
}

// ServicePathMapping es cada uno de los servicePathMappings de un serviceMapping
type ServicePathMapping struct {
	OriginalServicePath string          `json:"originalServicePath,omitempty"`
	NewServicePath      string          `json:"newServicePath,omitempty"`
	EntityMappings      []EntityMapping `json:"entityMappings"`
}

// EntityMaping es cada uno de los EntityMappings de un servicePathMapping
type EntityMapping struct {
	OriginalEntityId   string             `json:"originalEntityId,omitempty"`
	NewEntityId        string             `json:"newEntityId,omitempty"`
	OriginalEntityType string             `json:"originalEntityType,omitempty"`
	NewEntityType      string             `json:"newEntityType,omitempty"`
	AttributeMappings  []AttributeMapping `json:"attributeMappings"`
}

// AttributeMapping es cada uno de los AttributeMappings de un EntityMapping
type AttributeMapping struct {
	OriginalAttributeName string `json:"originalAttributeName,omitempty"`
	OriginalAttributeType string `json:"originalAttributeType,omitempty"`
	NewAttributeName      string `json:"newAttributeName,omitempty"`
	NewAttributeType      string `json:"newAttributeType,omitempty"`
}

// Suscription representa una suscripcion
type Suscription struct {
	Description  string       `json:"description"`
	Status       string       `json:"status,omitempty"`
	Notification Notification `json:"notification"`
	Subject      Subject      `json:"subject"`
	SuscriptionStatus
}

// SuscriptionStatus agrupa los datos de estado de la suscripción
type SuscriptionStatus struct {
	ID string `json:"id,omitempty"`
}

// Notification es la configuración de notificación de la suscripción
type Notification struct {
	Attrs            []string            `json:"attrs,omitempty"`
	AttrsFormat      string              `json:"attrsFormat"`
	HTTP             *NotificationHTTP   `json:"http,omitempty"`
	HTTPCustom       *NotificationCustom `json:"httpCustom,omitempty"`
	OnlyChangedAttrs bool                `json:"onlyChangedAttrs,omitempty"`
	NotificationStatus
}

// NotificationStatus agrupa los datos de estado de la suscripción
type NotificationStatus struct {
	LastFailure       string `json:"lastFailure,omitempty"`
	LastFailureReason string `json:"lastFailureReason,omitempty"`
	LastNotification  string `json:"lastNotification,omitempty"`
	LastSuccess       string `json:"lastSuccess,omitempty"`
	LastSuccessCode   int    `json:"lastSuccessCode,omitempty"`
	TimesSent         int    `json:"timesSent,omitempty"`
}

// NotificationHTTP son los datos de una notificacion
type NotificationHTTP struct {
	URL string `json:"url"`
}

// NotificationHTTP son los datos de una notificacion
type NotificationCustom struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

// Subject es el sujeto de la suscripcion
type Subject struct {
	Condition SubjectCondition `json:"condition"`
	Entities  []SubjectEntity  `json:"entities"`
}

// SubjectCondition es la condicion del sujeto de la suscripcion
type SubjectCondition struct {
	Attrs      []string           `json:"attrs,omitempty"`
	Expression *SubjectExpression `json:"expression,omitempty"`
}

// SubjectExpression es la expresion en la condicion
type SubjectExpression struct {
	Q string `json:"q,omitempty"`
}

// SubjectEntity es la entidad sujeto de la suscripcion
type SubjectEntity struct {
	IdPattern string `json:"idPattern,omitempty"`
	Type      string `json:"type"`
}

// Table define algunos parámetros básicos de tablas a crear
type Table struct {
	Name       string        `json:"name"`
	Columns    []TableColumn `json:"columns"`
	PrimaryKey []string      `json:"primaryKey"`
	Indexes    []TableIndex  `json:"indexes"`
	LastData   bool          `json:"lastdata"` // True si queremos crear una vista lastdata adicional
}

// TableColumn describe una columna de una tabla
type TableColumn struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	NotNull bool   `json:"notNull,omitempty"`
	Default string `json:"default,omitempty"`
}

// TableIndex describe un indice
type TableIndex struct {
	Name     string   `json:"name"`
	Columns  []string `json:"columns"`
	Geometry bool     `json:"geometry,omitempty"`
}

// Service describe la provisión de un grupo de dispositivos
type Service struct {
	Resource           string            `json:"resource"`
	APIKey             string            `json:"apikey"`
	EntityType         string            `json:"entity_type"`
	Description        string            `json:"description,omitempty"`
	Protocol           string            `json:"protocol"`
	Transport          string            `json:"transport,omitempty"`
	Timestamp          bool              `json:"timestamp,omitempty"`
	ExplicitAttrs      bool              `json:"explicitAttrs,omitempty"`
	InternalAttributes []DeviceAttribute `json:"internal_attributes,omitempty"`
	Attributes         []DeviceAttribute `json:"attributes"`
	Lazy               []DeviceAttribute `json:"lazy,omitempty"`
	StaticAttributes   []DeviceAttribute `json:"static_attributes,omitempty"`
	Commands           []DeviceCommand   `json:"commands,omitempty"`
	ExpressionLanguage string            `json:"expressionLanguage,omitempty"`
	GroupStatus
}

// GroupStatus agrupa atributos de estado que no se usan al crear un Service
type GroupStatus struct {
	ID          string `json:"_id,omitempty"`
	V           int    `json:"__v,omitempty"`
	IOTAgent    string `json:"iotagent,omitempty"`
	ServicePath string `json:"service_path,omitempty"`
	Service     string `json:"service,omitempty"`
	CBHost      string `json:"cbHost,omitempty"`
}

// Device representa un dispositivo
type Device struct {
	DeviceId           string            `json:"device_id"`
	APIKey             string            `json:"apikey,omitempty"`
	EntityName         string            `json:"entity_name,omitempty"`
	EntityType         string            `json:"entity_type"`
	Polling            bool              `json:"polling,omitempty"`
	Transport          string            `json:"transport"`
	Timestamp          bool              `json:"timestamp,omitempty"`
	Endpoint           string            `json:"endpoint,omitempty"`
	Attributes         []DeviceAttribute `json:"attributes,omitempty"`
	Lazy               []DeviceAttribute `json:"lazy,omitempty"`
	Commands           []DeviceCommand   `json:"commands,omitempty"`
	StaticAttributes   []DeviceAttribute `json:"static_attributes,omitempty"`
	Protocol           string            `json:"protocol"`
	ExpressionLanguage string            `json:"expressionLanguage,omitempty"`
	ExplicitAttrs      bool              `json:"explicitAttrs,omitempty"`
	DeviceStatus
}

// GroupStatus agrupa atributos de estado que no se usan al crear un Device
type DeviceStatus struct {
	Service     string `json:"service,omitempty"`
	ServicePath string `json:"service_path,omitempty"`
}

// DeviceAttribute describe un atributo de dispositivo
type DeviceAttribute struct {
	ObjectId   string `json:"object_id"`
	Name       string `json:"name"`
	Type       string `json:"type,omitempty"`
	Expression string `json:"expression,omitempty"`
	EntityName string `json:"entity_name,omitempty"`
	EntityType string `json:"entity_type"`
}

// DeviceCommand describe un comando de dispositivo
type DeviceCommand struct {
	ObjectId string `json:"object_id,omitempty"`
	Name     string `json:"name,omitempty"`
	Type     string `json:"type,omitempty"`
}

type Rule struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Misc        string          `json:"misc,omitempty"`
	Text        string          `json:"text,omitempty"`
	VR          string          `json:"VR,omitempty"`
	Action      json.RawMessage `json:"action,omitempty"`   // TODO: estructurar de alguna forma?
	NoSignal    json.RawMessage `json:"nosignal,omitempty"` // TODO: estructurar de alguna forma?
	RuleStatus
}

// RuleStatus agrupa atributos de estado que no se usan al crear una Rule
type RuleStatus struct {
	Subservice string `json:"subservice,omitempty"`
	Service    string `json:"service,omitempty"`
	ID         string `json:"_id,omitempty"`
}
