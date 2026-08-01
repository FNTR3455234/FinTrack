package middleware

import (
	"log/slog"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/respuestas"
)

// Recuperacion atrapa los panics de cualquier handler.
//
// Sin esto, un panic (un indice fuera de rango, un puntero nulo) tumba la
// goroutine de esa peticion y el cliente recibe una conexion cortada, sin
// cuerpo. Con esto el servidor sigue vivo, queda la traza completa en la
// bitacora y el cliente recibe un 500 con el mismo formato que cualquier otro
// error de la API.
func Recuperacion() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if panico := recover(); panico != nil {
				slog.Error("panic recuperado",
					"id_peticion", c.GetString(ClaveIDPeticion),
					"ruta", c.Request.URL.Path,
					"panico", panico,
					"traza", string(debug.Stack()),
				)
				// Al cliente se le responde generico: la traza jamas sale.
				respuestas.Fallo(c, errores.Interno(nil))
			}
		}()
		c.Next()
	}
}
