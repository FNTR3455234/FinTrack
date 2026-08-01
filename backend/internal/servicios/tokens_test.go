package servicios

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	secretoAccesoPrueba   = "secreto_de_acceso_para_pruebas_1234567890"
	secretoRefrescoPrueba = "secreto_de_refresco_para_pruebas_09876543"
)

func tokensDePrueba() *Tokens {
	return NuevoTokens(secretoAccesoPrueba, secretoRefrescoPrueba, 15, 7)
}

func TestGenerarYValidarAcceso_DevuelveElMismoUsuario(t *testing.T) {
	tokens := tokensDePrueba()
	usuarioID := bson.NewObjectID()

	token, err := tokens.GenerarAcceso(usuarioID)
	require.NoError(t, err)

	obtenido, err := tokens.ValidarAcceso(token)

	require.NoError(t, err)
	assert.Equal(t, usuarioID, obtenido)
}

func TestGenerarYValidarRefresco_DevuelveElMismoUsuario(t *testing.T) {
	tokens := tokensDePrueba()
	usuarioID := bson.NewObjectID()

	token, err := tokens.GenerarRefresco(usuarioID)
	require.NoError(t, err)

	obtenido, err := tokens.ValidarRefresco(token)

	require.NoError(t, err)
	assert.Equal(t, usuarioID, obtenido)
}

func TestValidar_UnTokenDeRefrescoNoSirveComoTokenDeAcceso(t *testing.T) {
	// Esta es la razon de usar dos secretos distintos: aunque alguien mande el
	// token largo en el encabezado Authorization, no pasa.
	tokens := tokensDePrueba()
	refresco, err := tokens.GenerarRefresco(bson.NewObjectID())
	require.NoError(t, err)

	_, err = tokens.ValidarAcceso(refresco)

	assert.ErrorIs(t, err, ErrTokenInvalido)
}

func TestValidar_UnTokenDeAccesoNoSirveParaRefrescar(t *testing.T) {
	tokens := tokensDePrueba()
	acceso, err := tokens.GenerarAcceso(bson.NewObjectID())
	require.NoError(t, err)

	_, err = tokens.ValidarRefresco(acceso)

	assert.ErrorIs(t, err, ErrTokenInvalido)
}

func TestValidar_RechazaUnTokenFirmadoConOtroSecreto(t *testing.T) {
	otros := NuevoTokens("un_secreto_completamente_distinto_12345678", secretoRefrescoPrueba, 15, 7)
	token, err := otros.GenerarAcceso(bson.NewObjectID())
	require.NoError(t, err)

	_, err = tokensDePrueba().ValidarAcceso(token)

	assert.ErrorIs(t, err, ErrTokenInvalido)
}

func TestValidar_RechazaUnTokenAlteradoOBasura(t *testing.T) {
	tokens := tokensDePrueba()
	valido, err := tokens.GenerarAcceso(bson.NewObjectID())
	require.NoError(t, err)

	casos := map[string]string{
		"vacio":              "",
		"basura":             "esto-no-es-un-token",
		"firma cambiada":     valido[:len(valido)-4] + "AAAA",
		"solo dos secciones": "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ4In0",
	}

	for nombre, token := range casos {
		t.Run(nombre, func(t *testing.T) {
			_, err := tokens.ValidarAcceso(token)
			assert.ErrorIs(t, err, ErrTokenInvalido)
		})
	}
}

func TestValidar_DistingueUnTokenVencido(t *testing.T) {
	// Duracion negativa: nace vencido. El middleware necesita distinguir este
	// caso para decirle al frontend que intente refrescar.
	vencidos := &Tokens{
		secretoAcceso:  []byte(secretoAccesoPrueba),
		duracionAcceso: -time.Hour,
	}
	token, err := vencidos.GenerarAcceso(bson.NewObjectID())
	require.NoError(t, err)

	_, err = vencidos.ValidarAcceso(token)

	assert.ErrorIs(t, err, ErrTokenVencido)
}

func TestValidar_RechazaElAlgoritmoNone(t *testing.T) {
	// Ataque clasico: quitar la firma y declarar alg="none". El validador
	// exige HMAC, asi que no pasa.
	afirmaciones := Afirmaciones{
		Tipo: TipoAcceso,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   bson.NewObjectID().Hex(),
			Issuer:    emisor,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	sinFirma, err := jwt.NewWithClaims(jwt.SigningMethodNone, afirmaciones).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	_, err = tokensDePrueba().ValidarAcceso(sinFirma)

	assert.ErrorIs(t, err, ErrTokenInvalido)
}

func TestValidar_RechazaUnTokenDeOtroEmisor(t *testing.T) {
	afirmaciones := Afirmaciones{
		Tipo: TipoAcceso,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   bson.NewObjectID().Hex(),
			Issuer:    "otra-aplicacion",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	ajeno, err := jwt.NewWithClaims(jwt.SigningMethodHS256, afirmaciones).
		SignedString([]byte(secretoAccesoPrueba))
	require.NoError(t, err)

	_, err = tokensDePrueba().ValidarAcceso(ajeno)

	assert.ErrorIs(t, err, ErrTokenInvalido)
}

func TestValidar_RechazaUnSubjectQueNoEsObjectID(t *testing.T) {
	afirmaciones := Afirmaciones{
		Tipo: TipoAcceso,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "no-es-un-object-id",
			Issuer:    emisor,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	raro, err := jwt.NewWithClaims(jwt.SigningMethodHS256, afirmaciones).
		SignedString([]byte(secretoAccesoPrueba))
	require.NoError(t, err)

	_, err = tokensDePrueba().ValidarAcceso(raro)

	assert.ErrorIs(t, err, ErrTokenInvalido)
}

func TestSegundosDeAcceso_ReflejaLaConfiguracion(t *testing.T) {
	assert.Equal(t, 15*60, NuevoTokens("a", "b", 15, 7).SegundosDeAcceso())
	assert.Equal(t, 30*60, NuevoTokens("a", "b", 30, 7).SegundosDeAcceso())
}
