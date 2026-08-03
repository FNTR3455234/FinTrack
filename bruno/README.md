# La misma colección, para Bruno

[Bruno](https://www.usebruno.com/) es un cliente de API que guarda las peticiones como **archivos
de texto dentro del repositorio**, uno por petición, en vez de un único `.json` de 80 kB. Se lee
en un `diff`, se revisa en un *pull request* y no necesita cuenta ni nube.

Es la **misma** colección del [entregable 4](../postman/README.md): 53 peticiones en 11 carpetas.

| | |
|---|---|
| `bruno.json` | Marca la carpeta como colección |
| `collection.bru` | La autenticación *bearer* que heredan todas y la prueba que corre en todas |
| `environments/FinTrack local.bru` | `base_url` y las credenciales del usuario de ejemplo |
| `<Carpeta>/<Petición>.bru` | Una petición: URL, cuerpo, scripts y pruebas |
| `ejemplo.csv` | El archivo de muestra para la importación |
| `generar.js` | El conversor — ver abajo |

## Cómo se usa

Necesita la API viva y la semilla cargada, igual que la de Postman.

### Desde la aplicación

**Open Collection** → elige esta carpeta. Arriba a la derecha, el entorno **FinTrack local**.
Corre primero **Autenticación → Login (usuario demo)**; el token queda guardado y el resto de las
peticiones lo heredan.

### Desde la terminal

```bash
cd bruno
npx --yes @usebruno/cli run --env "FinTrack local"
```

Salida real de la última ejecución:

```
📊 Execution Summary
┌───────────────┬────────────────┐
│ Metric        │     Result     │
├───────────────┼────────────────┤
│ Status        │     ✓ PASS     │
├───────────────┼────────────────┤
│ Requests      │ 53 (53 Passed) │
├───────────────┼────────────────┤
│ Tests         │    141/141     │
├───────────────┼────────────────┤
│ Duration (ms) │      7774      │
└───────────────┴────────────────┘
```

## No se edita a mano: se genera

**La fuente de verdad es la colección de Postman.** Esta carpeta la escribe `generar.js` y se
borra entera en cada ejecución, así que un `.bru` editado a mano se pierde al día siguiente:

```bash
make bruno          # Windows:  .\make.ps1 bruno
```

Es una decisión deliberada y la explica la [046](../docs/decisiones.md): mantener dos
implementaciones de lo mismo significa que probar una no prueba la otra, y la que menos se usa se
pudre en silencio. Eso ya pasó con el `Makefile` y `make.ps1`. Con las dos colecciones el riesgo
era el mismo —arreglar una aserción en Postman y olvidarla en Bruno— así que aquí una **deriva**
de la otra en lugar de vivir a su lado.

## Qué cambia al traducir

Las aserciones son las mismas porque Bruno también trae `chai`; lo que cambia es cómo se llega a
la respuesta y a las variables:

| Postman | Bruno |
|---|---|
| `pm.test(...)` · `pm.expect(...)` | `test(...)` · `expect(...)` |
| `pm.response.to.have.status(200)` | `expect(res.getStatus()).to.equal(200)` |
| `pm.response.json()` | `res.getBody()` |
| `pm.response.headers.get("Content-Type")` | `res.getHeader("content-type")` |
| `pm.response.responseTime` | `res.getResponseTime()` |
| `pm.collectionVariables.set/get` | `bru.setVar/getVar` |

Tres detalles que costaron una ejecución en rojo cada uno:

- **`pm.response.text()` no tiene equivalente.** `res.getBody()` devuelve el objeto ya parseado
  cuando la respuesta es JSON, así que la prueba *"nunca devuelve la contraseña"* comparaba contra
  un objeto y fallaba. El conversor declara una variable `texto` cuando hace falta.
- **Los nombres de las cabeceras van en minúsculas**: Node las normaliza, y `res.getHeader` no
  busca sin distinguir mayúsculas.
- **El orden lo manda `seq`**, no el nombre del archivo. Importa: los identificadores viajan de
  una petición a la siguiente. Cada `.bru` lleva su número dentro de la carpeta y cada
  `folder.bru` el suyo dentro de la colección.

Si algún día aparece un `pm.*` que el conversor no conoce, **falla al generar** en vez de escribir
una colección que revienta en tiempo de ejecución.

## Dos aserciones más que en Postman

Bruno da 141 y Newman 139, y no es un descuadre: Postman **no puede guardar un archivo dentro de
la colección**, así que su petición *Importar desde CSV* llega con el campo vacío y sin pruebas —
hay que elegir el archivo a mano en cada ejecución. Bruno sí guarda la ruta (`@file(ejemplo.csv)`),
de modo que ahí la importación corre sola y se le pueden comprobar el `201` y las dos filas.

El precio está anotado en la propia petición: cada ejecución **añade de verdad** esos dos
movimientos, con fecha de agosto de 2026. Ninguna otra petición los mira —todas trabajan sobre
julio— pero se acumulan si la corres muchas veces.

## Por qué siguen las dos

Postman es lo que pide la rúbrica y lo que corre el CI (`make postman`, trabajo `contrato`).
Bruno se versiona en texto plano, que para revisar cambios es incomparablemente mejor. No compiten:
una se genera desde la otra.
