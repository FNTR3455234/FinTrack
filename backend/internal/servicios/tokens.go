package servicios

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Tipos de token. Van dentro del propio token para que quede escrito que es
// cada uno, ademas de que cada tipo se firma con un secreto distinto.
const (
	TipoAcceso   = "acceso"
	TipoRefresco = "refresco"
)

const emisor = "fintrack"

// Afirmaciones son los datos que viajan dentro del token.
//
// Subject lleva el id del usuario: es el unico dato con el que se decide a que
// documentos puede llegar quien hace la peticion. Nunca se toma del cuerpo.
type Afirmaciones struct {
	Tipo string `json:"tipo"`
	jwt.RegisteredClaims
}

// Tokens emite y valida los JWT de la aplicacion.
//
// Los dos tipos usan secretos distintos: asi un token de refresco no puede
// pasar por uno de acceso ni aunque alguien se equivoque al validarlo.
type Tokens struct {
	secretoAcceso   []byte
	secretoRefresco []byte
	duracionAcceso  time.Duration
	duracionRefres  time.Duration
}

// NuevoTokens construye el emisor con la configuracion del servidor.
func NuevoTokens(secretoAcceso, secretoRefresco string, minutosAcceso, diasRefresco int) *Tokens {
	return &Tokens{
		secretoAcceso:   []byte(secretoAcceso),
		secretoRefresco: []byte(secretoRefresco),
		duracionAcceso:  time.Duration(minutosAcceso) * time.Minute,
		duracionRefres:  time.Duration(diasRefresco) * 24 * time.Hour,
	}
}

// SegundosDeAcceso es la vida del token de acceso, para informarsela al cliente.
func (t *Tokens) SegundosDeAcceso() int { return int(t.duracionAcceso.Seconds()) }

// GenerarAcceso emite el token corto que acompaña a cada peticion.
func (t *Tokens) GenerarAcceso(usuarioID bson.ObjectID) (string, error) {
	return t.generar(usuarioID, TipoAcceso, t.secretoAcceso, t.duracionAcceso)
}

// GenerarRefresco emite el token largo que sirve para pedir uno de acceso nuevo.
func (t *Tokens) GenerarRefresco(usuarioID bson.ObjectID) (string, error) {
	return t.generar(usuarioID, TipoRefresco, t.secretoRefresco, t.duracionRefres)
}

// ValidarAcceso comprueba la firma y la vigencia de un token de acceso.
func (t *Tokens) ValidarAcceso(token string) (bson.ObjectID, error) {
	return t.validar(token, TipoAcceso, t.secretoAcceso)
}

// ValidarRefresco hace lo mismo con uno de refresco.
func (t *Tokens) ValidarRefresco(token string) (bson.ObjectID, error) {
	return t.validar(token, TipoRefresco, t.secretoRefresco)
}

func (t *Tokens) generar(usuarioID bson.ObjectID, tipo string, secreto []byte, duracion time.Duration) (string, error) {
	ahora := time.Now()
	afirmaciones := Afirmaciones{
		Tipo: tipo,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   usuarioID.Hex(),
			Issuer:    emisor,
			IssuedAt:  jwt.NewNumericDate(ahora),
			ExpiresAt: jwt.NewNumericDate(ahora.Add(duracion)),
		},
	}

	firmado, err := jwt.NewWithClaims(jwt.SigningMethodHS256, afirmaciones).SignedString(secreto)
	if err != nil {
		return "", fmt.Errorf("al firmar el token de %s: %w", tipo, err)
	}
	return firmado, nil
}

// ErrTokenVencido y ErrTokenInvalido dejan que el middleware distinga entre
// "tu sesion expiro, refresca" y "este token no sirve, vuelve a entrar".
var (
	ErrTokenVencido  = errors.New("el token expiro")
	ErrTokenInvalido = errors.New("el token no es valido")
)

func (t *Tokens) validar(token, tipoEsperado string, secreto []byte) (bson.ObjectID, error) {
	var afirmaciones Afirmaciones

	_, err := jwt.ParseWithClaims(token, &afirmaciones, func(t *jwt.Token) (any, error) {
		// Se comprueba el algoritmo: sin esto, un atacante podria mandar un
		// token firmado con "none" o con un algoritmo asimetrico.
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("algoritmo inesperado: %v", t.Header["alg"])
		}
		return secreto, nil
	}, jwt.WithIssuer(emisor), jwt.WithExpirationRequired())

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return bson.NilObjectID, ErrTokenVencido
		}
		return bson.NilObjectID, ErrTokenInvalido
	}

	if afirmaciones.Tipo != tipoEsperado {
		return bson.NilObjectID, ErrTokenInvalido
	}

	usuarioID, err := bson.ObjectIDFromHex(afirmaciones.Subject)
	if err != nil {
		return bson.NilObjectID, ErrTokenInvalido
	}
	return usuarioID, nil
}
