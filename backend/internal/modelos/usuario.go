// Package modelos define los documentos que se guardan en MongoDB y los DTO
// (los objetos que entran y salen por HTTP). Se separan a proposito: el
// documento tiene campos que nunca deben salir a la API, como el hash de la
// contraseña.
package modelos

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Usuario es el documento de la coleccion usuarios.
//
// Password lleva `json:"-"`: aunque alguien devuelva un Usuario completo por
// error, el hash jamas se serializa en una respuesta.
type Usuario struct {
	ID            bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Nombre        string        `bson:"nombre"         json:"nombre"`
	Email         string        `bson:"email"          json:"email"`
	Password      string        `bson:"password"       json:"-"`
	Moneda        string        `bson:"moneda"         json:"moneda"`
	FechaRegistro time.Time     `bson:"fecha_registro" json:"fecha_registro"`
	Activo        bool          `bson:"activo"         json:"activo"`
}

// MonedaPorDefecto se usa cuando el registro no indica una.
const MonedaPorDefecto = "MXN"

// PeticionRegistro es el cuerpo de POST /auth/registro.
//
// El maximo de 72 en la contraseña no es capricho: bcrypt solo toma los
// primeros 72 bytes, asi que aceptar mas daria una falsa sensacion de
// seguridad.
type PeticionRegistro struct {
	Nombre   string `json:"nombre"   binding:"required,min=2,max=80"`
	Email    string `json:"email"    binding:"required,email,max=120"`
	Password string `json:"password" binding:"required,min=8,max=72"`
	Moneda   string `json:"moneda"   binding:"omitempty,len=3,alpha"`
}

// PeticionLogin es el cuerpo de POST /auth/login.
type PeticionLogin struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// PeticionRefresco es el cuerpo de POST /auth/refresh.
type PeticionRefresco struct {
	TokenRefresco string `json:"token_refresco" binding:"required"`
}

// PeticionActualizarPerfil es el cuerpo de PUT /auth/perfil.
// El email no se puede cambiar: es la credencial de acceso y la llave unica.
type PeticionActualizarPerfil struct {
	Nombre string `json:"nombre" binding:"required,min=2,max=80"`
	Moneda string `json:"moneda" binding:"required,len=3,alpha"`
}

// RespuestaSesion es lo que devuelven el registro y el login.
type RespuestaSesion struct {
	TokenAcceso   string  `json:"token_acceso"`
	TokenRefresco string  `json:"token_refresco"`
	ExpiraEn      int     `json:"expira_en"` // segundos de vida del token de acceso
	Usuario       Usuario `json:"usuario"`
}

// RespuestaRefresco es lo que devuelve POST /auth/refresh: solo se renueva el
// token de acceso, el de refresco sigue siendo valido hasta que expire.
type RespuestaRefresco struct {
	TokenAcceso string `json:"token_acceso"`
	ExpiraEn    int    `json:"expira_en"`
}
