package handlers

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"

	fintrackErrores "github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
	"github.com/FNTR3455234/FinTrack/backend/internal/respuestas"
)

// idDeLaRuta lee el parametro :id y lo convierte a ObjectID. Si no es un
// identificador valido ya responde 400 y devuelve false.
func idDeLaRuta(c *gin.Context) (bson.ObjectID, bool) {
	id, err := bson.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		respuestas.Fallo(c, fintrackErrores.SolicitudInvalida(fintrackErrores.CodigoIDInvalido,
			"El identificador no es valido."))
		return bson.NilObjectID, false
	}
	return id, true
}

// booleano lee un parametro de consulta tipo bandera. Solo "true" cuenta como
// verdadero; cualquier otra cosa (o su ausencia) es falso.
func booleano(c *gin.Context, nombre string) bool {
	return strings.EqualFold(c.Query(nombre), "true")
}

// filtroDeTransacciones arma el filtro del listado a partir de la query.
//
// Los valores fuera de rango se ajustan en lugar de rechazarse (pagina 0 pasa a
// 1, limite 500 pasa a 100): un listado es una consulta de lectura y es mas
// util devolver algo razonable que un error.  Lo que si se rechaza es una fecha
// o un identificador mal escrito, porque ahi el resultado seria enganoso.
func filtroDeTransacciones(c *gin.Context) (modelos.FiltroTransacciones, bool) {
	filtro := modelos.FiltroTransacciones{
		Tipo:     c.Query("tipo"),
		Busqueda: strings.TrimSpace(c.Query("busqueda")),
		Pagina:   entero(c.Query("pagina"), 1),
		Limite:   entero(c.Query("limite"), modelos.LimitePorDefecto),
		Orden:    c.Query("orden"),
	}

	if filtro.Pagina < 1 {
		filtro.Pagina = 1
	}
	if filtro.Limite < 1 {
		filtro.Limite = modelos.LimitePorDefecto
	}
	if filtro.Limite > modelos.LimiteMaximo {
		filtro.Limite = modelos.LimiteMaximo
	}

	if filtro.Tipo != "" && filtro.Tipo != modelos.TipoIngreso && filtro.Tipo != modelos.TipoGasto {
		respuestas.Fallo(c, fintrackErrores.SolicitudInvalida(fintrackErrores.CodigoDatosInvalidos,
			"El filtro tipo debe ser ingreso o gasto."))
		return filtro, false
	}

	var ok bool
	if filtro.Desde, ok = fecha(c, "desde", false); !ok {
		return filtro, false
	}
	if filtro.Hasta, ok = fecha(c, "hasta", true); !ok {
		return filtro, false
	}
	if filtro.CategoriaID, ok = identificador(c, "categoria_id"); !ok {
		return filtro, false
	}
	if filtro.CuentaID, ok = identificador(c, "cuenta_id"); !ok {
		return filtro, false
	}

	return filtro, true
}

// fecha lee un parametro con formato AAAA-MM-DD.
//
// finDelDia hace que "hasta=2026-07-31" incluya todo ese dia y no se quede en
// las 00:00, que es el error clasico de los filtros por rango.
func fecha(c *gin.Context, nombre string, finDelDia bool) (*time.Time, bool) {
	valor := strings.TrimSpace(c.Query(nombre))
	if valor == "" {
		return nil, true
	}

	momento, err := time.Parse(time.DateOnly, valor)
	if err != nil {
		respuestas.Fallo(c, fintrackErrores.SolicitudInvalida(fintrackErrores.CodigoDatosInvalidos,
			"La fecha "+nombre+" debe tener el formato AAAA-MM-DD."))
		return nil, false
	}

	if finDelDia {
		momento = momento.Add(24*time.Hour - time.Nanosecond)
	}
	return &momento, true
}

// identificador lee un parametro de consulta que debe ser un ObjectID.
func identificador(c *gin.Context, nombre string) (*bson.ObjectID, bool) {
	valor := strings.TrimSpace(c.Query(nombre))
	if valor == "" {
		return nil, true
	}

	id, err := bson.ObjectIDFromHex(valor)
	if err != nil {
		respuestas.Fallo(c, fintrackErrores.SolicitudInvalida(fintrackErrores.CodigoIDInvalido,
			"El parametro "+nombre+" no es un identificador valido."))
		return nil, false
	}
	return &id, true
}

// entero convierte una cadena a numero, o devuelve el valor por defecto.
func entero(valor string, porDefecto int) int {
	numero, err := strconv.Atoi(strings.TrimSpace(valor))
	if err != nil {
		return porDefecto
	}
	return numero
}
