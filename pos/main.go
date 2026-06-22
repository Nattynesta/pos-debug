package main

import (
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

//go:embed templates
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

//go:embed schema.sql
var schemaSQL string

//go:embed seed.sql
var seedSQL string



var db *sql.DB
var tmpl *template.Template
var pageTmpls map[string]*template.Template

func main() {
	var err error

	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".abarrotes-pdv")
	os.MkdirAll(dataDir, 0755)
	dbPath := filepath.Join(dataDir, "pdv.db")

	db, err = sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)")
	if err != nil {
		log.Fatalf("Error opening DB: %v", err)
	}
	defer db.Close()

	if err = migrate(db); err != nil {
		log.Fatalf("Error migrating: %v", err)
	}

	sub, _ := fs.Sub(templateFS, "templates")
	baseBytes, _ := fs.ReadFile(sub, "base.html")
	loginBytes, _ := fs.ReadFile(sub, "login.html")
	fmap := template.FuncMap{
		"formatMoney": func(f float64) string { return fmt.Sprintf("$%.2f", f) },
		"formatTime":  func(s string) string { return s },
		"yesno":       func(s string) string { if s == "t" { return "Sí" }; return "No" },
	}
	// Standalone templates (login.html) in shared set
	tmpl = template.New("").Funcs(fmap)
	template.Must(tmpl.New("login.html").Parse(string(loginBytes)))
	// Page templates: each gets its own namespace to avoid define "content" collision
	pageTmpls = make(map[string]*template.Template)
	fs.WalkDir(sub, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil { return err }
		if d.IsDir() || !strings.HasSuffix(path, ".html") { return nil }
		if path == "base.html" || path == "login.html" { return nil }
		b, err := fs.ReadFile(sub, path)
		if err != nil { return err }
		combined := `{{define "base.html"}}` + string(baseBytes) + `{{end}}` + string(b)
		t := template.Must(template.New(path).Funcs(fmap).Parse(combined))
		pageTmpls[path] = t
		return nil
	})

	mux := http.NewServeMux()

	staticSub, _ := fs.Sub(staticFS, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))

	mux.HandleFunc("GET /api/productos", handleProductosList)
	mux.HandleFunc("POST /api/productos", handleProductosCreate)
	mux.HandleFunc("GET /api/productos/{codigo}", handleProductosGet)
	mux.HandleFunc("PUT /api/productos/{codigo}", handleProductosUpdate)
	mux.HandleFunc("DELETE /api/productos/{codigo}", handleProductosDelete)
	mux.HandleFunc("POST /api/productos/{codigo}/imagen", handleProductoUploadImagen)

	mux.HandleFunc("GET /api/clientes", handleClientesList)
	mux.HandleFunc("POST /api/clientes", handleClientesCreate)
	mux.HandleFunc("GET /api/clientes/{numero}", handleClientesGet)
	mux.HandleFunc("PUT /api/clientes/{numero}", handleClientesUpdate)
	mux.HandleFunc("DELETE /api/clientes/{numero}", handleClientesDelete)
	mux.HandleFunc("GET /api/clientes/search", handleClientesSearch)

	mux.HandleFunc("GET /api/proveedores", handleProveedoresList)
	mux.HandleFunc("POST /api/proveedores", handleProveedoresCreate)
	mux.HandleFunc("GET /api/proveedores/{num}", handleProveedoresGet)
	mux.HandleFunc("PUT /api/proveedores/{num}", handleProveedoresUpdate)

	mux.HandleFunc("GET /api/departamentos", handleDepartamentosList)
	mux.HandleFunc("POST /api/departamentos", handleDepartamentosCreate)
	mux.HandleFunc("PUT /api/departamentos/{id}", handleDepartamentosUpdate)

	mux.HandleFunc("GET /api/medidas", handleMedidasList)
	mux.HandleFunc("POST /api/medidas", handleMedidasCreate)

	mux.HandleFunc("GET /api/usuarios", handleUsuariosList)
	mux.HandleFunc("POST /api/usuarios", handleUsuariosCreate)
	mux.HandleFunc("PUT /api/usuarios/{id}", handleUsuariosUpdate)

	mux.HandleFunc("GET /api/cajas", handleCajasList)
	mux.HandleFunc("GET /api/cajas/default", handleCajaDefault)
	mux.HandleFunc("POST /api/cajas", handleCajasCreate)

	mux.HandleFunc("GET /api/operaciones", handleOperacionesList)
	mux.HandleFunc("POST /api/operaciones/abrir", handleOperacionAbrir)
	mux.HandleFunc("POST /api/operaciones/cerrar/{id}", handleOperacionCerrar)
	mux.HandleFunc("GET /api/operaciones/activa", handleOperacionActiva)

	mux.HandleFunc("GET /api/tickets", handleTicketsList)
	mux.HandleFunc("POST /api/tickets", handleTicketCrear)
	mux.HandleFunc("GET /api/tickets/{id}", handleTicketGet)
	mux.HandleFunc("POST /api/tickets/{id}/articulo", handleTicketAddArticulo)
	mux.HandleFunc("DELETE /api/tickets/{id}/articulo/{artId}", handleTicketRemoveArticulo)
	mux.HandleFunc("POST /api/tickets/{id}/cobrar", handleTicketCobrar)
	mux.HandleFunc("POST /api/tickets/{id}/cancelar", handleTicketCancelar)
	mux.HandleFunc("PUT /api/tickets/{id}/prioridad", handleTicketActualizarPrioridad)
	mux.HandleFunc("DELETE /api/tickets/{id}", handleTicketDelete)

	mux.HandleFunc("GET /api/movimientos", handleMovimientosList)
	mux.HandleFunc("POST /api/movimientos", handleMovimientoCrear)

	mux.HandleFunc("GET /api/inventario/historial", handleHistorialInventario)
	mux.HandleFunc("POST /api/inventario/ajustar", handleInventarioAjustar)

	mux.HandleFunc("GET /api/impuestos", handleImpuestosList)
	mux.HandleFunc("POST /api/impuestos", handleImpuestosCreate)
	mux.HandleFunc("PUT /api/impuestos/{id}", handleImpuestosUpdate)

	mux.HandleFunc("GET /api/promociones", handlePromocionesList)
	mux.HandleFunc("POST /api/promociones", handlePromocionesCreate)
	mux.HandleFunc("DELETE /api/promociones/{id}", handlePromocionesDelete)

	mux.HandleFunc("GET /api/off/sync", withAdmin(handleOffSync))
	mux.HandleFunc("POST /api/off/sync", withAdmin(handleOffSync))

	mux.HandleFunc("GET /api/reportes/dashboard", handleReportesDashboard)
	mux.HandleFunc("GET /api/reportes/ventas-diarias", handleReportesVentasDiarias)
	mux.HandleFunc("GET /api/reportes/productos-mas-vendidos", handleReportesTopProductos)

	mux.HandleFunc("GET /", handleIndex)

	mux.HandleFunc("GET /login", handleLoginPage)
	mux.HandleFunc("POST /login", handleLogin)
	mux.HandleFunc("GET /logout", handleLogout)

	mux.HandleFunc("GET /productos", withAdmin(handleProductosPage))
	mux.HandleFunc("GET /productos/nuevo", withAdmin(handleProductoFormPage))
	mux.HandleFunc("GET /productos/{codigo}/editar", withAdmin(handleProductoEditPage))

	mux.HandleFunc("GET /ventas", handleVentasPage)
	mux.HandleFunc("GET /ventas/pos", handlePOSPage)

	mux.HandleFunc("GET /clientes", withAdmin(handleClientesPage))
	mux.HandleFunc("GET /clientes/nuevo", withAdmin(handleClienteFormPage))
	mux.HandleFunc("GET /clientes/{numero}/editar", withAdmin(handleClienteEditPage))

	mux.HandleFunc("GET /tickets", handleTicketsPage)
	mux.HandleFunc("GET /tickets/{id}", handleTicketDetailPage)

	mux.HandleFunc("GET /cajas", withAdmin(handleCajasPage))
	mux.HandleFunc("GET /reportes", withAdmin(handleReportesPage))
	mux.HandleFunc("GET /proveedores", withAdmin(handleProveedoresPage))
	mux.HandleFunc("GET /usuarios", withAdmin(handleUsuariosPage))
	mux.HandleFunc("GET /departamentos", withAdmin(handleDepartamentosPage))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := fmt.Sprintf("0.0.0.0:%s", port)
	log.Printf("Abarrotes PDV corriendo en http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, withCORS(withAuth(mux))))
}

func migrate(db *sql.DB) error {
	clean := regexp.MustCompile(`(?m)^\s*--.*$`).ReplaceAllString(schemaSQL, "")
	clean = regexp.MustCompile(`/\*.*?\*/`).ReplaceAllString(clean, "")
	statements := strings.Split(clean, ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("error executing %q: %w", stmt[:min(len(stmt), 60)], err)
		}
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM USUARIOS").Scan(&count)
	if count == 0 {
		db.Exec(`INSERT INTO USUARIOS (usuario, clave, activo, created_on, rol) VALUES (?, ?, 't', ?, 'admin')`,
			"admin", hashPassword("admin"), time.Now().Format("2006-01-02 15:04:05"))
	}
	var helperExists int
	db.QueryRow("SELECT COUNT(*) FROM USUARIOS WHERE usuario='helper'").Scan(&helperExists)
	if helperExists == 0 {
		db.Exec(`INSERT INTO USUARIOS (usuario, clave, activo, created_on, rol) VALUES (?, ?, 't', ?, 'helper')`,
			"helper", hashPassword("helper"), time.Now().Format("2006-01-02 15:04:05"))
	}

	var hasRol int
	db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('USUARIOS') WHERE name='rol'").Scan(&hasRol)
	if hasRol == 0 {
		db.Exec(`ALTER TABLE USUARIOS ADD COLUMN rol TEXT DEFAULT 'helper'`)
	}

	productoColumns := []string{"imagen_local", "marca", "categorias", "ingredientes", "nutriscore", "cantidad_presentacion", "nutricion", "off_image_url", "off_image_small"}
	for _, col := range productoColumns {
		var hasCol int
		db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('PRODUCTOS') WHERE name=?", col).Scan(&hasCol)
		if hasCol == 0 {
			db.Exec("ALTER TABLE PRODUCTOS ADD COLUMN "+col+" TEXT DEFAULT ''")
		}
	}

	var hasOff int
	db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='productos_openfoods'").Scan(&hasOff)
	if hasOff == 0 {
		db.Exec(`CREATE TABLE productos_openfoods (codigo TEXT PRIMARY KEY, nombre TEXT, marca TEXT, categorias TEXT, ingredientes TEXT, nutricion TEXT, nutriscore TEXT, cantidad_presentacion TEXT, imagen_url TEXT, imagen_small TEXT, imagen_grande TEXT, updated_at TEXT)`)
	}
	// Legacy table from old schema
	db.Exec("CREATE TABLE IF NOT EXISTS PRODUCTOS_OFF (codigo TEXT PRIMARY KEY, image_url TEXT, image_small TEXT, name TEXT, last_sync TEXT)")

	var hasPrioridad int
	db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('VENTATICKETS') WHERE name='prioridad'").Scan(&hasPrioridad)
	if hasPrioridad == 0 {
		db.Exec(`ALTER TABLE VENTATICKETS ADD COLUMN prioridad INTEGER DEFAULT 0`)
	}

	var productCount int
	db.QueryRow("SELECT COUNT(*) FROM PRODUCTOS").Scan(&productCount)
	if productCount == 0 {
		cleanSeed := regexp.MustCompile(`(?m)^\s*--.*$`).ReplaceAllString(seedSQL, "")
		seedStatements := strings.Split(cleanSeed, ";")
		for _, stmt := range seedStatements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := db.Exec(stmt); err != nil {
				return fmt.Errorf("error seeding: %q: %w", stmt[:min(len(stmt), 60)], err)
			}
		}
	}

	return nil
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(200)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/static/") || strings.HasPrefix(r.URL.Path, "/login") || strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		cookie, err := r.Cookie("session")
		if err != nil || cookie.Value == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleCookie, err := r.Cookie("role")
		if err != nil || roleCookie.Value != "admin" {
			http.Redirect(w, r, "/ventas/pos", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

func isAdmin(r *http.Request) bool {
	roleCookie, err := r.Cookie("role")
	return err == nil && roleCookie.Value == "admin"
}

func isHelperOrAdmin(r *http.Request) bool {
	roleCookie, err := r.Cookie("role")
	if err != nil {
		return false
	}
	return roleCookie.Value == "admin" || roleCookie.Value == "helper"
}

func render(w http.ResponseWriter, r *http.Request, name string, data PageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if cookie, err := r.Cookie("session"); err == nil {
		data.User = cookie.Value
	}
	if rc, err := r.Cookie("role"); err == nil {
		data.Role = rc.Value
	}
	var err error
	if t, ok := pageTmpls[name]; ok {
		err = t.Execute(w, data)
	} else {
		err = tmpl.ExecuteTemplate(w, name, data)
	}
	if err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, err.Error(), 500)
	}
}

func jsonResp(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonErr(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func queryInt(db *sql.DB, q string, args ...interface{}) int {
	var v int
	db.QueryRow(q, args...).Scan(&v)
	return v
}

func queryFloat(db *sql.DB, q string, args ...interface{}) float64 {
	var v float64
	db.QueryRow(q, args...).Scan(&v)
	return v
}

func parseFormFloat(r *http.Request, key string) (float64, error) {
	return strconv.ParseFloat(r.FormValue(key), 64)
}
