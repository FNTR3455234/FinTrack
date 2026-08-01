package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// texto devuelve la variable de entorno o el valor por defecto si viene vacia.
func texto(clave, porDefecto string) string {
	valor := strings.TrimSpace(os.Getenv(clave))
	if valor == "" {
		return porDefecto
	}
	return valor
}

// entero convierte la variable a numero. Si trae basura devuelve error en vez
// de caer al valor por defecto en silencio: un typo debe notarse al arrancar.
func entero(clave string, porDefecto int) (int, error) {
	valor := strings.TrimSpace(os.Getenv(clave))
	if valor == "" {
		return porDefecto, nil
	}
	numero, err := strconv.Atoi(valor)
	if err != nil {
		return 0, fmt.Errorf("%s debe ser un numero entero, se recibio %q", clave, valor)
	}
	return numero, nil
}

// lista parte la variable por comas y limpia los espacios de cada elemento.
// Se usa para CORS_ORIGENES: "http://a.com, http://b.com".
func lista(clave, porDefecto string) []string {
	valor := texto(clave, porDefecto)
	partes := strings.Split(valor, ",")

	limpias := make([]string, 0, len(partes))
	for _, p := range partes {
		if p = strings.TrimSpace(p); p != "" {
			limpias = append(limpias, p)
		}
	}
	return limpias
}
