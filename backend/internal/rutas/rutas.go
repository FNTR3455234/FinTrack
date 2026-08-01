// Package rutas arma el router: engancha los middlewares y registra los
// endpoints. Es el mapa de la API en un solo archivo.
package rutas

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/FNTR3455234/FinTrack/backend/internal/config"
	"github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/handlers"
	"github.com/FNTR3455234/FinTrack/backend/internal/middleware"
	"github.com/FNTR3455234/FinTrack/backend/internal/respuestas"
)

// Dependencias son las piezas que los handlers necesitan. Se pasan explicitas
// en vez de usar variables globales, para poder armar el router con dobles de
// prueba en los tests.
type Dependencias struct {
	BD handlers.VerificadorBD
}

// Configurar devuelve el router listo para servir.
func Configurar(cfg *config.Config, deps Dependencias) *gin.Engine {
	gin.SetMode(cfg.GinModo)

	// gin.New() y no gin.Default(): Default trae su propio logger y su propio
	// recovery, y aqui se usan los propios.
	router := gin.New()

	// Sin esto, un POST a una ruta que solo acepta GET responde 404 en vez de
	// 405, que es lo correcto y lo que espera la coleccion de Postman.
	router.HandleMethodNotAllowed = true

	// El orden importa: el id de peticion se genera primero para que la
	// bitacora y la recuperacion ya lo tengan; la recuperacion envuelve a todo
	// lo que venga despues.
	router.Use(middleware.IDPeticion())
	router.Use(middleware.Bitacora())
	router.Use(middleware.Recuperacion())
	router.Use(middleware.CORS(cfg.CORSOrigenes))

	// Una ruta inexistente tambien responde con el formato uniforme, no con el
	// texto plano "404 page not found" de Gin.
	router.NoRoute(func(c *gin.Context) {
		respuestas.Fallo(c, errores.NoEncontrado(
			errores.CodigoRutaNoEncontrada,
			"La ruta "+c.Request.Method+" "+c.Request.URL.Path+" no existe.",
		))
	})
	router.NoMethod(func(c *gin.Context) {
		respuestas.Fallo(c, &errores.ErrorDominio{
			Codigo:  errores.CodigoMetodoNoPermitido,
			Mensaje: "El metodo " + c.Request.Method + " no esta permitido en esta ruta.",
			HTTP:    http.StatusMethodNotAllowed,
		})
	})

	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", handlers.Salud(deps.BD, config.Version))

		// Fase 3: /auth/registro, /auth/login, /auth/refresh, /auth/perfil
		// Fase 4: /cuentas, /categorias, /transacciones
		// Fase 5: /presupuestos, /reportes
	}

	return router
}
