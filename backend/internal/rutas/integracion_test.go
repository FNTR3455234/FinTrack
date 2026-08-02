package rutas

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FNTR3455234/FinTrack/backend/internal/config"
	"github.com/FNTR3455234/FinTrack/backend/internal/db"
	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
	"github.com/FNTR3455234/FinTrack/backend/internal/repositorios"
	"github.com/FNTR3455234/FinTrack/backend/internal/servicios"
)

// Pruebas de integracion de la API completa contra MongoDB de verdad: router,
// middlewares, handlers, servicios y repositorios, sin ningun doble.
//
// Se saltan solas si no esta MONGO_URI_PRUEBAS. Para correrlas:
//
//	make up
//	MONGO_URI_PRUEBAS="mongodb://fintrack_admin:fintrack_dev_2026@localhost:27017/?authSource=admin" \
//	  go test ./internal/rutas/...
const bdDeIntegracion = "fintrack_pruebas_api"

// api es un cliente minimo para hablar con el router en las pruebas.
type api struct {
	router *gin.Engine
	t      *testing.T
}

// routerReal arma la API completa apuntando a una base de pruebas que se borra
// al terminar.
func routerReal(t *testing.T) *api {
	t.Helper()

	uri := os.Getenv("MONGO_URI_PRUEBAS")
	if uri == "" {
		t.Skip("sin MONGO_URI_PRUEBAS: se omiten las pruebas de integracion")
	}

	ctx, cancelar := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelar()

	conexion, err := db.Conectar(ctx, uri, bdDeIntegracion)
	require.NoError(t, err)
	require.NoError(t, db.CrearIndices(ctx, conexion.BD))

	t.Cleanup(func() {
		limpieza, cancelar := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancelar()
		_ = conexion.BD.Drop(limpieza)
		_ = conexion.Cerrar(limpieza)
	})

	tokens := servicios.NuevoTokens(
		"secreto_de_acceso_para_pruebas_1234567890",
		"secreto_de_refresco_para_pruebas_09876543", 15, 7)

	repoCuentas := repositorios.NuevoCuentas(conexion.BD)
	repoCategorias := repositorios.NuevoCategorias(conexion.BD)
	repoTransacciones := repositorios.NuevoTransacciones(conexion.BD)
	repoPresupuestos := repositorios.NuevoPresupuestos(conexion.BD)
	repoReportes := repositorios.NuevoReportes(conexion.BD)

	cfg := &config.Config{GinModo: gin.TestMode, CORSOrigenes: []string{"http://localhost:5173"}}
	router := Configurar(cfg, Dependencias{
		BD:            conexion,
		Auth:          servicios.NuevoAuth(repositorios.NuevoUsuarios(conexion.BD), tokens),
		Validador:     tokens,
		Cuentas:       servicios.NuevoCuentas(repoCuentas, repoTransacciones),
		Categorias:    servicios.NuevoCategorias(repoCategorias, repoTransacciones, repoPresupuestos),
		Transacciones: servicios.NuevoTransacciones(repoTransacciones, repoCuentas, repoCategorias, repoReportes),
		Presupuestos:  servicios.NuevoPresupuestos(repoPresupuestos, repoCategorias),
		Reportes:      servicios.NuevoReportes(repoReportes),
		CSV:           servicios.NuevoCSV(repoTransacciones, repoCuentas, repoCategorias),
	})

	return &api{router: router, t: t}
}

// llamar hace una peticion y devuelve la respuesta cruda.
func (a *api) llamar(metodo, ruta, token string, cuerpo any) *httptest.ResponseRecorder {
	a.t.Helper()
	return pedir(a.router, metodo, ruta, token, cuerpo)
}

// datos hace la peticion, exige el codigo esperado y deserializa el campo datos.
func (a *api) datos(metodo, ruta, token string, cuerpo any, esperado int, destino any) {
	a.t.Helper()

	grabadora := a.llamar(metodo, ruta, token, cuerpo)
	require.Equal(a.t, esperado, grabadora.Code, "%s %s -> %s", metodo, ruta, grabadora.Body.String())

	if destino == nil {
		return
	}
	var sobre struct {
		Datos json.RawMessage `json:"datos"`
	}
	require.NoError(a.t, json.Unmarshal(grabadora.Body.Bytes(), &sobre))
	require.NoError(a.t, json.Unmarshal(sobre.Datos, destino))
}

// nuevoUsuario registra un usuario y devuelve su token de acceso.
func (a *api) nuevoUsuario(email string) string {
	a.t.Helper()
	return registrar(a.t, a.router, email).TokenAcceso
}

// crearCuenta da de alta una cuenta y devuelve su id.
func (a *api) crearCuenta(token, nombre string) string {
	a.t.Helper()

	saldo := 1000.0
	var cuenta modelos.Cuenta
	a.datos(http.MethodPost, "/api/v1/cuentas", token, modelos.PeticionCuenta{
		Nombre: nombre, Tipo: modelos.CuentaDebito, SaldoInicial: &saldo, Color: "#2563EB",
	}, http.StatusCreated, &cuenta)
	return cuenta.ID.Hex()
}

// crearCategoria da de alta una categoria y devuelve su id.
func (a *api) crearCategoria(token, nombre, tipo string) string {
	a.t.Helper()

	var categoria modelos.Categoria
	a.datos(http.MethodPost, "/api/v1/categorias", token, modelos.PeticionCategoria{
		Nombre: nombre, Tipo: tipo, Color: "#EA580C", Icono: "🛒",
	}, http.StatusCreated, &categoria)
	return categoria.ID.Hex()
}

// movimiento arma el cuerpo de una transaccion.
func movimiento(cuentaID, categoriaID, tipo string, monto float64, descripcion string, fecha time.Time) modelos.PeticionTransaccion {
	return modelos.PeticionTransaccion{
		CuentaID: cuentaID, CategoriaID: categoriaID, Tipo: tipo,
		Monto: monto, Descripcion: descripcion, Fecha: fecha,
	}
}

func fechaUTC(anio int, mes time.Month, dia int) time.Time {
	return time.Date(anio, mes, dia, 12, 0, 0, 0, time.UTC)
}

// --- CRUD completo ----------------------------------------------------------

func TestIntegracion_CRUDDeCuentas(t *testing.T) {
	a := routerReal(t)
	token := a.nuevoUsuario("cuentas@fintrack.mx")

	// Crear
	saldo := 18000.0
	var creada modelos.Cuenta
	a.datos(http.MethodPost, "/api/v1/cuentas", token, modelos.PeticionCuenta{
		Nombre: "BBVA Debito", Tipo: modelos.CuentaDebito, SaldoInicial: &saldo, Color: "#2563EB",
	}, http.StatusCreated, &creada)
	assert.Equal(t, "BBVA Debito", creada.Nombre)
	assert.Equal(t, 18000.0, creada.SaldoInicial)

	// Leer una
	var leida modelos.Cuenta
	a.datos(http.MethodGet, "/api/v1/cuentas/"+creada.ID.Hex(), token, nil, http.StatusOK, &leida)
	assert.Equal(t, creada.ID, leida.ID)

	// Listar
	var lista []modelos.Cuenta
	a.datos(http.MethodGet, "/api/v1/cuentas", token, nil, http.StatusOK, &lista)
	assert.Len(t, lista, 1)

	// Actualizar
	nuevoSaldo := 20000.0
	var actualizada modelos.Cuenta
	a.datos(http.MethodPut, "/api/v1/cuentas/"+creada.ID.Hex(), token, modelos.PeticionCuenta{
		Nombre: "BBVA Nomina", Tipo: modelos.CuentaDebito, SaldoInicial: &nuevoSaldo, Color: "#111111",
	}, http.StatusOK, &actualizada)
	assert.Equal(t, "BBVA Nomina", actualizada.Nombre)
	assert.Equal(t, 20000.0, actualizada.SaldoInicial)

	// Borrar
	a.datos(http.MethodDelete, "/api/v1/cuentas/"+creada.ID.Hex(), token, nil, http.StatusNoContent, nil)
	a.datos(http.MethodGet, "/api/v1/cuentas/"+creada.ID.Hex(), token, nil, http.StatusNotFound, nil)
}

func TestIntegracion_CRUDDeCategorias(t *testing.T) {
	a := routerReal(t)
	token := a.nuevoUsuario("categorias@fintrack.mx")

	var creada modelos.Categoria
	a.datos(http.MethodPost, "/api/v1/categorias", token, modelos.PeticionCategoria{
		Nombre: "Supermercado", Tipo: modelos.TipoGasto, Color: "#EA580C", Icono: "🛒",
	}, http.StatusCreated, &creada)

	a.crearCategoria(token, "Nomina", modelos.TipoIngreso)

	// El filtro por tipo deja fuera la de ingreso.
	var gastos []modelos.Categoria
	a.datos(http.MethodGet, "/api/v1/categorias?tipo=gasto", token, nil, http.StatusOK, &gastos)
	require.Len(t, gastos, 1)
	assert.Equal(t, "Supermercado", gastos[0].Nombre)

	var todas []modelos.Categoria
	a.datos(http.MethodGet, "/api/v1/categorias", token, nil, http.StatusOK, &todas)
	assert.Len(t, todas, 2)

	// Archivar y comprobar que desaparece del listado normal.
	var archivada modelos.Categoria
	a.datos(http.MethodPut, "/api/v1/categorias/"+creada.ID.Hex(), token, modelos.PeticionCategoria{
		Nombre: "Supermercado", Tipo: modelos.TipoGasto, Color: "#EA580C", Icono: "🛒", Archivada: true,
	}, http.StatusOK, &archivada)
	assert.True(t, archivada.Archivada)

	var visibles []modelos.Categoria
	a.datos(http.MethodGet, "/api/v1/categorias", token, nil, http.StatusOK, &visibles)
	assert.Len(t, visibles, 1)

	var conArchivadas []modelos.Categoria
	a.datos(http.MethodGet, "/api/v1/categorias?incluir_archivadas=true", token, nil, http.StatusOK, &conArchivadas)
	assert.Len(t, conArchivadas, 2)

	a.datos(http.MethodDelete, "/api/v1/categorias/"+creada.ID.Hex(), token, nil, http.StatusNoContent, nil)
}

func TestIntegracion_CRUDDeTransacciones(t *testing.T) {
	a := routerReal(t)
	token := a.nuevoUsuario("transacciones@fintrack.mx")
	cuentaID := a.crearCuenta(token, "Efectivo")
	categoriaID := a.crearCategoria(token, "Supermercado", modelos.TipoGasto)

	var creada modelos.Transaccion
	a.datos(http.MethodPost, "/api/v1/transacciones", token,
		movimiento(cuentaID, categoriaID, modelos.TipoGasto, 850.50, "Despensa", fechaUTC(2026, time.July, 3)),
		http.StatusCreated, &creada)
	assert.Equal(t, 850.50, creada.Monto)
	assert.False(t, creada.CreadoEn.IsZero())

	var leida modelos.Transaccion
	a.datos(http.MethodGet, "/api/v1/transacciones/"+creada.ID.Hex(), token, nil, http.StatusOK, &leida)
	assert.Equal(t, "Despensa", leida.Descripcion)

	var actualizada modelos.Transaccion
	a.datos(http.MethodPut, "/api/v1/transacciones/"+creada.ID.Hex(), token,
		movimiento(cuentaID, categoriaID, modelos.TipoGasto, 900, "Despensa grande", fechaUTC(2026, time.July, 4)),
		http.StatusOK, &actualizada)
	assert.Equal(t, 900.0, actualizada.Monto)
	assert.Equal(t, creada.CreadoEn.UTC(), actualizada.CreadoEn.UTC(), "creado_en no cambia")

	a.datos(http.MethodDelete, "/api/v1/transacciones/"+creada.ID.Hex(), token, nil, http.StatusNoContent, nil)
	a.datos(http.MethodGet, "/api/v1/transacciones/"+creada.ID.Hex(), token, nil, http.StatusNotFound, nil)
}

func TestIntegracion_LaBaseRechazaLoQueElEsquemaNoPermite(t *testing.T) {
	// El $jsonSchema de MongoDB es la segunda linea de defensa. Aqui se
	// comprueba que la primera (el validador de Go) no deja llegar nada malo.
	a := routerReal(t)
	token := a.nuevoUsuario("esquema@fintrack.mx")
	cuentaID := a.crearCuenta(token, "Efectivo")
	categoriaID := a.crearCategoria(token, "Supermercado", modelos.TipoGasto)

	casos := map[string]modelos.PeticionTransaccion{
		"monto en cero":    movimiento(cuentaID, categoriaID, modelos.TipoGasto, 0, "X", fechaUTC(2026, time.July, 1)),
		"monto negativo":   movimiento(cuentaID, categoriaID, modelos.TipoGasto, -50, "X", fechaUTC(2026, time.July, 1)),
		"sin descripcion":  movimiento(cuentaID, categoriaID, modelos.TipoGasto, 100, "", fechaUTC(2026, time.July, 1)),
		"tipo desconocido": movimiento(cuentaID, categoriaID, "transferencia", 100, "X", fechaUTC(2026, time.July, 1)),
	}

	for nombre, cuerpo := range casos {
		t.Run(nombre, func(t *testing.T) {
			a.datos(http.MethodPost, "/api/v1/transacciones", token, cuerpo, http.StatusBadRequest, nil)
		})
	}
}
