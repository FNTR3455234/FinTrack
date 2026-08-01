package db

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Estas son pruebas de integracion: necesitan un MongoDB de verdad.
//
// Se saltan solas si no esta la variable MONGO_URI_PRUEBAS, para que
// "go test ./..." siga funcionando en una maquina sin Docker. Para correrlas:
//
//	make up
//	MONGO_URI_PRUEBAS="mongodb://fintrack_admin:fintrack_dev_2026@localhost:27017/?authSource=admin" go test ./internal/db/...
const bdDePruebas = "fintrack_pruebas_db"

func uriDePruebas(t *testing.T) string {
	t.Helper()
	uri := os.Getenv("MONGO_URI_PRUEBAS")
	if uri == "" {
		t.Skip("sin MONGO_URI_PRUEBAS: se omiten las pruebas de integracion")
	}
	return uri
}

// conectarParaPrueba abre la conexion contra una base aparte y se encarga de
// borrarla y cerrar todo al terminar.
func conectarParaPrueba(t *testing.T) *Conexion {
	t.Helper()

	ctx, cancelar := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelar()

	conexion, err := Conectar(ctx, uriDePruebas(t), bdDePruebas)
	require.NoError(t, err)

	t.Cleanup(func() {
		limpieza, cancelar := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancelar()
		_ = conexion.BD.Drop(limpieza)
		_ = conexion.Cerrar(limpieza)
	})
	return conexion
}

func TestConectar_HaceUnPingAlServidor(t *testing.T) {
	conexion := conectarParaPrueba(t)

	assert.Equal(t, bdDePruebas, conexion.BD.Name())
	assert.NoError(t, conexion.Ping(context.Background()))
}

func TestConectar_FallaRapidoSiElServidorNoExiste(t *testing.T) {
	uriDePruebas(t) // solo para saltarse la prueba si no hay entorno

	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelar()

	inicio := time.Now()
	// Puerto donde no hay nada escuchando.
	_, err := Conectar(ctx, "mongodb://localhost:27099/?directConnection=true", bdDePruebas)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no hay respuesta de MongoDB")
	// No debe quedarse esperando los 30 s por defecto del driver.
	assert.Less(t, time.Since(inicio), 15*time.Second)
}

func TestCrearIndices_CreaLosSeisIndicesDeFinTrack(t *testing.T) {
	conexion := conectarParaPrueba(t)
	ctx := context.Background()

	require.NoError(t, CrearIndices(ctx, conexion.BD))

	esperados := map[string][]string{
		"usuarios":      {"idx_usuarios_email_unico"},
		"cuentas":       {"idx_cuentas_usuario"},
		"categorias":    {"idx_categorias_usuario"},
		"transacciones": {"idx_transacciones_usuario_fecha", "idx_transacciones_usuario_categoria"},
		"presupuestos":  {"idx_presupuestos_unico_periodo"},
	}

	for coleccion, nombres := range esperados {
		existentes := nombresDeIndices(t, conexion, coleccion)
		for _, nombre := range nombres {
			assert.Contains(t, existentes, nombre, "falta el indice en %q", coleccion)
		}
	}
}

func TestCrearIndices_SePuedeCorrerEnCadaArranque(t *testing.T) {
	conexion := conectarParaPrueba(t)
	ctx := context.Background()

	require.NoError(t, CrearIndices(ctx, conexion.BD))
	primeraVez := nombresDeIndices(t, conexion, "transacciones")

	// El servidor llama a CrearIndices en cada arranque: la segunda vez no
	// puede fallar ni duplicar nada.
	require.NoError(t, CrearIndices(ctx, conexion.BD))
	segundaVez := nombresDeIndices(t, conexion, "transacciones")

	assert.Equal(t, primeraVez, segundaVez)
}

func TestCrearIndices_ElIndiceDeEmailImpideDuplicados(t *testing.T) {
	conexion := conectarParaPrueba(t)
	ctx := context.Background()
	require.NoError(t, CrearIndices(ctx, conexion.BD))

	usuarios := conexion.BD.Collection("usuarios")
	_, err := usuarios.InsertOne(ctx, bson.M{"email": "repetido@fintrack.mx"})
	require.NoError(t, err)

	_, err = usuarios.InsertOne(ctx, bson.M{"email": "repetido@fintrack.mx"})

	require.Error(t, err, "el segundo insert con el mismo email debe fallar")
	assert.Contains(t, err.Error(), "duplicate key")
}

// nombresDeIndices devuelve los nombres de los indices de una coleccion.
func nombresDeIndices(t *testing.T, conexion *Conexion, coleccion string) []string {
	t.Helper()

	cursor, err := conexion.BD.Collection(coleccion).Indexes().List(context.Background())
	require.NoError(t, err)

	var indices []struct {
		Nombre string `bson:"name"`
	}
	require.NoError(t, cursor.All(context.Background(), &indices))

	nombres := make([]string, 0, len(indices))
	for _, i := range indices {
		nombres = append(nombres, i.Nombre)
	}
	return nombres
}
