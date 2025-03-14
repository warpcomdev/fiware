// El paquete fiware define los modelos de datos básicos de warpcom-fiware
// El principal modelo es el de Vertical, que es el punto de entrada a
// partir del cual se accede al conjunto de datos de la vertical. El resto
// de tipos corresponden a sub-atributos dentro de la vertical.
package fiware

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/warpcomdev/fiware/internal/serialize"
)

// Tipos de datos que se usan para relacionarse con la vertical.
// Todos los tipos deben implementar la interfaz `Serializable`,
// para poder ser exportados a diferentes formatos (json, jsonnet,
// starlark, etc).
// La interfaz se implementa automáticamente con el siguiente generador:
//go:generate go run cmd/generate/generate.go

// Manifest representa un manifiesto de vertical
type Manifest struct {
	Name       string `json:"name,omitempty"`       // `tourism`, `wifi`, `watermeter`, etc
	Subservice string `json:"subservice,omitempty"` // `turismo`, `wifi`, `contadores`, etc.
	// Tipos de entidad definidos en la vertical.
	// El ID y los valores de los atributos son opcionales.
	EntityTypes []EntityType `json:"entityTypes,omitempty"`
	// Entidades específicas de alguno de los tipos anteriores
	Entities []Entity `json:"entities,omitempty"`
	// Contenidos compatibles con urbo-deployer
	Environment    Environment             `json:"environment,omitempty"`
	Deployment     DeploymentManifest      `json:"deployment,omitempty"`
	ManifestPanels PanelManifest           `json:"panels,omitempty"`
	Subscriptions  map[string]Subscription `json:"subscriptions,omitempty"`
	Rules          map[string]Rule         `json:"rules,omitempty"`
	Verticals      map[string]Vertical     `json:"verticals,omitempty"`
	Services       []Service               `json:"services,omitempty"`
	Devices        []Device                `json:"devices,omitempty"`
	Registrations  []Registration          `json:"registrations,omitempty"`
	// Solo por compatibilidad con urbo-deployer, no se usan
	SQL  json.RawMessage `json:"sql,omitempty"`
	Cdas json.RawMessage `json:"cdas,omitempty"`
	Etls json.RawMessage `json:"etls,omitempty"`
	// Otros datos de estado no asociados al manifest
	ServiceMappings []ServiceMapping     `json:"serviceMappings,omitempty"`
	Projects        []Project            `json:"projects,omitempty"`
	Domains         []Domain             `json:"domains,omitempty"`
	Panels          map[string]UrboPanel `json:"urboPanels,omitempty"`
	Tables          []Table              `json:"tables,omitempty"`
	Views           []View               `json:"views,omitempty"`
	Users           []User               `json:"users,omitempty"`
	Groups          []Group              `json:"groups,omitempty"`
	Roles           []Role               `json:"roles,omitempty"`
	Assignments     []RoleAssignment     `json:"assignments,omitempty"`
}

// SummaryOf makes a summary of every item in the list
func SummaryOf[V any](items map[string]V, summary func(k string, v V) string) []string {
	values := make([]string, 0, len(items))
	for k, item := range items {
		values = append(values, summary(k, item))
	}
	return values
}

func ValuesOf[V any](items map[string]V) []V {
	values := make([]V, 0, len(items))
	for _, item := range items {
		values = append(values, item)
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
	for k, v := range m.Services {
		v.ServiceStatus = ServiceStatus{}
		m.Services[k] = v
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
func (e Environment) IsEmpty() bool {
	return len(e.NotificationEndpoints) <= 0
}

// UrboPanel representa un panel de Urbo
type UrboPanel struct {
	Name          string                 `json:"name"`
	Description   string                 `json:"description,omitempty"`
	Slug          string                 `json:"slug"`
	LowercaseSlug string                 `json:"lowercaseSlug,omitempty"`
	WidgetCount   int                    `json:"widgetCount,omitempty"`
	IsShadow      serialize.OptionalBool `json:"isShadow,omitempty"`
	Section       string                 `json:"section,omitempty"`
}

// Vertical representa una vertical de Urbo
type Vertical struct {
	Panels       []string        `json:"panels,omitempty" compact:"true"`
	ShadowPanels []string        `json:"shadowPanels,omitempty" compact:"true"`
	Slug         string          `json:"slug"`
	Name         string          `json:"name"`
	Icon         string          `json:"icon,omitempty"`
	I18n         json.RawMessage `json:"i18n,omitempty"`
	UrboVerticalStatus
}

// Return all Panels of the vertical, regular and shadow
func (v Vertical) AllPanels() []string {
	result := make([]string, 0, len(v.Panels)+len(v.ShadowPanels))
	return append(append(result, v.Panels...), v.ShadowPanels...)
}

// UrboVerticalStatus contains detailed vertical status
type UrboVerticalStatus struct {
	PanelsObjects       []UrboPanel `json:"panelsObjects,omitempty"`
	ShadowPanelsObjects []UrboPanel `json:"shadowPanelsObjects,omitempty"`
}

// EntityType representa un tipo de entidad
type EntityType struct {
	ID   string `json:"entityID,omitempty"`
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
	Description []string        `json:"description,omitempty"`
	Value       json.RawMessage `json:"value,omitempty" compact:"true"`
	Metadatas   json.RawMessage `json:"metadatas,omitempty" compact:"true"`
	// Si la entidad es Singleton, este atributo se puede marcar
	// como parte de la identidad del singleton, y se añadirá a la
	// primary key de la tabla.
	SingletonKey bool `json:"singletonKey,omitempty"`
	// Indica si este atributo forma parte de la simulación
	Simulated bool `json:"simulated,omitempty"`
	// Indica si este atributo debe conservarse de alguna forma en longterm
	Longterm LongtermKind `json:"longterm,omitempty"`
	// Si longterm == LongtermEnum, estas serían las opciones
	LongtermOptions []string `json:"longtermOptions,omitempty" sort:"true"`
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
	MetaDatas map[string]json.RawMessage `json:"metadatas,omitempty" compact:"true"`
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

// Registration representa un registro
type Registration struct {
	ID           string          `json:"id"`
	Description  string          `json:"description,omitempty"`
	DataProvided json.RawMessage `json:"dataProvided,omitempty"`
	Provider     json.RawMessage `json:"provider,omitempty"`
	RegistrationStatus
}

type RegistrationStatus struct {
	Status string `json:"status"`
}

// Subscription representa una suscripcion
type Subscription struct {
	Description  string       `json:"description"`
	Status       string       `json:"status,omitempty"`
	Expires      string       `json:"expires,omitempty"`
	Notification Notification `json:"notification"`
	Subject      Subject      `json:"subject"`
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
	ID            string `json:"id,omitempty"`
	Documentation string `json:"documentation,omitempty"`
}

// Notification es la configuración de notificación de la suscripción
type Notification struct {
	Attrs            []string               `json:"attrs,omitempty" sort:"true" compact:"true"`
	ExceptAttrs      []string               `json:"exceptAttrs,omitempty" sort:"true" compact:"true"`
	AttrsFormat      string                 `json:"attrsFormat,omitempty"`
	HTTP             NotificationHTTP       `json:"http,omitempty"`
	HTTPCustom       NotificationCustom     `json:"httpCustom,omitempty"`
	MQTT             NotificationMQTT       `json:"mqtt,omitempty"`
	MQTTCustom       NotificationMQTTCustom `json:"mqttCustom,omitempty"`
	OnlyChangedAttrs serialize.OptionalBool `json:"onlyChangedAttrs,omitempty"`
	Covered          serialize.OptionalBool `json:"covered,omitempty"`
	NotificationStatus
}

// NotificationStatus agrupa los datos de estado de la suscripción
type NotificationStatus struct {
	LastFailure       string `json:"lastFailure,omitempty"`
	LastFailureReason string `json:"lastFailureReason,omitempty"`
	LastNotification  string `json:"lastNotification,omitempty"`
	LastSuccess       string `json:"lastSuccess,omitempty"`
	LastSuccessCode   int    `json:"lastSuccessCode,omitempty"`
	FailsCounter      int    `json:"failsCounter,omitempty"`
	TimesSent         int    `json:"timesSent,omitempty"`
}

// NotificationHTTP son los datos de una notificacion
type NotificationHTTP struct {
	URL     string `json:"url"`
	Timeout int    `json:"timeout,omitempty"`
}

func (n NotificationHTTP) IsEmpty() bool {
	return n.URL == ""
}

// NotificationHTTP son los datos de una notificacion
type NotificationCustom struct {
	URL     string            `json:"url"`
	Timeout int               `json:"timeout,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Qs      map[string]string `json:"qs,omitempty"`
	Method  string            `json:"method,omitempty"`
	Payload json.RawMessage   `json:"payload,omitempty" compact:"true"`
	Json    json.RawMessage   `json:"json,omitempty" compact:"true"`
	NGSI    json.RawMessage   `json:"ngsi,omitempty" compact:"true"`
}

func (n NotificationCustom) IsEmpty() bool {
	return n.URL == ""
}

// NotificationMQTT son los datos de una notificacion MQTT
type NotificationMQTT struct {
	URL    string `json:"url"`
	Topic  string `json:"topic"`
	QoS    string `json:"qos,omitempty"`
	User   string `json:"user,omitempty"`
	Passwd string `json:"passwd,omitempty"`
}

func (n NotificationMQTT) IsEmpty() bool {
	return n.URL == "" || n.Topic == ""
}

// NotificationMQTTCustom son los datos de una notificacion MQTT Custom
type NotificationMQTTCustom struct {
	URL     string          `json:"url"`
	Topic   string          `json:"topic"`
	QoS     int             `json:"qos,omitempty"`
	User    string          `json:"user,omitempty"`
	Passwd  string          `json:"passwd,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty" compact:"true"`
	Json    json.RawMessage `json:"json,omitempty" compact:"true"`
	NGSI    json.RawMessage `json:"ngsi,omitempty" compact:"true"`
}

func (n NotificationMQTTCustom) IsEmpty() bool {
	return n.URL == "" || n.Topic == ""
}

// Subject es el sujeto de la suscripcion
type Subject struct {
	Condition SubjectCondition `json:"condition"`
	Entities  []SubjectEntity  `json:"entities" compact:"true"`
}

// SubjectCondition es la condicion del sujeto de la suscripcion
type SubjectCondition struct {
	Attrs                  []string               `json:"attrs" sort:"true"`
	Expression             SubjectExpression      `json:"expression,omitempty"`
	AlterationTypes        []string               `json:"alterationTypes,omitempty"`
	NotifyOnMetadataChange serialize.OptionalBool `json:"notifyOnMetadataChange,omitempty"`
}

// SubjectExpression es la expresion en la condicion
type SubjectExpression struct {
	Q string `json:"q,omitempty"`
}

func (s SubjectExpression) IsEmpty() bool {
	return s.Q == ""
}

// SubjectEntity es la entidad sujeto de la suscripcion
type SubjectEntity struct {
	ID        string `json:"id,omitempty"`
	IdPattern string `json:"idPattern,omitempty"`
	Type      string `json:"type"`
}

// Table define algunos parámetros básicos de tablas a crear
type Table struct {
	Name       string        `json:"name"`
	Columns    []TableColumn `json:"columns"`
	PrimaryKey []string      `json:"primaryKey"`
	Indexes    []TableIndex  `json:"indexes"`
	LastData   bool          `json:"lastdata"`            // True si queremos crear una vista lastdata adicional
	Singleton  []string      `json:"singleton,omitempty"` // Lista de campos únicos, si la entidad es un singleton.
}

// MaterializedView define los parámetros de las vistas materializadas
type View struct {
	Materialized bool         `json:"materialized,omitempty"`
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
	Resource           string                 `json:"resource"`
	APIKey             string                 `json:"apikey"`
	Token              string                 `json:"token,omitempty"` // fully legacy
	EntityType         string                 `json:"entity_type"`
	Description        string                 `json:"description,omitempty"`
	Protocol           string                 `json:"protocol"`
	Transport          string                 `json:"transport,omitempty"`
	Timestamp          serialize.OptionalBool `json:"timestamp,omitempty"`
	ExplicitAttrs      json.RawMessage        `json:"explicitAttrs,omitempty"`
	InternalAttributes []DeviceAttribute      `json:"internal_attributes,omitempty"`
	Attributes         []DeviceAttribute      `json:"attributes"`
	Lazy               []DeviceAttribute      `json:"lazy,omitempty"`
	StaticAttributes   []DeviceAttribute      `json:"static_attributes,omitempty"`
	Commands           []DeviceCommand        `json:"commands,omitempty"`
	ExpressionLanguage string                 `json:"expressionLanguage,omitempty"`
	EntityNameExp      string                 `json:"entityNameExp,omitempty"`
	PayloadType        string                 `json:"PayloadType,omitempty"`
	AutoProvision      bool                   `json:"autoprovision,omitempty"`
	ServiceStatus
}

// ServiceStatus agrupa atributos de estado que no se usan al crear un Service
type ServiceStatus struct {
	ID          string `json:"_id,omitempty"`
	V           int    `json:"__v,omitempty"`
	IOTAgent    string `json:"iotagent,omitempty"`
	ServicePath string `json:"service_path,omitempty"`
	Service     string `json:"service,omitempty"`
	CBHost      string `json:"cbHost,omitempty"`
}

// Device representa un dispositivo
type Device struct {
	DeviceId           string                 `json:"device_id"`
	APIKey             string                 `json:"apikey,omitempty"`
	EntityName         string                 `json:"entity_name,omitempty"`
	EntityType         string                 `json:"entity_type"`
	Polling            serialize.OptionalBool `json:"polling,omitempty"`
	Transport          string                 `json:"transport"`
	Timestamp          serialize.OptionalBool `json:"timestamp,omitempty"`
	Endpoint           string                 `json:"endpoint,omitempty"`
	Attributes         []DeviceAttribute      `json:"attributes,omitempty"`
	Lazy               []DeviceAttribute      `json:"lazy,omitempty"`
	Commands           []DeviceCommand        `json:"commands,omitempty"`
	StaticAttributes   []DeviceAttribute      `json:"static_attributes,omitempty"`
	Protocol           string                 `json:"protocol"`
	ExpressionLanguage string                 `json:"expressionLanguage,omitempty"`
	ExplicitAttrs      json.RawMessage        `json:"explicitAttrs,omitempty"`
	DeviceStatus
}

// GroupStatus agrupa atributos de estado que no se usan al crear un Device
type DeviceStatus struct {
	Service     string `json:"service,omitempty"`
	ServicePath string `json:"service_path,omitempty"`
}

// DeviceAttribute describe un atributo de dispositivo
type DeviceAttribute struct {
	ObjectId   string                 `json:"object_id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type,omitempty"`
	Value      json.RawMessage        `json:"value,omitempty"` // para los staticAttribs
	Expression string                 `json:"expression,omitempty"`
	SkipValue  serialize.OptionalBool `json:"skipValue,omitempty"`
	EntityName string                 `json:"entity_name,omitempty"`
	EntityType string                 `json:"entity_type,omitempty"`
}

// DeviceCommand describe un comando de dispositivo
type DeviceCommand struct {
	ObjectId   string          `json:"object_id,omitempty"`
	Name       string          `json:"name,omitempty"`
	Type       string          `json:"type,omitempty"`
	Value      string          `json:"value,omitempty"`
	Expression string          `json:"expression,omitempty"`
	MQTT       json.RawMessage `json:"mqtt,omitempty"`
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
	Subservice string `json:"subservice,omitempty"`
	Service    string `json:"service,omitempty"`
	ID         string `json:"_id,omitempty"`
}

type Project struct {
	IsDomain    bool            `json:"is_domain"`
	Description string          `json:"description,omitempty"`
	Tags        json.RawMessage `json:"tags,omitempty"`
	Options     json.RawMessage `json:"options,omitempty"`
	Enabled     bool            `json:"enabled"`
	Name        string          `json:"name"`
	ParentId    string          `json:"parent_id,omitempty"`
	DomainId    string          `json:"domain_id,omitempty"`
	ProjectStatus
}

type ProjectStatus struct {
	Links  json.RawMessage `json:"links,omitempty"`
	ID     string          `json:"id,omitempty"`
	Parent string          `json:"parent,omitempty"`
	Domain string          `json:"domain,omitempty"`
}

type Domain struct {
	Description string `json:"description,omitempty"`
	Enabled     bool   `json:"enabled"`
	Name        string `json:"name"`
	DomainStatus
}

type DomainStatus struct {
	Links json.RawMessage `json:"links,omitempty"`
	ID    string          `json:"id"`
}

type User struct {
	Name        string                     `json:"name"`
	Description string                     `json:"description,omitempty"`
	Enabled     bool                       `json:"enabled"`
	Email       string                     `json:"email,omitempty"`
	Options     map[string]json.RawMessage `json:"options,omitempty"`
	DomainID    string                     `json:"domain_id"`
	UserStatus
}

type UserStatus struct {
	Links   json.RawMessage `json:"links,omitempty"`
	ID      string          `json:"id,omitempty"`
	Domain  string          `json:"domain,omitempty"`
	Expires json.RawMessage `json:"password_expires_at,omitempty"`
}

type Group struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	DomainID    string `json:"domain_id"`
	GroupStatus
}

type GroupStatus struct {
	Links     json.RawMessage `json:"links,omitempty"`
	ID        string          `json:"id,omitempty"`
	Domain    string          `json:"domain,omitempty"`
	Users     []string        `json:"users,omitempty"`
	UserNames []string        `json:"userNames,omitempty"`
}

type Role struct {
	Description string          `json:"description,omitempty"`
	Name        string          `json:"name"`
	DomainID    string          `json:"domain_id"`
	Options     json.RawMessage `json:"options,omitempty"`
	RoleStatus
}

type RoleStatus struct {
	Links  json.RawMessage `json:"links,omitempty"`
	ID     string          `json:"id"`
	Domain string          `json:"domain"`
}

type RoleAssignment struct {
	Scope json.RawMessage `json:"scope,omitempty"`
	Role  AssignmentID    `json:"role,omitempty"`
	User  AssignmentID    `json:"user,omitempty"`
	Group AssignmentID    `json:"group,omitempty"`
	RoleAssignmentStatus
}

type RoleAssignmentStatus struct {
	Links     json.RawMessage `json:"links,omitempty"`
	Inherited string          `json:"inherited"`
	ProjectID string          `json:"project_id,omitempty"`
	DomainID  string          `json:"domain_id,omitempty"`
	ScopeName string          `json:"scope_name,omitempty"`
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
	ID      string          `json:"id,omitempty"`
	Name    string          `json:"name,omitempty"`
	Domain  json.RawMessage `json:"domain,omitempty"`
	Project json.RawMessage `json:"project,omitempty"`
}

type DeploymentManifest struct {
	Sources map[string]ManifestSource `json:"sources,omitempty"`
}

func (d DeploymentManifest) IsEmpty() bool {
	return len(d.Sources) <= 0
}

type PanelManifest struct {
	Sources map[string]ManifestSource `json:"sources,omitempty"`
}

func (p PanelManifest) IsEmpty() bool {
	return len(p.Sources) <= 0
}

type ManifestSource struct {
	Path  string   `json:"path,omitempty"`
	Files []string `json:"files,omitempty"`
}
