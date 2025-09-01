# Instalación

La aplicación fiware es un ejecutable autocontenido, no tiene dependencias y está compilado tanto para linux como para windows.

- La release más reciente siempre estará en github, en la [página de releases](https://github.com/warpcomdev/fiware/releases).

- Cada release tiene al menos dos ficheros comprimidos. Se debe descargar el correspondiente al sistema operativo que se vaya a usar:
  - La versión para Windows, `fiware_X.Y.Z_Windows_x86_64.zip`,
  - O la versión para Linux, `fiware_X.Y.Z_Linux_x86_64.gz`

- Dentro del fichero comprimido se encuentra un ejecutable (`fiware` en el caso linux y `fiware.exe` en el caso windows). Es un ejecutable de **línea de comandos**, no basta hacer doble click en él; hay que abrir la línea de comandos y escribir `fiware`
  - Si el ejecutable se ha copiado alguna ruta dentro del path del sistema (p.e. `/usr/local/bin/` en linux, `C:\Windows\system32\` en windows), podrá lanzarse simplemente con la orden `fiware`
  - En otro caso, será necesario ejecutarlo indicando la ruta completa al ejecutable.
