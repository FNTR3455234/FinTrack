package modelos

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Aportacion es dinero que el usuario aparta para una meta.
//
// NO es una transaccion y no toca el saldo de ninguna cuenta: apartar dinero
// para un viaje no es gastarlo. Si se registrara como gasto, el total de gastos
// del mes incluiria dinero que solo cambio de sitio, y tanto el resumen como el
// semaforo de presupuestos mentirian.
//
// Una aportacion no significa nada sin su meta: al borrar la meta se borran
// tambien sus aportaciones (composicion, no asociacion).
type Aportacion struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UsuarioID bson.ObjectID `bson:"usuario_id"    json:"-"`
	MetaID    bson.ObjectID `bson:"meta_id"       json:"meta_id"`
	Monto     float64       `bson:"monto"         json:"monto"`
	Fecha     time.Time     `bson:"fecha"         json:"fecha"`
	Nota      *string       `bson:"nota"          json:"nota"`
	CreadoEn  time.Time     `bson:"creado_en"     json:"creado_en"`
}

// PeticionAportacion es el cuerpo de POST /metas/{id}/aportaciones.
//
// La meta no viene aqui: llega en la ruta. Aceptarla tambien en el cuerpo
// abriria la puerta a que las dos no coincidieran.
type PeticionAportacion struct {
	Monto float64   `json:"monto" binding:"required,gt=0"`
	Fecha time.Time `json:"fecha" binding:"required"`
	Nota  string    `json:"nota"  binding:"omitempty,max=140"`
}

// MaximoAportacionesPorMeta es el tope de aportaciones que devuelve el detalle
// de una meta. Con el uso normal no se roza: es un limite de seguridad para que
// una sola peticion no pueda pedir una coleccion entera.
const MaximoAportacionesPorMeta = 500
