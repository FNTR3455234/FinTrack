package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS permite que el frontend, que corre en otro origen (Vite en el 5173),
// pueda llamar a la API.
//
// Se escribe a mano en vez de usar gin-contrib/cors por dos razones: son treinta
// lineas que se entienden completas, y evita una dependencia mas.
//
// La lista de origenes permitidos viene de la variable CORS_ORIGENES. Se acepta
// "*" para desarrollo, pero entonces no se permiten credenciales, porque el
// navegador rechaza esa combinacion.
func CORS(origenesPermitidos []string) gin.HandlerFunc {
	permitidos := make(map[string]bool, len(origenesPermitidos))
	comodin := false
	for _, o := range origenesPermitidos {
		if o == "*" {
			comodin = true
		}
		permitidos[o] = true
	}

	metodos := strings.Join([]string{
		http.MethodGet, http.MethodPost, http.MethodPut,
		http.MethodDelete, http.MethodOptions,
	}, ", ")

	cabeceras := strings.Join([]string{
		"Authorization", "Content-Type", "Accept", CabeceraIDPeticion,
	}, ", ")

	return func(c *gin.Context) {
		origen := c.GetHeader("Origin")

		// Sin encabezado Origin no es una peticion entre origenes (curl,
		// Postman, el propio servidor): no hay nada que autorizar.
		if origen != "" && (comodin || permitidos[origen]) {
			c.Header("Access-Control-Allow-Origin", origen)
			c.Header("Access-Control-Allow-Methods", metodos)
			c.Header("Access-Control-Allow-Headers", cabeceras)
			c.Header("Access-Control-Expose-Headers", CabeceraIDPeticion)
			if !comodin {
				c.Header("Access-Control-Allow-Credentials", "true")
			}
			// Le dice a las caches que la respuesta cambia segun el origen.
			c.Header("Vary", "Origin")
			// El navegador puede guardar el resultado del preflight 10 minutos.
			c.Header("Access-Control-Max-Age", "600")
		}

		// El preflight se contesta aqui mismo: no tiene que llegar al handler.
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
