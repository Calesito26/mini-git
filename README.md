# Mini-Git en Go

Mini-Git es una implementacion academica de un sistema de control de versiones inspirado en Git. El objetivo es mostrar, de forma simple, como se puede guardar historial de archivos usando hashes, commits y una carpeta interna del repositorio.

## Requisitos

- Windows, Linux o macOS.
- Go instalado.
- Terminal PowerShell, CMD, Bash o similar.

Para verificar que Go esta instalado:

```powershell
go version
```

Si el comando no existe, instala Go desde:

```text
https://go.dev/dl/
```

En Windows tambien se puede instalar con:

```powershell
winget install --id GoLang.Go --exact
```

Despues de instalar Go, abre una terminal nueva para que el PATH se actualice.

## Preparar el proyecto

Clona el repositorio o entra a la carpeta del proyecto:

```powershell
cd "D:\UNIVERSIDAD\CICLO 9\Taller de lenguajes de programacion\RETO2\mini-git"
```

Verifica que el proyecto compile:

```powershell
go test ./...
```

## Instalacion recomendada en Windows

Para no escribir rutas largas ni depender de `.\minigit.exe`, instala el comando en tu usuario:

```powershell
.\scripts\install-windows.ps1
```

Luego abre una terminal nueva y verifica:

```powershell
minigit help
```

Desde ese momento puedes usar `minigit` en cualquier carpeta.

## Compilar manualmente

Si no quieres instalarlo, puedes compilar el ejecutable dentro del proyecto:

```powershell
go build -o minigit.exe ./cmd/minigit
```

En Linux o macOS:

```bash
go build -o minigit ./cmd/minigit
```

## Como ejecutarlo

Hay tres formas.

Instalado como comando global:

```powershell
minigit help
```

Con Go directamente:

```powershell
go run ./cmd/minigit help
```

Con el ejecutable compilado:

```powershell
.\minigit.exe help
```

Recomendacion: para trabajar mas simple, usa la instalacion global y ejecuta `minigit` desde la carpeta del proyecto que quieras versionar.

## Donde se crea el repositorio

Mini-Git siempre trabaja sobre la carpeta actual de la terminal.

Ejemplo:

```powershell
cd "D:\mis-proyectos\proyecto-demo"
minigit init
```

Eso crea:

```text
D:\mis-proyectos\proyecto-demo\.minigit
```

No importa donde este instalado el programa. Lo importante es la carpeta donde estas parado cuando ejecutas el comando.

## Demo rapida

Inicializa Mini-Git en la carpeta actual:

```powershell
minigit init
```

Crea un archivo de prueba:

```powershell
Set-Content archivo.txt "Hola mundo"
```

Si el archivo no existe, `minigit add archivo.txt` mostrara error. Primero debes crear el archivo o elegir uno que ya exista en tu proyecto.

Agrega el archivo al index:

```powershell
minigit add archivo.txt
```

Para agregar todos los archivos de la carpeta actual:

```powershell
minigit add .
```

Crea un commit:

```powershell
minigit commit -m "primer commit"
```

Consulta el estado:

```powershell
minigit status
```

Consulta el historial:

```powershell
minigit log
```

Modifica el archivo:

```powershell
Set-Content archivo.txt "Hola mundo modificado"
```

Revisa los cambios:

```powershell
minigit status
```

Mira las diferencias antes de volver a agregar el archivo:

```powershell
minigit diff archivo.txt
```

Lista los archivos agregados al index junto con su hash:

```powershell
minigit ls-files
```

Muestra el contenido guardado en un objeto usando un hash de `ls-files`:

```powershell
minigit cat-file <hash>
```

Guarda una nueva version:

```powershell
minigit add archivo.txt
minigit commit -m "actualice archivo"
```

Restaura una version anterior usando el ID que aparece en `log`:

```powershell
minigit checkout <commit-id>
```

## Comandos disponibles

```text
minigit init
minigit add <archivo|carpeta> [...]
minigit commit -m "mensaje"
minigit status
minigit diff [archivo]
minigit ls-files
minigit cat-file <hash>
minigit log
minigit checkout <commit-id>
```

## Arquitectura

```text
mini-git/
|-- cmd/
|   `-- minigit/
|       `-- main.go
|-- scripts/
|   `-- install-windows.ps1
|-- go.mod
|-- README.md
|-- .minigitignore
`-- .gitignore
```

El archivo principal es:

```text
cmd/minigit/main.go
```

Alli se encuentran los comandos, la lectura y escritura de archivos, el calculo de hashes y la logica de commits.

## Estructura interna del repositorio Mini-Git

Cuando se ejecuta `minigit init`, se crea una carpeta oculta:

```text
.minigit/
|-- HEAD
|-- index.json
|-- objects/
`-- commits/
```

Significado:

- `.minigit/objects`: guarda el contenido de archivos usando SHA-1 como nombre.
- `.minigit/index.json`: guarda los archivos agregados con `add`.
- `.minigit/commits`: guarda los commits como archivos JSON.
- `.minigit/HEAD`: guarda el ID del ultimo commit activo.

## Ignorar archivos

Mini-Git soporta un archivo llamado `.minigitignore`, parecido a `.gitignore`.

Ejemplo:

```text
*.exe
*.log
bin/
node_modules/
```

Cuando ejecutas:

```powershell
minigit add .
```

Mini-Git no agregara los archivos que coincidan con esas reglas.

## Flujo interno

Cuando se ejecuta:

```powershell
minigit add archivo.txt
```

Mini-Git:

1. Lee el contenido del archivo.
2. Calcula un hash SHA-1.
3. Guarda el contenido en `.minigit/objects`.
4. Registra la relacion archivo-hash en `.minigit/index.json`.

Cuando se ejecuta:

```powershell
minigit diff archivo.txt
```

Mini-Git compara la version guardada en el index contra el archivo actual del directorio de trabajo.

Cuando se ejecuta:

```powershell
minigit ls-files
```

Mini-Git muestra los archivos registrados en `.minigit/index.json` junto con el hash del objeto guardado.

Cuando se ejecuta:

```powershell
minigit cat-file <hash>
```

Mini-Git busca ese hash dentro de `.minigit/objects` y muestra el contenido guardado.

Cuando se ejecuta:

```powershell
minigit commit -m "mensaje"
```

Mini-Git:

1. Lee el index.
2. Crea un snapshot de los archivos agregados.
3. Genera un ID para el commit.
4. Guarda el commit en `.minigit/commits`.
5. Actualiza `.minigit/HEAD`.

## Alcance actual

La demo implementa:

- Inicializacion de repositorio.
- Agregado de archivos y carpetas.
- Archivo `.minigitignore`.
- Commits con mensaje.
- Diff basico de archivos.
- Listado de archivos del index con `ls-files`.
- Lectura de objetos con `cat-file`.
- Historial de commits.
- Estado de archivos.
- Restauracion de commits.

Limitaciones actuales:

- No maneja ramas.
- No maneja merge.
- No detecta conflictos.
- No comprime objetos.
- No elimina automaticamente archivos que no existen en el commit restaurado.

## Notas para subir al repositorio

No se debe subir la carpeta `.minigit` ni el ejecutable compilado. Ya estan ignorados en `.gitignore`:

```text
.minigit/
minigit.exe
```

Cada persona puede compilar el ejecutable localmente con:

```powershell
go build -o minigit.exe ./cmd/minigit
```
