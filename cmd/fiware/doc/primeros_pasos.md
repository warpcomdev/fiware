# Primeros pasos con la aplicación

Asumimos que ya se ha instalado la aplicación utilizando el procedimiento de [instalación](./instalacion.md)

Los primeros pasos a seguir con la aplicación son:

- Configurar el entorno fiware con el que vas a trabajar, creando [contextos](#contextos)
- [Autenticarte](#autenticación) en el entorno
- Hacer [consultas](#consultas) al entorno.

## Contextos

Para poder empezar a utilizar la aplicación, lo primero es definir un **contexto**. Los contextos definen:

- Los parámetros de conexión a la plataforma (URLs de keystone, orion, etc)
- El servicio y subservicio que gestiona dicho contexto
- Variables y parámetros adicionales útiles para la operación, como las URLs de cygnus.

Los contextos se almacenan en el directorio `~/.config/fiware.d` del usuario. Inicialmente, la lista de contextos estará vacía:

```
$ fiware context list

```

### Crear el primer contexto

Crearemos un contexto nuevo con `fiware context create <nombre del contexto>`

```
$ fiware context create lab_alcobendas
Using context lab_alcobendas
```

El contexto inicialmente está vacío, y es necesario rellenarlo con todos sus parámetros de conexión. Los parámetros que utiliza el contexto pueden verse con `fiware context info`:

```
# La aplicación reconoce "ctx" como abreviatura de "context"
$ fiware ctx info
{
  "biConnection": "",
  "customer": "",
  "database": "",
  "iotam": "",
  "jenkins": "",
  "jenkinsFolder": "",
  "jenkinsLabel": "",
  "keystone": "",
  "name": "lab_alcobendas",
  "orch": "",
  "orion": "",
  "pentaho": "",
  "perseo": "",
  "postgis": "",
  "schema": "",
  "service": "",
  "subservice": "",
  "type": "",
  "urbo": "",
  "username": ""
}
> fiware context set biConnection "" customer "" database "" iotam "" jenkins "" jenkinsFolder "" jenkinsLabel "" keystone "" name "lab_alcobendas" orch "" orion "" pentaho "" perseo "" postgis "" schema "" service "" subservice "" type "" urbo "" username ""
```

Es necesario configurar el contexto para que sea útil. Los parámetros no son obligatorios, sólo hay que rellenar los que hagan falta para la operación que queramos hacer con él. Por ejemplo,

- Si queremos ver y editar suscripciones, necesitaremos:

  - `keystone`: la URL del servidor keystone
  - `orion`: la URL del servidor orion
  - `service`: el nombre del servicio de plataforma.
  - `subservice`: el nombre del subservicio
  - `username`: el nombre de un usuario con permisos para operar con ese servicio y subservicio.

Podemos configurar todos estos parámetros con un solo comando:

```
$ fiware ctx set keystone "http://keystone.url.com:5000" orion "http://orion.url.com:2026" service testservice subservice /riego username lab_admin
using context lab_alcobendas
updated fields: {
  keystone: http://keystone.url.com:5000
  orion: http://orion.url.com:2026
  service: testservice
  subservice: /riego
  username: lab_admin
}
```

- Si además queremos trabajar con reglas de CEP, necesitaremos añadir:

  - perseo: url del pep perseo-fe

```
$ fiware ctx set perseo "http://pep-perseo-fe.url.com:9090"
using context lab_alcobendas
updated fields: {
  perseo: http://pep-perseo-fe.url.com:9090
}
```

- Si queremos también trabaajr con grupos y dispositivos del IoT agent, necesitaremos la URL del IoTA Manager:

  - iotam: url del iota-manager

```
$ fiware ctx set iotam "http://iotam.url.com:8082"
using context lab_alcobendas
updated fields: {
  iotam: http://iotam.url.com:8082
}
```

- Para trabajar con los dashboards de urbo. necesiaremos la URL del servidor:

  - urbo: url de urbo

```
$ fiware ctx set urbo "http://urbo.url.com:8082"
using context lab_alcobendas
updated fields: {
  urbo: http://urbo.url.com:8082
}
```

Podemos ver el estado en que ha quedado nuestro contexto con `fiware ctx info`:

```
$ fiware ctx info
{
  "biConnection": "testservice",
  "customer": "lab_alcobendas",
  "database": "urbo2",
  "iotam": "http://iotam.url.com:8082",
  "jenkins": "",
  "jenkinsFolder": "lab_alcobendas",
  "jenkinsLabel": "lab_alcobendas",
  "keystone": "http://keystone.url.com:5000",
  "name": "lab_alcobendas",
  "orch": "",
  "orion": "http://orion.url.com:2026",
  "pentaho": "",
  "perseo": "",
  "postgis": "",
  "schema": "testservice",
  "service": "testservice",
  "subservice": "/riego",
  "type": "DEV",
  "urbo": "http://urbo.url.com:8082",
  "username": "lab_admin"
}
> fiware context set biConnection "testservice" customer "lab_alcobendas" database "urbo2" iotam "http://iotam.url.com:8082" jenkins "" jenkinsFolder "lab_alcobendas" jenkinsLabel "lab_alcobendas" keystone "http://keystone.url.com:5000" name "lab_alcobendas" orch "" orion "http://orion.url.com:2026" pentaho "" perseo "" postgis "" schema "testservice" service "testservice" subservice "/riego" type "DEV" urbo "http://urbo.url.com:8082" username "lab_admin"
```

### Copiar un contexto

Para simplificar la creación de un contexto nuevo, es posible hacer una copia de uno existente, con `fiware context dup <nombre del nuevo contexto>`

```
# En primer lugar, debemos seleccionar el contexto que queremos copiar
$ fiware ctx use lab_alcobendas
using context lab_alcobendas

# A continuación, lo copiamos
fiware ctx dup pre_alcobendas
Using context pre_alcobendas
```

Podemos ver los contextos que tenemos creados con `fiware context list`. El contexto activo aparecerá marcado con un asterisco:

```
# La aplicación renoce ls como abreviatura de list
$ fiware ctx ls
lab_alcobendas
* pre_alcobendas
```

Una vez clonado, podemos modificar los parámetros que sean distintos en este entorno, por ejemplo el nombre de usuario y de servicio:

```
$ fiware ctx set service alcobendas_pre username pre_admin_alcobendas
```

### Cambiar de contexto

Para seleccionar y activar un contexto, basta con utilizar la orden `fiware ctx use <nombre del contexto>`:

```
$ fiware ctx use lab_alcobendas
using context lab_alcobendas
```

La lista completa de contextos puede verse con `fiware context ls`. El entorno activo aparecerá marcado con un `*`:

```
$ fiware ctx ls
* lab_alcobendas
pre_alcobendas
```

La información sobre el entorno activo puede verse con `fiware context info`:

```
$ fiware ctx info
{
  "biConnection": "testservice",
  "customer": "lab_alcobendas",
  "database": "urbo2",
  "iotam": "http://iotam.url.com:8082",
  "jenkins": "",
  "jenkinsFolder": "lab_alcobendas",
  "jenkinsLabel": "lab_alcobendas",
  "keystone": "http://keystone.url.com:5000",
  "name": "lab_alcobendas",
  "orch": "",
  "orion": "http://orion.url.com:2026",
  "pentaho": "",
  "perseo": "",
  "postgis": "",
  "schema": "testservice",
  "service": "testservice",
  "subservice": "/riego",
  "type": "DEV",
  "urbo": "http://urbo.url.com:8082",
  "username": "lab_admin"
}
> fiware context set biConnection "testservice" customer "lab_alcobendas" database "urbo2" iotam "http://iotam.url.com:8082" jenkins "" jenkinsFolder "lab_alcobendas" jenkinsLabel "lab_alcobendas" keystone "http://keystone.url.com:5000" name "lab_alcobendas" orch "" orion "http://orion.url.com:2026" pentaho "" perseo "" postgis "" schema "testservice" service "testservice" subservice "/riego" type "DEV" urbo "http://urbo.url.com:8082" username "lab_admin"
```

## Autenticación

Una vez tenemos seleccionado un contexto, nos podemos autenticar en él. El inicio de sesión se hace con la orden `fiware auth`:

```
$ fiware auth
Environment: http://keystone.url.com:5001
Username@Service: lab_alcobendas@alcobendas
Password: 
export FIWARE_TOKEN=gAAAAA...g5A8
export URBO_TOKEN=eyJh...bRPsA
```

(la salida completa del comando se ha omitido por brevedad). La aplicación pedirá el nombre de usuario para el servicio configurado en el contexto activo. Si el contexto tiene configuradas tanto la URL de keystone como la de Urbo, la aplicación obtiene tokens para ambas; si por el contrario solo estuviera definida en el contexto la URL de uno de los dos sistemas, solo se obtendría el token para ese sistema.

Por defecto, la aplicación muestra por pantalla los tokens obtenidos, pero no los almacena en ningún sitio. Es el usuario el que debe copiar y pegar los tokens como variables de entorno, para usarlos con cualquier comando que necesite autenticación:

```
$ export FIWARE_TOKEN=gAAAAA...g5A8
$ export URBO_TOKEN=eyJh...bRPsA
```

Por conveniencia, si los tokens sólo se van a utilizar desde la aplicación, se puede especificar el parámetro `--save` al autenticar, y la aplicación cacheará los tokens para uso propio, en lugar de mostrarlos:

```
$ fiware auth --save
Environment: http://keystone.url.com:5001
Username@Service: lab_admin@alcobendas
Password: 
tokens for context lab_alcobendas cached
```

Los tokens tienen la duración típica que les pone la plataforma, aproximadamente una hora.

## Consultas

Una vez conectados a la plataforma, con los tokens en caché, ya podemos hacer consultas al entorno. Las consultas se realizan con la orden `fiware get <recurso>`. Las siguientes secciones tienen algunos ejemplos.

### ¿Qué suscripciones hay en el servicio "/riego"?

Para poder hacer esta consulta, el contexto necesita tener al menos los siguientes datos:

- keystone: URL de keystone
- orion: URL de orionç
- service: nombre de servicio
- username: nombre de usuario

También debe tener cacheado un token reciente, generado con `fiware auth --save`

```
# Nos aseguramos de que el contexto apunta al subservicio de riego
$ fiware ctx set subservice /riego
using context lab_alcobendas
  subservice: /riego

# Y pedimos las suscripciones
$ fiware get subscriptions
{
  "subservice": "alumbrado",
  "environment": {
    "notificationEndpoints": {
      "HISTORIC": "http://iot-cygnus:5057/notify",
      "LASTDATA": "http://iot-cygnus:5082/notify",
      "RULES": "http://iot-perseo-fe:9090/notices",
      "http:iot-cygnus:5055:notify": "http://iot-cygnus:5055/notify",
      "http:iot-cygnus:5056:notify": "http://iot-cygnus:5056/notify"
    }
  },
  "subscriptions": {
    "Envio a CEP para comandado de Cuadro cmdIlluminanceLevel": {
      "description": "Envio a CEP para comandado de Cuadro cmdIlluminanceLevel",
      "status": "active",
      "notification": {
        "attrs": [
          "cmdIlluminanceLevel"
        ],
        "attrsFormat": "normalized",
        "http": {
          "url": "RULES"
        },
        "lastNotification": "2022-11-15T09:52:15.000Z",
        "lastSuccess": "2022-11-15T09:52:15.000Z",
        "lastSuccessCode": 200,
        "timesSent": 126
      },
      "subject": {
        "condition": {
          "attrs": [
            "cmdIlluminanceLevel"
          ]
        },
        "entities": [
          {
            "idPattern": ".*",
            "type": "StreetlightControlCabinet"
          }
        ]
      },
      "id": "..."
    },
    ... etc. omitido por brevedad ...
  }
}
```

Una cosa muy interesante que hace la aplicación es que, además de volcarnos todas las suscripciones, **nos agrupa todas las URLs de notificación** en un campo `notificationEndpoints` del json generado. Así podemos de un vistazo saber qué URLs estamos usando, o modificarlas para replicar estas reglas en otro entorno con otras URLs.

En vez de volcar por pantalla el resultado, se le puede pedir a la aplicación que lo guarde en un fichero, con la opción `-o`:

```
# La aplicación reconoce "subs" como abreviatura de "subscriptions"
# también reconoce el parámetro `--subservice`, `--ss`
$ fiware get --ss /riego -o subs_lab_alcobendas.json subs
writing output to file subs_lab_alcobendas.json
```

### ¿Qué reglas de CEP hay en el servicio "/alumbrado"?

Para poder hacer esta consulta, el contexto necesita tener al menos los siguientes datos:

- keystone: URL de keystone
- orion: URL de orion
- perseo: URL de perseo-fe o pep-perseo-fe
- service: nombre de servicio
- username: nombre de usuario

También debe tener cacheado un token reciente, generado con `fiware auth --save`

```
$ fiware get -ss /alumbrado -o rules_alumbrado.json rules
writing output to file rules_alumbrado.json

$ cat rules_alumbrado.sjon
...
```

### ¿Qué registros de dispositivos hay en el subservicio de medioambiente?

Para poder hacer esta consulta, el contexto necesita tener al menos los siguientes datos:

- keystone: URL de keystone
- orion: URL de orion
- iotam: URL del iota manager
- service: nombre de servicio
- username: nombre de usuario

También debe tener cacheado un token reciente, generado con `fiware auth --save`

```
$ fiware get -ss /medioambiente -o registros_medioambiente.json registrations
writing output to file registros_medioambiente.json

$ cat registros_medioambiente.json
...
```

### ¿Qué entidades de tipo `Zone` hay en la vertical de /aforo?

Para poder hacer esta consulta, el contexto necesita tener al menos los siguientes datos:

- keystone: URL de keystone
- orion: URL de orion
- service: nombre de servicio
- username: nombre de usuario

También debe tener cacheado un token reciente, generado con `fiware auth --save`

```
$ fiware get -ss /aforo -o entidades_zona.json --filter-type Zone entities
writing output to file rules_alumbrado.json

$ cat registros_medioambiente.json
...
```

### ¿Qué entidades hay en el subservicio tráfico con ID que empiece por `Cam`?

Para poder hacer esta consulta, el contexto necesita tener al menos los siguientes datos:

- keystone: URL de keystone
- orion: URL de orion
- service: nombre de servicio
- username: nombre de usuario

También debe tener cacheado un token reciente, generado con `fiware auth --save`

```
$ fiware ctx set subservice /trafico
$ fiware get -o camaras_trafico.json --filter-id "Cam*" entities
writing output to file camaras_trafico.json

$ cat camaras_trafico.json
...
```
