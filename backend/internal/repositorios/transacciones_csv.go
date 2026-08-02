package repositorios

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// Todas devuelve las transacciones que cumplen el filtro, sin paginar.
//
// Es lo que necesita la exportacion: el usuario pide "mis gastos de julio" y
// espera el archivo completo, no la primera pagina. El tope de
// MaximoFilasExportar evita que una peticion sin filtros se traiga la coleccion
// entera a memoria.
func (r *Transacciones) Todas(ctx context.Context, usuarioID bson.ObjectID, filtro modelos.FiltroTransacciones) ([]modelos.Transaccion, error) {
	opciones := options.Find().
		SetSort(construirOrden(filtro.Orden)).
		SetLimit(modelos.MaximoFilasExportar)

	cursor, err := r.coleccion.Find(ctx, construirFiltro(usuarioID, filtro), opciones)
	if err != nil {
		return nil, traducir(err, "al exportar las transacciones")
	}

	transacciones := []modelos.Transaccion{}
	if err := cursor.All(ctx, &transacciones); err != nil {
		return nil, traducir(err, "al leer las transacciones a exportar")
	}
	return transacciones, nil
}

// CrearVarias inserta un lote de transacciones y devuelve cuantas entraron.
//
// InsertMany en modo ordenado (el de por defecto): si una falla, se detiene ahi
// en vez de seguir metiendo las demas. Aun asi, el servicio valida el archivo
// completo ANTES de llamar aqui, justo para que esto no pase: sin transacciones
// multidocumento (ver docs/decisiones.md, decision 003) un fallo a mitad
// dejaria el archivo importado por la mitad.
func (r *Transacciones) CrearVarias(ctx context.Context, transacciones []modelos.Transaccion) (int, error) {
	if len(transacciones) == 0 {
		return 0, nil
	}

	documentos := make([]any, 0, len(transacciones))
	for i := range transacciones {
		documentos = append(documentos, transacciones[i])
	}

	resultado, err := r.coleccion.InsertMany(ctx, documentos)
	if err != nil {
		return 0, traducir(err, "al insertar las transacciones importadas")
	}
	return len(resultado.InsertedIDs), nil
}
