package modelos

import "go.mongodb.org/mongo-driver/v2/bson"

// Categoria clasifica un movimiento. Su tipo (ingreso o gasto) tiene que
// coincidir con el de las transacciones que la usan; esa regla la valida el
// servicio, no el esquema de MongoDB.
type Categoria struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UsuarioID bson.ObjectID `bson:"usuario_id"    json:"-"`
	Nombre    string        `bson:"nombre"        json:"nombre"`
	Tipo      string        `bson:"tipo"          json:"tipo"`
	Color     string        `bson:"color"         json:"color"`
	Icono     string        `bson:"icono"         json:"icono"`
	Archivada bool          `bson:"archivada"     json:"archivada"`
}

// Tipos de movimiento. Los usan categorias y transacciones.
const (
	TipoIngreso = "ingreso"
	TipoGasto   = "gasto"
)

// PeticionCategoria es el cuerpo de POST y PUT /categorias.
type PeticionCategoria struct {
	Nombre    string `json:"nombre"    binding:"required,min=1,max=60"`
	Tipo      string `json:"tipo"      binding:"required,oneof=ingreso gasto"`
	Color     string `json:"color"     binding:"required,len=7,hexcolor"`
	Icono     string `json:"icono"     binding:"required,max=40"`
	Archivada bool   `json:"archivada"`
}
