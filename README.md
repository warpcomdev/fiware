# FIWARE toolkit

## Herramienta de línea de comandos

Este repositorio es el hogar de la aplicación [warpcom-fiware](cmd/fiware/README.md), que simplifica la operación de los recursos gestionados por un entorno de Thiking Cities basado en plataforma FIWARE. La aplicación permite:

- Mantener un inventario de entornos FIWARE (SaaS, on premise, IaaS), con diferentes servicios y usuarios.
- Descargar recursos (entidades, suscripciones, reglas de CEP, etc) de un entorno FIWARE.
- Subir recursos a un entorno FIWARE.
- Borrar recursos de un entorno FIWARE.

## Modelo de datos

La apicación utiliza las APIs públicas de la plataforma FIWARE para implementar sus funcionalidades. Cada recurso de la plataforma se modela como un objeto son, descrito por el módulo golang [github.com/warpcomdev/fiware](./models.go). Este modelo está representa reecursos como:

- Name-mappings de cygnus
- Suscripciones de Context-Broker
- `Services` y `Devices` del IoTAgent-manager
- Reglas CEP de Perseo
- Tablas de la base de datos

El modelo se ha construido con cuatro premisas:

- El modelo global debe ser serializable a json.

- El modelo debe ser compatible con la API del componente de plataforma relevante en cada caso.

  Por ejemplo, para los recursos relacionados con el IoTAgent-Manager, se modelan dos objetos separados (`Services` y `Devices`) siguiendo la lógica de la API del IoTAgent-Manager, que tiene dos endpoints separados para gestionar estos recursos (`/iot/services` e `/iot/devices`). El schema de los objetos `Services` y `Devices` coincide con el formato que esperan ambas operaciones de la API del IoTAgent-manager.

- El modelo debe ser útil tanto para leer recursos de la plataforma, como para actualizarlos.

  Los atributos de cada modelo que sólo son relevantes para lectura (por ejemplo, la fecha de último disparo de una suscripción, que puede leerse pero no escribirse) se modelan como una sub-estructura opcional dentro de la estructura que define el recurso.

- El modelo debe ser compatible con urbo-deployer.

El schema del modelo resultante se ha documentado formalmente en el fichero [serializations.cue](./serializations.cue). La especificación utiliza el lenguaje [cue](https://cuelang.org/). La versión actual del schema se ha generado automáticamente a partir del código, aunque a futuro podría ser alrevés y ser el código el que se autogenerase en función del schema.

## Cheatsheet

- Extraer las suscripciones de los ficheros de assets generados por builder:

```bash
$ cat *.json | jq -s 'add | map(.[].subscriptions) | add | { subscriptions: . }' > subs.json
```
