// FinTrack - Datos semilla.
//
// Crea un usuario demo con 3 cuentas, 10 categorias, 6 presupuestos y 120
// transacciones repartidas en 6 meses (febrero a julio de 2026).
//
// Es IDEMPOTENTE: borra primero todo lo del usuario demo y lo vuelve a insertar,
// asi que se puede correr las veces que haga falta sin duplicar nada.
// Los _id son fijos para que las pruebas puedan apoyarse en datos conocidos.
//
// Acceso demo:  demo@fintrack.mx  /  Demo1234!
//
// Uso:  make seed   /   .\make.ps1 seed

const bd = db.getSiblingDB("fintrack");

// Ultimo mes que cubre la semilla. Cambiar estas dos lineas re-ancla los 6 meses
// de datos si la demostracion se hace mas adelante en el tiempo.
const ANIO_FINAL = 2026;
const MES_FINAL = 7; // julio
const MESES = 6;

const ID_USUARIO = ObjectId("650000000000000000000001");

// Redondea a centavos y lo devuelve como Double de BSON.
// El Double() es necesario: mongosh guarda los numeros enteros de JavaScript
// como int32, y el esquema exige double porque es lo que manda Go (float64).
function pesos(n) {
  return Double(Math.round(n * 100) / 100);
}

// Construye una fecha UTC al mediodia (para que el dia del calendario sea el
// mismo en Mexico y en UTC) y recorta el dia si el mes es mas corto.
function fechaUTC(anio, mes, dia) {
  const ultimoDia = new Date(Date.UTC(anio, mes, 0)).getUTCDate();
  return new Date(Date.UTC(anio, mes - 1, Math.min(dia, ultimoDia), 12, 0, 0));
}

print("Limpiando datos previos del usuario demo...");
bd.aportaciones.deleteMany({ usuario_id: ID_USUARIO });
bd.metas.deleteMany({ usuario_id: ID_USUARIO });
bd.transacciones.deleteMany({ usuario_id: ID_USUARIO });
bd.presupuestos.deleteMany({ usuario_id: ID_USUARIO });
bd.categorias.deleteMany({ usuario_id: ID_USUARIO });
bd.cuentas.deleteMany({ usuario_id: ID_USUARIO });
bd.usuarios.deleteOne({ _id: ID_USUARIO });

// --- usuario ----------------------------------------------------------------
bd.usuarios.insertOne({
  _id: ID_USUARIO,
  nombre: "Usuario Demo",
  email: "demo@fintrack.mx",
  // Hash bcrypt (cost 10) de "Demo1234!". Nunca se guarda la contraseña en claro.
  password: "$2a$10$76BgwHbDgsucfxkem2e6UO198KG2nLVp4uX6lFQ4WC/20ojmSigni",
  moneda: "MXN",
  fecha_registro: fechaUTC(ANIO_FINAL, MES_FINAL - MESES + 1, 1),
  activo: true,
});

// --- cuentas ----------------------------------------------------------------
const cuentas = {
  efectivo: { _id: ObjectId("650100000000000000000001"), nombre: "Efectivo", tipo: "efectivo", saldo_inicial: pesos(2500), color: "#22C55E" },
  debito: { _id: ObjectId("650100000000000000000002"), nombre: "BBVA Debito", tipo: "debito", saldo_inicial: pesos(18000), color: "#2563EB" },
  credito: { _id: ObjectId("650100000000000000000003"), nombre: "Santander Credito", tipo: "credito", saldo_inicial: pesos(-4500), color: "#DC2626" },
};
bd.cuentas.insertMany(
  Object.values(cuentas).map((c) => Object.assign({ usuario_id: ID_USUARIO, archivada: false }, c))
);

// --- categorias -------------------------------------------------------------
const categorias = {
  nomina: { _id: ObjectId("650200000000000000000001"), nombre: "Nomina", tipo: "ingreso", color: "#16A34A", icono: "💼" },
  freelance: { _id: ObjectId("650200000000000000000002"), nombre: "Freelance", tipo: "ingreso", color: "#0D9488", icono: "💻" },
  renta: { _id: ObjectId("650200000000000000000003"), nombre: "Renta", tipo: "gasto", color: "#7C3AED", icono: "🏠" },
  super: { _id: ObjectId("650200000000000000000004"), nombre: "Supermercado", tipo: "gasto", color: "#EA580C", icono: "🛒" },
  transporte: { _id: ObjectId("650200000000000000000005"), nombre: "Transporte", tipo: "gasto", color: "#0891B2", icono: "🚗" },
  restaurantes: { _id: ObjectId("650200000000000000000006"), nombre: "Restaurantes", tipo: "gasto", color: "#DB2777", icono: "🍽️" },
  servicios: { _id: ObjectId("650200000000000000000007"), nombre: "Servicios", tipo: "gasto", color: "#CA8A04", icono: "💡" },
  entretenimiento: { _id: ObjectId("650200000000000000000008"), nombre: "Entretenimiento", tipo: "gasto", color: "#9333EA", icono: "🎬" },
  salud: { _id: ObjectId("650200000000000000000009"), nombre: "Salud", tipo: "gasto", color: "#E11D48", icono: "🩺" },
  educacion: { _id: ObjectId("65020000000000000000000a"), nombre: "Educacion", tipo: "gasto", color: "#4F46E5", icono: "📚" },
};
bd.categorias.insertMany(
  Object.values(categorias).map((c) => Object.assign({ usuario_id: ID_USUARIO, archivada: false }, c))
);

// --- transacciones ----------------------------------------------------------
// Un mes tipico del usuario demo: 20 movimientos que se repiten cada mes.
// Los marcados como fijo (renta, nomina, suscripciones) valen igual todos los
// meses; el resto varia segun el factor mensual de mas abajo.
const PLANTILLA_MENSUAL = [
  { dia: 1, cat: "renta", cuenta: "debito", monto: 9500, desc: "Renta del departamento", fijo: true },
  { dia: 3, cat: "super", cuenta: "debito", monto: 850, desc: "Despensa de la semana" },
  { dia: 5, cat: "freelance", cuenta: "debito", monto: 3500, desc: "Proyecto freelance" },
  { dia: 6, cat: "transporte", cuenta: "efectivo", monto: 320, desc: "Gasolina" },
  { dia: 8, cat: "restaurantes", cuenta: "credito", monto: 380, desc: "Comida fuera de casa" },
  { dia: 9, cat: "servicios", cuenta: "debito", monto: 599, desc: "Internet", fijo: true },
  { dia: 10, cat: "super", cuenta: "debito", monto: 1240, desc: "Despensa quincenal" },
  { dia: 11, cat: "educacion", cuenta: "debito", monto: 1200, desc: "Curso en linea", fijo: true },
  { dia: 12, cat: "servicios", cuenta: "debito", monto: 780, desc: "Recibo de luz" },
  { dia: 14, cat: "restaurantes", cuenta: "credito", monto: 620, desc: "Cena con amigos" },
  { dia: 15, cat: "nomina", cuenta: "debito", monto: 12500, desc: "Nomina primera quincena", fijo: true },
  { dia: 16, cat: "entretenimiento", cuenta: "credito", monto: 299, desc: "Suscripciones de streaming", fijo: true },
  { dia: 17, cat: "super", cuenta: "efectivo", monto: 690, desc: "Frutas y verduras" },
  { dia: 18, cat: "servicios", cuenta: "debito", monto: 240, desc: "Recibo de agua" },
  { dia: 20, cat: "transporte", cuenta: "efectivo", monto: 450, desc: "Transporte publico y taxis" },
  { dia: 21, cat: "salud", cuenta: "debito", monto: 850, desc: "Consulta y medicamentos" },
  { dia: 22, cat: "restaurantes", cuenta: "efectivo", monto: 245, desc: "Cafe y antojos" },
  { dia: 24, cat: "super", cuenta: "debito", monto: 1105, desc: "Despensa de fin de mes" },
  { dia: 27, cat: "entretenimiento", cuenta: "credito", monto: 450, desc: "Cine y salidas" },
  { dia: 30, cat: "nomina", cuenta: "debito", monto: 12500, desc: "Nomina segunda quincena", fijo: true },
];

// Factor por el que se multiplican los gastos variables de cada mes, del mas
// antiguo al mas reciente. Son valores fijos (no aleatorios) para que la
// semilla sea reproducible y las pruebas puedan esperar totales exactos.
const VARIACION = [0.95, 1.08, 0.9, 1.12, 1.03, 1.06];

const transacciones = [];
const ahora = new Date();

for (let i = 0; i < MESES; i++) {
  // i = 0 es el mes mas antiguo; i = MESES-1 es ANIO_FINAL/MES_FINAL.
  const desplazamiento = MES_FINAL - (MESES - 1 - i);
  const anio = ANIO_FINAL + Math.floor((desplazamiento - 1) / 12);
  const mes = ((desplazamiento - 1 + 12) % 12) + 1;

  PLANTILLA_MENSUAL.forEach((m) => {
    const categoria = categorias[m.cat];
    transacciones.push({
      usuario_id: ID_USUARIO,
      cuenta_id: cuentas[m.cuenta]._id,
      categoria_id: categoria._id,
      tipo: categoria.tipo,
      monto: pesos(m.fijo ? m.monto : m.monto * VARIACION[i]),
      descripcion: m.desc,
      fecha: fechaUTC(anio, mes, m.dia),
      notas: null,
      creado_en: ahora,
      actualizado_en: ahora,
    });
  });
}
bd.transacciones.insertMany(transacciones);

// --- presupuestos -----------------------------------------------------------
// Limites del ultimo mes de la semilla, elegidos a proposito para que el
// reporte de estado muestre los tres estados: ok, alerta y excedido.
const presupuestos = [
  { cat: "super", limite: 4000 },          // gastado 4118.10 -> excedido
  { cat: "restaurantes", limite: 1500 },   // gastado 1319.70 -> alerta
  { cat: "servicios", limite: 2000 },      // gastado 1680.20 -> alerta
  { cat: "entretenimiento", limite: 1000 },// gastado  776.00 -> ok
  { cat: "transporte", limite: 1200 },     // gastado  816.20 -> ok
  { cat: "salud", limite: 1500 },          // gastado  901.00 -> ok
];
bd.presupuestos.insertMany(
  presupuestos.map((p, i) => ({
    _id: ObjectId("65030000000000000000000" + (i + 1)),
    usuario_id: ID_USUARIO,
    categoria_id: categorias[p.cat]._id,
    monto_limite: pesos(p.limite),
    mes: NumberInt(MES_FINAL),
    anio: NumberInt(ANIO_FINAL),
  }))
);

// --- metas de ahorro --------------------------------------------------------
// Tres metas elegidas para que el reporte de progreso muestre los tres estados:
// una cumplida, una en curso y una ya vencida sin completar.
//
// Las fechas se calculan a partir del ultimo mes de la semilla, no a mano, para
// que sigan teniendo sentido si se re-ancla la semilla (ANIO_FINAL / MES_FINAL).
const metas = [
  {
    _id: ObjectId("650400000000000000000001"),
    nombre: "Fondo de emergencia",
    objetivo: 30000,
    mesesDesdeElFinal: 6, // vence dentro de medio año
    color: "#0891B2",
    notas: "Tres meses de gastos fijos.",
    // 18 500 de 30 000 -> 61.7 %, en curso
    aportaciones: [
      { mesesAntes: 5, dia: 15, monto: 4000, nota: "Aguinaldo" },
      { mesesAntes: 3, dia: 15, monto: 6000, nota: null },
      { mesesAntes: 1, dia: 10, monto: 5500, nota: null },
      { mesesAntes: 0, dia: 5, monto: 3000, nota: null },
    ],
  },
  {
    _id: ObjectId("650400000000000000000002"),
    nombre: "Laptop nueva",
    objetivo: 22000,
    mesesDesdeElFinal: 2,
    color: "#7C3AED",
    notas: null,
    // 22 500 de 22 000 -> 102.3 %, cumplida
    aportaciones: [
      { mesesAntes: 4, dia: 20, monto: 7500, nota: null },
      { mesesAntes: 2, dia: 20, monto: 7500, nota: null },
      { mesesAntes: 0, dia: 20, monto: 7500, nota: "Ultima aportacion" },
    ],
  },
  {
    _id: ObjectId("650400000000000000000003"),
    nombre: "Viaje de fin de año",
    objetivo: 25000,
    mesesDesdeElFinal: -1, // vencio el mes pasado
    color: "#EA580C",
    notas: "No alcanzo: se pospone al siguiente periodo.",
    // 9 000 de 25 000 -> 36 %, vencida
    aportaciones: [
      { mesesAntes: 4, dia: 28, monto: 5000, nota: null },
      { mesesAntes: 2, dia: 28, monto: 4000, nota: null },
    ],
  },
];

// mesRelativo desplaza N meses respecto al ultimo mes de la semilla y devuelve
// { anio, mes } normalizados, para no pelearse con los cambios de año.
function mesRelativo(desplazamiento) {
  const total = ANIO_FINAL * 12 + (MES_FINAL - 1) + desplazamiento;
  return { anio: Math.floor(total / 12), mes: (total % 12) + 1 };
}

const documentosMetas = [];
const documentosAportaciones = [];

metas.forEach((m) => {
  const limite = mesRelativo(m.mesesDesdeElFinal);
  documentosMetas.push({
    _id: m._id,
    usuario_id: ID_USUARIO,
    nombre: m.nombre,
    monto_objetivo: pesos(m.objetivo),
    fecha_limite: fechaUTC(limite.anio, limite.mes, 28),
    color: m.color,
    notas: m.notas,
    archivada: false,
    creado_en: ahora,
    actualizado_en: ahora,
  });

  m.aportaciones.forEach((a) => {
    const cuando = mesRelativo(-a.mesesAntes);
    documentosAportaciones.push({
      usuario_id: ID_USUARIO,
      meta_id: m._id,
      monto: pesos(a.monto),
      fecha: fechaUTC(cuando.anio, cuando.mes, a.dia),
      nota: a.nota,
      creado_en: ahora,
    });
  });
});

bd.metas.insertMany(documentosMetas);
bd.aportaciones.insertMany(documentosAportaciones);

print("Semilla cargada para demo@fintrack.mx (clave: Demo1234!):");
print("  cuentas:       " + bd.cuentas.countDocuments({ usuario_id: ID_USUARIO }));
print("  categorias:    " + bd.categorias.countDocuments({ usuario_id: ID_USUARIO }));
print("  transacciones: " + bd.transacciones.countDocuments({ usuario_id: ID_USUARIO }));
print("  presupuestos:  " + bd.presupuestos.countDocuments({ usuario_id: ID_USUARIO }));
print("  metas:         " + bd.metas.countDocuments({ usuario_id: ID_USUARIO }));
print("  aportaciones:  " + bd.aportaciones.countDocuments({ usuario_id: ID_USUARIO }));
