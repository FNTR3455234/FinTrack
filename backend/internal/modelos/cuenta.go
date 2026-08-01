package modelos

import "go.mongodb.org/mongo-driver/v2/bson"

// Cuenta es de donde sale o a donde entra el dinero.
//
// UsuarioID lleva `json:"-"`: el cliente nunca lo manda ni lo recibe. Su valor
// sale siempre del token, y esconderlo del JSON deja claro que no es un dato
// que el cliente pueda elegir.
//
// No se guarda el saldo actual, solo el inicial: el saldo se calcula sumando
// las transacciones (fase 5), asi no hay dos fuentes de verdad.
type Cuenta struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UsuarioID    bson.ObjectID `bson:"usuario_id"    json:"-"`
	Nombre       string        `bson:"nombre"        json:"nombre"`
	Tipo         string        `bson:"tipo"          json:"tipo"`
	SaldoInicial float64       `bson:"saldo_inicial" json:"saldo_inicial"`
	Color        string        `bson:"color"         json:"color"`
	Archivada    bool          `bson:"archivada"     json:"archivada"`
}

// Tipos de cuenta permitidos. Coinciden con el enum del $jsonSchema.
const (
	CuentaEfectivo = "efectivo"
	CuentaDebito   = "debito"
	CuentaCredito  = "credito"
	CuentaAhorro   = "ahorro"
)

// PeticionCuenta es el cuerpo de POST y PUT /cuentas.
//
// SaldoInicial es un puntero a proposito: con float64 a secas, la regla
// `required` rechazaria un saldo de 0, que es perfectamente valido en una
// cuenta nueva. Con puntero, `required` solo exige que el campo venga.
//
// El color se valida con len=7 ademas de hexcolor porque hexcolor tambien
// acepta la forma corta (#abc) y el esquema de MongoDB exige los seis digitos.
type PeticionCuenta struct {
	Nombre       string   `json:"nombre"        binding:"required,min=1,max=60"`
	Tipo         string   `json:"tipo"          binding:"required,oneof=efectivo debito credito ahorro"`
	SaldoInicial *float64 `json:"saldo_inicial" binding:"required"`
	Color        string   `json:"color"         binding:"required,len=7,hexcolor"`
	Archivada    bool     `json:"archivada"`
}
