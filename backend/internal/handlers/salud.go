// Package handlers traduce entre HTTP y los servicios: lee la peticion, llama
// al servicio correspondiente y escribe la respuesta. No lleva reglas de
// negocio ni consultas a la base.
package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/FNTR3455234/FinTrack/backend/internal/respuestas"
)

// VerificadorBD es lo unico que /health necesita saber de la base de datos.
// Al depender de esta interfaz y no de *db.Conexion, el handler se puede probar
// con un doble que simule una base caida.
type VerificadorBD interface {
	Ping(ctx context.Context) error
}

// EstadoSalud es lo que devuelve GET /health.
type EstadoSalud struct {
	Estado  string    `json:"estado"`
	Mongo   string    `json:"mongo"`
	Version string    `json:"version"`
	Hora    time.Time `json:"hora"`
}

// Salud responde el estado del servicio y hace ping a MongoDB.
//
// Devuelve 200 si todo responde y 503 si la base no contesta, para que un
// orquestador (Docker, un balanceador) pueda decidir si mandarle trafico.
func Salud(bd VerificadorBD, version string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// El ping no puede colgar la peticion: dos segundos son de sobra para
		// una base sana y suficientes para detectar una caida.
		ctx, cancelar := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancelar()

		estado := EstadoSalud{
			Estado:  "ok",
			Mongo:   "ok",
			Version: version,
			Hora:    time.Now().UTC(),
		}

		if err := bd.Ping(ctx); err != nil {
			estado.Estado = "degradado"
			estado.Mongo = "sin_respuesta"
			c.JSON(http.StatusServiceUnavailable, respuestas.Sobre{Datos: estado})
			return
		}

		respuestas.OK(c, estado)
	}
}
