package rutas

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// presupuestar da de alta un limite y devuelve el presupuesto creado.
func (a *api) presupuestar(token, categoriaID string, limite float64, mes, anio int) modelos.Presupuesto {
	a.t.Helper()

	var presupuesto modelos.Presupuesto
	a.datos(http.MethodPost, "/api/v1/presupuestos", token, modelos.PeticionPresupuesto{
		CategoriaID: categoriaID, MontoLimite: limite, Mes: mes, Anio: anio,
	}, http.StatusCreated, &presupuesto)
	return presupuesto
}

func TestIntegracion_CRUDDePresupuestos(t *testing.T) {
	a := routerReal(t)
	token := a.nuevoUsuario("presupuestos@fintrack.mx")
	categoriaID := a.crearCategoria(token, "Supermercado", modelos.TipoGasto)

	creado := a.presupuestar(token, categoriaID, 4000, 7, 2026)
	assert.Equal(t, 4000.0, creado.MontoLimite)
	assert.Equal(t, modelos.Periodo{Mes: 7, Anio: 2026}, creado.Periodo())

	var leido modelos.Presupuesto
	a.datos(http.MethodGet, "/api/v1/presupuestos/"+creado.ID.Hex(), token, nil, http.StatusOK, &leido)
	assert.Equal(t, creado.ID, leido.ID)

	var actualizado modelos.Presupuesto
	a.datos(http.MethodPut, "/api/v1/presupuestos/"+creado.ID.Hex(), token, modelos.PeticionPresupuesto{
		CategoriaID: categoriaID, MontoLimite: 4500, Mes: 7, Anio: 2026,
	}, http.StatusOK, &actualizado)
	assert.Equal(t, 4500.0, actualizado.MontoLimite)

	a.datos(http.MethodDelete, "/api/v1/presupuestos/"+creado.ID.Hex(), token, nil, http.StatusNoContent, nil)
	a.datos(http.MethodGet, "/api/v1/presupuestos/"+creado.ID.Hex(), token, nil, http.StatusNotFound, nil)
}

func TestIntegracion_PresupuestosFiltraPorPeriodo(t *testing.T) {
	a := routerReal(t)
	token := a.nuevoUsuario("periodos@fintrack.mx")
	categoriaID := a.crearCategoria(token, "Supermercado", modelos.TipoGasto)

	a.presupuestar(token, categoriaID, 4000, 7, 2026)
	a.presupuestar(token, categoriaID, 4200, 8, 2026)

	var todos []modelos.Presupuesto
	a.datos(http.MethodGet, "/api/v1/presupuestos", token, nil, http.StatusOK, &todos)
	assert.Len(t, todos, 2, "sin periodo se devuelven todos")

	var deAgosto []modelos.Presupuesto
	a.datos(http.MethodGet, "/api/v1/presupuestos?mes=8&anio=2026", token, nil, http.StatusOK, &deAgosto)
	require.Len(t, deAgosto, 1)
	assert.Equal(t, 4200.0, deAgosto[0].MontoLimite)

	a.datos(http.MethodGet, "/api/v1/presupuestos?mes=13&anio=2026", token, nil, http.StatusBadRequest, nil)
}

func TestIntegracion_ElIndiceUnicoImpideDosPresupuestosDelMismoMes(t *testing.T) {
	// Esto lo decide MongoDB, no el codigo: no se consulta antes de insertar,
	// porque entre la consulta y el insert cabe otra peticion.
	a := routerReal(t)
	token := a.nuevoUsuario("duplicado@fintrack.mx")
	categoriaID := a.crearCategoria(token, "Supermercado", modelos.TipoGasto)

	a.presupuestar(token, categoriaID, 4000, 7, 2026)

	grabadora := a.llamar(http.MethodPost, "/api/v1/presupuestos", token, modelos.PeticionPresupuesto{
		CategoriaID: categoriaID, MontoLimite: 9999, Mes: 7, Anio: 2026,
	})

	assert.Equal(t, http.StatusConflict, grabadora.Code)
	assert.Contains(t, grabadora.Body.String(), "PRESUPUESTO_DUPLICADO")
}

func TestIntegracion_NoSePresupuestaUnaCategoriaDeIngreso(t *testing.T) {
	a := routerReal(t)
	token := a.nuevoUsuario("ingreso@fintrack.mx")
	categoriaID := a.crearCategoria(token, "Nomina", modelos.TipoIngreso)

	grabadora := a.llamar(http.MethodPost, "/api/v1/presupuestos", token, modelos.PeticionPresupuesto{
		CategoriaID: categoriaID, MontoLimite: 4000, Mes: 7, Anio: 2026,
	})

	assert.Equal(t, http.StatusBadRequest, grabadora.Code)
	assert.Contains(t, grabadora.Body.String(), "TIPO_NO_COINCIDE")
}

func TestIntegracion_NoSeBorraUnaCategoriaPresupuestada(t *testing.T) {
	a := routerReal(t)
	token := a.nuevoUsuario("categoria-presupuestada@fintrack.mx")
	categoriaID := a.crearCategoria(token, "Supermercado", modelos.TipoGasto)

	a.presupuestar(token, categoriaID, 4000, 7, 2026)

	grabadora := a.llamar(http.MethodDelete, "/api/v1/categorias/"+categoriaID, token, nil)

	assert.Equal(t, http.StatusConflict, grabadora.Code)
	assert.Contains(t, grabadora.Body.String(), "CATEGORIA_CON_PRESUPUESTOS")
}

func TestIntegracion_LosPresupuestosNoSeCruzanEntreUsuarios(t *testing.T) {
	a := routerReal(t)
	ana := a.nuevoUsuario("ana-presupuestos@fintrack.mx")
	beto := a.nuevoUsuario("beto-presupuestos@fintrack.mx")

	deAna := a.presupuestar(ana, a.crearCategoria(ana, "Supermercado", modelos.TipoGasto), 4000, 7, 2026)
	a.presupuestar(beto, a.crearCategoria(beto, "Supermercado", modelos.TipoGasto), 1500, 7, 2026)

	// Beto solo ve el suyo.
	var deBeto []modelos.Presupuesto
	a.datos(http.MethodGet, "/api/v1/presupuestos", beto, nil, http.StatusOK, &deBeto)
	require.Len(t, deBeto, 1)
	assert.Equal(t, 1500.0, deBeto[0].MontoLimite)

	// Y con el id de Ana en la mano no llega a ninguna parte: 404 y no 403,
	// porque un 403 confirmaria que el presupuesto existe.
	ruta := "/api/v1/presupuestos/" + deAna.ID.Hex()
	a.datos(http.MethodGet, ruta, beto, nil, http.StatusNotFound, nil)
	a.datos(http.MethodDelete, ruta, beto, nil, http.StatusNotFound, nil)

	// El de Ana sigue intacto.
	var intacto modelos.Presupuesto
	a.datos(http.MethodGet, ruta, ana, nil, http.StatusOK, &intacto)
	assert.Equal(t, 4000.0, intacto.MontoLimite)
}

// --- alertas al registrar un gasto ------------------------------------------

func TestIntegracion_AlertaAlRegistrarUnGasto(t *testing.T) {
	a := routerReal(t)
	token := a.nuevoUsuario("alertas@fintrack.mx")
	cuentaID := a.crearCuenta(token, "BBVA Debito")
	categoriaID := a.crearCategoria(token, "Supermercado", modelos.TipoGasto)
	a.presupuestar(token, categoriaID, 4000, 7, 2026)

	gastar := func(monto float64) modelos.TransaccionCreada {
		var creada modelos.TransaccionCreada
		a.datos(http.MethodPost, "/api/v1/transacciones", token,
			movimiento(cuentaID, categoriaID, modelos.TipoGasto, monto,
				fmt.Sprintf("Despensa de %.0f", monto), fechaUTC(2026, time.July, 5)),
			http.StatusCreated, &creada)
		return creada
	}

	// 1000 de 4000: 25 %, todavia no hay nada que avisar.
	assert.Nil(t, gastar(1000).Alerta)

	// Acumulado 3200: justo el 80 %, el umbral entra.
	enAlerta := gastar(2200)
	require.NotNil(t, enAlerta.Alerta)
	assert.Equal(t, modelos.EstadoAlerta, enAlerta.Alerta.Estado)
	assert.Equal(t, 3200.0, enAlerta.Alerta.Gastado, "suma el mes entero, no solo este gasto")
	assert.Equal(t, 80.0, enAlerta.Alerta.PorcentajeUsado)
	assert.Equal(t, 800.0, enAlerta.Alerta.Disponible)
	assert.Contains(t, enAlerta.Alerta.Mensaje, "Supermercado")

	// Acumulado 4300: se paso.
	excedido := gastar(1100)
	require.NotNil(t, excedido.Alerta)
	assert.Equal(t, modelos.EstadoExcedido, excedido.Alerta.Estado)
	assert.Equal(t, 4300.0, excedido.Alerta.Gastado)
	assert.Equal(t, -300.0, excedido.Alerta.Disponible)
	assert.Equal(t, 107.5, excedido.Alerta.PorcentajeUsado)
}

func TestIntegracion_SinPresupuestoNoHayAlerta(t *testing.T) {
	a := routerReal(t)
	token := a.nuevoUsuario("sin-alerta@fintrack.mx")
	cuentaID := a.crearCuenta(token, "Efectivo")
	categoriaID := a.crearCategoria(token, "Supermercado", modelos.TipoGasto)

	var creada modelos.TransaccionCreada
	a.datos(http.MethodPost, "/api/v1/transacciones", token,
		movimiento(cuentaID, categoriaID, modelos.TipoGasto, 99999, "Sin techo", fechaUTC(2026, time.July, 5)),
		http.StatusCreated, &creada)

	assert.Nil(t, creada.Alerta)
}

func TestIntegracion_LaAlertaMiraElMesDelMovimiento(t *testing.T) {
	a := routerReal(t)
	token := a.nuevoUsuario("alerta-mes@fintrack.mx")
	cuentaID := a.crearCuenta(token, "Efectivo")
	categoriaID := a.crearCategoria(token, "Supermercado", modelos.TipoGasto)
	a.presupuestar(token, categoriaID, 4000, 7, 2026)

	// Un gasto grande, pero de agosto: julio no se entera y agosto no tiene techo.
	var deAgosto modelos.TransaccionCreada
	a.datos(http.MethodPost, "/api/v1/transacciones", token,
		movimiento(cuentaID, categoriaID, modelos.TipoGasto, 5000, "De agosto", fechaUTC(2026, time.August, 2)),
		http.StatusCreated, &deAgosto)
	assert.Nil(t, deAgosto.Alerta)

	// Y el presupuesto de julio sigue en cero.
	var estados []modelos.EstadoPresupuesto
	a.datos(http.MethodGet, "/api/v1/reportes/estado-presupuestos?mes=7&anio=2026", token, nil, http.StatusOK, &estados)
	require.Len(t, estados, 1)
	assert.Zero(t, estados[0].Gastado)
}
