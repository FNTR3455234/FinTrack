package rutas

import (
	"net/http"

	"github.com/gin-gonic/gin"
	archivosSwagger "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/handlers"
	"github.com/FNTR3455234/FinTrack/backend/internal/respuestas"
)

// Como se registran los grupos de rutas. Vive aparte de rutas.go para que ese
// archivo se lea de un vistazo como lo que es: el mapa de la API.

// crud son los cinco handlers que comparten los recursos con CRUD.
type crud struct {
	listar     gin.HandlerFunc
	obtener    gin.HandlerFunc
	crear      gin.HandlerFunc
	actualizar gin.HandlerFunc
	eliminar   gin.HandlerFunc
}

// recursoCRUD es lo que cumplen los grupos de handlers con las cinco
// operaciones de siempre.
type recursoCRUD interface {
	Listar(*gin.Context)
	Obtener(*gin.Context)
	Crear(*gin.Context)
	Actualizar(*gin.Context)
	Eliminar(*gin.Context)
}

func crudDe(r recursoCRUD) crud {
	return crud{listar: r.Listar, obtener: r.Obtener, crear: r.Crear, actualizar: r.Actualizar, eliminar: r.Eliminar}
}

// registrarCRUD engancha las cinco rutas de siempre sobre una ruta base.
// Cuentas, categorias, transacciones, presupuestos y metas tienen exactamente
// la misma forma, asi que registrarlas a mano cinco veces solo invita a que se
// desalineen.
func registrarCRUD(grupo *gin.RouterGroup, base string, h crud) {
	grupo.GET(base, h.listar)
	grupo.POST(base, h.crear)
	grupo.GET(base+"/:id", h.obtener)
	grupo.PUT(base+"/:id", h.actualizar)
	grupo.DELETE(base+"/:id", h.eliminar)
}

// registrarMetas engancha el CRUD de metas y su sub-recurso de aportaciones.
//
// Es el unico recurso que no se registra con registrarCRUD a secas: las
// aportaciones cuelgan de su meta y no existen por su cuenta, asi que su ruta
// lleva dentro el identificador de la meta a la que pertenecen.
func registrarMetas(grupo *gin.RouterGroup, h *handlers.Metas) {
	registrarCRUD(grupo, "/metas", crudDe(h))

	grupo.POST("/metas/:id/aportaciones", h.Aportar)
	grupo.DELETE("/metas/:id/aportaciones/:aportacion", h.QuitarAportacion)
}

// registrarReportes engancha las consultas de analisis.
//
// Van todas bajo /reportes y todas son de solo lectura: aqui no se escribe
// nada, se responden preguntas sobre lo que ya esta guardado.
//
// La tercera consulta relacional no esta aqui: vive en GET /metas, porque el
// progreso de una meta no es un reporte aparte, es la meta misma.
func registrarReportes(grupo *gin.RouterGroup, h *handlers.Reportes) {
	reportes := grupo.Group("/reportes")
	{
		// Las consultas relacionales 1 y 2.
		reportes.GET("/gastos-por-categoria", h.GastosPorCategoria)
		reportes.GET("/estado-presupuestos", h.EstadoPresupuestos)

		reportes.GET("/resumen", h.Resumen)
		reportes.GET("/tendencia", h.Tendencia)
		reportes.GET("/saldos", h.Saldos)
	}
}

// registrarSwagger publica la documentacion interactiva en /swagger/index.html.
//
// Va fuera de /api/v1 y sin autenticacion: describe la API, no expone datos.
// Aun asi, en un despliegue publico de verdad conviene dejarla solo en los
// entornos internos; queda anotado en backend/README.md.
func registrarSwagger(router *gin.Engine) {
	router.GET("/swagger/*any", ginSwagger.WrapHandler(archivosSwagger.Handler))

	// Atajo para no tener que escribir /index.html a mano.
	router.GET("/swagger", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})
}

// registrarErroresDeRuta hace que un 404 o un 405 respondan con el mismo
// formato JSON que el resto de la API, y no con el texto plano de Gin.
func registrarErroresDeRuta(router *gin.Engine) {
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
}
