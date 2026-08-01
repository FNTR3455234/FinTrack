# Equivalente del Makefile para Windows, donde "make" no viene instalado.
# Mismos targets y mismos comandos que el Makefile: si cambias uno, cambia el otro.
#
# Uso:  .\make.ps1 up   /   .\make.ps1 test   /   .\make.ps1 (muestra la ayuda)

param(
    [Parameter(Position = 0)]
    [string]$Target = "help"
)

# Se detiene ante el primer error para no encadenar comandos sobre un paso fallido.
$ErrorActionPreference = "Stop"

# Todas las rutas son relativas a la raiz del repo, no al directorio donde se invoca.
$raiz = $PSScriptRoot
$composeDev = @("compose", "-f", "$raiz\docker-compose.dev.yml")
$mongosh = @("mongosh", "-u", "fintrack_admin", "-p", "fintrack_dev_2026", "--authenticationDatabase", "admin")

function Invoke-Ayuda {
    Write-Host "FinTrack - atajos disponibles:"
    Write-Host "  .\make.ps1 up      Levanta MongoDB (compose de desarrollo)"
    Write-Host "  .\make.ps1 down    Detiene MongoDB"
    Write-Host "  .\make.ps1 dev     Levanta Mongo y arranca la API en modo desarrollo   [fase 2]"
    Write-Host "  .\make.ps1 test    Corre las pruebas de Go con cobertura               [fase 3]"
    Write-Host "  .\make.ps1 lint    Corre go vet y golangci-lint                        [fase 2]"
    Write-Host "  .\make.ps1 seed    Carga los datos semilla en MongoDB                  [fase 1]"
    Write-Host "  .\make.ps1 build   Compila el backend y el frontend                    [fase 7]"
}

function Invoke-Up {
    & docker @composeDev up -d
    Write-Host "MongoDB escuchando en localhost:27017"
}

function Invoke-Down {
    & docker @composeDev down
}

function Invoke-Dev {
    Invoke-Up
    Push-Location "$raiz\backend"
    try { & go run ./cmd/api } finally { Pop-Location }
}

function Invoke-Test {
    Push-Location "$raiz\backend"
    try { & go test ./... -cover -coverprofile=coverage.out } finally { Pop-Location }
}

function Invoke-Lint {
    Push-Location "$raiz\backend"
    try {
        & go vet ./...
        if ($null -eq (Get-Command golangci-lint -ErrorAction SilentlyContinue)) {
            Write-Host "golangci-lint no instalado, se omite"
        } else {
            & golangci-lint run
        }
    } finally { Pop-Location }
}

function Invoke-Seed {
    # Se envia el script por stdin al mongosh que corre dentro del contenedor,
    # asi no hace falta tener mongosh instalado en Windows.
    Get-Content "$raiz\database\02_insertar_datos.js" | & docker @composeDev exec -T mongo @mongosh fintrack
}

function Invoke-Build {
    Push-Location "$raiz\backend"
    try { & go build -o bin/api ./cmd/api } finally { Pop-Location }
    Push-Location "$raiz\frontend"
    try {
        & npm ci
        & npm run build
    } finally { Pop-Location }
}

switch ($Target.ToLower()) {
    "help"  { Invoke-Ayuda }
    "up"    { Invoke-Up }
    "down"  { Invoke-Down }
    "dev"   { Invoke-Dev }
    "test"  { Invoke-Test }
    "lint"  { Invoke-Lint }
    "seed"  { Invoke-Seed }
    "build" { Invoke-Build }
    default {
        Write-Host "Target desconocido: $Target"
        Invoke-Ayuda
        exit 1
    }
}
