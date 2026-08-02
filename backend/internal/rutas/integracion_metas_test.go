package rutas

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// Pruebas de la consulta relacional 3 (metas cruzadas con aportaciones) contra
// MongoDB de verdad, sobre cifras que se pueden sumar a mano.

// hoyDeLasPruebas es el reloj con el que corre el servicio de metas en las
// pruebas de integracion. Sin fijarlo, "faltan 90 dias" seria una afirmacion
// que deja de ser cierta mañana.
var hoyDeLasPruebas = time.Date(2026, time.August, 2, 12, 0, 0, 0, time.UTC)

// ahorros es el escenario de las metas:
//
//	Fondo de emergencia  30 000 para el 31/10/2026   4 000 + 6 000 + 5 500 = 15 500
//	Laptop nueva         22 000 para el 30/09/2026   22 500 (se paso)
//	Viaje                25 000 para el 28/07/2026   9 000 (la fecha ya paso)
type ahorros struct {
	api                    *api
	token                  string
	fondo, laptop, viaje   string
	aportacionesDelFondo   int
	primeraAportacionViaje string
}

func escenarioDeMetas(t *testing.T, correo string) ahorros {
	t.Helper()

	a := routerReal(t)
	token := a.nuevoUsuario(correo)

	s := ahorros{api: a, token: token}
	s.fondo = a.crearMeta(token, "Fondo de emergencia", 30000, fechaUTC(2026, time.October, 31))
	s.laptop = a.crearMeta(token, "Laptop nueva", 22000, fechaUTC(2026, time.September, 30))
	s.viaje = a.crearMeta(token, "Viaje", 25000, fechaUTC(2026, time.July, 28))

	a.aportar(token, s.fondo, 4000, fechaUTC(2026, time.March, 15))
	a.aportar(token, s.fondo, 6000, fechaUTC(2026, time.May, 15))
	a.aportar(token, s.fondo, 5500, fechaUTC(2026, time.July, 10))
	s.aportacionesDelFondo = 3

	a.aportar(token, s.laptop, 22500, fechaUTC(2026, time.June, 20))
	s.primeraAportacionViaje = a.aportar(token, s.viaje, 9000, fechaUTC(2026, time.April, 28))

	return s
}

// listar pide GET /metas y deserializa el listado con su resumen.
func (s ahorros) listar(consulta string) modelos.ListadoMetas {
	var listado modelos.ListadoMetas
	s.api.datos(http.MethodGet, "/api/v1/metas"+consulta, s.token, nil, http.StatusOK, &listado)
	return listado
}

func TestIntegracion_ProgresoDeMetas(t *testing.T) {
	s := escenarioDeMetas(t, "metas@fintrack.mx")

	listado := s.listar("")
	require.Len(t, listado.Metas, 3)

	// Ordenadas por fecha limite ascendente: lo que vence antes va primero.
	porNombre := map[string]modelos.ProgresoMeta{}
	for _, m := range listado.Metas {
		porNombre[m.Nombre] = m
	}
	assert.Equal(t, "Viaje", listado.Metas[0].Nombre)
	assert.Equal(t, "Fondo de emergencia", listado.Metas[2].Nombre)

	fondo := porNombre["Fondo de emergencia"]
	assert.Equal(t, 15500.0, fondo.Ahorrado, "4000 + 6000 + 5500")
	assert.Equal(t, 14500.0, fondo.Restante)
	assert.InDelta(t, 51.67, fondo.Porcentaje, 0.01)
	assert.Equal(t, 3, fondo.Aportaciones)
	assert.Equal(t, modelos.EstadoMetaEnCurso, fondo.Estado)
	assert.Equal(t, 90, fondo.DiasRestantes, "del 2 de agosto al 31 de octubre")
	assert.Equal(t, 4833.33, fondo.RitmoMensual, "14 500 repartidos en tres meses")
	require.NotNil(t, fondo.UltimaFecha)
	assert.Equal(t, fechaUTC(2026, time.July, 10), fondo.UltimaFecha.UTC())

	laptop := porNombre["Laptop nueva"]
	assert.Equal(t, 22500.0, laptop.Ahorrado)
	assert.Zero(t, laptop.Restante, "juntar de mas deja el restante en cero, no en negativo")
	assert.InDelta(t, 102.27, laptop.Porcentaje, 0.01)
	assert.Equal(t, modelos.EstadoMetaCumplida, laptop.Estado)
	assert.Zero(t, laptop.RitmoMensual)

	viaje := porNombre["Viaje"]
	assert.Equal(t, 9000.0, viaje.Ahorrado)
	assert.Equal(t, modelos.EstadoMetaVencida, viaje.Estado)
	assert.Equal(t, -5, viaje.DiasRestantes)
	assert.Equal(t, 16000.0, viaje.RitmoMensual, "vencida: lo que falta se pide entero")
}

func TestIntegracion_ResumenDeMetas(t *testing.T) {
	s := escenarioDeMetas(t, "resumen-metas@fintrack.mx")

	resumen := s.listar("").Resumen

	assert.Equal(t, 3, resumen.Total)
	assert.Equal(t, 1, resumen.Cumplidas)
	assert.Equal(t, 1, resumen.Vencidas)
	assert.Equal(t, 77000.0, resumen.Objetivo, "30000 + 22000 + 25000")
	assert.Equal(t, 47000.0, resumen.Ahorrado, "15500 + 22500 + 9000")
}

func TestIntegracion_MetaSinAportacionesSaleEnCero(t *testing.T) {
	s := escenarioDeMetas(t, "meta-vacia@fintrack.mx")
	s.api.crearMeta(s.token, "Coche", 180000, fechaUTC(2027, time.June, 30))

	var vacia modelos.ProgresoMeta
	for _, m := range s.listar("").Metas {
		if m.Nombre == "Coche" {
			vacia = m
		}
	}

	// El $lookup devuelve un arreglo vacio y el $ifNull lo convierte en 0: la
	// meta tiene que aparecer igual, con su barra a cero.
	assert.Equal(t, "Coche", vacia.Nombre)
	assert.Zero(t, vacia.Ahorrado)
	assert.Zero(t, vacia.Aportaciones)
	assert.Equal(t, 180000.0, vacia.Restante)
	assert.Nil(t, vacia.UltimaFecha, "nunca no es una fecha")
}

func TestIntegracion_DetalleDeUnaMeta(t *testing.T) {
	s := escenarioDeMetas(t, "detalle-meta@fintrack.mx")

	var detalle modelos.MetaConAportaciones
	s.api.datos(http.MethodGet, "/api/v1/metas/"+s.fondo, s.token, nil, http.StatusOK, &detalle)

	assert.Equal(t, 15500.0, detalle.Ahorrado)
	require.Len(t, detalle.Detalle, 3)
	// De la mas reciente a la mas vieja.
	assert.Equal(t, 5500.0, detalle.Detalle[0].Monto)
	assert.Equal(t, 4000.0, detalle.Detalle[2].Monto)
}

func TestIntegracion_BorrarLaMetaSeLlevaSusAportaciones(t *testing.T) {
	s := escenarioDeMetas(t, "borrar-meta@fintrack.mx")

	s.api.datos(http.MethodDelete, "/api/v1/metas/"+s.viaje, s.token, nil, http.StatusNoContent, nil)

	// La meta ya no esta...
	grabadora := s.api.llamar(http.MethodGet, "/api/v1/metas/"+s.viaje, s.token, nil)
	assert.Equal(t, http.StatusNotFound, grabadora.Code)

	// ...y las demas siguen intactas, con sus propias aportaciones.
	listado := s.listar("")
	assert.Len(t, listado.Metas, 2)
	assert.Equal(t, 38000.0, listado.Resumen.Ahorrado, "15500 + 22500")
}

func TestIntegracion_LasMetasDeOtroUsuarioNoExisten(t *testing.T) {
	s := escenarioDeMetas(t, "duenio-metas@fintrack.mx")
	intruso := s.api.nuevoUsuario("intruso-metas@fintrack.mx")

	// El listado del intruso esta vacio aunque la base tenga tres metas.
	var suyas modelos.ListadoMetas
	s.api.datos(http.MethodGet, "/api/v1/metas", intruso, nil, http.StatusOK, &suyas)
	assert.Empty(t, suyas.Metas)
	assert.Zero(t, suyas.Resumen.Ahorrado)

	// Y ninguna operacion sobre una meta ajena funciona: 404, no 403, porque un
	// 403 confirmaria que existe.
	for _, caso := range []struct{ metodo, ruta string }{
		{http.MethodGet, "/api/v1/metas/" + s.fondo},
		{http.MethodDelete, "/api/v1/metas/" + s.fondo},
		{http.MethodDelete, "/api/v1/metas/" + s.viaje + "/aportaciones/" + s.primeraAportacionViaje},
	} {
		grabadora := s.api.llamar(caso.metodo, caso.ruta, intruso, nil)
		assert.Equal(t, http.StatusNotFound, grabadora.Code, "%s %s", caso.metodo, caso.ruta)
	}

	// Y la meta del dueño sigue con sus tres aportaciones.
	assert.Equal(t, 15500.0, s.listar("").Metas[2].Ahorrado)
}

func TestIntegracion_CRUDDeMetas(t *testing.T) {
	a := routerReal(t)
	token := a.nuevoUsuario("crud-metas@fintrack.mx")

	// Crear
	var creada modelos.Meta
	a.datos(http.MethodPost, "/api/v1/metas", token, modelos.PeticionMeta{
		Nombre: "Coche", MontoObjetivo: 180000, FechaLimite: fechaUTC(2027, time.June, 30),
		Color: "#0891B2", Notas: "Enganche",
	}, http.StatusCreated, &creada)
	assert.Equal(t, "Coche", creada.Nombre)
	require.NotNil(t, creada.Notas)
	assert.Equal(t, "Enganche", *creada.Notas)

	// Actualizar: se sube el objetivo y se adelanta la fecha.
	var actualizada modelos.Meta
	a.datos(http.MethodPut, "/api/v1/metas/"+creada.ID.Hex(), token, modelos.PeticionMeta{
		Nombre: "Coche usado", MontoObjetivo: 150000, FechaLimite: fechaUTC(2027, time.March, 31),
		Color: "#7C3AED",
	}, http.StatusOK, &actualizada)
	assert.Equal(t, "Coche usado", actualizada.Nombre)
	assert.Equal(t, 150000.0, actualizada.MontoObjetivo)
	assert.Equal(t, fechaUTC(2027, time.March, 31), actualizada.FechaLimite.UTC())
	assert.Nil(t, actualizada.Notas, "quitar las notas las deja en null")
	assert.Equal(t, creada.CreadoEn.UTC(), actualizada.CreadoEn.UTC(), "creado_en no cambia")

	// Archivar: deja de salir en el listado, salvo que se pidan las archivadas.
	a.datos(http.MethodPut, "/api/v1/metas/"+creada.ID.Hex(), token, modelos.PeticionMeta{
		Nombre: "Coche usado", MontoObjetivo: 150000, FechaLimite: fechaUTC(2027, time.March, 31),
		Color: "#7C3AED", Archivada: true,
	}, http.StatusOK, nil)

	var visibles, todas modelos.ListadoMetas
	a.datos(http.MethodGet, "/api/v1/metas", token, nil, http.StatusOK, &visibles)
	assert.Empty(t, visibles.Metas)
	a.datos(http.MethodGet, "/api/v1/metas?incluir_archivadas=true", token, nil, http.StatusOK, &todas)
	assert.Len(t, todas.Metas, 1)

	// Una meta archivada si se puede abrir por su id: se llego a ella por la
	// direccion, no por el listado.
	var detalle modelos.MetaConAportaciones
	a.datos(http.MethodGet, "/api/v1/metas/"+creada.ID.Hex(), token, nil, http.StatusOK, &detalle)
	assert.True(t, detalle.Archivada)

	// Borrar
	a.datos(http.MethodDelete, "/api/v1/metas/"+creada.ID.Hex(), token, nil, http.StatusNoContent, nil)
	grabadora := a.llamar(http.MethodGet, "/api/v1/metas/"+creada.ID.Hex(), token, nil)
	assert.Equal(t, http.StatusNotFound, grabadora.Code)
}

func TestIntegracion_IdentificadoresMalEscritosEnLaRutaDeMetas(t *testing.T) {
	s := escenarioDeMetas(t, "ids-metas@fintrack.mx")

	casos := []struct {
		nombre string
		ruta   string
	}{
		{"la meta no es un ObjectID", "/api/v1/metas/no-es-un-id/aportaciones/" + s.primeraAportacionViaje},
		{"la aportacion no es un ObjectID", "/api/v1/metas/" + s.viaje + "/aportaciones/tampoco"},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			grabadora := s.api.llamar(http.MethodDelete, caso.ruta, s.token, nil)
			// 400 y no 404: la ruta no se pudo ni interpretar.
			assert.Equal(t, http.StatusBadRequest, grabadora.Code)
			assert.Contains(t, grabadora.Body.String(), "ID_INVALIDO")
		})
	}
}

func TestIntegracion_NoSeBorraUnaAportacionPorLaMetaEquivocada(t *testing.T) {
	s := escenarioDeMetas(t, "aportacion-cruzada@fintrack.mx")

	// La aportacion existe, pero es del viaje, no del fondo.
	ruta := "/api/v1/metas/" + s.fondo + "/aportaciones/" + s.primeraAportacionViaje
	grabadora := s.api.llamar(http.MethodDelete, ruta, s.token, nil)

	assert.Equal(t, http.StatusNotFound, grabadora.Code)
	assert.Equal(t, 9000.0, s.listar("").Metas[0].Ahorrado, "el viaje conserva su aportacion")
}
