# Gestión de verticales

## Decode

El comando `fiware decode` infiere el modelo de una vertical a partir o bien de su fichero `models/ngsi/README.md`, o de un `CSV` de entidades volcado del Context Broker, y genera una versión *estandar* del modelo descriptivo de la vertical con el conjunto mínimo de name-mappings, suscripciones y tablas de base de datos que la vertical requiere.

```
$ fiware decode -h
NAME:
   fiware decode - decode NGSI README.md or CSV file

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

## Template

El comando `fiware template` lee el modelo de datos de un fichero proporcionado con el flag `-d FILE`, y le aplica un template para generar un fichero de texto. Actualmente, los formatos soportados son:

- Fichero de datos:
   - [cue](https://cuelang.org/) (ficheros *.cue*)
   - [jsonnet](https://jsonnet.org/) (otros)
   - [startlark](https://github.com/bazelbuild/starlark) (ficheros *.star*, *.py*)
   - [csv]: Solo soportado para entidades descargadas del portal
- Template:
   - [golang text/template](https://pkg.go.dev/text/template).

```
$ fiware template -h
NAME:
   fiware template - template for vertical data

USAGE:
   fiware template [command options] [arguments...]

CATEGORY:
   template

OPTIONS:
   --data FILE, -d FILE    read vertical data from FILE
   --lib DIR, -l DIR       load data modules / libs from DIR
   --output FILE, -o FILE  write template output to FILE
   --help, -h              show help (default: false)
```

La aplicación tiene varias plantillas predefinidas que pueden servir de punto de partida rápido para generar la documentación de un vertical:

- `dump.tmpl`: Convierte la entrada `jsonnet` en una salida `json` estándar.
- `default_cygnus.tmpl`: Genera un fichero de name_mappings para cygnus.
- `default_subs.tmpl`: Genera el típico fichero de suscripciones de la vertical.
- `default_readme.tmpl`: Genera el típico fichero `README.md` del modelo de datos la vertical.
- `default_ddls.tmpl`: Genera el típico fichero SQL de la vertical, con un conjunto típico de tablas y vistas.
- `default_csv.tmpl`: Genera el típico CSV de entidades.
- `default_lastdata.tmpl`: Genera un script SQL para hacer una carga inicial de las tablas lastdata.

Las plantillas predefinidas también publican algunos bloques reutilizables, en particular:

- `default_ddls_sets`: Genera los comandos `\set` del fichero SQL.
- `default_ddls_tables`: Genera los comandos SQL `create table` (y las vistas lastdata) del fichero SQL.
- `dump`: Vuelca todos los parámetros del template en formato json.

Por último, se conservan algunas plantillas *legacy* adaptadas a los formatos antiguos de tablas y suscripciones:

- `legacy_ddls.tmpl`: Genera el típico fichero de DDLs pero usando vistas para lastdata, en vez de tablas
- `legacy_readme.tmpl`: El típico README de las DDLs, pero usando vistas lastdata en vez de tablas
- `legacy_subs.tmpl`: El fichero de suscripciones sin la suscripción a lastdata
 
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

*NOTA sobre starlark*

En el caso de los ficheros *starlark*, se selecciona únicamente la variable global cuyo nombre coincida con el del fichero (sin la extensión), y esa variable es la que se utiliza como contexto al ejecutar el template.

Por ejemplo, si se especifica un fichero de datos *ejemplo.star* con el siguiente contenido:

```python
valores = [1, 2, 3]
ejemplo = {
   'dato1': "a",
   'valor': valores
}
```

El contexto con el que se ejecute el template contendrá los valores `{ "dato1": "a", "valor": [1, 2, 3] }`.

## Export

El comando `fiware export` lee el modelo de una vertical y lo vuelve a escribir, localizando los literales de texto que coincidan con alguno de los parámetros del contexto (ver [parámetros de contexto](#parámetros-de-contexto) y reemplazando esos literales por el parámetro de contexto correspondiente, para poder transportar de manera sencilla recursos de un contexto a otro.

En particular está pensado para transportar suscripciones, que pueden hacer referencia a diferentes URLs en diferentes contextos. Pero no se limita a suscripciones, sino que sustituye valores literales por parámetros en cualquiera d elos objetos soportados.

También puede usarse para transcodificar una configuración entre diferentes formatos (de jsonnet a cue, por ejemplo). El formato de entrada y de salida se determina a partir de la extensión de los ficheros correspondientes.

```
$ fiware export -h
NAME:
   fiware export - Read datafile and export with context params

USAGE:
   fiware export [command options] [arguments...]

CATEGORY:
   template

OPTIONS:
   --data FILE, -d FILE    read vertical data from FILE
   --lib DIR, -l DIR       load data modules / libs from DIR
   --output FILE, -o FILE  write output to FILE
   --help, -h              show help (default: false)
```

## Parámetros de contexto

Además de los atributos fijos que tiene cada contexto para poder conectar a los diferentes servidores del entorno, un contexto puede tener también una lista de *parámetros*.

Estos parámetros no los utiliza directamente la aplicación, sino que están pensados para que se usen desde los ficheros de datos o los templates.

Los parámetros se configuran con la orden `fiware context params ...`:

```
$ fiware context params
NAME:
   fiware context params - Set a template parameter

USAGE:
   fiware context params [command options] [arguments...]

OPTIONS:
   --help, -h  show help (default: false)
```

Para eliminar un parámetro, se debe establecer con el valor "".

Tanto los ficheros de datos como las plantillas pueden acceder a los párametros que se hayan definido en el contexto:

- Ficheros de datos *cue*: Los parámetros de contexto son accesibles a través de la variable `params: {[string]: string}`.

- Ficheros de datos *jsonnet*: los parámetros de contexto son accesibles mediante [std.extVar](https://jsonnet.org/ref/stdlib.html#std.extVar(x)).

- Ficheros de datos *starlark*: Si el fichero contiene una variable global con el mismo nombre que el fichero (sin extensión), y es `Callable`, la función es invocada con un diccionario que contiene todos los parámetros.

- Plantillas *golang text/template*: Los parámetros están accesibles en la variable global `{{ .params }}`.

Así, para compartir con el template un atributo como por ejemplo la URL del servidor cygnus, se puede añadir al contexto con el comando:

```fiware context params cygnus_url http://cygnus.fiware.com:8080```

Y ese valor será accesible desde *jsonnet*, *starlark* y las plantillas.
