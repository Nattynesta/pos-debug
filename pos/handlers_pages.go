package main

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"strconv"
)

type PageData struct {
	Title    string
	Active   string
	Data     interface{}
	User     string
	Role     string
	Error    string
	Success  string
	OperacionActiva bool
	UserID   int
}

func getOperacionActiva() bool {
	var count int
	db.QueryRow("SELECT COUNT(*) FROM OPERACIONES WHERE abierta='t'").Scan(&count)
	return count > 0
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	d := DashboardReport{}
	db.QueryRow(`SELECT COUNT(*) FROM VENTATICKETS WHERE DATE(creado_en)=DATE('now')`).Scan(&d.VentasHoy)
	db.QueryRow(`SELECT COALESCE(SUM(total),0) FROM VENTATICKETS WHERE DATE(creado_en)=DATE('now') AND esta_cancelado='f'`).Scan(&d.IngresosHoy)
	db.QueryRow(`SELECT COALESCE(SUM(ganancia),0) FROM VENTATICKETS WHERE DATE(creado_en)=DATE('now') AND esta_cancelado='f'`).Scan(&d.GananciaHoy)
	db.QueryRow(`SELECT COUNT(*) FROM PRODUCTOS WHERE COALESCE(dinventario,0) > 0`).Scan(&d.ProductosStock)
	db.QueryRow(`SELECT COALESCE(SUM(dinventario * pcosto),0) FROM PRODUCTOS WHERE COALESCE(dinventario,0) > 0`).Scan(&d.ValorInventario)
	d.OperacionActiva = getOperacionActiva()
	render(w, r, "dashboard.html", PageData{Title: "Dashboard", Active: "dashboard", Data: d, OperacionActiva: d.OperacionActiva})
}

func handleLoginPage(w http.ResponseWriter, r *http.Request) {
	render(w, r, "login.html", PageData{Title: "Iniciar Sesion"})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	user := r.FormValue("usuario")
	pw := r.FormValue("clave")
	var id int
	var hash string
	var rol string
	err := db.QueryRow("SELECT id, clave, COALESCE(rol,'helper') FROM USUARIOS WHERE usuario=? AND activo='t'", user).Scan(&id, &hash, &rol)
	if err != nil {
		render(w, r, "login.html", PageData{Title: "Iniciar Sesion", Error: "Usuario o clave incorrectos"})
		return
	}
	h := sha256.Sum256([]byte(pw))
	if fmt.Sprintf("%x", h) != hash {
		render(w, r, "login.html", PageData{Title: "Iniciar Sesion", Error: "Usuario o clave incorrectos"})
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "session", Value: user, Path: "/"})
	http.SetCookie(w, &http.Cookie{Name: "user_id", Value: strconv.Itoa(id), Path: "/"})
	http.SetCookie(w, &http.Cookie{Name: "role", Value: rol, Path: "/"})
	if rol == "helper" {
		http.Redirect(w, r, "/ventas/pos", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Path: "/", MaxAge: -1})
	http.SetCookie(w, &http.Cookie{Name: "role", Value: "", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func handleProductosPage(w http.ResponseWriter, r *http.Request) {
	render(w, r, "productos/list.html", PageData{Title: "Productos", Active: "productos", OperacionActiva: getOperacionActiva()})
}

func handleProductoFormPage(w http.ResponseWriter, r *http.Request) {
	render(w, r, "productos/form.html", PageData{Title: "Nuevo Producto", Active: "productos", OperacionActiva: getOperacionActiva()})
}

func handleProductoEditPage(w http.ResponseWriter, r *http.Request) {
	codigo := r.PathValue("codigo")
	render(w, r, "productos/form.html", PageData{Title: "Editar Producto", Active: "productos", Data: codigo, OperacionActiva: getOperacionActiva()})
}

func handleVentasPage(w http.ResponseWriter, r *http.Request) {
	render(w, r, "ventas/list.html", PageData{Title: "Ventas", Active: "ventas", OperacionActiva: getOperacionActiva()})
}

func handlePOSPage(w http.ResponseWriter, r *http.Request) {
	if !getOperacionActiva() {
		http.Redirect(w, r, "/cajas", http.StatusSeeOther)
		return
	}
	render(w, r, "ventas/pos.html", PageData{Title: "Punto de Venta", Active: "ventas", OperacionActiva: true})
}

func handleClientesPage(w http.ResponseWriter, r *http.Request) {
	render(w, r, "clientes/list.html", PageData{Title: "Clientes", Active: "clientes", OperacionActiva: getOperacionActiva()})
}

func handleClienteFormPage(w http.ResponseWriter, r *http.Request) {
	render(w, r, "clientes/form.html", PageData{Title: "Nuevo Cliente", Active: "clientes", OperacionActiva: getOperacionActiva()})
}

func handleClienteEditPage(w http.ResponseWriter, r *http.Request) {
	numero := r.PathValue("numero")
	render(w, r, "clientes/form.html", PageData{Title: "Editar Cliente", Active: "clientes", Data: numero, OperacionActiva: getOperacionActiva()})
}

func handleTicketsPage(w http.ResponseWriter, r *http.Request) {
	render(w, r, "tickets/list.html", PageData{Title: "Tickets", Active: "tickets", OperacionActiva: getOperacionActiva()})
}

func handleTicketDetailPage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	render(w, r, "tickets/detail.html", PageData{Title: "Ticket #" + id, Active: "tickets", Data: id, OperacionActiva: getOperacionActiva()})
}

func handleCajasPage(w http.ResponseWriter, r *http.Request) {
	render(w, r, "cajas/operacion.html", PageData{Title: "Caja", Active: "cajas", OperacionActiva: getOperacionActiva()})
}

func handleReportesPage(w http.ResponseWriter, r *http.Request) {
	render(w, r, "reportes/dashboard.html", PageData{Title: "Reportes", Active: "reportes", OperacionActiva: getOperacionActiva()})
}

func handleProveedoresPage(w http.ResponseWriter, r *http.Request) {
	render(w, r, "proveedores/list.html", PageData{Title: "Proveedores", Active: "proveedores", OperacionActiva: getOperacionActiva()})
}

func handleUsuariosPage(w http.ResponseWriter, r *http.Request) {
	render(w, r, "usuarios/list.html", PageData{Title: "Usuarios", Active: "usuarios", OperacionActiva: getOperacionActiva()})
}

func handleDepartamentosPage(w http.ResponseWriter, r *http.Request) {
	render(w, r, "departamentos/list.html", PageData{Title: "Departamentos", Active: "departamentos", OperacionActiva: getOperacionActiva()})
}

func handlePedidosPage(w http.ResponseWriter, r *http.Request) {
	render(w, r, "pedidos/list.html", PageData{Title: "Pedidos", Active: "pedidos", OperacionActiva: getOperacionActiva()})
}

func handleChatPage(w http.ResponseWriter, r *http.Request) {
	sessionCookie, _ := r.Cookie("session")
	var uid int
	if sessionCookie != nil {
		db.QueryRow("SELECT id FROM USUARIOS WHERE usuario=?", sessionCookie.Value).Scan(&uid)
	}
	render(w, r, "chat.html", PageData{Title: "Chat", Active: "chat", OperacionActiva: getOperacionActiva(), UserID: uid})
}


