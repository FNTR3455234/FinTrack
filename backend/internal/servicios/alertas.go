package servicios

import (
	"context"
	"fmt"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// EvaluadorPresupuesto dice como va el presupuesto de una categoria en un mes.
//
// Lo cumple el repositorio de reportes con la CONSULTA RELACIONAL 2, la misma
// que alimenta /reportes/estado-presupuestos. Reusarla es lo que garantiza que
// la alerta y el tablero nunca digan cosas distintas del mismo presupuesto.
type EvaluadorPresupuesto interface {
	EstadoDeCategoria(ctx context.Context, usuarioID, categoriaID bson.ObjectID, periodo modelos.Periodo) (*modelos.EstadoPresupuesto, error)
}

// alertaDe avisa si el gasto recien registrado dejo su categoria cerca del
// limite del mes o por encima de el.
//
// Devuelve nil cuando no hay nada que avisar: si el movimiento es un ingreso,
// si esa categoria no tiene presupuesto ese mes, o si todavia va holgada.
//
// Un fallo al calcular la alerta NO tumba la peticion: la transaccion ya se
// guardo, y responder 500 haria creer al usuario que no se registro y que
// tiene que volver a capturarla. Se anota en la bitacora y se responde sin
// alerta.
func (s *Transacciones) alertaDe(ctx context.Context, usuarioID bson.ObjectID, t *modelos.Transaccion) *modelos.AlertaPresupuesto {
	if t.Tipo != modelos.TipoGasto {
		return nil
	}

	// El periodo sale de la fecha del movimiento, no del dia de hoy: registrar
	// hoy un gasto del mes pasado tiene que revisar el presupuesto del mes pasado.
	periodo := modelos.PeriodoDe(t.Fecha)

	estado, err := s.presupuestos.EstadoDeCategoria(ctx, usuarioID, t.CategoriaID, periodo)
	if err != nil {
		slog.Warn("no se pudo revisar el presupuesto de la transaccion",
			"transaccion", t.ID.Hex(), "detalle", err)
		return nil
	}
	if estado == nil || estado.Estado == modelos.EstadoOK {
		return nil
	}

	return &modelos.AlertaPresupuesto{
		CategoriaID:     estado.CategoriaID,
		Nombre:          estado.Nombre,
		MontoLimite:     estado.MontoLimite,
		Gastado:         estado.Gastado,
		Disponible:      estado.Disponible,
		PorcentajeUsado: estado.PorcentajeUsado,
		Estado:          estado.Estado,
		Mensaje:         mensajeDeAlerta(*estado),
	}
}

// mensajeDeAlerta redacta el aviso ya listo para mostrar.
//
// El texto se arma aqui y no en el frontend porque el idioma y el redondeo del
// dinero son los mismos para cualquier cliente de la API.
func mensajeDeAlerta(estado modelos.EstadoPresupuesto) string {
	if estado.Estado == modelos.EstadoExcedido {
		return fmt.Sprintf(
			"Te pasaste del presupuesto de %s: llevas %.2f de %.2f (%.2f%%), %.2f de mas.",
			estado.Nombre, estado.Gastado, estado.MontoLimite,
			estado.PorcentajeUsado, -estado.Disponible)
	}
	return fmt.Sprintf(
		"Vas en el %.2f%% del presupuesto de %s: llevas %.2f de %.2f y te quedan %.2f.",
		estado.PorcentajeUsado, estado.Nombre, estado.Gastado,
		estado.MontoLimite, estado.Disponible)
}
