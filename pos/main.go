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

var db *sql.DB
var tmpl *template.Template

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
	tmpl = template.Must(template.New("").Funcs(template.FuncMap{
		"formatMoney": func(f float64) string { return fmt.Sprintf("$%.2f", f) },
		"formatTime":  func(s string) string { return s },
		"yesno":       func(s string) string { if s == "t" { return "Sí" }; return "No" },
	}).ParseFS(sub,
		"base.html", "login.html", "dashboard.html",
		"productos/list.html", "productos/form.html",
		"ventas/list.html", "ventas/pos.html",
		"clientes/list.html", "clientes/form.html",
		"tickets/list.html", "tickets/detail.html",
		"cajas/operacion.html",
		"proveedores/list.html",
		"reportes/dashboard.html",
		"usuarios/list.html",
	))

	mux := http.NewServeMux()

	staticSub, _ := fs.Sub(staticFS, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))

	mux.HandleFunc("GET /api/productos", handleProductosList)
	mux.HandleFunc("POST /api/productos", handleProductosCreate)
	mux.HandleFunc("GET /api/productos/{codigo}", handleProductosGet)
	mux.HandleFunc("PUT /api/productos/{codigo}", handleProductosUpdate)
	mux.HandleFunc("DELETE /api/productos/{codigo}", handleProductosDelete)

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

	mux.HandleFunc("GET /api/reportes/dashboard", handleReportesDashboard)
	mux.HandleFunc("GET /api/reportes/ventas-diarias", handleReportesVentasDiarias)
	mux.HandleFunc("GET /api/reportes/productos-mas-vendidos", handleReportesTopProductos)

	mux.HandleFunc("GET /", handleIndex)

	mux.HandleFunc("GET /login", handleLoginPage)
	mux.HandleFunc("POST /login", handleLogin)
	mux.HandleFunc("GET /logout", handleLogout)

	mux.HandleFunc("GET /productos", handleProductosPage)
	mux.HandleFunc("GET /productos/nuevo", handleProductoFormPage)
	mux.HandleFunc("GET /productos/{codigo}/editar", handleProductoEditPage)

	mux.HandleFunc("GET /ventas", handleVentasPage)
	mux.HandleFunc("GET /ventas/pos", handlePOSPage)

	mux.HandleFunc("GET /clientes", handleClientesPage)
	mux.HandleFunc("GET /clientes/nuevo", handleClienteFormPage)
	mux.HandleFunc("GET /clientes/{numero}/editar", handleClienteEditPage)

	mux.HandleFunc("GET /tickets", handleTicketsPage)
	mux.HandleFunc("GET /tickets/{id}", handleTicketDetailPage)

	mux.HandleFunc("GET /cajas", handleCajasPage)
	mux.HandleFunc("GET /reportes", handleReportesPage)
	mux.HandleFunc("GET /proveedores", handleProveedoresPage)
	mux.HandleFunc("GET /usuarios", handleUsuariosPage)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Abarrotes PDV corriendo en http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, withCORS(withAuth(mux))))
}

func migrate(db *sql.DB) error {
	statements := strings.Split(schemaSQL, ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("error executing %q: %w", stmt[:min(len(stmt), 60)], err)
		}
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM USUARIOS").Scan(&count)
	if count == 0 {
		db.Exec(`INSERT INTO USUARIOS (usuario, clave, activo, created_on) VALUES (?, ?, 't', ?)`,
			"admin", hashPassword("admin"), time.Now().Format("2006-01-02 15:04:05"))
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

func render(w http.ResponseWriter, name string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
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
