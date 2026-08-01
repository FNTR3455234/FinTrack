#!/usr/bin/env bash
#
# FinTrack - Respaldo y restauracion de la base de datos.
#
# Usa mongodump/mongorestore DENTRO del contenedor, asi que no hace falta tener
# las herramientas de MongoDB instaladas en la maquina. El respaldo viaja por la
# salida estandar (--archive) y se guarda como un solo archivo comprimido.
#
#   ./database/03_respaldo.sh respaldar
#   ./database/03_respaldo.sh restaurar respaldos/fintrack_20260731_2215.archive.gz
#   ./database/03_respaldo.sh listar
#
# La carpeta respaldos/ esta en .gitignore: los respaldos no se versionan.

set -euo pipefail

CONTENEDOR="${CONTENEDOR:-fintrack-mongo-dev}"
BD="${BD:-fintrack}"
USUARIO="${MONGO_USUARIO:-fintrack_admin}"
CLAVE="${MONGO_CLAVE:-fintrack_dev_2026}"

# Las rutas son relativas a la raiz del repo, se invoque desde donde se invoque.
RAIZ="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DESTINO="$RAIZ/respaldos"

# Falla temprano y con un mensaje claro si el contenedor no esta arriba.
function verificar_contenedor() {
  if ! docker ps --format '{{.Names}}' | grep -q "^${CONTENEDOR}$"; then
    echo "Error: el contenedor '${CONTENEDOR}' no esta corriendo. Levantalo con: make up" >&2
    exit 1
  fi
}

function respaldar() {
  verificar_contenedor
  mkdir -p "$DESTINO"
  local archivo="$DESTINO/fintrack_$(date +%Y%m%d_%H%M%S).archive.gz"

  docker exec -i "$CONTENEDOR" mongodump \
    --username "$USUARIO" --password "$CLAVE" --authenticationDatabase admin \
    --db "$BD" --archive --gzip > "$archivo"

  echo "Respaldo creado: $archivo ($(du -h "$archivo" | cut -f1))"
}

function restaurar() {
  local archivo="${1:-}"
  if [[ -z "$archivo" ]]; then
    echo "Error: falta el archivo. Uso: $0 restaurar <archivo.archive.gz>" >&2
    exit 1
  fi
  if [[ ! -f "$archivo" ]]; then
    echo "Error: no existe el archivo '$archivo'" >&2
    exit 1
  fi
  verificar_contenedor

  # --drop borra cada coleccion antes de restaurarla, para que el resultado sea
  # exactamente el contenido del respaldo y no una mezcla con lo que ya habia.
  docker exec -i "$CONTENEDOR" mongorestore \
    --username "$USUARIO" --password "$CLAVE" --authenticationDatabase admin \
    --archive --gzip --drop < "$archivo"

  echo "Base '$BD' restaurada desde: $archivo"
}

function listar() {
  if [[ ! -d "$DESTINO" ]]; then
    echo "Todavia no hay respaldos."
    exit 0
  fi
  ls -lh "$DESTINO"
}

case "${1:-}" in
  respaldar) respaldar ;;
  restaurar) restaurar "${2:-}" ;;
  listar)    listar ;;
  *)
    echo "Uso: $0 {respaldar|restaurar <archivo>|listar}" >&2
    exit 1
    ;;
esac
