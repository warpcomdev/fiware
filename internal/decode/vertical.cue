import (
    "strings"
    "list"
)

{
    // Datos básicos de la vertical
    name: string
    subservice: string
    entityTypes: [...#entityType]

    // Tipos de entidad
    #entityType: {
        entityID: string
        entityType: string
        attrs: [...#attr]

        #attr: {
            name: string
            type: string
            value: string | *""
        }
    }

    // BEGIN REPLACE
    // Esta sección de la configuración es de ejemplo.
    // Será reemplazada por los datos reales extraidos de la vertical.
    // END REPLACE

    let _subservice = subservice
    serviceMappings: [{
        servicePathMappings: [{
            originalServicePath: "/\(_subservice)",
            newServicePath: "/",
            entityMappings: #entityMappings
        }, {
            // Servicios mancomunados
            originalServicePath: "/(.*)/\(_subservice)",
            newServicePath: "/$1",
            entityMappings: #entityMappings
        }]
    }],

    // Formato de entity mapping por defecto.
    let _verticalName = name
    #entityMappings: [for _, entityType in entityTypes {{
        originalEntityId: "(.+)",
        newEntityId: strings.ToLower(_verticalName),
        originalEntityType: entityType.entityType,
        newEntityType: strings.ToLower(entityType.entityType),
        attributeMappings: [],
    }}]

    // Suscripciones por defecto: Una suscripcion a postgres y otra a lastdata
    // por cada entidad
    #subscription: {
        _entityType: #entityType
        description: string
        subject: {
            entities: [{
                idPattern: ".*",
                type: _entityType.entityType,
            }],
            condition: {
                attrs: [ "TimeInstant" ],
            }
        }
        notification: {
            attrs: [for attr in _entityType.attrs
            // Por defecto, no notificamos los atributos
            // relacionados con comandos
            if !strings.HasPrefix(attr.type, "command") {
                attr.name
            }]
            http: url: string
        }
    }
    
    suscriptions: [for entityType in entityTypes {
        #subscription & {
            _entityType: entityType,
            description: "Suscripción a postgres para " + entityType.entityType
            notification: http: url: "http://iot-cygnus:5057/notify"
        }
    }] + [for entityType in entityTypes {
        #subscription & {
            _entityType: entityType,
            description: "Suscripción a postgres (lastdata) para " + entityType.entityType
            notification: http: url: "http://iot-cygnus:5059/notify"
        }
    }]

    tables: [for entityType in entityTypes {{
        name: _verticalName + "_" + strings.ToLower(entityType.entityType)
        primaryKey: [ "timeinstant", "entityid" ]
        lastdata: true // Añadir tabla de lastdata
        columns: [...#column]
        indexes: [...#index]

        columns: [for attr in entityType.attrs {{
            _attr: attr
        }}]
    }}]

    #column: {
        _attr: #entityType.#attr
        name: strings.ToLower(_attr.name)
        if name == "timeinstant" || name == "municipality" {
            notNull: true
        }
        if name == "municipality" {
            default: "NA"
        }
        let _type= _attr.type
        type: [
        if name == "timeinstant" {
            "timestamp with time zone"
        },
        if _type == "number" {
            "double precision"
        },
        if _type == "datetime" {
            "timestamp with time zone"
        },
        if name == "location" {
            "geometry(Point)"
        },
        if strings.HasPrefix(_type, "geo") {
            "geometry"
        },
        if strings.HasPrefix(_type, "bool") {
            "bool"
        },
        "text"][0]
    }

    #index: {
        _tablename: string
        _suffix: string
        name: _tablename + _suffix
        columns: [...string]
        geometry: bool | *false
    }

    // Indexable columns we expect to find
    _indexable: [string]: #index
    _indexable: {
        "timeinstant": {
            _suffix: "_ld_idx",
            columns: ["entityid", "timeinstant DESC"],
        }
        "location": {
            _suffix: "_gm_idx",
            columns: ["location"],
            geometry: true,
        }
        "municipality": {
            _suffix: "_mun_idx",
            columns: ["municipality", "timeinstant"],
        }
    }

    // Build table indexes
    tables: [...{
        name: string
        columns: [...#column]
        let _colNames = [for _, col in columns {col.name}]
        indexes: [for k, v in _indexable if list.Contains(_colNames, k) {
            v & {_tablename: name}
        }]
    }]
}
