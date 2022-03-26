# FIWARE toolkit

Este repositorio contiene el módulo golang [github.com/warpcomdev/fiware](./models.go), que define un modelo de datos para describir los recursos relacionados con una vertical:

- Name-mappings de cygnus
- Suscripciones de Context-Broker
- `Services` y `Devices` del IoTAgent-manager
- Reglas CEP de Perseo
- Tablas de la base de datos

Este modelo se ha construido con tres premisas:

- El modelo global debe ser serializable a json.

- El modelo debe ser compatible con la API del componente de plataforma relevante en cada caso.

  Por ejemplo, para los recursos relacionados con el IoTAgent-Manager, se modelan dos objetos separados (`Services` y `Devices`) siguiendo la lógica de la API del IoTAgent-Manager, que tiene dos endpoints separados para gestionar estos recursos (`/iot/services` e `/iot/devices`). El schema de los objetos `Services` y `Devices` coincide con el formato que esperan ambas operaciones de la API del IoTAgent-manager.

- El modelo debe ser útil tanto para leer recursos de la plataforma, como para actualizarlos.

  Los atributos de cada modelo que sólo son relevantes para lectura (por ejemplo, la fecha de último disparo de una suscripción, que puede leerse pero no escribirse) se modelan como una sub-estructura opcional dentro de la estructura que define el recurso.

A partir de este modelo, se proporciona una herramienta de línea de comandos [fiware](cmd/fiware/README.md) que permite consumir el modelo y operar con él en instancias de la plataforma Fiware que implementen las APIs de Orion, Perseo, Keystone, IoTA, etc.
