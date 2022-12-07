package fiware

import (
	"github.com/warpcomdev/fiware/internal/serialize"
)

// Autogenerated file - DO NOT EDIT

func (x Manifest) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x Manifest) Serialize(s serialize.Serializer) {
	if x.Name != "" {
		s.KeyString("name", string(x.Name))
	}
	if x.Subservice != "" {
		s.KeyString("subservice", string(x.Subservice))
	}
	if len(x.EntityTypes) > 0 {
		s.BeginList("entityTypes")
		for _, y := range x.EntityTypes {
			s.BeginBlock("")
			s.Serialize(y)
			s.EndBlock()
		}
		s.EndList()
	}
	if len(x.Entities) > 0 {
		s.BeginList("entities")
		for _, y := range x.Entities {
			s.BeginBlock("")
			s.Serialize(y)
			s.EndBlock()
		}
		s.EndList()
	}
	if !x.Environment.IsEmpty() {
		s.BeginBlock("environment")
		x.Environment.Serialize(s)
		s.EndBlock()
	}
	if !x.Deployment.IsEmpty() {
		s.BeginBlock("deployment")
		x.Deployment.Serialize(s)
		s.EndBlock()
	}
	if !x.ManifestPanels.IsEmpty() {
		s.BeginBlock("panels")
		x.ManifestPanels.Serialize(s)
		s.EndBlock()
	}
	if len(x.Subscriptions) > 0 {
		s.BeginBlock("subscriptions")
		for _, k := range serialize.Keys(x.Subscriptions) {
			v := x.Subscriptions[k]
			s.BeginBlock(k)
			s.Serialize(v)
			s.EndBlock()
		}
		s.EndBlock()
	}
	if len(x.Rules) > 0 {
		s.BeginBlock("rules")
		for _, k := range serialize.Keys(x.Rules) {
			v := x.Rules[k]
			s.BeginBlock(k)
			s.Serialize(v)
			s.EndBlock()
		}
		s.EndBlock()
	}
	if len(x.Verticals) > 0 {
		s.BeginBlock("verticals")
		for _, k := range serialize.Keys(x.Verticals) {
			v := x.Verticals[k]
			s.BeginBlock(k)
			s.Serialize(v)
			s.EndBlock()
		}
		s.EndBlock()
	}
	if len(x.Services) > 0 {
		s.BeginList("services")
		for _, y := range x.Services {
			s.BeginBlock("")
			s.Serialize(y)
			s.EndBlock()
		}
		s.EndList()
	}
	if len(x.Devices) > 0 {
		s.BeginList("devices")
		for _, y := range x.Devices {
			s.BeginBlock("")
			s.Serialize(y)
			s.EndBlock()
		}
		s.EndList()
	}
	if len(x.ServiceMappings) > 0 {
		s.BeginList("serviceMappings")
		for _, y := range x.ServiceMappings {
			s.BeginBlock("")
			s.Serialize(y)
			s.EndBlock()
		}
		s.EndList()
	}
	if len(x.Projects) > 0 {
		s.BeginList("projects")
		for _, y := range x.Projects {
			s.BeginBlock("")
			s.Serialize(y)
			s.EndBlock()
		}
		s.EndList()
	}
	if len(x.Registrations) > 0 {
		s.BeginList("registrations")
		for _, y := range x.Registrations {
			s.BeginBlock("")
			s.Serialize(y)
			s.EndBlock()
		}
		s.EndList()
	}
	if len(x.Panels) > 0 {
		s.BeginBlock("urboPanels")
		for _, k := range serialize.Keys(x.Panels) {
			v := x.Panels[k]
			s.BeginBlock(k)
			s.Serialize(v)
			s.EndBlock()
		}
		s.EndBlock()
	}
	if len(x.Tables) > 0 {
		s.BeginList("tables")
		for _, y := range x.Tables {
			s.BeginBlock("")
			s.Serialize(y)
			s.EndBlock()
		}
		s.EndList()
	}
	if len(x.Views) > 0 {
		s.BeginList("views")
		for _, y := range x.Views {
			s.BeginBlock("")
			s.Serialize(y)
			s.EndBlock()
		}
		s.EndList()
	}
}

func (x EntityType) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x EntityType) Serialize(s serialize.Serializer) {
	if x.ID != "" {
		s.KeyString("entityID", string(x.ID))
	}
	s.KeyString("entityType", string(x.Type))
	s.BeginList("attrs")
	for _, y := range x.Attrs {
		s.BeginBlock("")
		s.Serialize(y)
		s.EndBlock()
	}
	s.EndList()
}

func (x Attribute) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x Attribute) Serialize(s serialize.Serializer) {
	s.KeyString("name", string(x.Name))
	s.KeyString("type", string(x.Type))
	if len(x.Description) > 0 {
		s.BeginList("description")
		for _, y := range x.Description {
			s.String(y)
		}
		s.EndList()
	}
	if len(x.Value) > 0 {
		s.KeyRaw("value", x.Value, true)
	}
	if len(x.Metadatas) > 0 {
		s.KeyRaw("metadatas", x.Metadatas, true)
	}
	if x.SingletonKey {
		s.KeyBool("singletonKey", x.SingletonKey)
	}
	if x.Simulated {
		s.KeyBool("simulated", x.Simulated)
	}
	if x.Longterm != "" {
		s.KeyString("longterm", string(x.Longterm))
	}
	if len(x.LongtermOptions) > 0 {
		s.BeginList("longtermOptions")
		for _, y := range serialize.Sorted(x.LongtermOptions) {
			s.String(y)
		}
		s.EndList()
	}
}

func (x Entity) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x Entity) Serialize(s serialize.Serializer) {
	s.KeyString("entityID", string(x.ID))
	s.KeyString("entityType", string(x.Type))
	s.BeginBlock("attrs")
	for _, k := range serialize.Keys(x.Attrs) {
		v := x.Attrs[k]
		s.KeyRaw(k, v, true)
	}
	s.EndBlock()
	if len(x.MetaDatas) > 0 {
		s.BeginBlock("metadatas")
		for _, k := range serialize.Keys(x.MetaDatas) {
			v := x.MetaDatas[k]
			s.KeyRaw(k, v, true)
		}
		s.EndBlock()
	}
}

func (x Environment) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x Environment) Serialize(s serialize.Serializer) {
	s.BeginBlock("notificationEndpoints")
	for _, k := range serialize.Keys(x.NotificationEndpoints) {
		v := x.NotificationEndpoints[k]
		s.KeyString(k, string(v))
	}
	s.EndBlock()
}

func (x DeploymentManifest) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x DeploymentManifest) Serialize(s serialize.Serializer) {
	if len(x.Sources) > 0 {
		s.BeginBlock("sources")
		for _, k := range serialize.Keys(x.Sources) {
			v := x.Sources[k]
			s.BeginBlock(k)
			s.Serialize(v)
			s.EndBlock()
		}
		s.EndBlock()
	}
}

func (x ManifestSource) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x ManifestSource) Serialize(s serialize.Serializer) {
	if x.Path != "" {
		s.KeyString("path", string(x.Path))
	}
	if len(x.Files) > 0 {
		s.BeginList("files")
		for _, y := range x.Files {
			s.String(y)
		}
		s.EndList()
	}
}

func (x PanelManifest) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x PanelManifest) Serialize(s serialize.Serializer) {
	if len(x.Sources) > 0 {
		s.BeginBlock("sources")
		for _, k := range serialize.Keys(x.Sources) {
			v := x.Sources[k]
			s.BeginBlock(k)
			s.Serialize(v)
			s.EndBlock()
		}
		s.EndBlock()
	}
}

func (x Subscription) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x Subscription) Serialize(s serialize.Serializer) {
	s.KeyString("description", string(x.Description))
	if x.Status != "" {
		s.KeyString("status", string(x.Status))
	}
	if x.Expires != "" {
		s.KeyString("expires", string(x.Expires))
	}
	s.BeginBlock("notification")
	x.Notification.Serialize(s)
	s.EndBlock()
	s.BeginBlock("subject")
	x.Subject.Serialize(s)
	s.EndBlock()
	x.SubscriptionStatus.Serialize(s)
}

func (x Notification) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x Notification) Serialize(s serialize.Serializer) {
	if len(x.Attrs) > 0 {
		s.BeginList("attrs")
		for _, y := range serialize.Sorted(x.Attrs) {
			s.String(y)
		}
		s.EndList()
	}
	if len(x.ExceptAttrs) > 0 {
		s.BeginList("exceptAttrs")
		for _, y := range serialize.Sorted(x.ExceptAttrs) {
			s.String(y)
		}
		s.EndList()
	}
	s.KeyString("attrsFormat", string(x.AttrsFormat))
	if !x.HTTP.IsEmpty() {
		s.BeginBlock("http")
		x.HTTP.Serialize(s)
		s.EndBlock()
	}
	if !x.HTTPCustom.IsEmpty() {
		s.BeginBlock("httpCustom")
		x.HTTPCustom.Serialize(s)
		s.EndBlock()
	}
	if !x.MQTT.IsEmpty() {
		s.BeginBlock("mqtt")
		x.MQTT.Serialize(s)
		s.EndBlock()
	}
	if !x.MQTTCustom.IsEmpty() {
		s.BeginBlock("mqttCustom")
		x.MQTTCustom.Serialize(s)
		s.EndBlock()
	}
	if x.OnlyChangedAttrs {
		s.KeyBool("onlyChangedAttrs", x.OnlyChangedAttrs)
	}
	if x.Covered {
		s.KeyBool("covered", x.Covered)
	}
	x.NotificationStatus.Serialize(s)
}

func (x NotificationHTTP) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x NotificationHTTP) Serialize(s serialize.Serializer) {
	s.KeyString("url", string(x.URL))
	if x.Timeout != 0 {
		s.KeyInt("timeout", x.Timeout)
	}
}

func (x NotificationCustom) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x NotificationCustom) Serialize(s serialize.Serializer) {
	s.KeyString("url", string(x.URL))
	if x.Timeout != 0 {
		s.KeyInt("timeout", x.Timeout)
	}
	if len(x.Headers) > 0 {
		s.BeginBlock("headers")
		for _, k := range serialize.Keys(x.Headers) {
			v := x.Headers[k]
			s.KeyString(k, string(v))
		}
		s.EndBlock()
	}
	if len(x.Qs) > 0 {
		s.BeginBlock("qs")
		for _, k := range serialize.Keys(x.Qs) {
			v := x.Qs[k]
			s.KeyString(k, string(v))
		}
		s.EndBlock()
	}
	if x.Method != "" {
		s.KeyString("method", string(x.Method))
	}
	if len(x.Payload) > 0 {
		s.KeyRaw("payload", x.Payload, false)
	}
	if len(x.Json) > 0 {
		s.KeyRaw("json", x.Json, false)
	}
	if len(x.NGSI) > 0 {
		s.KeyRaw("ngsi", x.NGSI, false)
	}
}

func (x NotificationMQTT) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x NotificationMQTT) Serialize(s serialize.Serializer) {
	s.KeyString("url", string(x.URL))
	s.KeyString("string", string(x.Topic))
	if x.QoS != "" {
		s.KeyString("qos", string(x.QoS))
	}
	if x.User != "" {
		s.KeyString("user", string(x.User))
	}
	if x.Password != "" {
		s.KeyString("password", string(x.Password))
	}
}

func (x NotificationMQTTCustom) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x NotificationMQTTCustom) Serialize(s serialize.Serializer) {
	s.KeyString("url", string(x.URL))
	s.KeyString("string", string(x.Topic))
	if x.QoS != "" {
		s.KeyString("qos", string(x.QoS))
	}
	if x.User != "" {
		s.KeyString("user", string(x.User))
	}
	if x.Password != "" {
		s.KeyString("password", string(x.Password))
	}
	if len(x.Payload) > 0 {
		s.KeyRaw("payload", x.Payload, false)
	}
	if len(x.Json) > 0 {
		s.KeyRaw("json", x.Json, false)
	}
	if len(x.NGSI) > 0 {
		s.KeyRaw("ngsi", x.NGSI, false)
	}
}

func (x NotificationStatus) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x NotificationStatus) Serialize(s serialize.Serializer) {
	if x.LastFailure != "" {
		s.KeyString("lastFailure", string(x.LastFailure))
	}
	if x.LastFailureReason != "" {
		s.KeyString("lastFailureReason", string(x.LastFailureReason))
	}
	if x.LastNotification != "" {
		s.KeyString("lastNotification", string(x.LastNotification))
	}
	if x.LastSuccess != "" {
		s.KeyString("lastSuccess", string(x.LastSuccess))
	}
	if x.LastSuccessCode != 0 {
		s.KeyInt("lastSuccessCode", x.LastSuccessCode)
	}
	if x.FailsCounter != 0 {
		s.KeyInt("failsCounter", x.FailsCounter)
	}
	if x.TimesSent != 0 {
		s.KeyInt("timesSent", x.TimesSent)
	}
}

func (x Subject) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x Subject) Serialize(s serialize.Serializer) {
	s.BeginBlock("condition")
	x.Condition.Serialize(s)
	s.EndBlock()
	s.BeginList("entities")
	for _, y := range x.Entities {
		s.BeginBlock("")
		s.Serialize(y)
		s.EndBlock()
	}
	s.EndList()
}

func (x SubjectCondition) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x SubjectCondition) Serialize(s serialize.Serializer) {
	s.BeginList("attrs")
	for _, y := range serialize.Sorted(x.Attrs) {
		s.String(y)
	}
	s.EndList()
	if !x.Expression.IsEmpty() {
		s.BeginBlock("expression")
		x.Expression.Serialize(s)
		s.EndBlock()
	}
}

func (x SubjectExpression) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x SubjectExpression) Serialize(s serialize.Serializer) {
	if x.Q != "" {
		s.KeyString("q", string(x.Q))
	}
}

func (x SubjectEntity) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x SubjectEntity) Serialize(s serialize.Serializer) {
	if x.IdPattern != "" {
		s.KeyString("idPattern", string(x.IdPattern))
	}
	s.KeyString("type", string(x.Type))
}

func (x SubscriptionStatus) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x SubscriptionStatus) Serialize(s serialize.Serializer) {
	if x.ID != "" {
		s.KeyString("id", string(x.ID))
	}
}

func (x Rule) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x Rule) Serialize(s serialize.Serializer) {
	s.KeyString("name", string(x.Name))
	if x.Description != "" {
		s.KeyString("description", string(x.Description))
	}
	if x.Misc != "" {
		s.KeyString("misc", string(x.Misc))
	}
	if x.Text != "" {
		s.KeyString("text", string(x.Text))
	}
	if x.VR != "" {
		s.KeyString("VR", string(x.VR))
	}
	if len(x.Action) > 0 {
		s.KeyRaw("action", x.Action, false)
	}
	if len(x.NoSignal) > 0 {
		s.KeyRaw("nosignal", x.NoSignal, false)
	}
	x.RuleStatus.Serialize(s)
}

func (x RuleStatus) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x RuleStatus) Serialize(s serialize.Serializer) {
	if x.Subservice != "" {
		s.KeyString("subservice", string(x.Subservice))
	}
	if x.Service != "" {
		s.KeyString("service", string(x.Service))
	}
	if x.ID != "" {
		s.KeyString("_id", string(x.ID))
	}
}

func (x Vertical) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x Vertical) Serialize(s serialize.Serializer) {
	if len(x.Panels) > 0 {
		s.BeginList("panels")
		for _, y := range x.Panels {
			s.String(y)
		}
		s.EndList()
	}
	if len(x.ShadowPanels) > 0 {
		s.BeginList("shadowPanels")
		for _, y := range x.ShadowPanels {
			s.String(y)
		}
		s.EndList()
	}
	s.KeyString("slug", string(x.Slug))
	s.KeyString("name", string(x.Name))
	if x.Icon != "" {
		s.KeyString("icon", string(x.Icon))
	}
	if len(x.I18n) > 0 {
		s.KeyRaw("i18n", x.I18n, false)
	}
	x.UrboVerticalStatus.Serialize(s)
}

func (x UrboVerticalStatus) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x UrboVerticalStatus) Serialize(s serialize.Serializer) {
	if len(x.PanelsObjects) > 0 {
		s.BeginList("panelsObjects")
		for _, y := range x.PanelsObjects {
			s.BeginBlock("")
			s.Serialize(y)
			s.EndBlock()
		}
		s.EndList()
	}
	if len(x.ShadowPanelsObjects) > 0 {
		s.BeginList("shadowPanelsObjects")
		for _, y := range x.ShadowPanelsObjects {
			s.BeginBlock("")
			s.Serialize(y)
			s.EndBlock()
		}
		s.EndList()
	}
}

func (x UrboPanel) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x UrboPanel) Serialize(s serialize.Serializer) {
	s.KeyString("name", string(x.Name))
	if x.Description != "" {
		s.KeyString("description", string(x.Description))
	}
	s.KeyString("slug", string(x.Slug))
	if x.LowercaseSlug != "" {
		s.KeyString("lowercaseSlug", string(x.LowercaseSlug))
	}
	if x.WidgetCount != 0 {
		s.KeyInt("widgetCount", x.WidgetCount)
	}
	if x.IsShadow {
		s.KeyBool("isShadow", x.IsShadow)
	}
	if x.Section != "" {
		s.KeyString("section", string(x.Section))
	}
}

func (x Service) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x Service) Serialize(s serialize.Serializer) {
	s.KeyString("resource", string(x.Resource))
	s.KeyString("apikey", string(x.APIKey))
	s.KeyString("entity_type", string(x.EntityType))
	if x.Description != "" {
		s.KeyString("description", string(x.Description))
	}
	s.KeyString("protocol", string(x.Protocol))
	if x.Transport != "" {
		s.KeyString("transport", string(x.Transport))
	}
	if x.Timestamp {
		s.KeyBool("timestamp", x.Timestamp)
	}
	if len(x.ExplicitAttrs) > 0 {
		s.KeyRaw("explicitAttrs", x.ExplicitAttrs, false)
	}
	if len(x.InternalAttributes) > 0 {
		s.BeginList("internal_attributes")
		for _, y := range x.InternalAttributes {
			s.BeginBlock("")
			s.Serialize(y)
			s.EndBlock()
		}
		s.EndList()
	}
	s.BeginList("attributes")
	for _, y := range x.Attributes {
		s.BeginBlock("")
		s.Serialize(y)
		s.EndBlock()
	}
	s.EndList()
	if len(x.Lazy) > 0 {
		s.BeginList("lazy")
		for _, y := range x.Lazy {
			s.BeginBlock("")
			s.Serialize(y)
			s.EndBlock()
		}
		s.EndList()
	}
	if len(x.StaticAttributes) > 0 {
		s.BeginList("static_attributes")
		for _, y := range x.StaticAttributes {
			s.BeginBlock("")
			s.Serialize(y)
			s.EndBlock()
		}
		s.EndList()
	}
	if len(x.Commands) > 0 {
		s.BeginList("commands")
		for _, y := range x.Commands {
			s.BeginBlock("")
			s.Serialize(y)
			s.EndBlock()
		}
		s.EndList()
	}
	if x.ExpressionLanguage != "" {
		s.KeyString("expressionLanguage", string(x.ExpressionLanguage))
	}
	if x.EntityNameExp != "" {
		s.KeyString("entityNameExp", string(x.EntityNameExp))
	}
	x.GroupStatus.Serialize(s)
}

func (x DeviceAttribute) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x DeviceAttribute) Serialize(s serialize.Serializer) {
	s.KeyString("object_id", string(x.ObjectId))
	s.KeyString("name", string(x.Name))
	if x.Type != "" {
		s.KeyString("type", string(x.Type))
	}
	if len(x.Value) > 0 {
		s.KeyRaw("value", x.Value, false)
	}
	if x.Expression != "" {
		s.KeyString("expression", string(x.Expression))
	}
	if x.EntityName != "" {
		s.KeyString("entity_name", string(x.EntityName))
	}
	if x.EntityType != "" {
		s.KeyString("entity_type", string(x.EntityType))
	}
}

func (x DeviceCommand) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x DeviceCommand) Serialize(s serialize.Serializer) {
	if x.ObjectId != "" {
		s.KeyString("object_id", string(x.ObjectId))
	}
	if x.Name != "" {
		s.KeyString("name", string(x.Name))
	}
	if x.Type != "" {
		s.KeyString("type", string(x.Type))
	}
	if x.Value != "" {
		s.KeyString("value", string(x.Value))
	}
	if len(x.MQTT) > 0 {
		s.KeyRaw("mqtt", x.MQTT, false)
	}
}

func (x GroupStatus) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x GroupStatus) Serialize(s serialize.Serializer) {
	if x.ID != "" {
		s.KeyString("_id", string(x.ID))
	}
	if x.V != 0 {
		s.KeyInt("__v", x.V)
	}
	if x.IOTAgent != "" {
		s.KeyString("iotagent", string(x.IOTAgent))
	}
	if x.ServicePath != "" {
		s.KeyString("service_path", string(x.ServicePath))
	}
	if x.Service != "" {
		s.KeyString("service", string(x.Service))
	}
	if x.CBHost != "" {
		s.KeyString("cbHost", string(x.CBHost))
	}
}

func (x Device) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x Device) Serialize(s serialize.Serializer) {
	s.KeyString("device_id", string(x.DeviceId))
	if x.APIKey != "" {
		s.KeyString("apikey", string(x.APIKey))
	}
	if x.EntityName != "" {
		s.KeyString("entity_name", string(x.EntityName))
	}
	s.KeyString("entity_type", string(x.EntityType))
	if x.Polling {
		s.KeyBool("polling", x.Polling)
	}
	s.KeyString("transport", string(x.Transport))
	if x.Timestamp {
		s.KeyBool("timestamp", x.Timestamp)
	}
	if x.Endpoint != "" {
		s.KeyString("endpoint", string(x.Endpoint))
	}
	if len(x.Attributes) > 0 {
		s.BeginList("attributes")
		for _, y := range x.Attributes {
			s.BeginBlock("")
			s.Serialize(y)
			s.EndBlock()
		}
		s.EndList()
	}
	if len(x.Lazy) > 0 {
		s.BeginList("lazy")
		for _, y := range x.Lazy {
			s.BeginBlock("")
			s.Serialize(y)
			s.EndBlock()
		}
		s.EndList()
	}
	if len(x.Commands) > 0 {
		s.BeginList("commands")
		for _, y := range x.Commands {
			s.BeginBlock("")
			s.Serialize(y)
			s.EndBlock()
		}
		s.EndList()
	}
	if len(x.StaticAttributes) > 0 {
		s.BeginList("static_attributes")
		for _, y := range x.StaticAttributes {
			s.BeginBlock("")
			s.Serialize(y)
			s.EndBlock()
		}
		s.EndList()
	}
	s.KeyString("protocol", string(x.Protocol))
	if x.ExpressionLanguage != "" {
		s.KeyString("expressionLanguage", string(x.ExpressionLanguage))
	}
	if len(x.ExplicitAttrs) > 0 {
		s.KeyRaw("explicitAttrs", x.ExplicitAttrs, false)
	}
	x.DeviceStatus.Serialize(s)
}

func (x DeviceStatus) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x DeviceStatus) Serialize(s serialize.Serializer) {
	if x.Service != "" {
		s.KeyString("service", string(x.Service))
	}
	if x.ServicePath != "" {
		s.KeyString("service_path", string(x.ServicePath))
	}
}

func (x ServiceMapping) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x ServiceMapping) Serialize(s serialize.Serializer) {
	if x.OriginalService != "" {
		s.KeyString("originalService", string(x.OriginalService))
	}
	if x.NewService != "" {
		s.KeyString("newService", string(x.NewService))
	}
	s.BeginList("servicePathMappings")
	for _, y := range x.ServicePathMappings {
		s.BeginBlock("")
		s.Serialize(y)
		s.EndBlock()
	}
	s.EndList()
}

func (x ServicePathMapping) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x ServicePathMapping) Serialize(s serialize.Serializer) {
	if x.OriginalServicePath != "" {
		s.KeyString("originalServicePath", string(x.OriginalServicePath))
	}
	if x.NewServicePath != "" {
		s.KeyString("newServicePath", string(x.NewServicePath))
	}
	s.BeginList("entityMappings")
	for _, y := range x.EntityMappings {
		s.BeginBlock("")
		s.Serialize(y)
		s.EndBlock()
	}
	s.EndList()
}

func (x EntityMapping) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x EntityMapping) Serialize(s serialize.Serializer) {
	if x.OriginalEntityId != "" {
		s.KeyString("originalEntityId", string(x.OriginalEntityId))
	}
	if x.NewEntityId != "" {
		s.KeyString("newEntityId", string(x.NewEntityId))
	}
	if x.OriginalEntityType != "" {
		s.KeyString("originalEntityType", string(x.OriginalEntityType))
	}
	if x.NewEntityType != "" {
		s.KeyString("newEntityType", string(x.NewEntityType))
	}
	s.BeginList("attributeMappings")
	for _, y := range x.AttributeMappings {
		s.BeginBlock("")
		s.Serialize(y)
		s.EndBlock()
	}
	s.EndList()
}

func (x AttributeMapping) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x AttributeMapping) Serialize(s serialize.Serializer) {
	if x.OriginalAttributeName != "" {
		s.KeyString("originalAttributeName", string(x.OriginalAttributeName))
	}
	if x.OriginalAttributeType != "" {
		s.KeyString("originalAttributeType", string(x.OriginalAttributeType))
	}
	if x.NewAttributeName != "" {
		s.KeyString("newAttributeName", string(x.NewAttributeName))
	}
	if x.NewAttributeType != "" {
		s.KeyString("newAttributeType", string(x.NewAttributeType))
	}
}

func (x Project) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x Project) Serialize(s serialize.Serializer) {
	s.KeyBool("is_domain", x.IsDomain)
	if x.Description != "" {
		s.KeyString("description", string(x.Description))
	}
	if len(x.Tags) > 0 {
		s.KeyRaw("tags", x.Tags, false)
	}
	s.KeyBool("enabled", x.Enabled)
	s.KeyString("id", string(x.ID))
	if x.ParentId != "" {
		s.KeyString("parent_id", string(x.ParentId))
	}
	if x.DomainId != "" {
		s.KeyString("domain_id", string(x.DomainId))
	}
	s.KeyString("name", string(x.Name))
	x.ProjectStatus.Serialize(s)
}

func (x ProjectStatus) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x ProjectStatus) Serialize(s serialize.Serializer) {
	if len(x.Links) > 0 {
		s.KeyRaw("links", x.Links, false)
	}
}

func (x Registration) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x Registration) Serialize(s serialize.Serializer) {
	s.KeyString("id", string(x.ID))
	s.KeyString("description", string(x.Description))
	if len(x.DataProvided) > 0 {
		s.KeyRaw("dataProvided", x.DataProvided, false)
	}
	if len(x.Provider) > 0 {
		s.KeyRaw("provider", x.Provider, false)
	}
	x.RegistrationStatus.Serialize(s)
}

func (x RegistrationStatus) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x RegistrationStatus) Serialize(s serialize.Serializer) {
	s.KeyString("status", string(x.Status))
}

func (x Table) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x Table) Serialize(s serialize.Serializer) {
	s.KeyString("name", string(x.Name))
	s.BeginList("columns")
	for _, y := range x.Columns {
		s.BeginBlock("")
		s.Serialize(y)
		s.EndBlock()
	}
	s.EndList()
	s.BeginList("primaryKey")
	for _, y := range x.PrimaryKey {
		s.String(y)
	}
	s.EndList()
	s.BeginList("indexes")
	for _, y := range x.Indexes {
		s.BeginBlock("")
		s.Serialize(y)
		s.EndBlock()
	}
	s.EndList()
	s.KeyBool("lastdata", x.LastData)
	if len(x.Singleton) > 0 {
		s.BeginList("singleton")
		for _, y := range x.Singleton {
			s.String(y)
		}
		s.EndList()
	}
}

func (x TableColumn) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x TableColumn) Serialize(s serialize.Serializer) {
	s.KeyString("name", string(x.Name))
	s.KeyString("type", string(x.Type))
	if x.NotNull {
		s.KeyBool("notNull", x.NotNull)
	}
	if x.Default != "" {
		s.KeyString("default", string(x.Default))
	}
}

func (x TableIndex) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x TableIndex) Serialize(s serialize.Serializer) {
	s.KeyString("name", string(x.Name))
	s.BeginList("columns")
	for _, y := range x.Columns {
		s.String(y)
	}
	s.EndList()
	if x.Geometry {
		s.KeyBool("geometry", x.Geometry)
	}
}

func (x View) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x View) Serialize(s serialize.Serializer) {
	if x.Materialized {
		s.KeyBool("materialized", x.Materialized)
	}
	s.KeyString("name", string(x.Name))
	s.KeyString("from", string(x.From))
	s.BeginList("group")
	for _, y := range x.Group {
		s.String(y)
	}
	s.EndList()
	s.BeginList("columns")
	for _, y := range x.Columns {
		s.BeginBlock("")
		s.Serialize(y)
		s.EndBlock()
	}
	s.EndList()
}

func (x ViewColumn) MarshalJSON() ([]byte, error) {
	return serialize.MarshalJSON(x)
}

func (x ViewColumn) Serialize(s serialize.Serializer) {
	s.KeyString("name", string(x.Name))
	s.KeyString("expression", string(x.Expression))
}
