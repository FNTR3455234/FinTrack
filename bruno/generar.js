// Genera la coleccion de Bruno a partir de la de Postman.
//
//   node bruno/generar.js        (o: make bruno / .\make.ps1 bruno)
//
// La fuente de verdad es postman/FinTrack.postman_collection.json. Esta carpeta
// se REGENERA entera, asi que editar un .bru a mano no sirve de nada: el cambio
// va en la coleccion de Postman y despues se corre esto.
//
// El porque de no mantener las dos a mano esta en la decision 046: dos
// implementaciones de lo mismo significan que probar una no prueba la otra, y
// que la que menos se usa se pudre en silencio.
//
// No es un conversor general de Postman a Bruno: cubre exactamente lo que usa
// esta coleccion (peticiones http, cuerpo json y multipart, cabeceras,
// parametros de consulta, scripts previos y de prueba, y descripciones).

const fs = require('fs')
const path = require('path')

const AQUI = __dirname
const POSTMAN = path.join(AQUI, '..', 'postman')
const coleccion = JSON.parse(fs.readFileSync(path.join(POSTMAN, 'FinTrack.postman_collection.json'), 'utf8'))
const entorno = JSON.parse(fs.readFileSync(path.join(POSTMAN, 'FinTrack.postman_environment.json'), 'utf8'))

// Postman no puede guardar un archivo dentro del .json y deja el campo del CSV
// vacio; Bruno si guarda la ruta, asi que aqui la peticion corre sola y se le
// pueden poner aserciones que en Postman no existen.
const AJUSTES = {
  'Importar desde CSV': {
    tests: [
      'test("responde 201", function () {',
      '    expect(res.getStatus()).to.equal(201);',
      '});',
      'test("importa las dos filas del archivo", function () {',
      '    expect(res.getBody().datos.importadas).to.eql(2);',
      '});'
    ].join('\n'),
    docs: [
      'El archivo ya viene enganchado (ejemplo.csv, al lado de bruno.json), asi que esta',
      'peticion corre sin tocar nada. En Postman hay que elegirlo a mano en cada ejecucion.',
      '',
      'O entra el archivo completo o no entra nada: si una sola fila falla, responde 400 con',
      'la lista de errores y no guarda ninguna. Para verlo, cambia el nombre de la cuenta en',
      'ejemplo.csv por uno que no exista.',
      '',
      'OJO: cada ejecucion añade de verdad los dos movimientos, con fecha de agosto de 2026.',
      'Ninguna otra peticion de la coleccion los mira -- todas trabajan sobre julio -- pero se',
      'acumulan en la base si la corres muchas veces.'
    ].join('\n')
  }
}

// --- Traduccion de los scripts ---------------------------------------------

// Bruno trae chai (expect) y su propio test(), asi que las aserciones se
// quedan igual; lo que cambia es como se llega a la respuesta y a las
// variables. Si algun dia aparece un pm.* nuevo, esto revienta en vez de
// generar una coleccion que falla en tiempo de ejecucion.
// Postman distingue json() de text(); Bruno tiene un solo getBody() que devuelve
// el objeto ya parseado cuando la respuesta es JSON y la cadena cuando no lo es.
// Sin esta variable, `expect(res.getBody()).to.not.include("password")` se
// aplicaba a un objeto en vez de a una cadena y la prueba del perfil fallaba.
const AYUDANTE_TEXTO = 'const texto = typeof res.getBody() === "string" ? res.getBody() : JSON.stringify(res.getBody());'

function traducir (js) {
  const necesitaTexto = js.includes('pm.response.text()')
  const salida = js
    .replace(/pm\.response\.to\.have\.status\((\d+)\)/g, 'expect(res.getStatus()).to.equal($1)')
    .replace(/pm\.response\.json\(\)/g, 'res.getBody()')
    .replace(/pm\.response\.text\(\)/g, 'texto')
    // Node normaliza los nombres de las cabeceras a minusculas.
    .replace(/pm\.response\.headers\.get\((["'])([^"']+)\1\)/g, (_m, _q, h) => `res.getHeader("${h.toLowerCase()}")`)
    .replace(/pm\.response\.responseTime/g, 'res.getResponseTime()')
    .replace(/pm\.collectionVariables\.set\(/g, 'bru.setVar(')
    .replace(/pm\.collectionVariables\.get\(/g, 'bru.getVar(')
    .replace(/pm\.environment\.get\(/g, 'bru.getEnvVar(')
    .replace(/pm\.expect\(/g, 'expect(')
    .replace(/pm\.test\(/g, 'test(')
  if (/\bpm\./.test(salida)) {
    throw new Error('quedo un pm.* sin traducir:\n' + salida)
  }
  return necesitaTexto ? AYUDANTE_TEXTO + '\n\n' + salida : salida
}

// --- Utilidades del formato .bru -------------------------------------------

// Bruno le quita la sangria comun al contenido de cada bloque al leerlo, asi
// que meter dos espacios es justo lo que hace su propio serializador.
const sangrar = (texto) => texto.split('\n').map((l) => (l.trim() === '' ? '' : '  ' + l)).join('\n')
const bloque = (nombre, cuerpo) => `${nombre} {\n${sangrar(cuerpo)}\n}\n`
const diccionario = (nombre, pares) => bloque(nombre, pares.map(([k, v]) => `${k}: ${v}`).join('\n'))

// Windows no admite \ / : * ? " < > | en un nombre de archivo.
const nombreArchivo = (n) => n.replace(/[\\/:*?"<>|]/g, '-').trim()

const url = (r) => (typeof r.url === 'string' ? r.url : r.url.raw)
const consulta = (r) => (typeof r.url === 'string' || !r.url.query ? [] : r.url.query.map((q) => [q.key, q.value]))

function eventos (item, escucha) {
  return (item.event || [])
    .filter((e) => e.listen === escucha)
    .map((e) => (e.script.exec || []).join('\n').trim())
    .filter(Boolean)
    .join('\n\n')
}

// --- Una peticion ----------------------------------------------------------

function generarPeticion (item, seq) {
  const r = item.request
  const ajuste = AJUSTES[item.name] || {}
  const metodo = r.method.toLowerCase()
  const tipoCuerpo = !r.body ? 'none' : r.body.mode === 'raw' ? 'json' : 'multipartForm'
  // Las publicas llevan auth: noauth en Postman; el resto hereda el bearer de
  // la coleccion, igual que alli.
  const auth = r.auth && r.auth.type === 'noauth' ? 'none' : 'inherit'

  let bru = diccionario('meta', [['name', item.name], ['type', 'http'], ['seq', String(seq)]]) + '\n'
  bru += diccionario(metodo, [['url', url(r)], ['body', tipoCuerpo], ['auth', auth]]) + '\n'

  const q = consulta(r)
  if (q.length) bru += diccionario('params:query', q) + '\n'

  const cabeceras = (r.header || []).filter((h) => !h.disabled)
  if (cabeceras.length) bru += diccionario('headers', cabeceras.map((h) => [h.key, h.value])) + '\n'

  if (tipoCuerpo === 'json') {
    bru += bloque('body:json', r.body.raw) + '\n'
  } else if (tipoCuerpo === 'multipartForm') {
    const campos = r.body.formdata.map((f) => [f.key, f.type === 'file' ? '@file(ejemplo.csv)' : f.value])
    bru += diccionario('body:multipart-form', campos) + '\n'
  }

  const previo = eventos(item, 'prerequest')
  if (previo) bru += bloque('script:pre-request', traducir(previo)) + '\n'

  const pruebas = ajuste.tests || traducir(eventos(item, 'test'))
  if (pruebas) bru += bloque('tests', pruebas) + '\n'

  const docs = ajuste.docs || r.description
  if (docs) bru += bloque('docs', docs) + '\n'

  return bru
}

// --- Recorrido -------------------------------------------------------------

let peticiones = 0

function generarCarpeta (item, destino, seq) {
  fs.mkdirSync(destino, { recursive: true })
  // El orden importa: los identificadores viajan de una peticion a la
  // siguiente. Bruno ordena por este seq, no por el nombre del archivo.
  fs.writeFileSync(path.join(destino, 'folder.bru'),
    diccionario('meta', [['name', item.name], ['seq', String(seq)]]), 'utf8')

  item.item.forEach((hijo, i) => {
    if (hijo.item) {
      generarCarpeta(hijo, path.join(destino, nombreArchivo(hijo.name)), i + 1)
      return
    }
    fs.writeFileSync(path.join(destino, nombreArchivo(hijo.name) + '.bru'), generarPeticion(hijo, i + 1), 'utf8')
    peticiones++
  })
}

// Se borra solo lo generado: este script y el README de la carpeta se quedan.
for (const generado of ['collection.bru', 'bruno.json', 'ejemplo.csv', 'environments', ...coleccion.item.map((i) => nombreArchivo(i.name))]) {
  fs.rmSync(path.join(AQUI, generado), { recursive: true, force: true })
}

fs.mkdirSync(path.join(AQUI, 'environments'), { recursive: true })

fs.writeFileSync(path.join(AQUI, 'bruno.json'), JSON.stringify({
  version: '1',
  name: coleccion.info.name,
  type: 'collection',
  ignore: ['node_modules', '.git']
}, null, 2) + '\n', 'utf8')

// collection.bru: el bearer que heredan todas y la prueba que corre en todas.
const token = coleccion.auth.bearer.find((b) => b.key === 'token').value
let raiz = diccionario('auth', [['mode', 'bearer']]) + '\n'
raiz += diccionario('auth:bearer', [['token', token]]) + '\n'
const global = (coleccion.event || [])
  .filter((e) => e.listen === 'test')
  .map((e) => e.script.exec.join('\n').trim())
  .join('\n\n')
if (global) raiz += bloque('tests', traducir(global)) + '\n'
fs.writeFileSync(path.join(AQUI, 'collection.bru'), raiz, 'utf8')

fs.writeFileSync(path.join(AQUI, 'environments', entorno.name + '.bru'),
  diccionario('vars', entorno.values.filter((v) => v.enabled !== false).map((v) => [v.key, v.value])), 'utf8')

coleccion.item.forEach((it, i) => generarCarpeta(it, path.join(AQUI, nombreArchivo(it.name)), i + 1))

// Bruno resuelve las rutas de @file() desde la raiz de la coleccion, asi que el
// CSV tiene que estar aqui dentro y no en ../postman.
fs.copyFileSync(path.join(POSTMAN, 'ejemplo.csv'), path.join(AQUI, 'ejemplo.csv'))

console.log(`Coleccion de Bruno generada: ${coleccion.item.length} carpetas, ${peticiones} peticiones.`)
