package rutas

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
	"github.com/FNTR3455234/FinTrack/backend/internal/respuestas"
)

// --- Filtros, orden y paginacion contra datos reales ------------------------

// escenarioDeFiltros deja 5 movimientos conocidos para probar los filtros.
func escenarioDeFiltros(t *testing.T) (*api, string) {
	t.Helper()

	a := routerReal(t)
	token := a.nuevoUsuario("filtros@fintrack.mx")
	efectivo := a.crearCuenta(token, "Efectivo")
	banco := a.crearCuenta(token, "Banco")
	super := a.crearCategoria(token, "Supermercado", modelos.TipoGasto)
	nomina := a.crearCategoria(token, "Nomina", modelos.TipoIngreso)

	movimientos := []modelos.PeticionTransaccion{
		movimiento(efectivo, super, modelos.TipoGasto, 100, "Despensa de enero", fechaUTC(2026, time.January, 10)),
		movimiento(efectivo, super, modelos.TipoGasto, 250, "Despensa de junio", fechaUTC(2026, time.June, 15)),
		movimiento(banco, super, modelos.TipoGasto, 900, "Despensa grande de julio", fechaUTC(2026, time.July, 20)),
		movimiento(banco, nomina, modelos.TipoIngreso, 12500, "Nomina de julio", fechaUTC(2026, time.July, 15)),
		movimiento(banco, nomina, modelos.TipoIngreso, 3500, "Bono de julio", fechaUTC(2026, time.July, 25)),
	}
	for _, m := range movimientos {
		a.datos(http.MethodPost, "/api/v1/transacciones", token, m, http.StatusCreated, nil)
	}
	return a, token
}

// listar devuelve las transacciones y la meta de paginacion de una consulta.
func (a *api) listar(token, consulta string) ([]modelos.Transaccion, respuestas.Meta) {
	a.t.Helper()

	grabadora := a.llamar(http.MethodGet, "/api/v1/transacciones?"+consulta, token, nil)
	require.Equal(a.t, http.StatusOK, grabadora.Code, grabadora.Body.String())

	var sobre struct {
		Datos []modelos.Transaccion `json:"datos"`
		Meta  respuestas.Meta       `json:"meta"`
	}
	require.NoError(a.t, json.Unmarshal(grabadora.Body.Bytes(), &sobre))
	return sobre.Datos, sobre.Meta
}

func TestIntegracion_FiltrosDelListadoDeTransacciones(t *testing.T) {
	a, token := escenarioDeFiltros(t)

	t.Run("sin filtros devuelve todo, de la mas reciente a la mas vieja", func(t *testing.T) {
		transacciones, meta := a.listar(token, "")

		require.Len(t, transacciones, 5)
		assert.Equal(t, int64(5), meta.Total)
		assert.Equal(t, "Bono de julio", transacciones[0].Descripcion)
		assert.Equal(t, "Despensa de enero", transacciones[4].Descripcion)
	})

	t.Run("rango de fechas", func(t *testing.T) {
		transacciones, meta := a.listar(token, "desde=2026-07-01&hasta=2026-07-31")

		assert.Equal(t, int64(3), meta.Total)
		assert.Len(t, transacciones, 3)
	})

	t.Run("hasta incluye el dia completo", func(t *testing.T) {
		// El movimiento del 20 de julio es a las 12:00. Si "hasta" cortara a
		// las 00:00, no saldria.
		_, meta := a.listar(token, "desde=2026-07-20&hasta=2026-07-20")

		assert.Equal(t, int64(1), meta.Total)
	})

	t.Run("por tipo", func(t *testing.T) {
		_, gastos := a.listar(token, "tipo=gasto")
		_, ingresos := a.listar(token, "tipo=ingreso")

		assert.Equal(t, int64(3), gastos.Total)
		assert.Equal(t, int64(2), ingresos.Total)
	})

	t.Run("busqueda por texto en la descripcion", func(t *testing.T) {
		_, meta := a.listar(token, "busqueda=despensa")

		assert.Equal(t, int64(3), meta.Total, "la busqueda no distingue mayusculas")
	})

	t.Run("la busqueda escapa los caracteres de expresion regular", func(t *testing.T) {
		// Sin QuoteMeta, ".*" traeria las cinco.
		_, meta := a.listar(token, "busqueda=.*")

		assert.Equal(t, int64(0), meta.Total)
	})

	t.Run("orden por monto", func(t *testing.T) {
		descendente, _ := a.listar(token, "orden=monto_desc")
		ascendente, _ := a.listar(token, "orden=monto_asc")

		assert.Equal(t, 12500.0, descendente[0].Monto)
		assert.Equal(t, 100.0, ascendente[0].Monto)
	})

	t.Run("orden por fecha ascendente", func(t *testing.T) {
		transacciones, _ := a.listar(token, "orden=fecha_asc")

		assert.Equal(t, "Despensa de enero", transacciones[0].Descripcion)
	})

	t.Run("filtros combinados", func(t *testing.T) {
		_, meta := a.listar(token, "desde=2026-07-01&hasta=2026-07-31&tipo=ingreso&busqueda=julio")

		assert.Equal(t, int64(2), meta.Total)
	})
}

func TestIntegracion_PaginacionDelListado(t *testing.T) {
	a, token := escenarioDeFiltros(t)

	primera, meta := a.listar(token, "limite=2&pagina=1")
	segunda, _ := a.listar(token, "limite=2&pagina=2")
	tercera, _ := a.listar(token, "limite=2&pagina=3")
	cuarta, _ := a.listar(token, "limite=2&pagina=4")

	assert.Equal(t, int64(5), meta.Total)
	assert.Equal(t, 3, meta.TotalPaginas, "5 entre 2 son 3 paginas")
	assert.Equal(t, 1, meta.Pagina)
	assert.Equal(t, 2, meta.Limite)

	assert.Len(t, primera, 2)
	assert.Len(t, segunda, 2)
	assert.Len(t, tercera, 1)
	assert.Empty(t, cuarta, "mas alla de la ultima pagina no hay nada")

	// Ninguna transaccion se repite ni se pierde entre paginas.
	vistas := map[string]bool{}
	for _, pagina := range [][]modelos.Transaccion{primera, segunda, tercera} {
		for _, t := range pagina {
			vistas[t.ID.Hex()] = true
		}
	}
	assert.Len(t, vistas, 5)
}

func TestIntegracion_NoSePuedeBorrarUnaCuentaConMovimientos(t *testing.T) {
	a := routerReal(t)
	token := a.nuevoUsuario("borrado@fintrack.mx")
	cuentaID := a.crearCuenta(token, "Con movimientos")
	categoriaID := a.crearCategoria(token, "Supermercado", modelos.TipoGasto)

	a.datos(http.MethodPost, "/api/v1/transacciones", token,
		movimiento(cuentaID, categoriaID, modelos.TipoGasto, 100, "Algo", fechaUTC(2026, time.July, 1)),
		http.StatusCreated, nil)

	grabadora := a.llamar(http.MethodDelete, "/api/v1/cuentas/"+cuentaID, token, nil)

	require.Equal(t, http.StatusConflict, grabadora.Code)
	var cuerpo respuestas.SobreError
	require.NoError(t, json.Unmarshal(grabadora.Body.Bytes(), &cuerpo))
	assert.Equal(t, errores.CodigoCuentaConTransacciones, cuerpo.Error.Codigo)

	// Lo mismo con la categoria.
	grabadora = a.llamar(http.MethodDelete, "/api/v1/categorias/"+categoriaID, token, nil)
	require.Equal(t, http.StatusConflict, grabadora.Code)
}

func TestIntegracion_TodasLasRutasDelCRUDExigenToken(t *testing.T) {
	a := routerReal(t)
	id := "650000000000000000000001"

	rutas := []struct{ metodo, ruta string }{
		{http.MethodGet, "/api/v1/cuentas"},
		{http.MethodPost, "/api/v1/cuentas"},
		{http.MethodGet, "/api/v1/cuentas/" + id},
		{http.MethodPut, "/api/v1/cuentas/" + id},
		{http.MethodDelete, "/api/v1/cuentas/" + id},
		{http.MethodGet, "/api/v1/categorias"},
		{http.MethodPost, "/api/v1/categorias"},
		{http.MethodGet, "/api/v1/categorias/" + id},
		{http.MethodPut, "/api/v1/categorias/" + id},
		{http.MethodDelete, "/api/v1/categorias/" + id},
		{http.MethodGet, "/api/v1/transacciones"},
		{http.MethodPost, "/api/v1/transacciones"},
		{http.MethodGet, "/api/v1/transacciones/" + id},
		{http.MethodPut, "/api/v1/transacciones/" + id},
		{http.MethodDelete, "/api/v1/transacciones/" + id},
	}

	for _, r := range rutas {
		t.Run(r.metodo+" "+r.ruta, func(t *testing.T) {
			grabadora := a.llamar(r.metodo, r.ruta, "", nil)
			assert.Equal(t, http.StatusUnauthorized, grabadora.Code)
		})
	}
}
