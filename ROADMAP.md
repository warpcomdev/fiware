# Roadmap automatización de verticales

# Fase 1. Modelado de recursos

## Modelado formal

Recursos que pueden modelarse en su totalidad:

- Suscripciones: Trabajo previo en https://github.com/telefonicasc/streetlight-vertical/pulls
- Reglas de CEP.
- Name-mappings de cygnus.
- Device groups (servicios) del IoTA manager.
- Devices del IoTA-Agent.

Se debe tener en cuenta tanto el modelado del recurso como el del **estado** del recurso, que puede ser útil para obtener información de la situación actual del recurso en la plataforma y actuar en consecuencia.

Por ejemplo, las suscripciones tienen un **id** que no forma parte del modelo de la suscripción (no es algo que se configure cuando defines la suscripción), pero sí de su estado, y es imprescindible para por ejemplo borrar una suscripción de una versión anterior, antes de desplegar la versión nueva.

## Modelado operativo

Recursos que no se puedan modelar porque tengan demasiados grados de libertad:

- Base de datos.
- Dashboards.
- ETLs.

En estos casos podría modelarse sólo el aspecto **operativo**: los parámetros que controlan cómo se despliegan en un entorno. Por ejemplo, las variables que se utilizan al instalar el schema de postgres dado en un fichero `ddls.sql`, la definición de qué paneles son primarios y qué paneles son secundarios en una vertical, los ficheros de configuración que necesita una ETL, su `requirements.txt`, la frecuencia con la que se debe lanzar, etc.

# Fase 2. Modelo de entorno

## Modelado de plataforma

Modelado de las características de un entorno smart cities donde se va a desplegar una vertical. El entorno se describiría mediante un conjunto de URLs:

- URLs de datasource: orion, CEP, pentaho, postgres (para los paneles)
- URLs de notificación: histórico, lastdata, perseo, ckan (para las suscripciones)
- Repositorios GIT de configuración de name-mappings, dashboards.

## Modelado de funcionalidades opcionales

Diferentes entornos pueden utilizar partes distintas de la vertical. Por ejemplo, un cliente con la vertical de Parkings que sólo tenga parkings de superficie y no subterráneos, no necesitará varias de las suscripciones ni de las reglas CEP de la vertical.

Sería conveniente tener un mecanismo para que el cliente pudiera especificar qué conjunto de funcionalidades de la vertical requiere en su caso. Estos parámetros personalizables deberían:

- Permitir activar o desactivar recursos particulares.
- Capturar las dependencias entre recursos. Por ejemplo, que no se pueda desactivar la suscripción al CEP para un determinado EntityType si no se desactivan también las reglas CEP que se disparan con esa suscripción.

Ojo porque esto en general no podrá resolverse desplegando simplemente todas las funciones que tenga la vertical. Algunas necesariamente dependen del cliente. Por ejemplo, un cliente que tenga la vertical de medioambiente con Inmótica usará el context adapter y no necesitará crear device groups ni devices para implementar los comandos, pero otro que lo tenga con datakorum o hopu usará el iot-agent y necesitará esos devices y groups.

## Seguridad

Modelado de las URLs de acceso exterior a las APIs: keystone, orion, perseo, iota manager (para las herramientas de operación), y en particular de la gestión de credenciales: cómo se vana  gestionar las credenciales necesarias para desplegar los objetos, incluyendo todo lo que entre en el alcance: APIs, pero también bases de datos, ETLs, etc.

# Fase 3 - distribución de modelos

Definir cómo se van a hacer accesibles los distintos tipos de modelos a los clientes / consumidores.

- Se espera que el resultado sea una estructura de carpetas en los repositorios git de las verticales y el proyecto.
- En esta fase se debe determinar cómo controlar el acceso desde las herramientas de operación a los repos privados de vertical y proyecto, y como unificar los modelos de vertical con los de funcionalidades opcionales y servicios.

# Fase 4. Viabilidad automatización

En función de todo lo anterior se definirá lo que se considera viable incluir en una herramienta de despliegue, y lo que se piense que va a requerir intervención manual. A priori se considera que:

- El despliegue de todos los recursos que utilizan APIs de plataforma (suscripciones, reglas CEP, iotas, etc) será automatizable.
- El despliegue de recursos que utilizan gitops (name mappings de CEP, paneles?) será automatizable.
- La instalación inicial de un schema de base de datos será automatizable.
- La instalación inicial de una ETL puede ser automatizable, imponiendo restricciones en el tipo de ETL y sus parámetros (trabajo previo en  https://github.com/telefonicasc/tech-transfer/pull/13)
- La actualización de un schema de base de datos requerirá intervención manual. La aplicación puede ayudar permitiendo importar un fichero SQL generado por un integrador.
- La actualización de una ETL no será en general automatizable, aunque puede depender de cómo evolucione el framework de ETLs de tech-transfer.

Estos puntos deberán acordarse con el resto de actores (equipos de desarrollo y operaciones)

# Fase 5. Vertical de despliegues

El objetivo final es que las herramientas de instalación se gestionen desde una **vertical de despliegues**, que sea como un servicio más de plataforma. Para esto el desarrollo se orientará a implementar un **bucle de operación** Urbo -> CEP -> Jenkins -> Orion -> Urbo:

- El estado de instalación de cada vertical será modelado por entidades del vertical de despliegue, que reflejarán al menos:

  - La versión de la vertical desplegada o a desplegar.
  - La selección de recursos opcionales.
  - La selección de recursos parametrizables.

- Urbo operará los despliegues creando o editando esas entidades.
- Los cambios en las entidades dispararán reglas de CEP que básicamente invocarán jobs en jenkins (trabajo previo en https://github.com/telefonicasc/tech-transfer/blob/master/topics/trigger_jenkins_jobs_by_api.md)
- Los jobs reaccionarán a los cambios de estado en las entidades operando la plataforma, y actualizarán la información en el Context Broker.
- Si el cliente no tiene contratada la funcionalidad de despliegue de verticales, se restringiría su acceso al subservicio.

Este bucle de operación debe implementarse para todos los recursos involucrados en el despliegue de una vertical:

1. Suscripciones: Trabajo previo en https://github.com/telefonicasc/streetlight-vertical/pulls
2. Device groups (servicios) del IoTA manager.
3. Devices del IoTA-Agent.
4. Reglas de CEP.
5. Name-mappings de cygnus.
6. Dashboards.
7. Base de datos.
8. ETLs.

El bucle de operación es el producto mínimo viable de la vertical de despliegues. A partir de este desarrollo, existen diferentes vías alternativas de avance:

1. Personalización: Añadir la capacidad de activar o desactivar componentes de la vertical en función de las características del proyecto.

    - Device o device groups opcionales on / off (por ejemplo: desactivar si se usa el context adapter de inmótica)
    - Reglas de CEP opcionales on / off (por ejemplo: desactivar reglas de parking onstreet si un cliente no tiene plazas sensorizadas individualmente).
    - Suscripciones opcionales on / off (idem).
    - Paneles opcionales on / off (Ejemplo: desactivar paneles de medioambiente - NoiseLevelObserved si las estaciones meteorológicas del cliente no reportan el nivel de ruido).
    - ETLs opcionales on / off (Por ejemplo: desactivar ETL de instagram si el cliente no tiene cuenta en esa red social)

2. Parametrización: Añadir la capacidad de parametrizar ciertos recursos de la vertical en función de las características del proyecto:

    - Device o device groups parametrizables (por ejemplo: URL del endpoint para comandado)
    - Reglas CEP parametrizables (ejemplo: dirección de correo a la que enviar alertas, franjas horarias en las que alertar de la ocupación de plazas de parking, etc)
    - ETLs parametrizables (ej: nombrer de usuario de twitter)

3. Actualizaciones: Permitir no sólo instalar o desinstalar una versión de la vertical, sino actualizar sus recursos.

La lista anterior está ordenada de menor a mayor complejidad a priori, pero no tiene por qué ser el orden en que se acometan las automatizaciones. En cualquier caso en 4 meses no creemos que sea viable realizarlo todo, por lo que habrá que empezar por el bucle de operación e ir priorizando a partir de ahí.
