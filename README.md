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

Compila el ejecutable:

```powershell
go build -o minigit.exe ./cmd/minigit
```

En Linux o macOS:

```bash
go build -o minigit ./cmd/minigit
```

## Como ejecutarlo

Hay dos formas.

Con Go directamente:

```powershell
go run ./cmd/minigit help
```

Con el ejecutable compilado:

```powershell
.\minigit.exe help
```

## Demo rapida

Inicializa Mini-Git en la carpeta actual:

```powershell
.\minigit.exe init
```

Crea un archivo de prueba:

```powershell
Set-Content archivo.txt "Hola mundo"
```

Agrega el archivo al index:

```powershell
.\minigit.exe add archivo.txt
```

Crea un commit:

```powershell
.\minigit.exe commit -m "primer commit"
```

Consulta el estado:

```powershell
.\minigit.exe status
```

Consulta el historial:

```powershell
.\minigit.exe log
```

Modifica el archivo:

```powershell
Set-Content archivo.txt "Hola mundo modificado"
```

Revisa los cambios:

```powershell
.\minigit.exe status
```

Guarda una nueva version:

```powershell
.\minigit.exe add archivo.txt
.\minigit.exe commit -m "actualice archivo"
```

Restaura una version anterior usando el ID que aparece en `log`:

```powershell
.\minigit.exe checkout <commit-id>
```

## Comandos disponibles

```text
minigit init
minigit add <archivo|carpeta> [...]
minigit commit -m "mensaje"
minigit status
minigit log
minigit checkout <commit-id>
```

## Arquitectura

```text
mini-git/
|-- cmd/
|   `-- minigit/
|       `-- main.go
|-- go.mod
|-- README.md
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

## Flujo interno

Cuando se ejecuta:

```powershell
.\minigit.exe add archivo.txt
```

Mini-Git:

1. Lee el contenido del archivo.
2. Calcula un hash SHA-1.
3. Guarda el contenido en `.minigit/objects`.
4. Registra la relacion archivo-hash en `.minigit/index.json`.

Cuando se ejecuta:

```powershell
.\minigit.exe commit -m "mensaje"
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
- Commits con mensaje.
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
