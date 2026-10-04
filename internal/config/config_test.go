package config

import (
	"strings"
	"testing"
	"time"
)

func entorno(m map[string]string) func(string) (string, bool) {
	return func(k string) (string, bool) {
		v, ok := m[k]
		return v, ok
	}
}

func TestCargarValoresPorDefecto(t *testing.T) {
	cfg, err := Cargar(entorno(map[string]string{
		"DATABASE_URL": "postgres://u:p@db:5432/x",
		"EXCEL_PATH":   "/app/data/x.xlsx",
	}))
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if cfg.SessionTTL != 8*time.Hour {
		t.Errorf("SessionTTL = %v, se esperaba 8h", cfg.SessionTTL)
	}
	if cfg.CookieSecure {
		t.Error("CookieSecure deberia ser false por defecto")
	}
}

func TestCargarValoresExplicitos(t *testing.T) {
	cfg, err := Cargar(entorno(map[string]string{
		"DATABASE_URL":  "postgres://u:p@db:5432/x",
		"EXCEL_PATH":    "/app/data/x.xlsx",
		"SESSION_TTL":   "30m",
		"COOKIE_SECURE": "true",
	}))
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if cfg.SessionTTL != 30*time.Minute || !cfg.CookieSecure {
		t.Errorf("valores mal leidos: %+v", cfg)
	}
}

func TestCargarFaltantesSeReportanJuntos(t *testing.T) {
	_, err := Cargar(entorno(map[string]string{"DATABASE_URL": "   "}))
	if err == nil {
		t.Fatal("se esperaba error")
	}
	for _, esperado := range []string{"DATABASE_URL", "EXCEL_PATH"} {
		if !strings.Contains(err.Error(), "falta la variable "+esperado) {
			t.Errorf("el mensaje no menciona %s: %v", esperado, err)
		}
	}
}

func TestCargarValoresInvalidos(t *testing.T) {
	casos := map[string]string{
		"SESSION_TTL":   "ocho horas",
		"COOKIE_SECURE": "quizas",
	}
	for nombre, valor := range casos {
		env := map[string]string{"DATABASE_URL": "x", "EXCEL_PATH": "y", nombre: valor}
		_, err := Cargar(entorno(env))
		if err == nil || !strings.Contains(err.Error(), nombre) {
			t.Errorf("%s=%q: se esperaba error que mencione la variable, llego %v", nombre, valor, err)
		}
	}
	env := map[string]string{"DATABASE_URL": "x", "EXCEL_PATH": "y", "SESSION_TTL": "-1h"}
	if _, err := Cargar(entorno(env)); err == nil {
		t.Error("una duracion negativa deberia fallar")
	}
}
