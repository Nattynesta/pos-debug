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
var negociosName string

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

	migrateLegacyPagos()

	negociosName = os.Getenv("NEGOCIO_NAME")
	if negociosName == "" {
		negociosName = "ABARROTES PDV"
	}

	if err := initCSRF(); err != nil {
		log.Fatalf("Error initializing CSRF: %v", err)
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
	localStatic := http.Dir("static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "" || r.URL.Path == "/" {
			http.NotFound(w, r)
			return
		}
		f, err := localStatic.Open(r.URL.Path)
		if err == nil {
			f.Close()
			http.FileServer(localStatic).ServeHTTP(w, r)
			return
		}
		http.FileServer(http.FS(staticSub)).ServeHTTP(w, r)
	})))

	mux.HandleFunc("GET /api/productos", handleProductosList)
	mux.HandleFunc("POST /api/productos", handleProductosCreate)
	mux.HandleFunc("GET /api/productos/{codigo}", handleProductosGet)
	mux.HandleFunc("PUT /api/productos/{codigo}", handleProductosUpdate)
	mux.HandleFunc("DELETE /api/productos/{codigo}", handleProductosDelete)
	mux.HandleFunc("POST /api/productos/{codigo}/imagen", handleProductoUploadImagen)
	mux.HandleFunc("GET /api/productos/barcode/{codigo}", handleBarcodeLookup)
	mux.HandleFunc("GET /api/productos/search", handleProductosSearch)

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
	mux.HandleFunc("GET /api/proveedores/{num}/productos", handleProveedoresProductos)
	mux.HandleFunc("POST /api/proveedores/{num}/recibir", handleProveedoresRecibir)

	mux.HandleFunc("GET /api/departamentos", handleDepartamentosList)
	mux.HandleFunc("POST /api/departamentos", handleDepartamentosCreate)
	mux.HandleFunc("PUT /api/departamentos/{id}", handleDepartamentosUpdate)

	mux.HandleFunc("GET /api/medidas", handleMedidasList)
	mux.HandleFunc("POST /api/medidas", handleMedidasCreate)

	mux.HandleFunc("GET /api/usuarios", handleUsuariosList)
	mux.HandleFunc("POST /api/usuarios", handleUsuariosCreate)
	mux.HandleFunc("PUT /api/usuarios/{id}", handleUsuariosUpdate)
	mux.HandleFunc("PUT /api/usuarios/{id}/password", handleUsuarioPassword)

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
	mux.HandleFunc("GET /api/tickets/{id}/pagos", handleTicketPagosList)
	mux.HandleFunc("GET /api/tickets/{id}/print", handleTicketPrint)
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

	mux.HandleFunc("GET /api/pedidos", handlePedidosList)
	mux.HandleFunc("POST /api/pedidos", handlePedidosCreate)
	mux.HandleFunc("PUT /api/pedidos/{id}/asignar", handlePedidosAsignar)
	mux.HandleFunc("PUT /api/pedidos/{id}/estado", handlePedidosEstado)
	mux.HandleFunc("GET /api/pedidos/estadisticas", handlePedidosStats)

	mux.HandleFunc("GET /api/off/sync", withAdmin(handleOffSync))
	mux.HandleFunc("POST /api/off/sync", withAdmin(handleOffSync))

	mux.HandleFunc("GET /api/reportes/dashboard", handleReportesDashboard)
	mux.HandleFunc("GET /api/reportes/ventas-diarias", handleReportesVentasDiarias)
	mux.HandleFunc("GET /api/reportes/productos-mas-vendidos", handleReportesTopProductos)
	mux.HandleFunc("POST /api/admin/reset-ventas", withAdmin(handleAdminResetVentas))
	mux.HandleFunc("GET /api/chat/canales", handleChatCanales)
	mux.HandleFunc("GET /api/chat/mensajes", handleChatMensajes)
	mux.HandleFunc("POST /api/chat/mensajes", handleChatMensajes)
	mux.HandleFunc("DELETE /api/chat/mensajes", handleChatMensajes)
	mux.HandleFunc("PUT /api/chat/leido", handleChatLeido)
	mux.HandleFunc("GET /api/chat/usuarios", handleChatUsuarios)
	mux.HandleFunc("GET /api/chat/online", handleChatOnline)
	mux.HandleFunc("GET /api/chat/ws", handleChatWS)

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
	mux.HandleFunc("GET /pedidos", handlePedidosPage)
	mux.HandleFunc("GET /chat", handleChatPage)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	go runWSHub()

	addr := fmt.Sprintf("0.0.0.0:%s", port)
	log.Printf("Abarrotes PDV corriendo en http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, withRateLimit(withCSRF(withCORS(withAuth(mux))))))
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
		adminHash, _ := HashPassword("admin")
		db.Exec(`INSERT INTO USUARIOS (usuario, clave, activo, created_on, rol) VALUES (?, ?, 't', ?, 'admin')`,
			"admin", adminHash, time.Now().Format("2006-01-02 15:04:05"))
	}
	var helperExists int
	db.QueryRow("SELECT COUNT(*) FROM USUARIOS WHERE usuario='helper'").Scan(&helperExists)
	if helperExists == 0 {
		helperHash, _ := HashPassword("helper")
		db.Exec(`INSERT INTO USUARIOS (usuario, clave, activo, created_on, rol) VALUES (?, ?, 't', ?, 'helper')`,
			"helper", helperHash, time.Now().Format("2006-01-02 15:04:05"))
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

	var hasPedidos int
	db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='PEDIDOS'").Scan(&hasPedidos)
	if hasPedidos == 0 {
		db.Exec(`CREATE TABLE PEDIDOS (id INTEGER PRIMARY KEY AUTOINCREMENT, items TEXT NOT NULL DEFAULT '[]', total REAL NOT NULL DEFAULT 0, prioridad TEXT NOT NULL DEFAULT 'media', notas TEXT DEFAULT '', cliente_nombre TEXT DEFAULT '', cliente_direccion TEXT DEFAULT '', cliente_telefono TEXT DEFAULT '', es_adeudo INTEGER DEFAULT 0, creado_por_id INTEGER NOT NULL, asignado_a_id INTEGER, estado TEXT NOT NULL DEFAULT 'pendiente', created_on TEXT NOT NULL DEFAULT (datetime('now','localtime')), completado_on TEXT, FOREIGN KEY (creado_por_id) REFERENCES USUARIOS(id), FOREIGN KEY (asignado_a_id) REFERENCES USUARIOS(id))`)
		db.Exec(`CREATE TABLE PEDIDOS_LOG (id INTEGER PRIMARY KEY AUTOINCREMENT, pedido_id INTEGER NOT NULL, usuario_id INTEGER NOT NULL, accion TEXT NOT NULL, created_on TEXT NOT NULL DEFAULT (datetime('now','localtime')), FOREIGN KEY (pedido_id) REFERENCES PEDIDOS(id), FOREIGN KEY (usuario_id) REFERENCES USUARIOS(id))`)
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

	// Chat tables
	db.Exec("CREATE TABLE IF NOT EXISTS chat_canales (id INTEGER PRIMARY KEY AUTOINCREMENT, nombre TEXT NOT NULL UNIQUE, icono TEXT DEFAULT 'hash', descripcion TEXT DEFAULT '', created_on TEXT DEFAULT (datetime('now','localtime')))")
	db.Exec("CREATE TABLE IF NOT EXISTS chat_leidos (usuario_id INTEGER NOT NULL, canal_id INTEGER NOT NULL, ultimo_leido_id INTEGER DEFAULT 0, PRIMARY KEY (usuario_id, canal_id), FOREIGN KEY (usuario_id) REFERENCES USUARIOS(id), FOREIGN KEY (canal_id) REFERENCES chat_canales(id))")

	var hasCanalID int
	db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('CHAT_MESSAGES') WHERE name='canal_id'").Scan(&hasCanalID)
	if hasCanalID == 0 {
		db.Exec("ALTER TABLE CHAT_MESSAGES ADD COLUMN canal_id INTEGER DEFAULT 1")
	}
	var hasTipo int
	db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('CHAT_MESSAGES') WHERE name='tipo'").Scan(&hasTipo)
	if hasTipo == 0 {
		db.Exec("ALTER TABLE CHAT_MESSAGES ADD COLUMN tipo TEXT DEFAULT ''")
	}
	var hasDatosJSON int
	db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('CHAT_MESSAGES') WHERE name='datos_json'").Scan(&hasDatosJSON)
	if hasDatosJSON == 0 {
		db.Exec("ALTER TABLE CHAT_MESSAGES ADD COLUMN datos_json TEXT DEFAULT ''")
	}

	// Seed default channels
	var canalCount int
	db.QueryRow("SELECT COUNT(*) FROM chat_canales").Scan(&canalCount)
	if canalCount == 0 {
		db.Exec("INSERT INTO chat_canales (nombre, icono, descripcion) VALUES ('General', 'hash', 'Chat general del equipo')")
		db.Exec("INSERT INTO chat_canales (nombre, icono, descripcion) VALUES ('Ventas', 'shopping-cart', 'Notificaciones y discusión de ventas')")
		db.Exec("INSERT INTO chat_canales (nombre, icono, descripcion) VALUES ('Inventario', 'package', 'Movimientos y ajustes de inventario')")
		db.Exec("INSERT INTO chat_canales (nombre, icono, descripcion) VALUES ('Admin', 'settings', 'Solo administradores')")
	}

	// FTS5 search index
	var hasFTS int
	db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='productos_fts'").Scan(&hasFTS)
	if hasFTS == 0 {
		db.Exec(`CREATE VIRTUAL TABLE productos_fts USING fts5(codigo, descripcion, categorias, marca, tokenize='unicode61')`)
	}
	db.Exec("DELETE FROM productos_fts")
	db.Exec(`INSERT INTO productos_fts (codigo, descripcion, categorias, marca) SELECT codigo, COALESCE(descripcion,''), COALESCE(categorias,''), COALESCE(p.marca,'') FROM PRODUCTOS p WHERE descripcion != '' OR codigo != ''`)

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
	tok := csrfToken()
	data.CSRFToken = tok
	http.SetCookie(w, &http.Cookie{Name: "csrf_token", Value: tok, Path: "/", HttpOnly: false})
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
