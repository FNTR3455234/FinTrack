// Package middleware contiene los middlewares propios de FinTrack.
package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ClaveIDPeticion es la llave con la que se guarda el identificador de la
// peticion en el contexto de Gin.
const ClaveIDPeticion = "id_peticion"

// CabeceraIDPeticion es el nombre del encabezado HTTP donde viaja.
const CabeceraIDPeticion = "X-Request-ID"

// IDPeticion asigna un identificador unico a cada peticion y lo devuelve en la
// respuesta. Sirve para seguir una peticion completa en la bitacora: cuando un
// usuario reporta un error, con ese id se encuentran todas sus lineas.
//
// Si el cliente ya mando uno (por ejemplo un proxy), se respeta.
func IDPeticion() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(CabeceraIDPeticion)
		if id == "" {
			id = generarID()
		}
		c.Set(ClaveIDPeticion, id)
		c.Header(CabeceraIDPeticion, id)
		c.Next()
	}
}

// generarID devuelve 16 caracteres hexadecimales aleatorios.
func generarID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// rand.Read practicamente no falla; si lo hace, un id basado en el
		// reloj es suficiente para no dejar la peticion sin identificar.
		return hex.EncodeToString([]byte(time.Now().Format("150405.000000")))
	}
	return hex.EncodeToString(bytes)
}

// Bitacora registra cada peticion con log/slog: metodo, ruta, estado, latencia
// e identificador. Reemplaza al gin.Logger() por defecto, que escribe texto sin
// estructura y no sirve para filtrar despues.
func Bitacora() gin.HandlerFunc {
	return func(c *gin.Context) {
		inicio := time.Now()
		ruta := c.Request.URL.Path
		consulta := c.Request.URL.RawQuery

		c.Next()

		estado := c.Writer.Status()
		atributos := []any{
			"id_peticion", c.GetString(ClaveIDPeticion),
			"metodo", c.Request.Method,
			"ruta", ruta,
			"estado", estado,
			"latencia_ms", time.Since(inicio).Milliseconds(),
			"ip", c.ClientIP(),
		}
		if consulta != "" {
			atributos = append(atributos, "consulta", consulta)
		}

		// El nivel depende del resultado: un 500 no puede quedar como una
		// linea informativa mas.
		switch {
		case estado >= http.StatusInternalServerError:
			slog.Error("peticion", atributos...)
		case estado >= http.StatusBadRequest:
			slog.Warn("peticion", atributos...)
		default:
			slog.Info("peticion", atributos...)
		}
	}
}
