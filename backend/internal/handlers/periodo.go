package handlers

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	fintrackErrores "github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
	"github.com/FNTR3455234/FinTrack/backend/internal/respuestas"
)

// periodoDeLaConsulta lee los parametros mes y anio.
//
// Si no vienen, se usa el mes en curso: abrir el tablero sin escribir nada es
// el caso normal y ahi lo que se quiere ver es "este mes".
//
// A diferencia de los filtros del listado, aqui un valor invalido SI se
// rechaza. Un listado con un filtro raro devuelve menos filas y se nota; un
// reporte del mes 13 devolveria ceros que se leerian como "no gastaste nada".
func periodoDeLaConsulta(c *gin.Context) (modelos.Periodo, bool) {
	ahora := time.Now().UTC()

	mes, mesOK := numeroOPorDefecto(c.Query("mes"), int(ahora.Month()))
	anio, anioOK := numeroOPorDefecto(c.Query("anio"), ahora.Year())
	periodo := modelos.Periodo{Mes: mes, Anio: anio}

	if !mesOK || !anioOK || !periodo.Valido() {
		respuestas.Fallo(c, fintrackErrores.SolicitudInvalida(fintrackErrores.CodigoPeriodoInvalido,
			"El periodo no es valido: mes va de 1 a 12 y anio de 2000 a 2100."))
		return periodo, false
	}
	return periodo, true
}

// numeroOPorDefecto convierte el valor de un parametro de consulta.
//
// Se usa en vez de entero() porque aqui hay que distinguir "no vino" de "vino
// mal escrito". entero() devuelve el valor por defecto en los dos casos, y con
// eso "mes=abc" acabaria pasando por el mes actual: el usuario veria un reporte
// que no es el que pidio y no tendria como notarlo.
func numeroOPorDefecto(valor string, porDefecto int) (int, bool) {
	limpio := strings.TrimSpace(valor)
	if limpio == "" {
		return porDefecto, true
	}

	numero, err := strconv.Atoi(limpio)
	if err != nil {
		return porDefecto, false
	}
	return numero, true
}

// periodoOpcional lee mes y anio solo si vienen los dos.
//
// Lo usa el listado de presupuestos, donde "sin periodo" significa "todos" y no
// "los de este mes": el usuario que entra a administrar sus presupuestos quiere
// verlos completos.
func periodoOpcional(c *gin.Context) (*modelos.Periodo, bool) {
	if c.Query("mes") == "" && c.Query("anio") == "" {
		return nil, true
	}

	periodo, ok := periodoDeLaConsulta(c)
	if !ok {
		return nil, false
	}
	return &periodo, true
}
