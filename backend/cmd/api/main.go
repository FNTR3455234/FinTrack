// Comando api: punto de entrada del servidor de FinTrack.
//
// Arranca en este orden: configuracion, bitacora, conexion a MongoDB, indices,
// router y servidor HTTP. Al recibir Ctrl+C o SIGTERM hace un apagado ordenado.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/FNTR3455234/FinTrack/backend/internal/config"
	"github.com/FNTR3455234/FinTrack/backend/internal/db"
	"github.com/FNTR3455234/FinTrack/backend/internal/rutas"
)

// tiempoApagado es lo que se espera a que terminen las peticiones en curso
// antes de cerrar a la fuerza.
const tiempoApagado = 10 * time.Second

func main() {
	cfg, err := config.Cargar()
	if err != nil {
		// Todavia no hay bitacora configurada, asi que se escribe directo.
		slog.Error("no se pudo cargar la configuracion", "detalle", err)
		os.Exit(1)
	}

	configurarBitacora(cfg)
	slog.Info("iniciando FinTrack", "version", config.Version, "modo", cfg.GinModo)

	// Contexto que se cancela solo con Ctrl+C (SIGINT) o con SIGTERM, que es lo
	// que manda Docker al detener el contenedor.
	ctx, detener := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer detener()

	conexion, err := db.Conectar(ctx, cfg.MongoURI, cfg.MongoBD)
	if err != nil {
		slog.Error("no se pudo conectar a MongoDB", "detalle", err)
		os.Exit(1)
	}
	slog.Info("conectado a MongoDB", "base", cfg.MongoBD)

	if err := db.CrearIndices(ctx, conexion.BD); err != nil {
		slog.Error("no se pudieron crear los indices", "detalle", err)
		os.Exit(1)
	}

	servidor := &http.Server{
		Addr:    cfg.Direccion(),
		Handler: rutas.Configurar(cfg, rutas.Dependencias{BD: conexion}),
		// Sin estos limites una conexion lenta puede quedarse tomada de forma
		// indefinida.
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// El servidor corre en su propia goroutine para que main pueda quedarse
	// esperando la señal de apagado.
	go func() {
		slog.Info("servidor escuchando", "direccion", servidor.Addr)
		if err := servidor.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("el servidor se detuvo por un error", "detalle", err)
			detener()
		}
	}()

	<-ctx.Done()
	apagar(servidor, conexion)
}

// apagar cierra primero el servidor HTTP y despues MongoDB.
//
// Ese orden importa: si se cerrara la base primero, las peticiones que todavia
// se estan atendiendo fallarian. Asi terminan de responder con la base viva.
func apagar(servidor *http.Server, conexion *db.Conexion) {
	slog.Info("apagando de forma ordenada", "espera", tiempoApagado.String())

	// Contexto nuevo: el anterior ya esta cancelado por la señal.
	ctx, cancelar := context.WithTimeout(context.Background(), tiempoApagado)
	defer cancelar()

	if err := servidor.Shutdown(ctx); err != nil {
		slog.Error("el servidor no cerro a tiempo", "detalle", err)
	} else {
		slog.Info("servidor HTTP cerrado")
	}

	if err := conexion.Cerrar(ctx); err != nil {
		slog.Error("la conexion a MongoDB no cerro bien", "detalle", err)
	} else {
		slog.Info("conexion a MongoDB cerrada")
	}

	slog.Info("hasta luego")
}

// configurarBitacora deja slog listo: JSON en produccion (para que lo lea una
// herramienta) y texto en desarrollo (para que lo lea una persona).
func configurarBitacora(cfg *config.Config) {
	nivel := slog.LevelDebug
	if cfg.EsProduccion() {
		nivel = slog.LevelInfo
	}
	opciones := &slog.HandlerOptions{Level: nivel}

	var manejador slog.Handler
	if cfg.EsProduccion() {
		manejador = slog.NewJSONHandler(os.Stdout, opciones)
	} else {
		manejador = slog.NewTextHandler(os.Stdout, opciones)
	}
	slog.SetDefault(slog.New(manejador))
}
