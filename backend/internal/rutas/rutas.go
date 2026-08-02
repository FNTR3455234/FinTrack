// Package rutas arma el router: engancha los middlewares y registra los
// endpoints. Este archivo es el mapa de la API; el como se registra cada grupo
// esta en registro.go.
package rutas

import (
	"time"

	"github.com/gin-gonic/gin"

	// La especificacion OpenAPI la genera swaggo a partir de las anotaciones de
	// los handlers (`make swagger`). Se importa solo por su init(), que la
	// registra para que la sirva /swagger.
	_ "github.com/FNTR3455234/FinTrack/backend/docs"
	"github.com/FNTR3455234/FinTrack/backend/internal/config"
	"github.com/FNTR3455234/FinTrack/backend/internal/handlers"
	"github.com/FNTR3455234/FinTrack/backend/internal/middleware"
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
	Presupuestos  handlers.ServicioPresupuestos
	Reportes      handlers.ServicioReportes
	CSV           handlers.ServicioCSV
	Metas         handlers.ServicioMetas
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

	registrarSwagger(router)
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
			registrarCRUD(privadas, "/presupuestos", crudDe(handlers.NuevoPresupuestos(deps.Presupuestos)))

			// Conviven con /transacciones/:id: Gin prefiere el tramo estatico
			// sobre el parametro, asi que "exportar" nunca se toma por un id.
			archivos := handlers.NuevoCSV(deps.CSV)
			privadas.GET("/transacciones/exportar", archivos.Exportar)
			privadas.POST("/transacciones/importar", archivos.Importar)

			registrarMetas(privadas, handlers.NuevoMetas(deps.Metas))

			registrarReportes(privadas, handlers.NuevoReportes(deps.Reportes))
		}
	}

	return router
}
