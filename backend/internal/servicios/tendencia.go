package servicios

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// Tendencia devuelve los ultimos `meses` meses terminando en `hasta`, del mas
// viejo al mas reciente.
func (s *Reportes) Tendencia(ctx context.Context, usuarioID bson.ObjectID, hasta modelos.Periodo, meses int) ([]modelos.PuntoTendencia, error) {
	if meses < 1 {
		meses = modelos.MesesTendenciaPorDefecto
	}
	if meses > modelos.MesesTendenciaMaximo {
		meses = modelos.MesesTendenciaMaximo
	}

	desde := hasta.Retroceder(meses - 1)

	puntos, err := s.repositorio.Tendencia(ctx, usuarioID, desde, hasta)
	if err != nil {
		return nil, errores.Interno(err)
	}
	return completarMeses(puntos, desde, meses), nil
}

// completarMeses devuelve la serie con TODOS los meses del rango, en orden y
// rellenando con ceros los que no traen ningun movimiento.
//
// MongoDB solo agrupa lo que existe: un mes sin transacciones sencillamente no
// sale de la agregacion. Si se dibujara asi, la grafica de barras juntaria
// febrero con mayo y mentiria sobre la forma de la serie, que es justo lo unico
// que una tendencia tiene que contar bien.
func completarMeses(puntos []modelos.PuntoTendencia, desde modelos.Periodo, meses int) []modelos.PuntoTendencia {
	// Se indexa lo que vino por su etiqueta para no recorrer la lista en cada mes.
	porMes := make(map[string]modelos.PuntoTendencia, len(puntos))
	for _, punto := range puntos {
		porMes[etiquetaDe(punto.Periodo)] = punto
	}

	serie := make([]modelos.PuntoTendencia, 0, meses)
	periodo := desde
	for i := 0; i < meses; i++ {
		etiqueta := etiquetaDe(periodo)

		punto, hubo := porMes[etiqueta]
		if !hubo {
			punto = modelos.PuntoTendencia{Periodo: periodo}
		}
		punto.Etiqueta = etiqueta
		punto.Balance = redondear(punto.Ingresos - punto.Gastos)

		serie = append(serie, punto)
		periodo = siguienteMes(periodo)
	}
	return serie
}

// etiquetaDe devuelve el periodo como "2026-07": ordenable, sin ambiguedad de
// formato y directamente pintable en el eje de la grafica.
func etiquetaDe(periodo modelos.Periodo) string {
	return fmt.Sprintf("%04d-%02d", periodo.Anio, periodo.Mes)
}

// siguienteMes avanza un mes, cambiando de año en diciembre.
func siguienteMes(periodo modelos.Periodo) modelos.Periodo {
	if periodo.Mes == 12 {
		return modelos.Periodo{Mes: 1, Anio: periodo.Anio + 1}
	}
	return modelos.Periodo{Mes: periodo.Mes + 1, Anio: periodo.Anio}
}
