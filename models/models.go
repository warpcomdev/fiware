// El paquete fiware define los modelos de datos básicos de warpcom-fiware
// El principal modelo es el de Vertical, que es el punto de entrada a
// partir del cual se accede al conjunto de datos de la vertical. El resto
// de tipos corresponden a sub-atributos dentro de la vertical.
package models

// Some bools in the IoTA APIs have different behaviour if they are
// undefined versus false. For instance, "timestamp === false" might
// not be the same as "timestamp === undefined", there is a global config
// parameter for that.
//
// For this reason, all those bools that are marked as `omitempty,omitzero`
// cannot be just omitted when set to 'false'. They must only be omitted
// if they are actually not defined.
//
// I achieve this using `*bool` instead of `bool` as the type for
// these settings, and using `omitempty` but not `omitzero`.

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Tipos de datos que se usan para relacionarse con la vertical.
// Todos los tipos deben implementar la interfaz `Serializable`,
// para poder ser exportados a diferentes formatos (json, jsonnet,
// starlark, etc).
// La interfaz se implementa automáticamente con el siguiente generador:
//go:generate go run ../cmd/generate/generate.go

// Manifest representa un manifiesto de vertical
type Manifest struct {
	Name       string `json:"name,omitempty,omitzero"`       // `tourism`, `wifi`, `watermeter`, etc
	Subservice string `json:"subservice,omitempty,omitzero"` // `turismo`, `wifi`, `contadores`, etc.
	// Tipos de entidad definidos en la vertical.
	// El ID y los valores de los atributos son opcionales.
	EntityTypes []EntityType `json:"entityTypes,omitempty,omitzero"`
	// Entidades específicas de alguno de los tipos anteriores
	Entities []Entity `json:"entities,omitempty,omitzero"`
	// Contenidos compatibles con urbo-deployer
	Environment    Environment             `json:"environment,omitempty,omitzero"`
	Deployment     DeploymentManifest      `json:"deployment,omitempty,omitzero"`
	ManifestPanels PanelManifest           `json:"panels,omitempty,omitzero"`
	Subscriptions  map[string]Subscription `json:"subscriptions,omitempty,omitzero"`
	Rules          map[string]Rule         `json:"rules,omitempty,omitzero"`
	Verticals      map[string]Vertical     `json:"verticals,omitempty,omitzero"`
	DeviceGroups   []DeviceGroup           `json:"deviceGroups,omitempty,omitzero"`
	Devices        []Device                `json:"devices,omitempty,omitzero"`
	Registrations  []Registration          `json:"registrations,omitempty,omitzero"`
	// Solo por compatibilidad con urbo-deployer, no se usan
	SQL  json.RawMessage `json:"sql,omitempty,omitzero"`
	Cdas json.RawMessage `json:"cdas,omitempty,omitzero"`
	Etls json.RawMessage `json:"etls,omitempty,omitzero"`
	// Otros datos de estado no asociados al manifest
	ServiceMappings []ServiceMapping     `json:"serviceMappings,omitempty,omitzero"`
	Projects        []Project            `json:"projects,omitempty,omitzero"`
	Domains         []Domain             `json:"domains,omitempty,omitzero"`
	Panels          map[string]UrboPanel `json:"urboPanels,omitempty,omitzero"`
	Tables          []Table              `json:"tables,omitempty,omitzero"`
	Views           []View               `json:"views,omitempty,omitzero"`
	Users           []User               `json:"users,omitempty,omitzero"`
	Groups          []Group              `json:"groups,omitempty,omitzero"`
	Roles           []Role               `json:"roles,omitempty,omitzero"`
	Assignments     []RoleAssignment     `json:"assignments,omitempty,omitzero"`
}

// SummaryOf makes a summary of every item in the list
func SummaryOf[V any](items map[string]V, summary func(k string, v V) string) []string {
	values := make([]string, 0, len(items))
	for k, item := range items {
		values = append(values, summary(k, item))
	}
	return values
}

func (m *Manifest) ClearStatus() {
	for k, v := range m.Subscriptions {
		v.SubscriptionStatus = SubscriptionStatus{}
		v.Notification.NotificationStatus = NotificationStatus{}
		m.Subscriptions[k] = v
	}
	for k, v := range m.Rules {
		v.RuleStatus = RuleStatus{}
		m.Rules[k] = v
	}
	for k, v := range m.DeviceGroups {
		v.ServiceStatus = ServiceStatus{}
		m.DeviceGroups[k] = v
	}
	for k, v := range m.Devices {
		v.DeviceStatus = DeviceStatus{}
		m.Devices[k] = v
	}
	// Remove known endpoints
	for k := range m.Environment.NotificationEndpoints {
		if !strings.Contains(k, ":") {
			delete(m.Environment.NotificationEndpoints, k)
		}
	}
}

// Environment settings
type Environment struct {
	NotificationEndpoints map[string]string `json:"notificationEndpoints"`
}

// Environment is empty?
func (e Environment) IsZero() bool {
	return len(e.NotificationEndpoints) <= 0
}

// UrboPanel representa un panel de Urbo
type UrboPanel struct {
	Name          string `json:"name"`
	Description   string `json:"description,omitempty,omitzero"`
	Slug          string `json:"slug"`
	LowercaseSlug string `json:"lowercaseSlug,omitempty,omitzero"`
	WidgetCount   int    `json:"widgetCount,omitempty,omitzero"`
	IsShadow      *bool  `json:"isShadow,omitempty"`
	Section       string `json:"section,omitempty,omitzero"`
}

// Vertical representa una vertical de Urbo
type Vertical struct {
	Panels       []string        `json:"panels,omitempty,omitzero" compact:"true"`
	ShadowPanels []string        `json:"shadowPanels,omitempty,omitzero" compact:"true"`
	Slug         string          `json:"slug"`
	Name         string          `json:"name"`
	Icon         string          `json:"icon,omitempty,omitzero"`
	I18n         json.RawMessage `json:"i18n,omitempty,omitzero"`
	UrboVerticalStatus
}

// Return all Panels of the vertical, regular and shadow
func (v Vertical) AllPanels() []string {
	result := make([]string, 0, len(v.Panels)+len(v.ShadowPanels))
	return append(append(result, v.Panels...), v.ShadowPanels...)
}

// UrboVerticalStatus contains detailed vertical status
type UrboVerticalStatus struct {
	PanelsObjects       []UrboPanel `json:"panelsObjects,omitempty,omitzero"`
	ShadowPanelsObjects []UrboPanel `json:"shadowPanelsObjects,omitempty,omitzero"`
}

// EntityType representa un tipo de entidad
type EntityType struct {
	ID   string `json:"entityID,omitempty,omitzero"`
	Type string `json:"entityType"`
	// Usamos una lista en vez de un map para poder
	// establecer un orden específico, por si nos interesa
	// conservar el orden de atributos para algo.
	Attrs []Attribute `json:"attrs"`
}

type LongtermKind string

const (
	LongtermNone      LongtermKind = ""
	LongtermCounter   LongtermKind = "counter"
	LongtermGauge     LongtermKind = "gauge"
	LongtermEnum      LongtermKind = "enum"
	LongtermModal     LongtermKind = "modal"
	LongtermDimension LongtermKind = "dimension"
)

// Attribute representa un atributo de una entidad
type Attribute struct {
	Name        string          `json:"name"`
	Type        string          `json:"type"`
	Description []string        `json:"description,omitempty,omitzero"`
	Value       json.RawMessage `json:"value,omitempty,omitzero" compact:"true"`
	Metadatas   json.RawMessage `json:"metadatas,omitempty,omitzero" compact:"true"`
	// Si la entidad es Singleton, este atributo se puede marcar
	// como parte de la identidad del singleton, y se añadirá a la
	// primary key de la tabla.
	SingletonKey bool `json:"singletonKey,omitempty,omitzero"`
	// Indica si este atributo forma parte de la simulación
	Simulated bool `json:"simulated,omitempty,omitzero"`
	// Indica si este atributo debe conservarse de alguna forma en longterm
	Longterm LongtermKind `json:"longterm,omitempty,omitzero"`
	// Si longterm == LongtermEnum, estas serían las opciones
	LongtermOptions []string `json:"longtermOptions,omitempty,omitzero" sort:"true"`
}

// Entity representa una instancia de EntityType
type Entity struct {
	ID   string `json:"entityID"`
	Type string `json:"entityType"`
	// Aquí no hace falta mantener el orden,
	// porque el orden correcto de los atributos ya
	// está en el EntityType.
	// Los Metadatas se sacan aparte porque generalmente
	// el json generado es más tratable de esta forma
	Attrs     map[string]json.RawMessage `json:"attrs" compact:"true"`
	MetaDatas map[string]json.RawMessage `json:"metadatas,omitempty,omitzero" compact:"true"`
}

// ServiceMapping es cada uno de los serviceMappings de cygnus
type ServiceMapping struct {
	OriginalService     string               `json:"originalService,omitempty,omitzero"`
	NewService          string               `json:"newService,omitempty,omitzero"`
	ServicePathMappings []ServicePathMapping `json:"servicePathMappings"`
}

// ServicePathMapping es cada uno de los servicePathMappings de un serviceMapping
type ServicePathMapping struct {
	OriginalServicePath string          `json:"originalServicePath,omitempty,omitzero"`
	NewServicePath      string          `json:"newServicePath,omitempty,omitzero"`
	EntityMappings      []EntityMapping `json:"entityMappings"`
}

// EntityMaping es cada uno de los EntityMappings de un servicePathMapping
type EntityMapping struct {
	OriginalEntityId   string             `json:"originalEntityId,omitempty,omitzero"`
	NewEntityId        string             `json:"newEntityId,omitempty,omitzero"`
	OriginalEntityType string             `json:"originalEntityType,omitempty,omitzero"`
	NewEntityType      string             `json:"newEntityType,omitempty,omitzero"`
	AttributeMappings  []AttributeMapping `json:"attributeMappings"`
}

// AttributeMapping es cada uno de los AttributeMappings de un EntityMapping
type AttributeMapping struct {
	OriginalAttributeName string `json:"originalAttributeName,omitempty,omitzero"`
	OriginalAttributeType string `json:"originalAttributeType,omitempty,omitzero"`
	NewAttributeName      string `json:"newAttributeName,omitempty,omitzero"`
	NewAttributeType      string `json:"newAttributeType,omitempty,omitzero"`
}

// Registration representa un registro
type Registration struct {
	ID           string          `json:"id"`
	Description  string          `json:"description,omitempty,omitzero"`
	DataProvided json.RawMessage `json:"dataProvided,omitempty,omitzero"`
	Provider     json.RawMessage `json:"provider,omitempty,omitzero"`
	RegistrationStatus
}

type RegistrationStatus struct {
	Status string `json:"status"`
}

// Subscription representa una suscripcion
type Subscription struct {
	Description  string       `json:"description"`
	Status       string       `json:"status,omitempty,omitzero"`
	Expires      string       `json:"expires,omitempty,omitzero"`
	Notification Notification `json:"notification"`
	Subject      Subject      `json:"subject"`
	Throttling   int          `json:"throttling,omitempty,omitzero"`
	SubscriptionStatus
}

// UpdateEndpoint updates the notification endpoint
func (subs Subscription) UpdateEndpoint(notificationEndpoints map[string]string) (Subscription, error) {
	result := subs
	var url *string
	if result.Notification.HTTP.URL != "" {
		url = &(result.Notification.HTTP.URL)
	}
	if result.Notification.HTTPCustom.URL != "" {
		url = &(result.Notification.HTTPCustom.URL)
	}
	if result.Notification.MQTT.URL != "" {
		url = &(result.Notification.MQTT.URL)
	}
	if result.Notification.MQTTCustom.URL != "" {
		url = &(result.Notification.MQTTCustom.URL)
	}
	if url == nil {
		return result, errors.New("subscription has no notification URL")
	}
	ep, ok := notificationEndpoints[*url]
	if !ok {
		return result, fmt.Errorf("notification endpoint %s not found", *url)
	}
	*url = ep
	return result, nil
}

// SubscriptionStatus agrupa los datos de estado de la suscripción
type SubscriptionStatus struct {
	ID            string `json:"id,omitempty,omitzero"`
	Documentation string `json:"documentation,omitempty,omitzero"`
}

// Notification es la configuración de notificación de la suscripción
type Notification struct {
	Attrs            []string               `json:"attrs,omitempty,omitzero" sort:"true" compact:"true"`
	ExceptAttrs      []string               `json:"exceptAttrs,omitempty,omitzero" sort:"true" compact:"true"`
	AttrsFormat      string                 `json:"attrsFormat,omitempty,omitzero"`
	HTTP             NotificationHTTP       `json:"http,omitempty,omitzero"`
	HTTPCustom       NotificationCustom     `json:"httpCustom,omitempty,omitzero"`
	MQTT             NotificationMQTT       `json:"mqtt,omitempty,omitzero"`
	MQTTCustom       NotificationMQTTCustom `json:"mqttCustom,omitempty,omitzero"`
	OnlyChangedAttrs *bool                  `json:"onlyChangedAttrs,omitempty"`
	Covered          *bool                  `json:"covered,omitempty"`
	NotificationStatus
}

// NotificationStatus agrupa los datos de estado de la suscripción
type NotificationStatus struct {
	LastFailure       string `json:"lastFailure,omitempty,omitzero"`
	LastFailureReason string `json:"lastFailureReason,omitempty,omitzero"`
	LastNotification  string `json:"lastNotification,omitempty,omitzero"`
	LastSuccess       string `json:"lastSuccess,omitempty,omitzero"`
	LastSuccessCode   int    `json:"lastSuccessCode,omitempty,omitzero"`
	FailsCounter      int    `json:"failsCounter,omitempty,omitzero"`
	TimesSent         int    `json:"timesSent,omitempty,omitzero"`
}

// NotificationHTTP son los datos de una notificacion
type NotificationHTTP struct {
	URL     string `json:"url"`
	Timeout int    `json:"timeout,omitempty,omitzero"`
}

func (n NotificationHTTP) IsZero() bool {
	return n.URL == ""
}

// NotificationHTTP son los datos de una notificacion
type NotificationCustom struct {
	URL     string            `json:"url"`
	Timeout int               `json:"timeout,omitempty,omitzero"`
	Headers map[string]string `json:"headers,omitempty,omitzero"`
	Qs      map[string]string `json:"qs,omitempty,omitzero"`
	Method  string            `json:"method,omitempty,omitzero"`
	Payload json.RawMessage   `json:"payload,omitempty,omitzero" compact:"true"`
	Json    json.RawMessage   `json:"json,omitempty,omitzero" compact:"true"`
	NGSI    json.RawMessage   `json:"ngsi,omitempty,omitzero" compact:"true"`
}

func (n NotificationCustom) IsZero() bool {
	return n.URL == ""
}

// NotificationMQTT son los datos de una notificacion MQTT
type NotificationMQTT struct {
	URL    string `json:"url"`
	Topic  string `json:"topic"`
	QoS    string `json:"qos,omitempty,omitzero"`
	User   string `json:"user,omitempty,omitzero"`
	Passwd string `json:"passwd,omitempty,omitzero"`
}

func (n NotificationMQTT) IsZero() bool {
	return n.URL == "" || n.Topic == ""
}

// NotificationMQTTCustom son los datos de una notificacion MQTT Custom
type NotificationMQTTCustom struct {
	URL     string          `json:"url"`
	Topic   string          `json:"topic"`
	QoS     int             `json:"qos,omitempty,omitzero"`
	User    string          `json:"user,omitempty,omitzero"`
	Passwd  string          `json:"passwd,omitempty,omitzero"`
	Payload json.RawMessage `json:"payload,omitempty,omitzero" compact:"true"`
	Json    json.RawMessage `json:"json,omitempty,omitzero" compact:"true"`
	NGSI    json.RawMessage `json:"ngsi,omitempty,omitzero" compact:"true"`
}

func (n NotificationMQTTCustom) IsZero() bool {
	return n.URL == "" || n.Topic == ""
}

// Subject es el sujeto de la suscripcion
type Subject struct {
	Condition SubjectCondition `json:"condition"`
	Entities  []SubjectEntity  `json:"entities" compact:"true"`
}

// SubjectCondition es la condicion del sujeto de la suscripcion
type SubjectCondition struct {
	Attrs                  []string          `json:"attrs" sort:"true"`
	Expression             SubjectExpression `json:"expression,omitempty,omitzero"`
	AlterationTypes        []string          `json:"alterationTypes,omitempty,omitzero"`
	NotifyOnMetadataChange *bool             `json:"notifyOnMetadataChange,omitempty"`
}

// SubjectExpression es la expresion en la condicion
type SubjectExpression struct {
	Q string `json:"q,omitempty,omitzero"`
}

func (s SubjectExpression) IsZero() bool {
	return s.Q == ""
}

// SubjectEntity es la entidad sujeto de la suscripcion
type SubjectEntity struct {
	ID        string `json:"id,omitempty,omitzero"`
	IdPattern string `json:"idPattern,omitempty,omitzero"`
	Type      string `json:"type"`
}

// Table define algunos parámetros básicos de tablas a crear
type Table struct {
	Name       string        `json:"name"`
	Columns    []TableColumn `json:"columns"`
	PrimaryKey []string      `json:"primaryKey"`
	Indexes    []TableIndex  `json:"indexes"`
	LastData   bool          `json:"lastdata"`                     // True si queremos crear una vista lastdata adicional
	Singleton  []string      `json:"singleton,omitempty,omitzero"` // Lista de campos únicos, si la entidad es un singleton.
}

// MaterializedView define los parámetros de las vistas materializadas
type View struct {
	Materialized bool         `json:"materialized,omitempty,omitzero"`
	Name         string       `json:"name"`
	From         string       `json:"from"`
	Group        []string     `json:"group"`
	Columns      []ViewColumn `json:"columns"`
}

// ViewColumn define las columnas de la vista
type ViewColumn struct {
	Name       string `json:"name"`
	Expression string `json:"expression"`
}

// TableColumn describe una columna de una tabla
type TableColumn struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	NotNull bool   `json:"notNull,omitempty,omitzero"`
	Default string `json:"default,omitempty,omitzero"`
}

// TableIndex describe un indice
type TableIndex struct {
	Name     string   `json:"name"`
	Columns  []string `json:"columns"`
	Geometry bool     `json:"geometry,omitempty,omitzero"`
}

// DeviceGroup describe la provisión de un grupo de dispositivos
type DeviceGroup struct {
	Resource           string            `json:"resource"`
	APIKey             string            `json:"apikey"`
	Token              string            `json:"token,omitempty,omitzero"` // fully legacy
	EntityType         string            `json:"entity_type"`
	Description        string            `json:"description,omitempty,omitzero"`
	Protocol           string            `json:"protocol"`
	Transport          string            `json:"transport,omitempty,omitzero"`
	Timestamp          *bool             `json:"timestamp,omitempty"`
	ExplicitAttrs      json.RawMessage   `json:"explicitAttrs,omitempty,omitzero"`
	InternalAttributes []DeviceAttribute `json:"internal_attributes,omitempty,omitzero"`
	Attributes         []DeviceAttribute `json:"attributes"`
	Lazy               []DeviceAttribute `json:"lazy,omitempty,omitzero"`
	StaticAttributes   []DeviceAttribute `json:"static_attributes,omitempty,omitzero"`
	Commands           []DeviceCommand   `json:"commands,omitempty,omitzero"`
	ExpressionLanguage string            `json:"expressionLanguage,omitempty,omitzero"`
	EntityNameExp      string            `json:"entityNameExp,omitempty,omitzero"`
	PayloadType        string            `json:"PayloadType,omitempty,omitzero"`
	AutoProvision      bool              `json:"autoprovision,omitempty,omitzero"`
	ServiceStatus
}

// ServiceStatus agrupa atributos de estado que no se usan al crear un Service
type ServiceStatus struct {
	ID          string `json:"_id,omitempty,omitzero"`
	V           int    `json:"__v,omitempty,omitzero"`
	IOTAgent    string `json:"iotagent,omitempty,omitzero"`
	ServicePath string `json:"service_path,omitempty,omitzero"`
	Service     string `json:"service,omitempty,omitzero"`
	CBHost      string `json:"cbHost,omitempty,omitzero"`
}

// Device representa un dispositivo
type Device struct {
	DeviceId           string            `json:"device_id"`
	APIKey             string            `json:"apikey,omitempty,omitzero"`
	EntityName         string            `json:"entity_name,omitempty,omitzero"`
	EntityType         string            `json:"entity_type"`
	Polling            *bool             `json:"polling,omitempty"`
	Transport          string            `json:"transport"`
	Timestamp          *bool             `json:"timestamp,omitempty"`
	Endpoint           string            `json:"endpoint,omitempty,omitzero"`
	Attributes         []DeviceAttribute `json:"attributes,omitempty,omitzero"`
	Lazy               []DeviceAttribute `json:"lazy,omitempty,omitzero"`
	Commands           []DeviceCommand   `json:"commands,omitempty,omitzero"`
	StaticAttributes   []DeviceAttribute `json:"static_attributes,omitempty,omitzero"`
	Protocol           string            `json:"protocol"`
	ExpressionLanguage string            `json:"expressionLanguage,omitempty,omitzero"`
	ExplicitAttrs      json.RawMessage   `json:"explicitAttrs,omitempty,omitzero"`
	DeviceStatus
}

// GroupStatus agrupa atributos de estado que no se usan al crear un Device
type DeviceStatus struct {
	Service     string `json:"service,omitempty,omitzero"`
	ServicePath string `json:"service_path,omitempty,omitzero"`
}

// DeviceAttribute describe un atributo de dispositivo
type DeviceAttribute struct {
	ObjectId   string          `json:"object_id"`
	Name       string          `json:"name"`
	Type       string          `json:"type,omitempty,omitzero"`
	Value      json.RawMessage `json:"value,omitempty,omitzero"` // para los staticAttribs
	Expression string          `json:"expression,omitempty,omitzero"`
	SkipValue  *bool           `json:"skipValue,omitempty"`
	EntityName string          `json:"entity_name,omitempty,omitzero"`
	EntityType string          `json:"entity_type,omitempty,omitzero"`
}

// DeviceCommand describe un comando de dispositivo
type DeviceCommand struct {
	ObjectId   string          `json:"object_id,omitempty,omitzero"`
	Name       string          `json:"name,omitempty,omitzero"`
	Type       string          `json:"type,omitempty,omitzero"`
	Value      string          `json:"value,omitempty,omitzero"`
	Expression string          `json:"expression,omitempty,omitzero"`
	MQTT       json.RawMessage `json:"mqtt,omitempty,omitzero"`
}

type Rule struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty,omitzero"`
	Misc        string          `json:"misc,omitempty,omitzero"`
	Text        string          `json:"text,omitempty,omitzero"`
	VR          string          `json:"VR,omitempty,omitzero"`
	Action      json.RawMessage `json:"action,omitempty,omitzero"`   // TODO: estructurar de alguna forma?
	NoSignal    json.RawMessage `json:"nosignal,omitempty,omitzero"` // TODO: estructurar de alguna forma?
	RuleStatus
}

// ActionList converts the action field into list of actions
func (rule Rule) ActionList() []interface{} {
	var actionList []interface{}
	if len(rule.Action) > 0 {
		var action interface{}
		if err := json.Unmarshal(rule.Action, &action); err == nil {
			switch action := action.(type) {
			case []interface{}:
				actionList = action
			default:
				actionList = append(make([]interface{}, 0, 1), action)
			}
		}
	}
	return actionList
}

// RuleStatus agrupa atributos de estado que no se usan al crear una Rule
type RuleStatus struct {
	Subservice string `json:"subservice,omitempty,omitzero"`
	Service    string `json:"service,omitempty,omitzero"`
	ID         string `json:"_id,omitempty,omitzero"`
}

type Project struct {
	IsDomain    bool            `json:"is_domain"`
	Description string          `json:"description,omitempty,omitzero"`
	Tags        json.RawMessage `json:"tags,omitempty,omitzero"`
	Options     json.RawMessage `json:"options,omitempty,omitzero"`
	Enabled     bool            `json:"enabled"`
	Name        string          `json:"name"`
	ParentId    string          `json:"parent_id,omitempty,omitzero"`
	DomainId    string          `json:"domain_id,omitempty,omitzero"`
	ProjectStatus
}

type ProjectStatus struct {
	Links  json.RawMessage `json:"links,omitempty,omitzero"`
	ID     string          `json:"id,omitempty,omitzero"`
	Parent string          `json:"parent,omitempty,omitzero"`
	Domain string          `json:"domain,omitempty,omitzero"`
}

type Domain struct {
	Description string `json:"description,omitempty,omitzero"`
	Enabled     bool   `json:"enabled"`
	Name        string `json:"name"`
	DomainStatus
}

type DomainStatus struct {
	Links json.RawMessage `json:"links,omitempty,omitzero"`
	ID    string          `json:"id"`
}

type User struct {
	Name        string                     `json:"name"`
	Description string                     `json:"description,omitempty,omitzero"`
	Enabled     bool                       `json:"enabled"`
	Email       string                     `json:"email,omitempty,omitzero"`
	Options     map[string]json.RawMessage `json:"options,omitempty,omitzero"`
	DomainID    string                     `json:"domain_id"`
	UserStatus
}

type UserStatus struct {
	Links   json.RawMessage `json:"links,omitempty,omitzero"`
	ID      string          `json:"id,omitempty,omitzero"`
	Domain  string          `json:"domain,omitempty,omitzero"`
	Expires json.RawMessage `json:"password_expires_at,omitempty,omitzero"`
}

type Group struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty,omitzero"`
	DomainID    string `json:"domain_id"`
	GroupStatus
}

type GroupStatus struct {
	Links     json.RawMessage `json:"links,omitempty,omitzero"`
	ID        string          `json:"id,omitempty,omitzero"`
	Domain    string          `json:"domain,omitempty,omitzero"`
	Users     []string        `json:"users,omitempty,omitzero"`
	UserNames []string        `json:"userNames,omitempty,omitzero"`
}

type Role struct {
	Description string          `json:"description,omitempty,omitzero"`
	Name        string          `json:"name"`
	DomainID    string          `json:"domain_id"`
	Options     json.RawMessage `json:"options,omitempty,omitzero"`
	RoleStatus
}

type RoleStatus struct {
	Links  json.RawMessage `json:"links,omitempty,omitzero"`
	ID     string          `json:"id"`
	Domain string          `json:"domain"`
}

type RoleAssignment struct {
	Scope json.RawMessage `json:"scope,omitempty,omitzero"`
	Role  AssignmentID    `json:"role,omitempty,omitzero"`
	User  AssignmentID    `json:"user,omitempty,omitzero"`
	Group AssignmentID    `json:"group,omitempty,omitzero"`
	RoleAssignmentStatus
}

type RoleAssignmentStatus struct {
	Links     json.RawMessage `json:"links,omitempty,omitzero"`
	Inherited string          `json:"inherited"`
	ProjectID string          `json:"project_id,omitempty,omitzero"`
	DomainID  string          `json:"domain_id,omitempty,omitzero"`
	ScopeName string          `json:"scope_name,omitempty,omitzero"`
}

func (r *RoleAssignment) ParseScope() error {
	var items map[string]json.RawMessage
	if err := json.Unmarshal(r.Scope, &items); err != nil {
		return err
	}
	var assignmentID AssignmentID
	if project, ok := items["project"]; ok {
		if err := json.Unmarshal(project, &assignmentID); err != nil {
			return err
		}
		r.ProjectID = assignmentID.ID
		r.ScopeName = assignmentID.Name
	}
	if domain, ok := items["domain"]; ok {
		if err := json.Unmarshal(domain, &assignmentID); err != nil {
			return err
		}
		r.DomainID = assignmentID.ID
		r.ScopeName = assignmentID.Name
	}
	if inherit, ok := items["OS-INHERIT:inherited_to"]; ok {
		if err := json.Unmarshal(inherit, &r.Inherited); err != nil {
			return err
		}
	}
	return nil
}

type AssignmentID struct {
	ID      string          `json:"id,omitempty,omitzero"`
	Name    string          `json:"name,omitempty,omitzero"`
	Domain  json.RawMessage `json:"domain,omitempty,omitzero"`
	Project json.RawMessage `json:"project,omitempty,omitzero"`
}

type DeploymentManifest struct {
	Sources map[string]ManifestSource `json:"sources,omitempty,omitzero"`
}

func (d DeploymentManifest) IsZero() bool {
	return len(d.Sources) <= 0
}

type PanelManifest struct {
	Sources map[string]ManifestSource `json:"sources,omitempty,omitzero"`
}

func (p PanelManifest) IsZero() bool {
	return len(p.Sources) <= 0
}

type ManifestSource struct {
	Path  string   `json:"path,omitempty,omitzero"`
	Files []string `json:"files,omitempty,omitzero"`
}
