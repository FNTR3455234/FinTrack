package modelos

import "go.mongodb.org/mongo-driver/v2/bson"

// Presupuesto es el limite de gasto que el usuario se pone en una categoria
// para un mes concreto.
//
// mes y anio se guardan como enteros y no como fecha porque el presupuesto es
// del periodo entero, no de un instante. Guardarlo como Date obligaria a
// recordar en cada consulta que "el 1 a las 00:00 representa todo el mes".
//
// La combinacion (usuario_id, categoria_id, mes, anio) es unica en la base: un
// indice unico impide dos presupuestos para la misma categoria y periodo.
type Presupuesto struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UsuarioID   bson.ObjectID `bson:"usuario_id"    json:"-"`
	CategoriaID bson.ObjectID `bson:"categoria_id"  json:"categoria_id"`
	MontoLimite float64       `bson:"monto_limite"  json:"monto_limite"`
	Mes         int           `bson:"mes"           json:"mes"`
	Anio        int           `bson:"anio"          json:"anio"`
}

// Periodo devuelve el mes y el año del presupuesto.
func (p Presupuesto) Periodo() Periodo {
	return Periodo{Mes: p.Mes, Anio: p.Anio}
}

// PeticionPresupuesto es el cuerpo de POST y PUT /presupuestos.
//
// La categoria llega como cadena hexadecimal; el servicio la convierte y
// comprueba que sea del usuario del token y que sea de gasto.
type PeticionPresupuesto struct {
	CategoriaID string  `json:"categoria_id" binding:"required,len=24,hexadecimal"`
	MontoLimite float64 `json:"monto_limite" binding:"required,gt=0"`
	Mes         int     `json:"mes"          binding:"required,min=1,max=12"`
	Anio        int     `json:"anio"         binding:"required,min=2000,max=2100"`
}

// Periodo devuelve el periodo que trae la peticion.
func (p PeticionPresupuesto) Periodo() Periodo {
	return Periodo{Mes: p.Mes, Anio: p.Anio}
}

// FiltroPresupuestos son los criterios del listado.
//
// Periodo es un puntero porque es opcional: sin el se devuelven todos los
// presupuestos del usuario, con el solo los de ese mes.
type FiltroPresupuestos struct {
	Periodo *Periodo
}
