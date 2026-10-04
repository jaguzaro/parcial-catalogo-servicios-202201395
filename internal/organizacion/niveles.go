// Package organizacion mantiene empresa, area, departamento, seccion y puesto. Las reglas
// estan en docs/diseno/reglas.md: codigo unico dentro del padre, nada colgando de un padre
// inactivo y baja rechazada mientras haya dependientes activos.
package organizacion

import "fmt"

// Nivel describe una unidad de la jerarquia. Los nombres de tabla y columna son
// constantes de este archivo: nunca vienen de la peticion.
type Nivel struct {
	Recurso    string // segmento de la ruta: "empresas"
	Tabla      string // tabla y entidad: "empresa"
	Nombre     string // para mensajes: "area"
	Plural     string
	Femenino   bool   // concordancia del adjetivo: "areas activas"
	El, Un     string // articulos, que no siempre siguen al genero: "el area"
	Padre      *Nivel
	CampoPadre string // columna y campo JSON del padre: "empresa_id"

	// dependientes son las consultas que bloquean la baja. Cada una devuelve id, codigo
	// y nombre, con $1 = id del registro.
	dependientes []dependiente
}

type dependiente struct {
	entidad  string
	singular string // "area activa"
	plural   string // "areas activas"
	consulta string
}

var (
	Empresa = &Nivel{Recurso: "empresas", Tabla: "empresa", Nombre: "empresa", Plural: "empresas",
		Femenino: true, El: "la", Un: "una"}
	Area = &Nivel{Recurso: "areas", Tabla: "area", Nombre: "area", Plural: "areas",
		Femenino: true, El: "el", Un: "un", Padre: Empresa, CampoPadre: "empresa_id"}
	Departamento = &Nivel{Recurso: "departamentos", Tabla: "departamento", Nombre: "departamento", Plural: "departamentos",
		El: "el", Un: "un", Padre: Area, CampoPadre: "area_id"}
	Seccion = &Nivel{Recurso: "secciones", Tabla: "seccion", Nombre: "seccion", Plural: "secciones",
		Femenino: true, El: "la", Un: "una", Padre: Departamento, CampoPadre: "departamento_id"}
	Puesto = &Nivel{Recurso: "puestos", Tabla: "puesto", Nombre: "puesto", Plural: "puestos",
		El: "el", Un: "un", Padre: Seccion, CampoPadre: "seccion_id"}

	// Niveles en orden, de la empresa al puesto.
	Niveles = []*Nivel{Empresa, Area, Departamento, Seccion, Puesto}
)

func init() {
	hijo := func(n *Nivel) dependiente {
		fem := "activos"
		if n.Femenino {
			fem = "activas"
		}
		return dependiente{
			entidad:  n.Tabla,
			singular: n.Nombre + " " + fem[:len(fem)-1],
			plural:   n.Plural + " " + fem,
			consulta: fmt.Sprintf(`SELECT id, codigo, nombre FROM %s WHERE %s = $1 AND activo`, n.Tabla, n.CampoPadre),
		}
	}
	Empresa.dependientes = []dependiente{hijo(Area)}
	Area.dependientes = []dependiente{hijo(Departamento)}
	Departamento.dependientes = []dependiente{hijo(Seccion)}
	// Un servicio con activo DESCONOCIDO bloquea igual que S (D22).
	Seccion.dependientes = []dependiente{hijo(Puesto), {
		entidad:  "servicio_n2",
		singular: "servicio activo a su cargo",
		plural:   "servicios activos a su cargo",
		consulta: `SELECT id, codigo, nombre FROM servicio_n2 WHERE seccion_responsable_id = $1 AND activo <> 'N'`,
	}}
	// Del usuario se informa el nombre de usuario como codigo.
	Puesto.dependientes = []dependiente{{
		entidad:  "usuario",
		singular: "usuario activo",
		plural:   "usuarios activos",
		consulta: `SELECT id, usuario AS codigo, nombre FROM usuario WHERE puesto_id = $1 AND activo`,
	}}
}

// ancestros devuelve los niveles superiores, del padre directo a la empresa.
func (n *Nivel) ancestros() []*Nivel {
	var out []*Nivel
	for p := n.Padre; p != nil; p = p.Padre {
		out = append(out, p)
	}
	return out
}

// el devuelve "la empresa" o "el area".
func (n *Nivel) el() string { return n.El + " " + n.Nombre }

// un devuelve "una empresa" o "un area".
func (n *Nivel) un() string { return n.Un + " " + n.Nombre }

// inactivo devuelve "inactiva" o "inactivo".
func (n *Nivel) inactivo() string {
	if n.Femenino {
		return "inactiva"
	}
	return "inactivo"
}
