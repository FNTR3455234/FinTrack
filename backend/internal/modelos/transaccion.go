package modelos

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Transaccion es un movimiento de dinero.
//
// El monto siempre es positivo: lo que decide si suma o resta es el campo Tipo.
// Guardar signos mezclados en la misma coleccion es una fuente clasica de
// errores al sumar.
//
// Notas es un puntero porque el esquema acepta el campo en null y asi se
// distingue "sin notas" de "notas vacias".
type Transaccion struct {
	ID            bson.ObjectID `bson:"_id,omitempty"   json:"id"`
	UsuarioID     bson.ObjectID `bson:"usuario_id"      json:"-"`
	CuentaID      bson.ObjectID `bson:"cuenta_id"       json:"cuenta_id"`
	CategoriaID   bson.ObjectID `bson:"categoria_id"    json:"categoria_id"`
	Tipo          string        `bson:"tipo"            json:"tipo"`
	Monto         float64       `bson:"monto"           json:"monto"`
	Descripcion   string        `bson:"descripcion"     json:"descripcion"`
	Fecha         time.Time     `bson:"fecha"           json:"fecha"`
	Notas         *string       `bson:"notas"           json:"notas"`
	CreadoEn      time.Time     `bson:"creado_en"       json:"creado_en"`
	ActualizadoEn time.Time     `bson:"actualizado_en"  json:"actualizado_en"`
}

// PeticionTransaccion es el cuerpo de POST y PUT /transacciones.
//
// Los identificadores llegan como cadena hexadecimal de 24 caracteres, que es
// como se ve un ObjectID en JSON. El servicio los convierte y comprueba que
// pertenezcan al usuario del token.
type PeticionTransaccion struct {
	CuentaID    string    `json:"cuenta_id"    binding:"required,len=24,hexadecimal"`
	CategoriaID string    `json:"categoria_id" binding:"required,len=24,hexadecimal"`
	Tipo        string    `json:"tipo"         binding:"required,oneof=ingreso gasto"`
	Monto       float64   `json:"monto"        binding:"required,gt=0"`
	Descripcion string    `json:"descripcion"  binding:"required,min=1,max=140"`
	Fecha       time.Time `json:"fecha"        binding:"required"`
	Notas       string    `json:"notas"        binding:"omitempty,max=500"`
}

// Ordenamientos permitidos en el listado de transacciones.
const (
	OrdenFechaDesc = "fecha_desc"
	OrdenFechaAsc  = "fecha_asc"
	OrdenMontoDesc = "monto_desc"
	OrdenMontoAsc  = "monto_asc"
)

// Limites de la paginacion.
const (
	LimitePorDefecto = 20
	LimiteMaximo     = 100
)

// FiltroTransacciones son los criterios del listado. Los campos opcionales van
// como puntero para poder distinguir "no se pidio" de "se pidio el valor cero".
type FiltroTransacciones struct {
	Desde       *time.Time
	Hasta       *time.Time
	Tipo        string
	CategoriaID *bson.ObjectID
	CuentaID    *bson.ObjectID
	Busqueda    string
	Pagina      int
	Limite      int
	Orden       string
}

// Saltar son los documentos que la consulta debe brincar para llegar a la
// pagina pedida.
func (f FiltroTransacciones) Saltar() int64 {
	return int64((f.Pagina - 1) * f.Limite)
}
