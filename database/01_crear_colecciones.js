// FinTrack - Creacion de colecciones, validacion de esquema e indices.
//
// Es IDEMPOTENTE: se puede correr las veces que haga falta. Si la coleccion no
// existe la crea, y si ya existe solo actualiza su validador con collMod.
// Los indices se crean con createIndex, que no hace nada si ya existen con el
// mismo nombre y las mismas llaves.
//
// Uso:  mongosh -u fintrack_admin -p fintrack_dev_2026 --authenticationDatabase admin < 01_crear_colecciones.js
//       (o automaticamente al iniciar el contenedor por primera vez)

const bd = db.getSiblingDB("fintrack");

// Crea la coleccion con su validador, o le aplica el validador si ya existia.
// validationAction "error" = MongoDB rechaza la escritura que no cumpla el esquema.
function definirColeccion(nombre, esquema) {
  const opciones = {
    validator: { $jsonSchema: esquema },
    validationLevel: "strict",
    validationAction: "error",
  };
  if (bd.getCollectionNames().includes(nombre)) {
    bd.runCommand(Object.assign({ collMod: nombre }, opciones));
    print("  coleccion actualizada: " + nombre);
  } else {
    bd.createCollection(nombre, opciones);
    print("  coleccion creada:      " + nombre);
  }
}

print("Definiendo colecciones en la base 'fintrack'...");

// --- usuarios ---------------------------------------------------------------
// Dueño de todos los demas documentos. El email es la credencial de acceso y es
// unico; la contraseña se guarda como hash bcrypt, nunca en claro.
definirColeccion("usuarios", {
  bsonType: "object",
  required: ["nombre", "email", "password", "moneda", "fecha_registro", "activo"],
  additionalProperties: false,
  properties: {
    _id: { bsonType: "objectId" },
    nombre: { bsonType: "string", minLength: 2, maxLength: 80 },
    email: { bsonType: "string", pattern: "^[^@\\s]+@[^@\\s]+\\.[^@\\s]+$" },
    password: { bsonType: "string", description: "hash bcrypt, 60 caracteres" },
    moneda: { bsonType: "string", pattern: "^[A-Z]{3}$", description: "codigo ISO, por defecto MXN" },
    fecha_registro: { bsonType: "date" },
    activo: { bsonType: "bool" },
  },
});

// --- cuentas ----------------------------------------------------------------
// De donde sale o a donde entra el dinero. No se guarda el saldo actual: se
// calcula sumando las transacciones sobre saldo_inicial, asi no hay dos
// fuentes de verdad que se puedan desincronizar.
definirColeccion("cuentas", {
  bsonType: "object",
  required: ["usuario_id", "nombre", "tipo", "saldo_inicial", "color", "archivada"],
  additionalProperties: false,
  properties: {
    _id: { bsonType: "objectId" },
    usuario_id: { bsonType: "objectId" },
    nombre: { bsonType: "string", minLength: 1, maxLength: 60 },
    tipo: { enum: ["efectivo", "debito", "credito", "ahorro"] },
    saldo_inicial: { bsonType: "double", description: "puede ser negativo en tarjetas de credito" },
    color: { bsonType: "string", pattern: "^#[0-9A-Fa-f]{6}$" },
    archivada: { bsonType: "bool" },
  },
});

// --- categorias -------------------------------------------------------------
// Clasifican el movimiento. El tipo de la categoria y el de la transaccion
// deben coincidir; esa regla la valida el servicio, no el esquema.
definirColeccion("categorias", {
  bsonType: "object",
  required: ["usuario_id", "nombre", "tipo", "color", "icono", "archivada"],
  additionalProperties: false,
  properties: {
    _id: { bsonType: "objectId" },
    usuario_id: { bsonType: "objectId" },
    nombre: { bsonType: "string", minLength: 1, maxLength: 60 },
    tipo: { enum: ["ingreso", "gasto"] },
    color: { bsonType: "string", pattern: "^#[0-9A-Fa-f]{6}$" },
    icono: { bsonType: "string", maxLength: 40 },
    archivada: { bsonType: "bool" },
  },
});

// --- transacciones ----------------------------------------------------------
// El movimiento de dinero. Es la coleccion que mas crece y sobre la que corren
// las dos consultas relacionales. El monto siempre es positivo: lo que define
// si suma o resta es el campo tipo.
definirColeccion("transacciones", {
  bsonType: "object",
  required: [
    "usuario_id", "cuenta_id", "categoria_id", "tipo",
    "monto", "descripcion", "fecha", "creado_en", "actualizado_en",
  ],
  additionalProperties: false,
  properties: {
    _id: { bsonType: "objectId" },
    usuario_id: { bsonType: "objectId" },
    cuenta_id: { bsonType: "objectId" },
    categoria_id: { bsonType: "objectId" },
    tipo: { enum: ["ingreso", "gasto"] },
    monto: { bsonType: "double", exclusiveMinimum: true, minimum: 0 },
    descripcion: { bsonType: "string", minLength: 1, maxLength: 140 },
    fecha: { bsonType: "date", description: "se guarda en UTC" },
    // Se acepta null para que el backend pueda enviar el campo vacio sin fallar.
    notas: { bsonType: ["string", "null"], maxLength: 500 },
    creado_en: { bsonType: "date" },
    actualizado_en: { bsonType: "date" },
  },
});

// --- presupuestos -----------------------------------------------------------
// Limite de gasto de una categoria en un mes concreto. mes y anio se guardan
// como enteros (no como fecha) porque el presupuesto es del periodo completo.
definirColeccion("presupuestos", {
  bsonType: "object",
  required: ["usuario_id", "categoria_id", "monto_limite", "mes", "anio"],
  additionalProperties: false,
  properties: {
    _id: { bsonType: "objectId" },
    usuario_id: { bsonType: "objectId" },
    categoria_id: { bsonType: "objectId" },
    monto_limite: { bsonType: "double", exclusiveMinimum: true, minimum: 0 },
    // int o long: Go envia int32 y mongosh necesita NumberInt() para no mandar double.
    mes: { bsonType: ["int", "long"], minimum: 1, maximum: 12 },
    anio: { bsonType: ["int", "long"], minimum: 2000, maximum: 2100 },
  },
});

print("Creando indices...");

// Un email no se puede repetir: es la credencial de acceso.
bd.usuarios.createIndex({ email: 1 }, { unique: true, name: "idx_usuarios_email_unico" });

// Todo listado filtra por dueño; estos indices evitan un COLLSCAN en cada consulta.
bd.cuentas.createIndex({ usuario_id: 1 }, { name: "idx_cuentas_usuario" });
bd.categorias.createIndex({ usuario_id: 1 }, { name: "idx_categorias_usuario" });

// El listado principal de transacciones: del usuario, ordenadas de la mas
// reciente a la mas vieja. El indice sirve para el filtro y para el orden.
bd.transacciones.createIndex({ usuario_id: 1, fecha: -1 }, { name: "idx_transacciones_usuario_fecha" });

// Lo usan la agregacion de gastos por categoria y el $lookup de presupuestos.
bd.transacciones.createIndex({ usuario_id: 1, categoria_id: 1 }, { name: "idx_transacciones_usuario_categoria" });

// Un usuario no puede tener dos presupuestos para la misma categoria y periodo.
bd.presupuestos.createIndex(
  { usuario_id: 1, categoria_id: 1, mes: 1, anio: 1 },
  { unique: true, name: "idx_presupuestos_unico_periodo" }
);

print("Listo. Colecciones e indices de FinTrack en su lugar.");
