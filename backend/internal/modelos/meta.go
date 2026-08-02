package modelos

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Meta es un objetivo de ahorro: cuanto se quiere juntar y para cuando.
//
// No guarda cuanto se lleva ahorrado. Eso se calcula sumando las aportaciones,
// igual que el saldo de una cuenta se calcula sumando sus transacciones: una
// sola fuente de verdad (ver docs/decisiones.md, decision 020).
//
// FechaLimite es obligatoria a proposito. Sin ella no se puede responder la
// pregunta que hace util a una meta —¿a que ritmo tengo que ahorrar?— y lo que
// queda es una lista de deseos.
type Meta struct {
	ID            bson.ObjectID `bson:"_id,omitempty"  json:"id"`
	UsuarioID     bson.ObjectID `bson:"usuario_id"     json:"-"`
	Nombre        string        `bson:"nombre"         json:"nombre"`
	MontoObjetivo float64       `bson:"monto_objetivo" json:"monto_objetivo"`
	FechaLimite   time.Time     `bson:"fecha_limite"   json:"fecha_limite"`
	Color         string        `bson:"color"          json:"color"`
	Notas         *string       `bson:"notas"          json:"notas"`
	Archivada     bool          `bson:"archivada"      json:"archivada"`
	CreadoEn      time.Time     `bson:"creado_en"      json:"creado_en"`
	ActualizadoEn time.Time     `bson:"actualizado_en" json:"actualizado_en"`
}

// PeticionMeta es el cuerpo de POST y PUT /metas.
type PeticionMeta struct {
	Nombre        string    `json:"nombre"         binding:"required,min=1,max=80"`
	MontoObjetivo float64   `json:"monto_objetivo" binding:"required,gt=0"`
	FechaLimite   time.Time `json:"fecha_limite"   binding:"required"`
	Color         string    `json:"color"          binding:"required,len=7,hexcolor"`
	Notas         string    `json:"notas"          binding:"omitempty,max=500"`
	Archivada     bool      `json:"archivada"`
}

// FiltroMetas son los criterios del listado.
//
// Por defecto las archivadas no salen: una meta archivada es una que ya no se
// esta persiguiendo, y verlas todas juntas convierte la pantalla en un archivo
// historico en vez de en un plan.
type FiltroMetas struct {
	IncluirArchivadas bool
}
