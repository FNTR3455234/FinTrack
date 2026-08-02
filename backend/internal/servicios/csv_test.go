package servicios

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	fintrackErrores "github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// --- importacion ------------------------------------------------------------

func TestCSVImportar_GuardaLasFilasValidas(t *testing.T) {
	e := nuevoEscenarioCSV(t)

	resultado, err := e.importar(encabezado +
		"2026-07-03,gasto,BBVA Debito,Supermercado,850.50,Despensa,con cupon\n" +
		"2026-07-04,ingreso,BBVA Debito,Nomina,20000,Quincena,\n")

	require.NoError(t, err)
	assert.Equal(t, 2, resultado.Importadas)
	assert.Len(t, e.repo.datos, 2)

	for _, t2 := range e.repo.datos {
		assert.Equal(t, e.usuarioID, t2.UsuarioID, "el dueño sale del token, no del archivo")
		assert.Equal(t, 12, t2.Fecha.Hour(), "la fecha se ancla a mediodia UTC")
	}
}

func TestCSVImportar_ElOrdenYLasMayusculasDelEncabezadoDanIgual(t *testing.T) {
	e := nuevoEscenarioCSV(t)

	resultado, err := e.importar(
		"Notas,MONTO,Fecha,Categoria,Cuenta,Tipo,Descripcion\n" +
			",850.50,2026-07-03,supermercado,bbva debito,gasto,Despensa\n")

	require.NoError(t, err)
	assert.Equal(t, 1, resultado.Importadas, "el archivo lo pudo armar una persona en Excel")
}

func TestCSVImportar_AceptaElFormatoDeMonedaDeExcel(t *testing.T) {
	e := nuevoEscenarioCSV(t)

	resultado, err := e.importar(encabezado +
		"2026-07-03,gasto,BBVA Debito,Supermercado,\"$1,250.50\",Despensa,\n")

	require.NoError(t, err)
	require.Equal(t, 1, resultado.Importadas)
	for _, t2 := range e.repo.datos {
		assert.Equal(t, 1250.50, t2.Monto)
	}
}

func TestCSVImportar_SaltaLasLineasEnBlancoDelFinal(t *testing.T) {
	e := nuevoEscenarioCSV(t)

	resultado, err := e.importar(encabezado +
		"2026-07-03,gasto,BBVA Debito,Supermercado,100,Despensa,\n" +
		",,,,,,\n" +
		",,,,,,\n")

	require.NoError(t, err)
	assert.Equal(t, 1, resultado.Importadas)
}

func TestCSVImportar_QuitaLaMarcaBOMQueEscribeLaExportacion(t *testing.T) {
	// Es lo que hace que un archivo exportado se pueda volver a subir tal cual.
	e := nuevoEscenarioCSV(t)

	resultado, err := e.importar("\uFEFF" + encabezado +
		"2026-07-03,gasto,BBVA Debito,Supermercado,100,Despensa,\n")

	require.NoError(t, err)
	assert.Equal(t, 1, resultado.Importadas)
}

func TestCSVImportar_UnaFilaMalaTumbaElArchivoCompleto(t *testing.T) {
	e := nuevoEscenarioCSV(t)

	_, err := e.importar(encabezado +
		"2026-07-03,gasto,BBVA Debito,Supermercado,100,Buena,\n" +
		"2026-07-04,gasto,BBVA Debito,Supermercado,-50,Monto negativo,\n" +
		"2026-07-05,gasto,BBVA Debito,Supermercado,100,Otra buena,\n")

	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Equal(t, fintrackErrores.CodigoCSVInvalido, dominio.Codigo)
	assert.Empty(t, e.repo.datos,
		"o entra el archivo completo o no entra nada: reintentar no puede duplicar lo que si entro")
	require.Len(t, dominio.Detalles, 1)
	assert.Contains(t, dominio.Detalles[0], "fila 3", "el numero es el que se ve en la hoja de calculo")
}

func TestCSVImportar_JuntaTodosLosErroresDeUnaVez(t *testing.T) {
	e := nuevoEscenarioCSV(t)

	_, err := e.importar(encabezado +
		"03/07/2026,gasto,BBVA Debito,Supermercado,100,Fecha al reves,\n" +
		"2026-07-04,transferencia,BBVA Debito,Supermercado,100,Tipo raro,\n" +
		"2026-07-05,gasto,Cuenta fantasma,Supermercado,100,Cuenta que no existe,\n")

	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Len(t, dominio.Detalles, 3, "quien sube un archivo con erratas prefiere verlas todas juntas")
}

func TestCSVImportar_RechazaCadaClaseDeCeldaInvalida(t *testing.T) {
	casos := map[string]struct {
		fila     string
		contiene string
	}{
		"fecha con otro formato": {"03-07-2026,gasto,BBVA Debito,Supermercado,100,X,", "AAAA-MM-DD"},
		"tipo desconocido":       {"2026-07-03,transferencia,BBVA Debito,Supermercado,100,X,", "no es ingreso ni gasto"},
		"monto en cero":          {"2026-07-03,gasto,BBVA Debito,Supermercado,0,X,", "mayor que cero"},
		"monto que no es numero": {"2026-07-03,gasto,BBVA Debito,Supermercado,abc,X,", "no es un numero"},
		"sin descripcion":        {"2026-07-03,gasto,BBVA Debito,Supermercado,100,,", "descripcion es obligatoria"},
		"cuenta inexistente":     {"2026-07-03,gasto,Otra cuenta,Supermercado,100,X,", `la cuenta "Otra cuenta" no existe`},
		"categoria inexistente":  {"2026-07-03,gasto,BBVA Debito,Viajes,100,X,", `la categoria "Viajes" no existe`},
		"tipo contra categoria":  {"2026-07-03,ingreso,BBVA Debito,Supermercado,100,X,", "es de tipo"},
	}

	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			e := nuevoEscenarioCSV(t)

			_, err := e.importar(encabezado + caso.fila + "\n")

			dominio, ok := fintrackErrores.Como(err)
			require.True(t, ok)
			assert.Equal(t, fintrackErrores.CodigoCSVInvalido, dominio.Codigo)
			require.Len(t, dominio.Detalles, 1)
			assert.Contains(t, dominio.Detalles[0], caso.contiene)
		})
	}
}

func TestCSVImportar_RechazaLosArchivosQueNiSiquieraSonLaTabla(t *testing.T) {
	casos := map[string]string{
		"vacio":                "",
		"solo el encabezado":   encabezado,
		"le faltan columnas":   "fecha,tipo,monto\n2026-07-03,gasto,100\n",
		"columnas desiguales":  encabezado + "2026-07-03,gasto,BBVA Debito\n",
		"no tiene encabezados": "2026-07-03,gasto,BBVA Debito,Supermercado,100,X,\n",
	}

	for nombre, contenido := range casos {
		t.Run(nombre, func(t *testing.T) {
			e := nuevoEscenarioCSV(t)

			_, err := e.importar(contenido)

			dominio, ok := fintrackErrores.Como(err)
			require.True(t, ok)
			assert.Equal(t, fintrackErrores.CodigoCSVInvalido, dominio.Codigo)
			assert.Equal(t, 400, dominio.HTTP)
		})
	}
}

func TestCSVImportar_NoAlcanzaLasCuentasDeOtroUsuario(t *testing.T) {
	e := nuevoEscenarioCSV(t)
	intruso := bson.NewObjectID()

	// El intruso escribe en su archivo el nombre exacto de la cuenta de otro.
	// Como el catalogo se arma filtrando por usuario, para el no existe.
	_, err := e.servicio.Importar(context.Background(), intruso,
		strings.NewReader(encabezado+"2026-07-03,gasto,BBVA Debito,Supermercado,100,Intento,\n"))

	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Contains(t, dominio.Detalles[0], "no existe")
	assert.Empty(t, e.repo.datos)
}

func TestCSVImportar_AvisaCuandoDosNombresNoSePuedenDistinguir(t *testing.T) {
	e := nuevoEscenarioCSV(t)
	// Dos categorias que solo se diferencian por las mayusculas: una fila que
	// diga "supermercado" no puede saber a cual se refiere.
	require.NoError(t, e.servicio.categorias.(*categoriasFalso).Crear(context.Background(),
		&modelos.Categoria{UsuarioID: e.usuarioID, Nombre: "SUPERMERCADO", Tipo: modelos.TipoGasto}))

	_, err := e.importar(encabezado + "2026-07-03,gasto,BBVA Debito,Supermercado,100,X,\n")

	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Contains(t, dominio.Detalles[0], "no se puede distinguir")
}
