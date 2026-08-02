package rutas

import (
	"bytes"
	"encoding/csv"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

const rutaExportar = "/api/v1/transacciones/exportar"
const rutaImportar = "/api/v1/transacciones/importar"

// subir manda un archivo por multipart, como lo haria un formulario del navegador.
func (a *api) subir(token, contenido string) *httptest.ResponseRecorder {
	a.t.Helper()

	var cuerpo bytes.Buffer
	formulario := multipart.NewWriter(&cuerpo)
	parte, err := formulario.CreateFormFile("archivo", "movimientos.csv")
	require.NoError(a.t, err)
	_, err = parte.Write([]byte(contenido))
	require.NoError(a.t, err)
	require.NoError(a.t, formulario.Close())

	peticion := httptest.NewRequest(http.MethodPost, rutaImportar, &cuerpo)
	peticion.Header.Set("Content-Type", formulario.FormDataContentType())
	if token != "" {
		peticion.Header.Set("Authorization", "Bearer "+token)
	}

	grabadora := httptest.NewRecorder()
	a.router.ServeHTTP(grabadora, peticion)
	return grabadora
}

// exportar descarga el CSV y lo devuelve ya interpretado.
func (a *api) exportar(token, consulta string) (*httptest.ResponseRecorder, [][]string) {
	a.t.Helper()

	grabadora := a.llamar(http.MethodGet, rutaExportar+consulta, token, nil)
	require.Equal(a.t, http.StatusOK, grabadora.Code, grabadora.Body.String())

	// La marca BOM del principio no es parte de la tabla: se quita antes de leer.
	texto := strings.TrimPrefix(grabadora.Body.String(), "\uFEFF")
	filas, err := csv.NewReader(strings.NewReader(texto)).ReadAll()
	require.NoError(a.t, err)
	return grabadora, filas
}

// conMovimientos deja un usuario con cuenta, categorias y tres movimientos.
func conMovimientos(a *api, correo string) string {
	a.t.Helper()

	token := a.nuevoUsuario(correo)
	cuenta := a.crearCuenta(token, "BBVA Debito")
	gasto := a.crearCategoria(token, "Supermercado", modelos.TipoGasto)
	ingreso := a.crearCategoria(token, "Nomina", modelos.TipoIngreso)

	a.datos(http.MethodPost, "/api/v1/transacciones", token,
		movimiento(cuenta, gasto, modelos.TipoGasto, 850.50, "Despensa", fechaUTC(2026, time.July, 3)),
		http.StatusCreated, nil)
	a.datos(http.MethodPost, "/api/v1/transacciones", token,
		movimiento(cuenta, ingreso, modelos.TipoIngreso, 20000, "Quincena", fechaUTC(2026, time.July, 1)),
		http.StatusCreated, nil)
	a.datos(http.MethodPost, "/api/v1/transacciones", token,
		movimiento(cuenta, gasto, modelos.TipoGasto, 300, "Despensa de junio", fechaUTC(2026, time.June, 18)),
		http.StatusCreated, nil)

	return token
}

// --- exportacion ------------------------------------------------------------

func TestIntegracion_ExportarCSV(t *testing.T) {
	a := routerReal(t)
	token := conMovimientos(a, "exportar@fintrack.mx")

	grabadora, filas := a.exportar(token, "")

	assert.Contains(t, grabadora.Header().Get("Content-Type"), "text/csv")
	assert.Contains(t, grabadora.Header().Get("Content-Disposition"), "attachment")
	assert.Contains(t, grabadora.Header().Get("Content-Disposition"), ".csv")
	assert.True(t, strings.HasPrefix(grabadora.Body.String(), "\uFEFF"),
		"sin la marca BOM, Excel parte los acentos")

	require.Len(t, filas, 4, "el encabezado y los tres movimientos")
	assert.Equal(t, modelos.ColumnasCSV, filas[0])

	// Van de la mas reciente a la mas vieja, igual que el listado.
	assert.Equal(t, []string{"2026-07-03", "gasto", "BBVA Debito", "Supermercado", "850.50", "Despensa", ""}, filas[1])
	assert.Equal(t, "2026-06-18", filas[3][0])
}

func TestIntegracion_ExportarCSVRespetaLosFiltros(t *testing.T) {
	a := routerReal(t)
	token := conMovimientos(a, "exportar-filtros@fintrack.mx")

	_, soloJulio := a.exportar(token, "?desde=2026-07-01&hasta=2026-07-31")
	assert.Len(t, soloJulio, 3, "el de junio queda fuera")

	_, soloGastos := a.exportar(token, "?tipo=gasto")
	assert.Len(t, soloGastos, 3)

	_, buscados := a.exportar(token, "?busqueda=quincena")
	require.Len(t, buscados, 2)
	assert.Equal(t, "Quincena", buscados[1][5])
}

func TestIntegracion_ExportarCSVNoSeLlevaNadaDeOtroUsuario(t *testing.T) {
	a := routerReal(t)
	conMovimientos(a, "ana-csv@fintrack.mx")
	beto := a.nuevoUsuario("beto-csv@fintrack.mx")

	_, filas := a.exportar(beto, "")

	assert.Len(t, filas, 1, "Beto no tiene movimientos: solo baja el encabezado")
}

func TestIntegracion_ExportarCSVExigeToken(t *testing.T) {
	a := routerReal(t)

	grabadora := a.llamar(http.MethodGet, rutaExportar, "", nil)

	assert.Equal(t, http.StatusUnauthorized, grabadora.Code)
}

// --- importacion ------------------------------------------------------------

func TestIntegracion_ImportarCSV(t *testing.T) {
	a := routerReal(t)
	token := a.nuevoUsuario("importar@fintrack.mx")
	a.crearCuenta(token, "BBVA Debito")
	a.crearCategoria(token, "Supermercado", modelos.TipoGasto)

	grabadora := a.subir(token, "fecha,tipo,cuenta,categoria,monto,descripcion,notas\n"+
		"2026-07-03,gasto,BBVA Debito,Supermercado,850.50,Despensa,con cupon\n"+
		"2026-07-10,gasto,BBVA Debito,Supermercado,120,Pan,\n")

	require.Equal(t, http.StatusCreated, grabadora.Code, grabadora.Body.String())
	assert.Contains(t, grabadora.Body.String(), `"importadas":2`)

	// Y quedan guardadas de verdad, no solo contadas.
	var guardadas []modelos.Transaccion
	a.datos(http.MethodGet, "/api/v1/transacciones", token, nil, http.StatusOK, &guardadas)
	// El listado va de la mas reciente a la mas vieja.
	require.Len(t, guardadas, 2)
	assert.Equal(t, 120.0, guardadas[0].Monto)
	assert.Equal(t, "2026-07-10", guardadas[0].Fecha.UTC().Format(time.DateOnly))
	assert.Equal(t, 850.50, guardadas[1].Monto)
	require.NotNil(t, guardadas[1].Notas)
	assert.Equal(t, "con cupon", *guardadas[1].Notas)
}

func TestIntegracion_ImportarCSVNoGuardaNadaSiUnaFilaFalla(t *testing.T) {
	a := routerReal(t)
	token := a.nuevoUsuario("importar-fallo@fintrack.mx")
	a.crearCuenta(token, "BBVA Debito")
	a.crearCategoria(token, "Supermercado", modelos.TipoGasto)

	grabadora := a.subir(token, "fecha,tipo,cuenta,categoria,monto,descripcion,notas\n"+
		"2026-07-03,gasto,BBVA Debito,Supermercado,100,Buena,\n"+
		"2026-07-04,gasto,Cuenta que no existe,Supermercado,100,Mala,\n")

	require.Equal(t, http.StatusBadRequest, grabadora.Code)
	assert.Contains(t, grabadora.Body.String(), "CSV_INVALIDO")
	assert.Contains(t, grabadora.Body.String(), "fila 3")

	var guardadas []modelos.Transaccion
	a.datos(http.MethodGet, "/api/v1/transacciones", token, nil, http.StatusOK, &guardadas)
	assert.Empty(t, guardadas, "ni siquiera la fila buena")
}

func TestIntegracion_ImportarCSVSinArchivo(t *testing.T) {
	a := routerReal(t)
	token := a.nuevoUsuario("importar-sin-archivo@fintrack.mx")

	grabadora := a.llamar(http.MethodPost, rutaImportar, token, nil)

	assert.Equal(t, http.StatusBadRequest, grabadora.Code)
	assert.Contains(t, grabadora.Body.String(), "ARCHIVO_REQUERIDO")
}

func TestIntegracion_ImportarCSVExigeToken(t *testing.T) {
	a := routerReal(t)

	grabadora := a.subir("", "fecha,tipo,cuenta,categoria,monto,descripcion,notas\n")

	assert.Equal(t, http.StatusUnauthorized, grabadora.Code)
}

// --- ida y vuelta -----------------------------------------------------------

func TestIntegracion_LoQueExportaLaAPISePuedeVolverAImportar(t *testing.T) {
	// Es la prueba que amarra las dos mitades: el archivo con su marca BOM, sus
	// nombres de cuenta y categoria y su formato de fecha y de monto tiene que
	// entrar de vuelta sin que nadie lo edite.
	a := routerReal(t)
	ana := conMovimientos(a, "ana-ida-vuelta@fintrack.mx")

	descarga, _ := a.exportar(ana, "")
	archivo := descarga.Body.String()

	// Se importa en OTRA cuenta con los mismos nombres, para no duplicar los de
	// Ana y de paso comprobar que el archivo no arrastra identificadores suyos.
	beto := a.nuevoUsuario("beto-ida-vuelta@fintrack.mx")
	a.crearCuenta(beto, "BBVA Debito")
	a.crearCategoria(beto, "Supermercado", modelos.TipoGasto)
	a.crearCategoria(beto, "Nomina", modelos.TipoIngreso)

	grabadora := a.subir(beto, archivo)
	require.Equal(t, http.StatusCreated, grabadora.Code, grabadora.Body.String())
	assert.Contains(t, grabadora.Body.String(), `"importadas":3`)

	// Y lo que queda en la cuenta de Beto es idéntico a lo que exportó Ana.
	_, deAna := a.exportar(ana, "")
	_, deBeto := a.exportar(beto, "")
	assert.Equal(t, deAna, deBeto)
}
