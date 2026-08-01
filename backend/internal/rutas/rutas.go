// Package rutas arma el router: engancha los middlewares y registra los
// endpoints. Es el mapa de la API en un solo archivo.
package rutas

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/FNTR3455234/FinTrack/backend/internal/config"
	"github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/handlers"
	"github.com/FNTR3455234/FinTrack/backend/internal/middleware"
	"github.com/FNTR3455234/FinTrack/backend/internal/respuestas"
)

// Limite de peticiones del grupo /auth: 20 por minuto y por IP.
//
// Es suficiente para alguien que se equivoca de contraseña varias veces o para
// el refresco automatico del frontend, y corta de inmediato un intento de
// probar contraseñas a fuerza bruta.
const (
	maximoPeticionesAuth = 20
	ventanaAuth          = time.Minute
)

// Dependencias son las piezas que los handlers necesitan. Se pasan explicitas
// en vez de usar variables globales, para poder armar el router con dobles de
// prueba en los tests.
type Dependencias struct {
	BD            handlers.VerificadorBD
	Auth          handlers.ServicioAuth
	Validador     middleware.ValidadorToken
	Cuentas       handlers.ServicioCuentas
	Categorias    handlers.ServicioCategorias
	Transacciones handlers.ServicioTransacciones
}

// Configurar devuelve el router listo para servir.
func Configurar(cfg *config.Config, deps Dependencias) *gin.Engine {
	gin.SetMode(cfg.GinModo)
	handlers.ConfigurarValidador()

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

	registrarErroresDeRuta(router)

	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", handlers.Salud(deps.BD, config.Version))

		auth := handlers.NuevoAuth(deps.Auth)
		// Publicas, pero con limite de peticiones por IP.
		publicas := v1.Group("/auth", middleware.LimitePeticiones(maximoPeticionesAuth, ventanaAuth))
		{
			publicas.POST("/registro", auth.Registro)
			publicas.POST("/login", auth.Login)
			publicas.POST("/refresh", auth.Refrescar)
		}

		// A partir de aqui hace falta un token de acceso valido.
		privadas := v1.Group("", middleware.Autenticacion(deps.Validador))
		{
			privadas.GET("/auth/perfil", auth.Perfil)
			privadas.PUT("/auth/perfil", auth.ActualizarPerfil)

			registrarCRUD(privadas, "/cuentas", crudDe(handlers.NuevoCuentas(deps.Cuentas)))
			registrarCRUD(privadas, "/categorias", crudDe(handlers.NuevoCategorias(deps.Categorias)))
			registrarCRUD(privadas, "/transacciones", crudDe(handlers.NuevoTransacciones(deps.Transacciones)))

			// Fase 5: /presupuestos, /reportes
		}
	}

	return router
}

// crud son los cinco handlers que comparten los tres recursos con CRUD.
type crud struct {
	listar     gin.HandlerFunc
	obtener    gin.HandlerFunc
	crear      gin.HandlerFunc
	actualizar gin.HandlerFunc
	eliminar   gin.HandlerFunc
}

// recursoCRUD es lo que cumplen los tres grupos de handlers de la fase 4.
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
// Cuentas, categorias y transacciones tienen exactamente la misma forma, asi
// que registrarlas a mano tres veces solo invita a que se desalineen.
func registrarCRUD(grupo *gin.RouterGroup, base string, h crud) {
	grupo.GET(base, h.listar)
	grupo.POST(base, h.crear)
	grupo.GET(base+"/:id", h.obtener)
	grupo.PUT(base+"/:id", h.actualizar)
	grupo.DELETE(base+"/:id", h.eliminar)
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
