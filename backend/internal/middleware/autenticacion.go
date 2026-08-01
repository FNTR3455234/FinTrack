package middleware

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"

	fintrackErrores "github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/respuestas"
	"github.com/FNTR3455234/FinTrack/backend/internal/servicios"
)

// ClaveUsuarioID es la llave con la que viaja el id del usuario autenticado
// dentro del contexto de Gin.
const ClaveUsuarioID = "usuario_id"

// ValidadorToken es lo unico que este middleware necesita saber de los tokens.
// La implementacion real es servicios.Tokens.
type ValidadorToken interface {
	ValidarAcceso(token string) (bson.ObjectID, error)
}

// Autenticacion exige un token de acceso valido y deja el id del usuario en el
// contexto.
//
// Este es el punto donde se decide de quien son los datos que va a ver la
// peticion. Los handlers toman el id de aqui con UsuarioID(c) y **nunca** del
// cuerpo ni de la query: si el id viniera del cliente, cualquiera podria pedir
// los datos de otro cambiando un numero.
func Autenticacion(validador ValidadorToken) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := tokenDelEncabezado(c)
		if err != nil {
			respuestas.Fallo(c, err)
			return
		}

		usuarioID, err := validador.ValidarAcceso(token)
		if err != nil {
			if errors.Is(err, servicios.ErrTokenVencido) {
				// Codigo propio para que el frontend sepa que aqui si vale la
				// pena intentar el refresco antes de mandar al login.
				respuestas.Fallo(c, fintrackErrores.NoAutorizado(
					fintrackErrores.CodigoTokenVencido, "Tu sesion expiro."))
				return
			}
			respuestas.Fallo(c, fintrackErrores.NoAutorizado(
				fintrackErrores.CodigoTokenInvalido, "El token no es valido."))
			return
		}

		c.Set(ClaveUsuarioID, usuarioID)
		c.Next()
	}
}

// tokenDelEncabezado saca el token de "Authorization: Bearer <token>".
func tokenDelEncabezado(c *gin.Context) (string, error) {
	encabezado := strings.TrimSpace(c.GetHeader("Authorization"))
	if encabezado == "" {
		return "", fintrackErrores.NoAutorizado(fintrackErrores.CodigoNoAutenticado,
			"Falta el encabezado Authorization.")
	}

	partes := strings.Fields(encabezado)
	if len(partes) != 2 || !strings.EqualFold(partes[0], "Bearer") {
		return "", fintrackErrores.NoAutorizado(fintrackErrores.CodigoNoAutenticado,
			"El encabezado Authorization debe tener la forma: Bearer <token>.")
	}
	return partes[1], nil
}

// UsuarioID devuelve el id del usuario autenticado que dejo el middleware.
//
// El segundo valor es false solo si se llama desde una ruta que no pasa por
// Autenticacion, que seria un error de programacion al armar las rutas.
func UsuarioID(c *gin.Context) (bson.ObjectID, bool) {
	valor, existe := c.Get(ClaveUsuarioID)
	if !existe {
		return bson.NilObjectID, false
	}
	id, ok := valor.(bson.ObjectID)
	return id, ok
}
