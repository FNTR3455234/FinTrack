package repositorios

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/db"
	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// Pruebas de integracion: necesitan MongoDB. Se saltan solas si no esta la
// variable MONGO_URI_PRUEBAS. Para correrlas:
//
//	make up
//	MONGO_URI_PRUEBAS="mongodb://fintrack_admin:fintrack_dev_2026@localhost:27017/?authSource=admin" \
//	  go test ./internal/repositorios/...
const bdDePruebas = "fintrack_pruebas_repos"

// repositorioDePrueba abre la conexion contra una base aparte, crea los indices
// y se encarga de borrar todo al terminar.
func repositorioDePrueba(t *testing.T) *Usuarios {
	t.Helper()

	uri := os.Getenv("MONGO_URI_PRUEBAS")
	if uri == "" {
		t.Skip("sin MONGO_URI_PRUEBAS: se omiten las pruebas de integracion")
	}

	ctx, cancelar := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelar()

	conexion, err := db.Conectar(ctx, uri, bdDePruebas)
	require.NoError(t, err)

	// Hacen falta los indices: sin el unico de email, la prueba de duplicados
	// no probaria nada.
	require.NoError(t, db.CrearIndices(ctx, conexion.BD))

	t.Cleanup(func() {
		limpieza, cancelar := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancelar()
		_ = conexion.BD.Drop(limpieza)
		_ = conexion.Cerrar(limpieza)
	})

	return NuevoUsuarios(conexion.BD)
}

func usuarioDePrueba(email string) *modelos.Usuario {
	return &modelos.Usuario{
		Nombre:        "Usuario de Prueba",
		Email:         email,
		Password:      "$2a$10$hashfalsoperoconlaformacorrecta1234567890abcdef",
		Moneda:        "MXN",
		FechaRegistro: time.Now().UTC().Truncate(time.Millisecond),
		Activo:        true,
	}
}

func TestCrear_GuardaElUsuarioYLeAsignaElID(t *testing.T) {
	repo := repositorioDePrueba(t)
	usuario := usuarioDePrueba("nuevo@fintrack.mx")

	err := repo.Crear(context.Background(), usuario)

	require.NoError(t, err)
	assert.NotEqual(t, bson.NilObjectID, usuario.ID, "MongoDB debe devolver el _id generado")
}

func TestCrear_ElIndiceUnicoImpideDosCorreosIguales(t *testing.T) {
	// Es la razon de no comprobar antes si el correo existe: dos registros
	// simultaneos pasarian la comprobacion los dos, pero el indice no falla.
	repo := repositorioDePrueba(t)
	require.NoError(t, repo.Crear(context.Background(), usuarioDePrueba("repetido@fintrack.mx")))

	err := repo.Crear(context.Background(), usuarioDePrueba("repetido@fintrack.mx"))

	assert.ErrorIs(t, err, ErrDuplicado)
}

func TestPorEmail_EncuentraAlUsuarioGuardado(t *testing.T) {
	repo := repositorioDePrueba(t)
	original := usuarioDePrueba("buscame@fintrack.mx")
	require.NoError(t, repo.Crear(context.Background(), original))

	encontrado, err := repo.PorEmail(context.Background(), "buscame@fintrack.mx")

	require.NoError(t, err)
	assert.Equal(t, original.ID, encontrado.ID)
	assert.Equal(t, "Usuario de Prueba", encontrado.Nombre)
	assert.Equal(t, original.Password, encontrado.Password, "el hash se guarda tal cual")
	assert.True(t, encontrado.Activo)
}

func TestPorEmail_DevuelveErrNoEncontradoCuandoNoExiste(t *testing.T) {
	repo := repositorioDePrueba(t)

	_, err := repo.PorEmail(context.Background(), "fantasma@fintrack.mx")

	assert.ErrorIs(t, err, ErrNoEncontrado)
}

func TestPorID_EncuentraAlUsuarioGuardado(t *testing.T) {
	repo := repositorioDePrueba(t)
	original := usuarioDePrueba("porid@fintrack.mx")
	require.NoError(t, repo.Crear(context.Background(), original))

	encontrado, err := repo.PorID(context.Background(), original.ID)

	require.NoError(t, err)
	assert.Equal(t, "porid@fintrack.mx", encontrado.Email)
}

func TestPorID_DevuelveErrNoEncontradoConUnIDInventado(t *testing.T) {
	repo := repositorioDePrueba(t)

	_, err := repo.PorID(context.Background(), bson.NewObjectID())

	assert.ErrorIs(t, err, ErrNoEncontrado)
}

func TestActualizar_CambiaNombreYMonedaYDevuelveLoNuevo(t *testing.T) {
	repo := repositorioDePrueba(t)
	original := usuarioDePrueba("editar@fintrack.mx")
	require.NoError(t, repo.Crear(context.Background(), original))

	actualizado, err := repo.Actualizar(context.Background(), original.ID, "Nombre Nuevo", "USD")

	require.NoError(t, err)
	assert.Equal(t, "Nombre Nuevo", actualizado.Nombre)
	assert.Equal(t, "USD", actualizado.Moneda)
	// Lo que no se pidio cambiar sigue igual.
	assert.Equal(t, "editar@fintrack.mx", actualizado.Email)
	assert.Equal(t, original.Password, actualizado.Password)

	// Y quedo guardado de verdad, no solo en la respuesta.
	releido, err := repo.PorID(context.Background(), original.ID)
	require.NoError(t, err)
	assert.Equal(t, "Nombre Nuevo", releido.Nombre)
}

func TestActualizar_ConUnIDQueNoExisteDevuelveErrNoEncontrado(t *testing.T) {
	repo := repositorioDePrueba(t)

	_, err := repo.Actualizar(context.Background(), bson.NewObjectID(), "X Y", "MXN")

	assert.ErrorIs(t, err, ErrNoEncontrado)
}

func TestCrear_RespetaElEsquemaDeLaColeccion(t *testing.T) {
	// La base valida con $jsonSchema. Este documento cumple, asi que debe pasar
	// y quedar legible con los mismos nombres de campo en español.
	repo := repositorioDePrueba(t)
	usuario := usuarioDePrueba("esquema@fintrack.mx")
	require.NoError(t, repo.Crear(context.Background(), usuario))

	var crudo bson.M
	err := repo.coleccion.FindOne(context.Background(), bson.M{"_id": usuario.ID}).Decode(&crudo)

	require.NoError(t, err)
	for _, campo := range []string{"nombre", "email", "password", "moneda", "fecha_registro", "activo"} {
		assert.Contains(t, crudo, campo, "el documento debe guardar el campo %q", campo)
	}
}
