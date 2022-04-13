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

Recursos que no pueden modelarse porque tienen demasiados grados de libertad:

- Base de datos.
- Dashboards.
- ETLs.

En este caso lo que debe modelarse es el aspecto **operativo**: los parámetros que controlan cómo se despliegan en un entorno. Por ejemplo, los ficheros de configuración que necesita una ETL, su `requirements.txt`, la frecuencia con la que se debe lanzar, etc.

# Fase 2. Modelo de entorno

## Modelado de servicios

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

## Modelado operativo

Básicamente consiste en las URLs de acceso exterior a las APIs: keystone, orion, perseo, iota manager (para las herramientas de operación).

La consideración principal de este apartado es la gestión de credenciales: cómo va a obtener el operador o Urbo las credenciales necesarias para desplegar los objetos necesarios, incluyendo todo lo que entre en el alcance: APIs, pero también bases de datos, ETLs, etc.

# Fase 3 - distribución de modelos

Definir cómo se van a hacer accesibles los distintos tipos de modelos a los clientes / consumidores.

- Se espera que el resultado sea una estructura de carpetas en los repositorios git de las verticales y el proyecto. 
- En esta fase se debe determinar cómo controlar el acceso desde las herramientas de operación a los repos privados de vertical y proyecto, y como unificar los modelos de vertical con los de funcionalidades opcionales y servicios.

# Fase 4. Viabilidad automatización

En función de todo lo anterior se definirá lo que se considera viable incluir en una herramienta de despliegue, y lo que se piense que va a requerir intervención manual. Por ejemplo, se considera que:

- El despliegue de todos los recursos que utilizan APIs de plataforma (suscripciones, reglas CEP, iotas, etc) será automatizable.
- El despliegue de recursos que utilizan gitops (name mappings de CEP, paneles?) será automatizable.
- La instalación inicial de un schema de base de datos será automatizable.
- La instalación inicial de una ETL puede ser automatizable, imponiendo restricciones en el tipo de ETL y sus parámetros (trabajo previo en  https://github.com/telefonicasc/tech-transfer/pull/13)
- La actualización de un schema de base de datos requerirá intervención manual. La aplicación puede ayudar permitiendo importar un fichero SQL generado por un integrador.
- La actualización de una ETL no se ha analizado.

# Fase 5. Desarrollo

El objetivo final es que las herramientas de instalación tengan:

- Una API REST que pueda ser utilizada desde urbo
- Una línea de comandos que pueda ser invocada desde herrameintas de operación (como jenkins)

Eventualmente la API REST de la herramienta de gestión pueden ser simplemente la API de Jenkins, al que Urbo llamaría para lanzar trabajos programados que invoquen a la línea de comandos. Así que la API REST se deja fuera de la fase 4. Sí que es necesario que todas las herramientas usen parámetros y formatos equivalentes para simplificar luego ponerle la API por encima.

La PoC de instalación utilizará una vertical y un proyecto de prueba para implementar todo el modelado definido y desarrollar las herramientas de despliegue de cada componente de la vertical:

1. Suscripciones: Trabajo previo en https://github.com/telefonicasc/streetlight-vertical/pulls
2. Device groups (servicios) del IoTA manager.
3. Devices del IoTA-Agent.
4. Reglas de CEP.
5. Name-mappings de cygnus.
6. Dashboards.
7. Personalización

    - Device o device groups opcionales on / off.
    - Reglas de CEP opcionales on / off.
    - Suscripciones opcionales on / off.
    - ETLs opcionales on / off

8. Base de datos.
9. ETLs.
10. Parametrización: 

    - Device o device groups parametrizables (por ejemplo: URL del endpoint para comandado)
    - Reglas CEP parametrizables (ejemplo: dirección de correo a la que enviar alertas, franjas horarias en las que alertar de la ocupación de plazas de parking, etc)
    - ETLs parametrizables (ej: nombrer de usuario de twitter)

La lista anterior está ordenada de menor a mayor complejidad (a priori).

# Fase 6. Explotación desde URBO

Para la explotación desde Urbo habría que resolver varias cosas:

- API REST de la herramienta: inicialmente se podría utilizar la API de Jenkins para lanzar tareas programadas, pero es una API asíncrona, Urbo tendría que lanzar el trabajo y comprobar periódicamente si ha finalizado o no, y si lo ha hecho con éxito. ¿Podría requerir el desarrollo de un nuevo widget?

Se podría considerar guardar información del estado de una instalación en entidades de plataforma (entidad tipo "vertical", por ejemplo), y que la interacción entre Urbo y Jenkins sea a través de esas entidades (Urbo lee y escribe estado de las entidades, CEP convierte los cambios de estado en llamadas a Jenkins). Se debe evaluar la complejidad de la sinteracciones.

- Control del consumo de la funcionalidad: Si se regula el estado del instalador a través de un subservicio de plataforma, se podría controlar el acceso de los clientes a la funcionalidad simplemente controlando el acceso al subservicio.

  - Clientes con contrato de soporte activo: tienen acceso al subservicio /verticales, donde están las entidades que controlan la interacción entr eUrbo y jenkins.
  - Clientes sin contrato de soporte: se les restirnge el acceso al subservicio, o se borra. Se deberían borrar también los jobs de jenkins y credenciales asociadas.

En función de lo anterior, habrá que definir:

1. Modelo de gobernanza: cómo se relaciona Urbo con el deployer. Esta propuesta se orienta hacia una "vertical" que utilice el CEP para comandar a Jenkins, aunque hay que valorar otras alternativas.
2. Reglas de CEP y comandado: implementar una PoC de las reglas de CEP para generar los jobs en jenkins, y del feedback que el proceso de instalación tiene que proporcionar a la plataforma para que Urbo lo muestre.
3. Widgets especializados? en función de las necesidades que se planteen durante la PoC.
4. Paneles especializados.

# Alcance estimado

Acotando el proyecto a 4 meses, creemos que se puede esperar llegar hasta:

- Fase 5: Punto 8 o 9.
- Fase 6: Punto 2, con algunos paneles básicos del punto 4.
