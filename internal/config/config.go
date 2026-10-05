// Package config lee y valida las variables de entorno del binario.
package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const sesionPorDefecto = 8 * time.Hour

// Config son los valores de entorno ya validados.
type Config struct {
	DatabaseURL  string
	ExcelPath    string
	SessionTTL   time.Duration
	CookieSecure bool
	// AutoInicio: al servir, importar el Excel y sembrar la demostracion. Por defecto true.
	AutoInicio bool
}

// Cargar lee la configuracion con la funcion de busqueda dada (os.LookupEnv en produccion).
// Reporta todos los problemas juntos, no solo el primero.
func Cargar(lookup func(string) (string, bool)) (*Config, error) {
	var problemas []string
	obtener := func(nombre string) string {
		v, _ := lookup(nombre)
		return strings.TrimSpace(v)
	}
	requerida := func(nombre string) string {
		v := obtener(nombre)
		if v == "" {
			problemas = append(problemas, fmt.Sprintf("falta la variable %s", nombre))
		}
		return v
	}

	cfg := &Config{
		DatabaseURL: requerida("DATABASE_URL"),
		ExcelPath:   requerida("EXCEL_PATH"),
		SessionTTL:  sesionPorDefecto,
		AutoInicio:  true,
	}

	if v := obtener("SESSION_TTL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			problemas = append(problemas, fmt.Sprintf("SESSION_TTL=%q no es una duracion positiva (ejemplo: 8h)", v))
		} else {
			cfg.SessionTTL = d
		}
	}
	if v := obtener("COOKIE_SECURE"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			problemas = append(problemas, fmt.Sprintf("COOKIE_SECURE=%q no es true ni false", v))
		} else {
			cfg.CookieSecure = b
		}
	}
	if v := obtener("AUTO_INICIO"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			problemas = append(problemas, fmt.Sprintf("AUTO_INICIO=%q no es true ni false", v))
		} else {
			cfg.AutoInicio = b
		}
	}

	if len(problemas) > 0 {
		return nil, errors.New("configuracion invalida: " + strings.Join(problemas, "; "))
	}
	return cfg, nil
}
