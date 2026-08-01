package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// parsear corre filtroDeTransacciones sobre una query dada.
func parsear(consulta string) (modelos.FiltroTransacciones, bool, *httptest.ResponseRecorder) {
	grabadora := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(grabadora)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/transacciones?"+consulta, nil)

	filtro, ok := filtroDeTransacciones(c)
	return filtro, ok, grabadora
}

func TestFiltro_SinParametrosUsaLosValoresPorDefecto(t *testing.T) {
	filtro, ok, _ := parsear("")

	require.True(t, ok)
	assert.Equal(t, 1, filtro.Pagina)
	assert.Equal(t, modelos.LimitePorDefecto, filtro.Limite)
	assert.Nil(t, filtro.Desde)
	assert.Nil(t, filtro.Hasta)
	assert.Empty(t, filtro.Tipo)
}

func TestFiltro_LeeTodosLosCriterios(t *testing.T) {
	categoriaID := bson.NewObjectID()
	cuentaID := bson.NewObjectID()

	// Los %20 son espacios: la busqueda llega con ellos y el filtro los recorta.
	filtro, ok, _ := parsear("desde=2026-07-01&hasta=2026-07-31&tipo=gasto&busqueda=%20%20despensa%20" +
		"&pagina=3&limite=50&orden=monto_desc" +
		"&categoria_id=" + categoriaID.Hex() + "&cuenta_id=" + cuentaID.Hex())

	require.True(t, ok)
	assert.Equal(t, modelos.TipoGasto, filtro.Tipo)
	assert.Equal(t, "despensa", filtro.Busqueda, "se recortan los espacios")
	assert.Equal(t, 3, filtro.Pagina)
	assert.Equal(t, 50, filtro.Limite)
	assert.Equal(t, modelos.OrdenMontoDesc, filtro.Orden)
	require.NotNil(t, filtro.CategoriaID)
	assert.Equal(t, categoriaID, *filtro.CategoriaID)
	require.NotNil(t, filtro.CuentaID)
	assert.Equal(t, cuentaID, *filtro.CuentaID)
}

func TestFiltro_HastaIncluyeElDiaCompleto(t *testing.T) {
	// El error clasico de los rangos: "hasta=2026-07-31" tiene que incluir los
	// movimientos de ese dia, no cortar a las 00:00.
	filtro, ok, _ := parsear("hasta=2026-07-31")

	require.True(t, ok)
	require.NotNil(t, filtro.Hasta)
	assert.Equal(t, 2026, filtro.Hasta.Year())
	assert.Equal(t, 31, filtro.Hasta.Day())
	assert.Equal(t, 23, filtro.Hasta.Hour())
	assert.Equal(t, 59, filtro.Hasta.Minute())
}

func TestFiltro_DesdeEmpiezaAlPrincipioDelDia(t *testing.T) {
	filtro, ok, _ := parsear("desde=2026-07-01")

	require.True(t, ok)
	require.NotNil(t, filtro.Desde)
	assert.Equal(t, 0, filtro.Desde.Hour())
	assert.Equal(t, 1, filtro.Desde.Day())
}

func TestFiltro_AjustaLaPaginacionFueraDeRango(t *testing.T) {
	// Un listado es una lectura: es mas util devolver algo razonable que un
	// error por pedir la pagina 0.
	casos := []struct {
		consulta       string
		pagina, limite int
	}{
		{"pagina=0", 1, modelos.LimitePorDefecto},
		{"pagina=-5", 1, modelos.LimitePorDefecto},
		{"limite=0", 1, modelos.LimitePorDefecto},
		{"limite=-1", 1, modelos.LimitePorDefecto},
		{"limite=500", 1, modelos.LimiteMaximo},
		{"pagina=abc&limite=xyz", 1, modelos.LimitePorDefecto},
	}

	for _, caso := range casos {
		t.Run(caso.consulta, func(t *testing.T) {
			filtro, ok, _ := parsear(caso.consulta)

			require.True(t, ok)
			assert.Equal(t, caso.pagina, filtro.Pagina)
			assert.Equal(t, caso.limite, filtro.Limite)
		})
	}
}

func TestFiltro_RechazaLoQueHariaEnganosoElResultado(t *testing.T) {
	// Aqui si se responde error: una fecha mal escrita que se ignorara en
	// silencio devolveria un listado que no es el que se pidio.
	casos := map[string]string{
		"fecha con formato raro":   "desde=31-07-2026",
		"fecha inventada":          "hasta=2026-13-45",
		"tipo desconocido":         "tipo=transferencia",
		"categoria_id no valido":   "categoria_id=abc",
		"cuenta_id no hexadecimal": "cuenta_id=zzzzzzzzzzzzzzzzzzzzzzzz",
	}

	for nombre, consulta := range casos {
		t.Run(nombre, func(t *testing.T) {
			_, ok, grabadora := parsear(consulta)

			assert.False(t, ok)
			assert.Equal(t, http.StatusBadRequest, grabadora.Code)
		})
	}
}

func TestBooleano_SoloTrueCuentaComoVerdadero(t *testing.T) {
	casos := map[string]bool{
		"incluir_archivadas=true":  true,
		"incluir_archivadas=TRUE":  true,
		"incluir_archivadas=1":     false,
		"incluir_archivadas=false": false,
		"":                         false,
	}

	for consulta, esperado := range casos {
		t.Run(consulta, func(t *testing.T) {
			grabadora := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(grabadora)
			c.Request = httptest.NewRequest(http.MethodGet, "/x?"+consulta, nil)

			assert.Equal(t, esperado, booleano(c, "incluir_archivadas"))
		})
	}
}

func TestIDDeLaRuta_RechazaUnIdentificadorMalFormado(t *testing.T) {
	grabadora := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(grabadora)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	c.Params = gin.Params{{Key: "id", Value: "no-es-un-id"}}

	_, ok := idDeLaRuta(c)

	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, grabadora.Code)
}

func TestIDDeLaRuta_AceptaUnObjectIDValido(t *testing.T) {
	esperado := bson.NewObjectID()
	grabadora := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(grabadora)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	c.Params = gin.Params{{Key: "id", Value: esperado.Hex()}}

	id, ok := idDeLaRuta(c)

	require.True(t, ok)
	assert.Equal(t, esperado, id)
}
