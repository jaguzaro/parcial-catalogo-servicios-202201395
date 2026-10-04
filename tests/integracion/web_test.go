package integracion

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	apihttp "catalogo/internal/http"
)

// TestWeb_RutasUsadasExisten lee las rutas que declara web/src/api.ts y exige que cada una
// este registrada en el servidor con el mismo metodo y el mismo patron. Si la interfaz
// llama a un endpoint que no existe, falla. Ademas exige que ningun otro archivo de la
// interfaz escriba rutas /api/ por su cuenta: toda peticion pasa por api.ts.
func TestWeb_RutasUsadasExisten(t *testing.T) {
	raiz := filepath.Join("..", "..", "web", "src")
	fuente, err := os.ReadFile(filepath.Join(raiz, "api.ts"))
	if err != nil {
		t.Fatalf("no se pudo leer web/src/api.ts: %v", err)
	}

	registradas := map[string]bool{}
	for _, r := range apihttp.RutasAPI() {
		registradas[r.Metodo+" "+r.Patron] = true
	}

	entrada := regexp.MustCompile(`metodo:\s*'([A-Z]+)',\s*ruta:\s*'([^']+)'`)
	usadas := entrada.FindAllStringSubmatch(string(fuente), -1)
	if len(usadas) == 0 {
		t.Fatal("api.ts no declara ninguna ruta con el formato { metodo: '...', ruta: '...' }")
	}
	for _, u := range usadas {
		if clave := u[1] + " " + u[2]; !registradas[clave] {
			t.Errorf("la interfaz usa %s y el servidor no la registra", clave)
		}
	}

	// Cada ruta /api/ que aparece en api.ts tiene que ser una de las declaradas arriba.
	declaradas := len(usadas)
	if todas := regexp.MustCompile(`'/api/[^']*'`).FindAllString(string(fuente), -1); len(todas) != declaradas {
		t.Errorf("api.ts tiene %d rutas /api/ entre comillas y solo %d declaradas con metodo", len(todas), declaradas)
	}

	err = filepath.WalkDir(raiz, func(ruta string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Base(ruta) == "api.ts" {
			return err
		}
		contenido, err := os.ReadFile(ruta)
		if err != nil {
			return err
		}
		if strings.Contains(string(contenido), "/api/") {
			t.Errorf("%s escribe una ruta /api/ suelta: debe pasar por api.ts", ruta)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
