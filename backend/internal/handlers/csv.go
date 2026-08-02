package handlers

import (
	"context"
	"encoding/csv"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"

	fintrackErrores "github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
	"github.com/FNTR3455234/FinTrack/backend/internal/respuestas"
)

// campoDelArchivo es el nombre del campo del formulario que trae el CSV.
const campoDelArchivo = "archivo"

// ServicioCSV es lo que los handlers necesitan del servicio.
type ServicioCSV interface {
	Exportar(ctx context.Context, usuarioID bson.ObjectID, f modelos.FiltroTransacciones) ([][]string, error)
	Importar(ctx context.Context, usuarioID bson.ObjectID, lector io.Reader) (*modelos.ResultadoImportacion, error)
}

// CSV agrupa los handlers de exportacion e importacion.
type CSV struct {
	servicio ServicioCSV
}

// NuevoCSV construye el grupo de handlers.
func NuevoCSV(servicio ServicioCSV) *CSV {
	return &CSV{servicio: servicio}
}

// Exportar atiende GET /transacciones/exportar.
//
// Acepta los mismos filtros que el listado (desde, hasta, tipo, categoria_id,
// cuenta_id, busqueda, orden) pero ignora la paginacion: quien pide "mis gastos
// de julio" espera el archivo completo, no la primera pagina.
//
//	@Summary		Exportar transacciones a CSV
//	@Description	Descarga un archivo con los mismos filtros del listado, sin paginar. Las cuentas y categorias salen por nombre.
//	@Tags			csv
//	@Produce		text/csv
//	@Security		BearerAuth
//	@Param			desde	query	string	false	"Fecha inicial AAAA-MM-DD"
//	@Param			hasta	query	string	false	"Fecha final AAAA-MM-DD, incluida"
//	@Param			tipo	query	string	false	"ingreso o gasto"	Enums(ingreso, gasto)
//	@Param			categoria_id	query	string	false	"Filtra por categoria"
//	@Param			cuenta_id	query	string	false	"Filtra por cuenta"
//	@Param			busqueda	query	string	false	"Texto en descripcion o notas"
//	@Param			orden	query	string	false	"Orden del listado"	Enums(fecha_desc, fecha_asc, monto_desc, monto_asc)
//	@Success		200	{file}	file	"Archivo CSV"
//	@Failure		400	{object}	respuestas.SobreError
//	@Failure		401	{object}	respuestas.SobreError
//	@Router			/transacciones/exportar [get]
func (h *CSV) Exportar(c *gin.Context) {
	usuarioID, ok := usuarioAutenticado(c)
	if !ok {
		return
	}
	filtro, ok := filtroDeTransacciones(c)
	if !ok {
		return
	}

	filas, err := h.servicio.Exportar(c.Request.Context(), usuarioID, filtro)
	if err != nil {
		respuestas.Fallo(c, err)
		return
	}

	escribirCSV(c, filas)
}

// nombreDelArchivo arma el nombre con el que el navegador guarda la descarga.
func nombreDelArchivo(momento time.Time) string {
	return "transacciones-" + momento.UTC().Format(time.DateOnly) + ".csv"
}

// escribirCSV manda las filas como archivo descargable.
func escribirCSV(c *gin.Context, filas [][]string) {
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="`+nombreDelArchivo(time.Now())+`"`)
	c.Status(http.StatusOK)

	// Excel supone que un .csv viene en la codificacion del sistema, no en
	// UTF-8, y sin esta marca al principio parte los acentos ("Educación" se ve
	// como "EducaciÃ³n"). La importacion la quita al leer, asi que el archivo
	// que exporta la API se puede volver a subir tal cual.
	_, _ = c.Writer.WriteString("\uFEFF")

	salida := csv.NewWriter(c.Writer)
	if err := salida.WriteAll(filas); err != nil {
		// El 200 y el encabezado ya se mandaron: aqui ya no se puede responder
		// un error en JSON. Solo queda dejarlo anotado en la bitacora.
		_ = c.Error(err)
		return
	}
	salida.Flush()
}

// Importar atiende POST /transacciones/importar con un formulario
// multipart que trae el archivo en el campo "archivo".
//
//	@Summary		Importar transacciones desde CSV
//	@Description	Sube un CSV con las columnas fecha, tipo, cuenta, categoria, monto, descripcion y notas. O entra el archivo completo o no entra nada.
//	@Tags			csv
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		BearerAuth
//	@Param			archivo	formData	file	true	"Archivo CSV, maximo 2 MiB"
//	@Success		201	{object}	respuestas.Sobre{datos=modelos.ResultadoImportacion}
//	@Failure		400	{object}	respuestas.SobreError
//	@Failure		401	{object}	respuestas.SobreError
//	@Router			/transacciones/importar [post]
func (h *CSV) Importar(c *gin.Context) {
	usuarioID, ok := usuarioAutenticado(c)
	if !ok {
		return
	}

	// Corta la lectura en el limite en vez de confiar en lo que diga
	// Content-Length, que lo pone el cliente.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, modelos.MaximoBytesCSV)

	encabezado, err := c.FormFile(campoDelArchivo)
	if err != nil {
		respuestas.Fallo(c, errorDeArchivo(err))
		return
	}

	archivo, err := encabezado.Open()
	if err != nil {
		respuestas.Fallo(c, fintrackErrores.SolicitudInvalida(fintrackErrores.CodigoArchivoRequerido,
			"No se pudo abrir el archivo."))
		return
	}
	defer func() { _ = archivo.Close() }()

	resultado, err := h.servicio.Importar(c.Request.Context(), usuarioID, archivo)
	if err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.Creado(c, resultado)
}

// errorDeArchivo distingue "no mandaste archivo" de "mandaste uno demasiado
// grande", que si no acabarian dando el mismo mensaje inutil.
func errorDeArchivo(err error) error {
	var muyGrande *http.MaxBytesError
	if errors.As(err, &muyGrande) {
		return fintrackErrores.SolicitudInvalida(fintrackErrores.CodigoArchivoMuyGrande,
			"El archivo pasa del tamaño maximo permitido.")
	}

	return fintrackErrores.SolicitudInvalida(fintrackErrores.CodigoArchivoRequerido,
		"Falta el archivo. Se manda como formulario multipart en el campo \""+campoDelArchivo+"\".")
}
