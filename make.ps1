# Equivalente del Makefile para Windows, donde "make" no viene instalado.
# Mismos targets y mismos comandos que el Makefile: si cambias uno, cambia el otro.
#
# Uso:  .\make.ps1 up   /   .\make.ps1 test   /   .\make.ps1 (muestra la ayuda)

param(
    [Parameter(Position = 0)]
    [string]$Target = "help"
)

# OJO: aqui NO se usa $ErrorActionPreference = "Stop".
#
# En Windows PowerShell 5.1, con "Stop" cualquier linea que docker, go o npm
# escriban en la salida de error se convierte en un error fatal, aunque el
# comando haya terminado bien (docker compose informa el progreso por stderr).
# En su lugar se revisa el codigo de salida real con Verificar.
$ErrorActionPreference = "Continue"

# Verificar corta el script si el ultimo programa devolvio un codigo distinto de 0.
function Verificar([string]$paso) {
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Fallo: $paso (codigo $LASTEXITCODE)"
        exit $LASTEXITCODE
    }
}

# Todas las rutas son relativas a la raiz del repo, no al directorio donde se invoca.
$raiz = $PSScriptRoot
$composeDev = @("compose", "-f", "$raiz\docker-compose.dev.yml")
$mongosh = @("mongosh", "-u", "fintrack_admin", "-p", "fintrack_dev_2026", "--authenticationDatabase", "admin", "--quiet")

function Invoke-Ayuda {
    Write-Host "FinTrack - atajos disponibles:"
    Write-Host "  .\make.ps1 up      Levanta MongoDB (compose de desarrollo)"
    Write-Host "  .\make.ps1 down    Detiene MongoDB"
    Write-Host "  .\make.ps1 dev     Levanta Mongo y arranca la API en modo desarrollo   [fase 2]"
    Write-Host "  .\make.ps1 web     Arranca el frontend de Vite en el 5173             [fase 7]"
    Write-Host "  .\make.ps1 test    Corre las pruebas de Go con cobertura"
    Write-Host "  .\make.ps1 test-integracion  Todas las pruebas, incluidas las de MongoDB"
    Write-Host "  .\make.ps1 lint    Corre go vet y golangci-lint                        [fase 2]"
    Write-Host "  .\make.ps1 seed    Recrea el esquema y carga los datos semilla"
    Write-Host "  .\make.ps1 build   Compila el backend y el frontend                    [fase 7]"
    Write-Host "  .\make.ps1 swagger Regenera la especificacion OpenAPI de backend/docs"
    Write-Host "  .\make.ps1 postman Corre la coleccion de Postman con newman (necesita la API viva)"
}

function Invoke-Up {
    & docker @composeDev up -d
    Verificar "levantar MongoDB"
    Write-Host "MongoDB escuchando en localhost:27017"
}

function Invoke-Down {
    & docker @composeDev down
    Verificar "detener MongoDB"
}

function Invoke-Dev {
    Invoke-Up
    Push-Location "$raiz\backend"
    try { & go run ./cmd/api } finally { Pop-Location }
}

function Invoke-Web {
    # El frontend habla con la API a traves del proxy de Vite, asi que hace falta
    # que la API ya este corriendo (.\make.ps1 dev en otra terminal).
    Push-Location "$raiz\frontend"
    try {
        & npm install --no-audit --no-fund
        Verificar "instalar las dependencias del frontend"
        & npm run dev
    } finally { Pop-Location }
}

# MostrarCobertura imprime la ultima linea del reporte, que es el total.
#
# La salida se guarda en una variable en vez de mandarla por una tuberia: en
# PowerShell 5.1, pasar argumentos con la forma -bandera=valor a un programa
# externo dentro de una tuberia los deja mal formados.
function MostrarCobertura {
    $reporte = & go tool cover "-func=coverage.out"
    Verificar "leer el reporte de cobertura"
    Write-Host $reporte[-1]
}

function Invoke-Test {
    # Las pruebas de integracion se saltan solas al no estar MONGO_URI_PRUEBAS.
    Push-Location "$raiz\backend"
    try {
        & go test ./... "-coverpkg=./cmd/...,./internal/..." "-coverprofile=coverage.out"
        Verificar "pruebas del backend"
        MostrarCobertura
    } finally { Pop-Location }
}

function Invoke-TestIntegracion {
    # Levanta MongoDB y corre TODO, incluidas las pruebas contra la base.
    Invoke-Up
    Push-Location "$raiz\backend"
    try {
        $env:MONGO_URI_PRUEBAS = "mongodb://fintrack_admin:fintrack_dev_2026@localhost:27017/?authSource=admin"
        & go test ./... "-coverpkg=./cmd/...,./internal/..." "-coverprofile=coverage.out"
        Verificar "pruebas del backend"
        MostrarCobertura
    } finally {
        $env:MONGO_URI_PRUEBAS = $null
        Pop-Location
    }
}

function Invoke-Lint {
    Push-Location "$raiz\backend"
    try {
        & go vet ./...
        Verificar "go vet"
        if ($null -eq (Get-Command golangci-lint -ErrorAction SilentlyContinue)) {
            Write-Host "golangci-lint no instalado, se omite"
        } else {
            & golangci-lint run
        }
    } finally { Pop-Location }
}

function Invoke-Seed {
    # Los scripts se ejecutan con el mongosh que ya vive dentro del contenedor
    # (asi no hace falta instalarlo en Windows). La carpeta database/ esta montada
    # en /scripts. Los dos scripts son idempotentes.
    & docker @composeDev exec -T mongo @mongosh --file /scripts/01_crear_colecciones.js
    Verificar "crear las colecciones"
    & docker @composeDev exec -T mongo @mongosh --file /scripts/02_insertar_datos.js
    Verificar "cargar la semilla"
}

function Invoke-Build {
    Push-Location "$raiz\backend"
    try { & go build -o bin/api ./cmd/api; Verificar "compilar el backend" } finally { Pop-Location }
    Push-Location "$raiz\frontend"
    try {
        & npm ci
        Verificar "npm ci"
        & npm run build
        Verificar "npm run build"
    } finally { Pop-Location }
}

function Invoke-Swagger {
    # swag lee las anotaciones de los handlers y regenera backend/docs, que es lo
    # que sirve /swagger. Si no esta instalado, se instala en GOPATH/bin.
    $swag = Join-Path (& go env GOPATH) "bin\swag.exe"
    if (-not (Test-Path $swag)) {
        Write-Host "Instalando swag..."
        & go install github.com/swaggo/swag/cmd/swag@latest
        Verificar "instalar swag"
    }
    Push-Location "$raiz\backend"
    try {
        & $swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal
        Verificar "generar la especificacion OpenAPI"
    } finally { Pop-Location }
    Write-Host "Listo. Levanta la API y abre http://localhost:8080/swagger"
}

function Invoke-Postman {
    # newman es el ejecutor de colecciones de Postman en linea de comandos.
    # La API tiene que estar corriendo y la semilla cargada.
    & npx --yes newman run "$raiz\postman\FinTrack.postman_collection.json" `
        -e "$raiz\postman\FinTrack.postman_environment.json"
    Verificar "correr la coleccion de Postman"
}

switch ($Target.ToLower()) {
    "help"  { Invoke-Ayuda }
    "up"    { Invoke-Up }
    "down"  { Invoke-Down }
    "dev"   { Invoke-Dev }
    "web"   { Invoke-Web }
    "test"  { Invoke-Test }
    "test-integracion" { Invoke-TestIntegracion }
    "lint"  { Invoke-Lint }
    "seed"  { Invoke-Seed }
    "build" { Invoke-Build }
    "swagger" { Invoke-Swagger }
    "postman" { Invoke-Postman }
    default {
        Write-Host "Target desconocido: $Target"
        Invoke-Ayuda
        exit 1
    }
}
