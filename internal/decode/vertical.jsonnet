// to read settings from external file:
// local settings = import("settings.libsonnet");
local settings = {
    cygnus_notification_url: "http://cygnus.fiware.com:5000/notify",
};

{
    /* BEGIN REPLACE */
    // Esta sección de la configuración es de ejemplo.
    // Será reemplazada por los datos reales extraidos de la vertical.
    name: "verticalName",
    subservice: "subService",
    entityTypes: [
        {
            entityID: "test1",
            entityType: "type1",
            attrs: [
                {
                    name: "attrib1",
                    type: "Text",
                    value: "text1",
                },
            ],
        },
    ],
    /* END REPLACE */

    // Formato de entity mapping por defecto.
    local defaultEntityMappings = [
        {
            originalEntityId: "(.+)",
            newEntityId: std.asciiLower($.name),
            originalEntityType: entityType.entityType,
            newEntityType: std.asciiLower(entityType.entityType),
            attributeMappings: [],
        }
        for entityType in $.entityTypes
    ],

    serviceMappings: [
        {
            servicePathMappings: [
                {
                    originalServicePath: "/" + $.subservice,
                    newServicePath: "/",
                    entityMappings: defaultEntityMappings
                },
                {
                    // Servicios mancomunados
                    originalServicePath: "/(.*)/" + $.subservice,
                    newServicePath: "/$1",
                    entityMappings: defaultEntityMappings
                },
            ],
        },
    ],

    // Suscripciones por defecto: Una suscripcion a postgres por cada entidad
    local defaultSuscriptions = [
        {
            subject: {
                entities: [
                    {
                        idPattern: ".*",
                        type: entityType.entityType,
                    },
                ],
                condition: {
                    attrs: [ "TimeInstant" ],
                },
            },
            notification: {
                http: {
                    url: settings.cygnus_notification_url,
                },
                attrs: [
                    // Por defecto, no notificamos los atributos
                    // relacionados con comandos
                    attr.name
                    for attr in entityType.attrs
                    if !std.startsWith(attr.type, "command")
                ],
            },
        }
        for entityType in $.entityTypes
    ],

    suscriptions: defaultSuscriptions,

    // Formato de columna de base de datos típico por atributo
    local defaultTableColumnFor(attr) = 
        local name = std.asciiLower(attr.name);
        local type = std.asciiLower(attr.type);
        if name == "timeinstant" then {
            name: name,
            type: "timestamp with time zone",
            notNull: true,
        } else if name == "municipality" then {
            name: name,
            type: "text",
            notNull: true,
            default: "NA",
        } else if type == "number" then {
            name: name,
            type: "double precision",
        } else if type == "datetime" then {
            name: name,
            type: "timestamp with time zone",
        } else if name == "location" then {
            name: name,
            type: "geometry(Point)",
        } else if std.startsWith(type, "geo") then {
            name: name,
            type: "geometry",
        } else if std.startsWith(type, "bool") then {
            name: name,
            type: "bool",
        } else {
            name: name,
            type: "text",
        },

    // Nombre de tabla por defecto
    local defaultTableNameFor(entityType) = $.name + "_" + std.asciiLower(entityType.entityType),

    // Columnas derivadas de los atributos de la entidad (exceptuando las fijas:
    // entityid, entityType, recvtime y fiwareServicePath)
    // No consideramos los atributos relacionados con comandos.
    local defaultTableColumnsFor(entityType) =
        [ 
            defaultTableColumnFor(attr)
            for attr in entityType.attrs
            if !std.startsWith(attr.type, "command")
        ],

    // Índices por defecto para un entiType
    local defaultTableIndexesFor(entityType, tableName, columns) = 
        local columnMap = { [col.name]: col for col in columns };
        [
            {
                name: tableName + "_ld_idx",
                columns: ["entityid", "timeinstant DESC"],
            }
        ] + (
        if std.objectHas(columnMap, "location") then [
            {
                name: tableName + "_gm_idx",
                columns: ["location"],
                geometry: true,
            }
        ] else []) + (
            if std.objectHas(columnMap, "municipality") then [
            {
                name: tableName + "_mun_idx",
                columns: ["municipality", "timeinstant"],
            }
        ] else []),

    // Tabla por defecto para un entityType
    local defaultTableFor(entityType, tableName, columns) =
        {
            name: tableName,
            columns: columns,
            indexes: defaultTableIndexesFor(entityType, tableName, columns),
            primaryKey: [ "timeinstant", "entityid" ],
            lastdata: true,
        },

    tables: [ 
        defaultTableFor(
            entityType,
            defaultTableNameFor(entityType),
            defaultTableColumnsFor(entityType),
        )
        for entityType in $.entityTypes
    ],
}
