package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// entornoMinimo deja en el entorno solo lo obligatorio, con valores validos.
// t.Setenv restaura el valor anterior al terminar cada prueba.
func entornoMinimo(t *testing.T) {
	t.Helper()
	t.Setenv("MONGO_URI", "mongodb://localhost:27017")
	t.Setenv("JWT_SECRETO_ACCESO", "secreto_de_acceso_para_pruebas_1234567890")
	t.Setenv("JWT_SECRETO_REFRESCO", "secreto_de_refresco_para_pruebas_098765432")
	// Se limpian las opcionales para que el entorno real de la maquina no
	// cambie el resultado de las pruebas.
	t.Setenv("PUERTO", "")
	t.Setenv("GIN_MODO", "")
	t.Setenv("MONGO_BD", "")
	t.Setenv("JWT_MINUTOS_ACCESO", "")
	t.Setenv("JWT_DIAS_REFRESCO", "")
	t.Setenv("CORS_ORIGENES", "")
}

func TestDesdeEntorno_UsaLosValoresPorDefecto(t *testing.T) {
	entornoMinimo(t)

	cfg, err := DesdeEntorno()

	require.NoError(t, err)
	assert.Equal(t, "8080", cfg.Puerto)
	assert.Equal(t, "debug", cfg.GinModo)
	assert.Equal(t, "fintrack", cfg.MongoBD)
	assert.Equal(t, 15, cfg.JWTMinutosAcceso)
	assert.Equal(t, 7, cfg.JWTDiasRefresco)
	assert.Equal(t, []string{"http://localhost:5173"}, cfg.CORSOrigenes)
}

func TestDesdeEntorno_LeeLosValoresDelEntorno(t *testing.T) {
	entornoMinimo(t)
	t.Setenv("PUERTO", "9000")
	t.Setenv("GIN_MODO", "release")
	t.Setenv("MONGO_BD", "fintrack_pruebas")
	t.Setenv("JWT_MINUTOS_ACCESO", "30")
	t.Setenv("JWT_DIAS_REFRESCO", "14")

	cfg, err := DesdeEntorno()

	require.NoError(t, err)
	assert.Equal(t, "9000", cfg.Puerto)
	assert.Equal(t, ":9000", cfg.Direccion())
	assert.True(t, cfg.EsProduccion())
	assert.Equal(t, "fintrack_pruebas", cfg.MongoBD)
	assert.Equal(t, 30, cfg.JWTMinutosAcceso)
	assert.Equal(t, 14, cfg.JWTDiasRefresco)
}

func TestDesdeEntorno_SeparaYLimpiaLosOrigenesCORS(t *testing.T) {
	entornoMinimo(t)
	t.Setenv("CORS_ORIGENES", " http://localhost:5173 , https://fintrack.mx ,, ")

	cfg, err := DesdeEntorno()

	require.NoError(t, err)
	assert.Equal(t, []string{"http://localhost:5173", "https://fintrack.mx"}, cfg.CORSOrigenes)
}

func TestDesdeEntorno_ReportaTodoLoQueFaltaDeUnaVez(t *testing.T) {
	t.Setenv("MONGO_URI", "")
	t.Setenv("JWT_SECRETO_ACCESO", "")
	t.Setenv("JWT_SECRETO_REFRESCO", "")

	cfg, err := DesdeEntorno()

	require.Error(t, err)
	assert.Nil(t, cfg)
	// El mensaje junta los tres problemas: se corrige el .env de una pasada.
	assert.Contains(t, err.Error(), "MONGO_URI es obligatoria")
	assert.Contains(t, err.Error(), "JWT_SECRETO_ACCESO es obligatoria")
	assert.Contains(t, err.Error(), "JWT_SECRETO_REFRESCO es obligatoria")
}

func TestDesdeEntorno_RechazaConfiguracionInvalida(t *testing.T) {
	casos := []struct {
		nombre   string
		variable string
		valor    string
		esperado string
	}{
		{"puerto no numerico", "PUERTO", "ocho mil", "PUERTO debe ser un numero"},
		{"modo desconocido", "GIN_MODO", "produccion", "GIN_MODO debe ser debug, release o test"},
		{"minutos no numericos", "JWT_MINUTOS_ACCESO", "quince", "JWT_MINUTOS_ACCESO debe ser un numero entero"},
		{"minutos en cero", "JWT_MINUTOS_ACCESO", "0", "JWT_MINUTOS_ACCESO debe ser mayor que 0"},
		{"dias negativos", "JWT_DIAS_REFRESCO", "-1", "JWT_DIAS_REFRESCO debe ser mayor que 0"},
		{"secreto corto", "JWT_SECRETO_ACCESO", "corto", "al menos 32 caracteres"},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			entornoMinimo(t)
			t.Setenv(caso.variable, caso.valor)

			_, err := DesdeEntorno()

			require.Error(t, err)
			assert.Contains(t, err.Error(), caso.esperado)
		})
	}
}

func TestDesdeEntorno_ExigeSecretosDistintos(t *testing.T) {
	entornoMinimo(t)
	mismo := "el_mismo_secreto_para_los_dos_1234567890"
	t.Setenv("JWT_SECRETO_ACCESO", mismo)
	t.Setenv("JWT_SECRETO_REFRESCO", mismo)

	_, err := DesdeEntorno()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "deben ser distintos")
}

func TestDesdeEntorno_NoDejaArrancarEnReleaseConLosSecretosDeEjemplo(t *testing.T) {
	entornoMinimo(t)
	t.Setenv("GIN_MODO", "release")
	t.Setenv("JWT_SECRETO_ACCESO", "cambia_este_secreto_de_acceso_en_produccion")

	_, err := DesdeEntorno()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "sigue con el valor de ejemplo")
}

func TestDesdeEntorno_EnDesarrolloSiAceptaLosSecretosDeEjemplo(t *testing.T) {
	entornoMinimo(t)
	t.Setenv("GIN_MODO", "debug")
	t.Setenv("JWT_SECRETO_ACCESO", "cambia_este_secreto_de_acceso_en_produccion")
	t.Setenv("JWT_SECRETO_REFRESCO", "cambia_este_secreto_de_refresco_en_produccion")

	_, err := DesdeEntorno()

	require.NoError(t, err)
}
