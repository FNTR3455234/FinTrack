package modelos

import "go.mongodb.org/mongo-driver/v2/bson"

// Estados del semaforo de un presupuesto.
//
// El umbral de alerta esta aqui, en un solo sitio, aunque quien lo aplica es la
// agregacion de MongoDB ($switch): asi el valor del codigo y el de la consulta
// no se pueden separar sin que se note.
const (
	EstadoOK       = "ok"
	EstadoAlerta   = "alerta"
	EstadoExcedido = "excedido"

	UmbralAlerta   = 80.0
	UmbralExcedido = 100.0
)

// Cuantos meses devuelve /reportes/tendencia.
const (
	MesesTendenciaPorDefecto = 6
	MesesTendenciaMaximo     = 24
)

// GastoPorCategoria es una fila de la consulta relacional 1: cuanto se gasto en
// una categoria durante el periodo y que peso tiene sobre el gasto total.
type GastoPorCategoria struct {
	CategoriaID bson.ObjectID `bson:"categoria_id" json:"categoria_id"`
	Nombre      string        `bson:"nombre"       json:"nombre"`
	Color       string        `bson:"color"        json:"color"`
	Icono       string        `bson:"icono"        json:"icono"`
	Total       float64       `bson:"total"        json:"total"`
	Cantidad    int           `bson:"cantidad"     json:"cantidad"`
	Porcentaje  float64       `bson:"porcentaje"   json:"porcentaje"`
}

// EstadoPresupuesto es una fila de la consulta relacional 2: lo presupuestado
// contra lo realmente gastado en una categoria durante un mes.
//
// Disponible puede ser negativo: es justo el caso que interesa ver.
type EstadoPresupuesto struct {
	PresupuestoID   bson.ObjectID `bson:"presupuesto_id"   json:"presupuesto_id"`
	CategoriaID     bson.ObjectID `bson:"categoria_id"     json:"categoria_id"`
	Nombre          string        `bson:"nombre"           json:"nombre"`
	Color           string        `bson:"color"            json:"color"`
	MontoLimite     float64       `bson:"monto_limite"     json:"monto_limite"`
	Gastado         float64       `bson:"gastado"          json:"gastado"`
	Disponible      float64       `bson:"disponible"       json:"disponible"`
	PorcentajeUsado float64       `bson:"porcentaje_usado" json:"porcentaje_usado"`
	Estado          string        `bson:"estado"           json:"estado"`
}

// Resumen son las cifras del mes que encabezan el tablero.
type Resumen struct {
	Periodo      Periodo         `json:"periodo"`
	Ingresos     float64         `json:"ingresos"`
	Gastos       float64         `json:"gastos"`
	Balance      float64         `json:"balance"`
	Movimientos  int             `json:"movimientos"`
	SaldoTotal   float64         `json:"saldo_total"`
	Presupuestos ContadorEstados `json:"presupuestos"`
}

// ContadorEstados resume el semaforo de todos los presupuestos del mes en tres
// numeros, para pintar el aviso del tablero sin recorrer la lista completa.
type ContadorEstados struct {
	Total     int `json:"total"`
	EnAlerta  int `json:"en_alerta"`
	Excedidos int `json:"excedidos"`
}

// PuntoTendencia es un mes de la serie de los ultimos meses.
//
// Los meses sin ningun movimiento tambien aparecen, con ceros: una grafica de
// barras a la que le faltan meses miente sobre la forma de la serie.
type PuntoTendencia struct {
	Periodo  Periodo `bson:"periodo"  json:"periodo"`
	Etiqueta string  `bson:"-"        json:"etiqueta"`
	Ingresos float64 `bson:"ingresos" json:"ingresos"`
	Gastos   float64 `bson:"gastos"   json:"gastos"`
	Balance  float64 `bson:"-"        json:"balance"`
	Cantidad int     `bson:"cantidad" json:"cantidad"`
}

// SaldoCuenta es el dinero que queda hoy en una cuenta.
//
// No se guarda en la coleccion cuentas: se calcula sumando sus movimientos al
// saldo inicial (ver docs/decisiones.md, decision 020). Asi no hay dos fuentes
// de verdad que se puedan desincronizar.
type SaldoCuenta struct {
	CuentaID     bson.ObjectID `bson:"cuenta_id"     json:"cuenta_id"`
	Nombre       string        `bson:"nombre"        json:"nombre"`
	Tipo         string        `bson:"tipo"          json:"tipo"`
	Color        string        `bson:"color"         json:"color"`
	Archivada    bool          `bson:"archivada"     json:"archivada"`
	SaldoInicial float64       `bson:"saldo_inicial" json:"saldo_inicial"`
	Ingresos     float64       `bson:"ingresos"      json:"ingresos"`
	Gastos       float64       `bson:"gastos"        json:"gastos"`
	Saldo        float64       `bson:"saldo"         json:"saldo"`
}

// AlertaPresupuesto es el aviso que acompaña a un gasto recien registrado
// cuando deja su categoria cerca del limite o por encima de el.
type AlertaPresupuesto struct {
	CategoriaID     bson.ObjectID `json:"categoria_id"`
	Nombre          string        `json:"nombre"`
	MontoLimite     float64       `json:"monto_limite"`
	Gastado         float64       `json:"gastado"`
	Disponible      float64       `json:"disponible"`
	PorcentajeUsado float64       `json:"porcentaje_usado"`
	Estado          string        `json:"estado"`
	Mensaje         string        `json:"mensaje"`
}
