# Guía de despliegue

Cómo poner FinTrack en un servidor y mantenerlo. Todo lo que hay aquí está probado contra el
stack de `docker-compose.yml`.

> **Alcance honesto.** Esto es un despliegue de un solo servidor, pensado para una demostración,
> un entorno interno o un uso personal. No cubre alta disponibilidad, réplicas ni escalado
> horizontal; al final hay una sección con lo que faltaría para eso.

## 1. Requisitos

| | Mínimo | Por qué |
|---|---|---|
| Docker Engine | 24+ | `docker compose up --wait` necesita Compose v2.1.1+ |
| RAM | 2 GB | MongoDB reserva memoria para su caché; la API y nginx ocupan poco |
| Disco | 5 GB | Las imágenes ocupan ~1.3 GB; el resto son datos y respaldos |
| Puertos | Uno público | Solo el frontend se publica; la API y la base quedan en la red interna |

No hace falta instalar Go, Node ni MongoDB en el servidor: todo se compila dentro de las
imágenes.

## 2. Primer despliegue

En un servidor Linux:

```bash
git clone https://github.com/FNTR3455234/FinTrack.git
cd FinTrack

make arriba               # genera el .env, construye y levanta, en un paso
```

O paso a paso, si el servidor no tiene `make`:

```bash
cp .env.example .env
# edita .env: cambia MONGO_PASSWORD y los dos secretos de JWT
docker compose up --build -d --wait
```

`--wait` devuelve el control solo cuando los tres servicios están *healthy*; si algo no arranca,
falla ahí y no en la primera petición. En una máquina normal tarda menos de un minuto.

Comprueba que responde:

```bash
curl -sf http://localhost:8080/api/v1/health
# {"datos":{"estado":"ok","mongo":"ok","version":"1.0.0", ...}}
```

### Los datos de ejemplo

La primera vez que se crea el volumen, MongoDB ejecuta solo los scripts de `database/` y carga el
usuario demo. **En un despliegue real eso no es lo que quieres.** Para arrancar con la base
vacía, comenta estas dos líneas de `docker-compose.yml` antes del primer `up`:

```yaml
# - ./database/02_insertar_datos.js:/docker-entrypoint-initdb.d/02_insertar_datos.js:ro
```

El primer script (`01_crear_colecciones.js`) **sí debe quedarse**: crea las colecciones con su
validación de esquema y sus índices. La API también los crea al arrancar, de forma idempotente,
así que ninguno de los dos sobra.

## 3. Variables de entorno

Todas viven en el `.env` de la raíz y las lee `docker-compose.yml`. Están documentadas en
[`.env.example`](../.env.example).

| Variable | Qué es | Cuidado |
|---|---|---|
| `MONGO_USUARIO`, `MONGO_PASSWORD` | Credenciales del administrador de MongoDB | **Solo se aplican al crear el volumen.** Cambiarlas después no cambia nada; ver §7 |
| `MONGO_BD` | Nombre de la base | |
| `JWT_SECRETO_ACCESO` | Firma los tokens de 15 minutos | Mínimo 32 caracteres |
| `JWT_SECRETO_REFRESCO` | Firma los de 7 días | **Distinto** del anterior; el servidor no arranca si son iguales |
| `JWT_MINUTOS_ACCESO`, `JWT_DIAS_REFRESCO` | Vida de cada token | |
| `PUERTO_WEB` | Puerto en la máquina anfitriona | |
| `CORS_ORIGENES` | Orígenes permitidos | Solo importa si algún día publicas el puerto de la API |
| `VERSION` | Se graba en el binario y etiqueta las imágenes | La responde `/health` |

**El servidor se niega a arrancar** si falta una obligatoria, si un secreto tiene menos de 32
caracteres, si los dos secretos son iguales, o si en modo `release` siguen empezando por
`cambia_este`. Reporta todos los problemas de una vez:

```
configuracion invalida:
  - JWT_SECRETO_ACCESO debe tener al menos 32 caracteres (tiene 5)
  - los secretos de acceso y de refresco deben ser distintos
```

### Rotar los secretos de JWT

Cambiar `JWT_SECRETO_ACCESO` o `JWT_SECRETO_REFRESCO` y reiniciar la API **cierra todas las
sesiones abiertas**: los tokens firmados con el secreto viejo dejan de validar. No hay pérdida de
datos, pero todo el mundo tiene que volver a entrar. Es también la única forma de revocar un
token de refresco antes de sus 7 días (ver §8).

## 4. HTTPS

El stack sirve **HTTP en claro**. Los tokens viajan en el encabezado `Authorization`, así que sin
TLS cualquiera en la red los puede leer. **En un despliegue accesible desde internet, TLS no es
opcional.**

La forma más simple es poner un proxy inverso delante que termine TLS y hable HTTP con el
contenedor. Con Caddy son cuatro líneas y renueva el certificado solo:

```caddyfile
fintrack.ejemplo.mx {
    reverse_proxy localhost:8080
}
```

Con el proxy delante, ajusta:

1. `PUERTO_WEB=127.0.0.1:8080` en el `.env`, para que el contenedor **solo** escuche en la
   interfaz local y no se pueda entrar saltándose el proxy. (Compose acepta la forma
   `ip:puerto:puerto` en `ports`.)
2. Añade HSTS en el proxy, no en nginx: la cabecera solo tiene sentido cuando la conexión ya es
   HTTPS.

   ```caddyfile
   header Strict-Transport-Security "max-age=31536000; includeSubDomains"
   ```

3. Cambia `CORS_ORIGENES` a `https://fintrack.ejemplo.mx`.

La `Content-Security-Policy` que emite nginx ya es `default-src 'self'` y no hace falta tocarla.

## 5. Respaldo y restauración

`database/03_respaldo.sh` usa `mongodump` dentro del contenedor, así que no hay que instalar nada
en el servidor. Para el stack de producción hay que decirle qué contenedor y qué credenciales:

```bash
export CONTENEDOR=fintrack-mongo
export MONGO_USUARIO=$(grep '^MONGO_USUARIO=' .env | cut -d= -f2)
export MONGO_CLAVE=$(grep '^MONGO_PASSWORD=' .env | cut -d= -f2)

./database/03_respaldo.sh respaldar        # deja un .archive.gz en respaldos/
./database/03_respaldo.sh listar
./database/03_respaldo.sh restaurar respaldos/fintrack_20260802_1830.archive.gz
```

**`restaurar` usa `--drop`: reemplaza el contenido actual.** No es una fusión.

Un respaldo diario con `cron`:

```cron
0 3 * * * cd /srv/fintrack && CONTENEDOR=fintrack-mongo MONGO_USUARIO=... MONGO_CLAVE=... ./database/03_respaldo.sh respaldar
```

> Un respaldo que no se ha restaurado nunca no es un respaldo. Prueba la restauración en otra
> máquina al menos una vez.

## 6. Actualizar

```bash
cd /srv/fintrack
git pull
docker compose up --build -d --wait
```

Compose recrea solo los contenedores cuya imagen cambió. El volumen de MongoDB no se toca, así
que **no se pierden datos**. Los índices y la validación de esquema se vuelven a aplicar al
arrancar la API, de forma idempotente.

Para comprobar qué versión quedó desplegada:

```bash
curl -s http://localhost:8080/api/v1/health | grep -o '"version":"[^"]*"'
```

Volver atrás es `git checkout <etiqueta>` y repetir el `up`. Como las imágenes se etiquetan con
`VERSION`, la anterior sigue en el disco local mientras no se limpie con `docker image prune`.

## 7. Problemas frecuentes

**La API reinicia en bucle con `AuthenticationFailed`.**
La imagen de MongoDB **solo crea el usuario administrador la primera vez**, con el directorio de
datos vacío. Si el volumen ya existía con otras credenciales, las nuevas se ignoran en silencio.
O restauras las viejas en el `.env`, o cambias la contraseña dentro de Mongo:

```bash
docker compose exec mongo mongosh -u <usuario_viejo> -p <clave_vieja> --authenticationDatabase admin \
  --eval 'db.getSiblingDB("admin").changeUserPassword("<usuario>", "<clave_nueva>")'
```

Es la misma trampa que hizo que los dos stacks de Compose se pisaran
([decisión 033](decisiones.md)); por eso cada archivo declara su `name:`.

**El puerto 8080 está ocupado.** Cambia `PUERTO_WEB` en el `.env` y vuelve a levantar.

**Recargo `/transacciones` y da 404.** Solo pasa si sirves el `dist/` con otro servidor web sin el
*fallback* de aplicación de una sola página. `nginx.conf` ya lo trae (`try_files … /index.html`).

**Ver qué está pasando.**

```bash
docker compose logs -f api        # la API escribe JSON estructurado en modo release
docker compose ps                 # estado y salud de los tres servicios
docker compose exec mongo mongosh -u ... --eval 'db.stats()'
```

Cada respuesta trae un `X-Request-Id` que aparece también en la bitácora: es la forma de atar lo
que vio el usuario con lo que pasó en el servidor.

## 8. Lo que este despliegue **no** resuelve

Dicho para que nadie se lo encuentre por sorpresa:

- **Sin alta disponibilidad.** Un solo nodo de cada cosa. Si el servidor cae, la app cae. MongoDB
  está en *standalone* sin réplica ([decisión 003](decisiones.md)), así que tampoco hay
  transacciones de varios documentos.
- **Los tokens de refresco no se pueden revocar** antes de sus 7 días, salvo rotando el secreto
  —lo que cierra todas las sesiones a la vez—. No hay lista de sesiones activas
  ([limitación anotada desde la fase 3](decisiones.md)).
- **Sin recuperación de contraseña.** No hay envío de correo, así que un usuario que la olvide
  necesita que alguien le cambie el hash en la base.
- **Sin métricas ni alertas.** Hay `/health` y bitácora estructurada, que es con lo que se puede
  enganchar un vigilante externo, pero no hay Prometheus ni nada que avise solo.
- **Sin límite de peticiones fuera de `/auth`.** El resto de la API confía en que el token es
  válido. Para exponerla a internet de verdad conviene un límite por token en el proxy.
- **Los respaldos se quedan en el mismo servidor.** Un respaldo en el disco que puede fallar no
  protege de que falle ese disco: hay que copiarlos fuera.
