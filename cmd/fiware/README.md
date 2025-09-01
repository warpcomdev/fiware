# Fiware

Esta aplicación permite conectar a un entorno fiware (de desarrollo, en la nube, u on-premise) que contenga los componentes estándar de plataforma (context broker, CEP, agent manager, IdM, etc), y operar sobre él:

- Enumerar suscripciones, device groups, devices y reglas de CEP.
- Crear suscripciones, device groups, devices y reglas de CEP.
- Eliminar suscripciones, device groups, devices, reglas de CEP y entidades.

Utilice el comando `fiware -h` o `go run fiware -h` para obtener detalles del modo de uso:

```
$ fiware -h
NAME:
   FIWARE CLI client - manage fiware environments

USAGE:
   FIWARE CLI client [global options] command [command options] [arguments...]

DESCRIPTION:
   Manage fiware verticals and environments

COMMANDS:
   upload, up  Upload panels to urbo
   help, h     Shows a list of commands or help for one command
   config:
     context, ctx  Manage contexts
   platform:
     login, auth          Login into keystone
     get                  Get some resource (services, devices, suscriptions, rules, projects, panels, verticals, entities, regitrations)
     download, down, dld  Download vertical or subservice
     post                 Post some resource (services, devices, suscriptions, rules, entities, verticals)
     delete               Delete some resource (services, devices, suscriptions, rules, entities)
     serve                Turn on http server
   template:
     decode, import  decode NGSI README.md or CSV file
     export          Read datafile and export with context params
     template        template for vertical data

GLOBAL OPTIONS:
   --context value, -c value  Path to the context configuration file (default: ${XDG_CONFIG_DIR}/fiware.json) [$FIWARE_CONTEXT]
   --help, -h                 show help (default: false)
```

## Tutoriales

- [Instalación](doc/instalacion.md)
- [Primeros pasos](doc/primeros_pasos.md)
- [Creando verticales](doc/gestion_verticales.md)
