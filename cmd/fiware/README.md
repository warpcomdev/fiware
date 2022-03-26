# Fiware

Esta aplicación permite conectar a un entorno fiware (de desarrollo, en la nube, u on-premise) que contenga los componentes estándar de esta plataforma (context broker, CEP, agent manager, IdM, etc), y operar sobre él:

- Enumerar suscripciones, device groups, devices y reglas de CEP.
- Crear suscripciones, device groups, devices y reglas de CEP.
- Eliminar suscripciones, device groups, devices y reglas de CEP.

Utilice el comando `fiware -h` o `go run fiware -h` para obtener detalles del modo de uso:

```bash
NAME:
   fiware - A new cli application

USAGE:
   fiware [global options] command [command options] [arguments...]

DESCRIPTION:
   Manage fiware verticals and environments

COMMANDS:
   help, h  Shows a list of commands or help for one command
   config:
     context, ctx  Manage contexts
   platform:
     login, auth  Login into keystone
     get          Get some resource (services, devices, suscriptions, rules)
     post         Post some resource (services, devices, suscriptions, rules)
     delete       Delete some resource (services, devices, suscriptions, rules)
   template:
     decode    decode NGSI README.md file
     template  template for vertical data

GLOBAL OPTIONS:
   --context value, -c value  Path to the context configuration file (default: ${XDG_CONFIG_DIR}/fiware.json) [$FIWARE_CONTEXT]
   --help, -h                 show help (default: false)
```

## Gestión de verticales

### Decode

El comando `fiware decode` lee el fichero `models/ngsi/README.md` de una vertical, y genera una versión *estandar* del modelo descriptivo de la vertical, con el conjunto mínimo de name-mappings, suscripciones y tablas de base de datos que la vertical requiere.

```bash
$ fiware decode -h
NAME:
   fiware decode - decode NGSI README.md file

USAGE:
   fiware decode [command options] [arguments...]

CATEGORY:
   template

OPTIONS:
   --output FILE, -o FILE          write output to FILE
   --vertical value, -v value      vertical name (without '-vertical' suffix) (default: vertical)
   --subservice value, --ss value  subservice name (without '/' prefix) (default: subservice)
   --help, -h                      show help (default: false)
```

### Template

El comando `fiware template` lee el modelo de datos de un fichero proporcionado con el flag `-d FILE`, y le aplica un template para generar un fichero de texto. Actualmente, los formatos soportados son:

- Fichero de datos:
   - json
   - [jsonnet](https://jsonnet.org/)
- Template:
   - [golang text/template](https://pkg.go.dev/text/template).

```bash
$ fiware template -h
NAME:
   fiware template - template for vertical data

USAGE:
   fiware template [command options] [arguments...]

CATEGORY:
   template

OPTIONS:
   --data FILE, -d FILE    read vertical data from FILE
   --output FILE, -o FILE  write template output to FILE
   --help, -h              show help (default: false)
```

Además de los datos definidos en el fichero de vertical, se añade al modelo un atributo `params` con todos los parámetros que se hayan definido en el contexto (ver [contextos](#contextos)). Así, para compartir con el template un atributo como por ejemplo la URL del servidor cygnus, se puede añadir al contexto con el comando:

```fiware context params cygnus_url http://cygnus.fiware.com:8080```

Y ese valor será accesible desde dentro del template, usando la ruta `{{ .params.cygnus_url }}`.

La aplicación tiene varias plantillas predefinidas que pueden servir de punto de partida rápido para generar la documentación de un vertical:

- `dump.tmpl`: Convierte la entrada `jsonnet` en una salida `json` estándar.
- `default_cygnus.tmpl`: Genera un fichero de name_mappings para cygnus.
- `default_subs.tmpl`: Genera el típico fichero de suscripciones de la vertical.
- `default_readme.tmpl`: Genera el típico fichero `README.md` del modelo de datos la vertical.
- `default_ddls.tmpl`: Genera el típico fichero SQL de la vertical, con un conjunto típico de tablas y vistas.

Las plantillas predefinidas también publican algunos bloques reutilizables, en particular:

- `default_ddls_sets`: Genera los comandos `\set` del fichero SQL.
- `default_ddls_tables`: Genera los comandos SQL `create table` (y las vistas lastdata) del fichero SQL.
- `dump`: Vuelca todos los parámetros del template en formato json.

De esta forma, es posible extender los formatos por defecto creando una plantilla personalizada para cada vertical en cuestión, que contenga únicamente las vistas personalizadas y reutilice los bloques comunes de las plantillas predefinidas.

Por ejemplo, una vertical con vistas personalizadas podría usar la siguiente plantilla para generar su fichero `ddls.sql`:

```sql
-- defino sets "custom" para mi vertical
\set my_custom_view 'my_custom_view'

-- Cargo los set y vistas "estándar"
{{ template "default_ddls_sets" . }}
{{ template "default_ddls_tables" . }}

-- Y por último, mis vistas customizadas
create view :target_schema.:scope:my_custom_view as (
   ...
);
```

## Configuración

### Contextos

La herramienta puede gestionar varios **contextos** de conexión, que representan diferentes entornos: URL de keystone, Orion, etc. Por ejemplo, un contexto para trabajar con el usuario *admin_user* en un entorno hipotético *fiware.platform.com* tendría estos datos:

```json
{
  "name": "fiware_demo",
  "keystone": "http://fiware.platform.com:5001",
  "orion": "http://fiware.platform.com:1026",
  "iotam": "http://fiware.platform.com:8082",
  "perseo": "http://fiware.platform.com:9090",
  "service": "demoservice",
  "subservice": "",
  "username": "admin_user"
}
```

La aplicación puede gestionar múltiples contextos usando el comando `fiware context`:

```bash
$ fiware context help
NAME:
   fiware context - Manage contexts

USAGE:
   fiware context [global options] command [command options] [arguments...]

COMMANDS:
   create      Create a new context
   delete, rm  Delete a context
   list, ls    List all contexts
   use         Use a context
   info, show  Show context configuration
   dup         Duplicate the current context
   set         Set a context variable
   params      Set a template parameter
   help, h     Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h  show help (default: false)
```

Para empezar a trabajar con la aplicación, el primer paso es crear un contexto:

```bash
$ fiware ctx create fiware_demo
Using new context fiware_demo
```

Una vez creado el contexto, se pueden configurar los distintos valores con `fiware ctx set`:

```bash
$ fiware ctx set keystone "http://fiware.platform.com:5001" orion "http://fiware.platform.com:1026" service demoservice
using context fiware_demo
context settings:
{
   "name": "fiware_demo",
   "keystone": "http://fiware.platform.com:5001",
   "orion": "http://fiware.platform.com:1026",
   "iotam": "",
   "perseo": "",
   "service": "demoservice",
   "subservice": "",
   "username": ""
}
```

Los valores se pueden cambiar tambien individualmente:

```bash
$ fiware ctx set perseo "http://fiware.platform.com:9090"
using context fiware_demo
context settings:
{
  "name": "fiware_demo",
  "keystone": "http://fiware.platform.com:5001",
  "orion": "http://fiware.platform.com:1026",
  "iotam": "",
  "perseo": "http://fiware.platform.com:9090",
  "service": "demoservice",
  "subservice": "",
  "username": ""
}
```

La lista de contextos configurados puede verse con el comando `fiware context list`, y el contexto seleccionado puede modificarse con `fiware context use`:

```bash
$ fiware ctx ls
* fiware_demo
lab_demoservice
saas

$ ./fiware ctx use lab_demoservice
using context lab_demoservice
```

## Operación de entornos

### Login

El comando `fiware login` inicia sesión usando el servidor keystone, servicio y usuario configurados en el contexto, solicitando el password en la terminal:

```bash
$ fiware auth -h
NAME:
   fiware login - Login into keystone

USAGE:
   fiware login [command options] [arguments...]

CATEGORY:
   platform

OPTIONS:
   --help, -h  show help (default: false)
```

La información de autenticación **no se almacena en el contexto ni en ningún fichero persistente**. Para poder efectuar operaciones en el entorno, es necesario configurar la variable de entorno **FIWARE_TOKEN** con el valor devuelto por el comando `fiware auth`.

### Get

El comando `fiware get` obtiene información sobre uno o varios tipos de objetos en la plataforma:

```bash
NAME:
   fiware get - Get some resource (services, devices, suscriptions, rules)

USAGE:
   fiware get [command options] [arguments...]

CATEGORY:
   platform

OPTIONS:
   --token value, -t value         authentication token (default: <empty>) [$FIWARE_TOKEN, $X_AUTH_TOKEN]
   --subservice value, --ss value  subservice name
   --output FILE, -o FILE          Write output to FILE
   --help, -h                      show help (default: false)
```

El resultado del comando se formatea usando el modelo descrito por el paquete ["github.com/warpcomdev/fiware"](../../models.go), con el objetivo de poder compararlo con otros modelos de vertical generados a partir de otra información (por ejemplo, la extraída por la aplicación [decode](../decode/README.md)).

```bash
$ fiware get groups
{
  "name": "",
  "subservice": "alumbrado",
  "services": [
    {
      "iotagent": "http://172.17.0.1:4052",
      "apikey": "sdgew ... egdgd",
      "entity_type": "device",
      "service_path": "/alumbrado",
      "service": "demoservice",
      "resource": "/iot/json",
      "description": "JSON_IoT_Agent_Node",
      "protocol": "IoTA-JSON",
      "_id": "6sfd ... 45h"
    }
  ]
}
```

El resultado también puede enviarse a un fichero, con la opción `-o FILE`:

```bash
$ fiware get -o manager.json groups devices
writing output to file manager.json
```

### Post

El comando `fiware post` envía una petición API POST para alguno de los tipos de objetos en la plataforma:

```bash
NAME:
   fiware post - Post some resource (services, devices, suscriptions, rules)

USAGE:
   fiware post [command options] [arguments...]

CATEGORY:
   platform

OPTIONS:
   --token value, -t value         authentication token (default: <empty>) [$FIWARE_TOKEN, $X_AUTH_TOKEN]
   --subservice value, --ss value  subservice name
   --data FILE, -d FILE            Read vertical data from FILE
   --help, -h
```

El comando lee los datos del fichero jsonnet que se le indique con el flag `-data` (**obligatorio**). El schema del fichero jsonnet debe seguir el modelo de datos descrito por el paquete ["github.com/warpcomdev/fiware"](../../models.go).

### Delete

El comando `fiware delete` envía una petición API DELETE para alguno de los tipos de objetos en la plataforma:

```bash
$ fiware delete -h
NAME:
   fiware delete - Delete some resource (services, devices, suscriptions, rules)

USAGE:
   fiware delete [command options] [arguments...]

CATEGORY:
   platform

OPTIONS:
   --token value, -t value         authentication token (default: <empty>) [$FIWARE_TOKEN, $X_AUTH_TOKEN]
   --subservice value, --ss value  subservice name
   --data FILE, -d FILE            Read vertical data from FILE
   --help, -h                      show help (default: false)
```

El comando lee los datos del fichero jsonnet que se le indique con el flag `-data` (**obligatorio**). El schema del fichero jsonnet debe seguir el modelo de datos descrito por el paquete ["github.com/warpcomdev/fiware"](../../models.go).
