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
        entityID?: string
        entityType: string
        attrs: [...#attr]

        #attr: {
            name: string
            type: string
            description?: [...string]
            singletonKey: bool | *false
            simulated: bool | *false
            longterm?: "gauge" | "counter" | "enum" | "modal" | "dimension"
            longtermOptions?: [...string]
            value: #any | *""
            metadatas?: [...#any]
        }
    }

    #any: _

    // BEGIN REPLACE
    // Esta sección de la configuración es de ejemplo.
    // Será reemplazada por los datos reales extraidos de la vertical.
    // END REPLACE

    #replaceId: {for et in entityTypes {
        (et.entityType): {
            attrs: [for attr in et.attrs if attr.singletonKey { attr.name }]
            text: strings.Join(attrs, "}_${")
        }
    }}
    
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
            attrsFormat: "normalized"
            attrs: [for attr in _entityType.attrs
            // Por defecto, no notificamos los atributos
            // relacionados con comandos
            if !strings.HasPrefix(attr.type, "command") {
                attr.name
            }]
            http?: url: string
            httpCustom?: {
                url: string
                ngsi?: {...}
            }
        }
    }
    
    environment: notificationEndpoints: {
        CYGNUS: "http://iot-cygnus:5057/notify"
        LASTDATA: "http://iot-cygnus:5059/notify"
    }

    subscriptions: {for entityType in entityTypes {
        "\(entityType.entityType):CYGNUS": #subscription & {
            _entityType: entityType,
            description: "Suscripción a POSTGRES para " + entityType.entityType
            // PREVIEW FEATURE: Cuando se soporte el parámetro "ngsi"
            // en las suscripciones, se podrá volver a habilitar replaceId
            if len(#replaceId[entityType.entityType].attrs) == 0 {
                notification: http: url: "CYGNUS"
            }
            if len(#replaceId[entityType.entityType].attrs) > 0 {
                notification: httpCustom: url: "CYGNUS"
                notification: httpCustom: ngsi: id: "${\(#replaceId[entityType.entityType].text)}"
            }
        }
    }}
    subscriptions: {for entityType in entityTypes {
        "\(entityType.entityType):LASTDATA": #subscription & {
            _entityType: entityType,
            description: "Suscripción a POSTGRES lastdata para " + entityType.entityType
            // PREVIEW FEATURE: Cuando se soporte el parámetro "ngsi"
            // en las suscripciones, se podrá volver a habilitar replaceId
            //if len(#replaceId[entityType.entityType].attrs) == 0 {
                notification: http: url: "LASTDATA"
            //}
            //if len(#replaceId[entityType.entityType].attrs) > 0 {
            //    notification: httpCustom: url: "LASTDATA"
            //    notification: httpCustom: ngsi: id: "${\(#replaceId[entityType.entityType].text)}"
            //}
        }
    }}

    tables: [for entityType in entityTypes {{
        name: _verticalName + "_" + strings.ToLower(entityType.entityType)
        lastdata: true // Añadir tabla de lastdata
        columns: [...#column]
        indexes: [...#index]
        // not deprecated yet: ngsi custom suscription does not work
        singleton: [for _attr in entityType.attrs if _attr.singletonKey { strings.ToLower(_attr.name) }]
        primaryKey: [ "timeinstant", "entityid" ] + singleton

        columns: [
            for attr in entityType.attrs
            if !strings.HasPrefix(attr.type, "command") {{
                _attr: attr
            }}
        ]
    }}]

    // Estos atributos son "municipality" y no pueden ser NULL
    let _municipalityAttribs = ["zip", "zone", "district", "municipality", "region", "province", "community", "country"]

    // Añado el atributo "longterm" a las columnas que lo necesiten.
    entityTypes: [...{
        attrs: [...{
            simulated: true
            name: string
            type: string
            if list.Contains(_municipalityAttribs, name) {
                longterm: "dimension"
            }
        }]
    }]

    // Materialized views
    _hasLongterm: {
        for entityType in entityTypes {
            for attr in entityType.attrs if attr.longterm != _|_ {
                if attr.longterm != "dimension" {
                    "\(entityType.entityType)": entityType
                }
            }
        }
    }
    views: [for _, entityType in _hasLongterm {{
        materialized: true,
        name: _verticalName + "_" + strings.ToLower(entityType.entityType) + "_mv"
        from: _verticalName + "_" + strings.ToLower(entityType.entityType)
	    _longterms: [
            for _attr in entityType.attrs
            if _attr.longterm != _|_ {{
                attr: _attr
                kind: _attr.longterm
            }}
        ]
	    group: [
            for col in _longterms
            if col.kind == "dimension" {
                col.attr.name
            }
        ]
        columns: list.FlattenN([
            for col in _longterms {
                let _lowerName = strings.ToLower(col.attr.name)
                if col.kind == "dimension" {
                    [{
                        name: _lowerName,
                        expression: _lowerName
                    }]
                }
                if col.kind == "gauge" || col.kind == "counter" {
                    [{
                        name: "min\(_lowerName)",
                        expression: "MIN(\(_lowerName))"
                    },{
                        name: "max\(_lowerName)",
                        expression: "MAX(\(_lowerName))"
                    },{
                        name: "avg\(_lowerName)",
                        expression: "AVG(\(_lowerName))"
                    },{
                        name: "sum\(_lowerName)",
                        expression: "SUM(\(_lowerName))"
                    },{
                        name: "cont\(_lowerName)",
                        expression: "COUNT(\(_lowerName))"
                    },{
                        name: "dev\(_lowerName)",
                        expression: "STDDEV(\(_lowerName))"
                    },{
                        name: "var\(_lowerName)",
                        expression: "VARIANCE(\(_lowerName))"
                    },{
                        name: "med\(_lowerName)",
                        expression: "PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY \(_lowerName))"
                    }]
                }
                if col.kind == "modal" {
                    [{
                        name: "mode\(_lowerName)",
                        expression: "MODE() WITHIN GROUP (ORDER BY \(_lowerName))"
                    }]
                }
                if col.kind == "enum" && col.attr.longtermOptions != _|_ {
                    [{
                        name: "mode\(_lowerName)",
                        expression: "MODE() WITHIN GROUP (ORDER BY \(_lowerName))"
                    }] + [for option in col.attr.longtermOptions {
                        name: "\(option)\(_lowerName)",
                        expression: "COUNT(\(_lowerName)) FILTER (WHERE \(_lowerName) = '\(option)')"
                    }]
                }
            }
        ], 1)
    }}]

    #column: {
        _attr: #entityType.#attr
        name: strings.ToLower(_attr.name)
        if name == "timeinstant" {
            notNull: true
        }
        // deprecated
        //if list.Contains(_municipalityAttribs, name) {
        //    notNull: true
        //    default: "NA"
        //}
        let _type= strings.ToLower(_attr.type)
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
        if _type == "json" {
            "json"
        },
        if strings.HasPrefix(_type, "list ") {
            "json"
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
        },
        "zip": {
            _suffix: "_zip_idx",
            columns: ["zip", "timeinstant"],
        },
        "zone": {
            _suffix: "_zon_idx",
            columns: ["zone", "timeinstant"],
        },
        "district": {
            _suffix: "_dis_idx",
            columns: ["district", "timeinstant"],
        },
        "region": {
            _suffix: "_reg_idx",
            columns: ["region", "timeinstant"],
        },
        "province": {
            _suffix: "_pro_idx",
            columns: ["province", "timeinstant"],
        },
        "community": {
            _suffix: "_com_idx",
            columns: ["community", "timeinstant"],
        },
        "country": {
            _suffix: "_cou_idx",
            columns: ["country", "timeinstant"],
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

    services: [for entityType in entityTypes {{
        resource: "/iot/json"
        apikey: "CHANGEME!"
        entity_type: entityType.entityType
        description: "JSON_IoT_Agent_Node"
        protocol: "IoTA-JSON"
        transport: "http"
        expressionLanguage: "jexl"
        attributes: [
            for attr in entityType.attrs
            if attr.type != "command" {{
                object_id: attr.name
                name: attr.name
                type: attr.type
            }}
        ]
    }}]
}
