package main

import (
	"crypto/sha256"
	"fmt"
	"net/http"
)

type PageData struct {
	Title    string
	Active   string
	Data     interface{}
	User     string
	Error    string
	Success  string
	OperacionActiva bool
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
	render(w, "dashboard.html", PageData{Title: "Dashboard", Active: "dashboard", Data: d, OperacionActiva: d.OperacionActiva})
}

func handleLoginPage(w http.ResponseWriter, r *http.Request) {
	render(w, "login.html", PageData{Title: "Iniciar Sesion"})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	user := r.FormValue("usuario")
	pw := r.FormValue("clave")
	var id int
	var hash string
	err := db.QueryRow("SELECT id, clave FROM USUARIOS WHERE usuario=? AND activo='t'", user).Scan(&id, &hash)
	if err != nil {
		render(w, "login.html", PageData{Title: "Iniciar Sesion", Error: "Usuario o clave incorrectos"})
		return
	}
	h := sha256.Sum256([]byte(pw))
	if fmt.Sprintf("%x", h) != hash {
		render(w, "login.html", PageData{Title: "Iniciar Sesion", Error: "Usuario o clave incorrectos"})
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "session", Value: user, Path: "/"})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func handleProductosPage(w http.ResponseWriter, r *http.Request) {
	render(w, "productos/list.html", PageData{Title: "Productos", Active: "productos", OperacionActiva: getOperacionActiva()})
}

func handleProductoFormPage(w http.ResponseWriter, r *http.Request) {
	render(w, "productos/form.html", PageData{Title: "Nuevo Producto", Active: "productos", OperacionActiva: getOperacionActiva()})
}

func handleProductoEditPage(w http.ResponseWriter, r *http.Request) {
	codigo := r.PathValue("codigo")
	render(w, "productos/form.html", PageData{Title: "Editar Producto", Active: "productos", Data: codigo, OperacionActiva: getOperacionActiva()})
}

func handleVentasPage(w http.ResponseWriter, r *http.Request) {
	render(w, "ventas/list.html", PageData{Title: "Ventas", Active: "ventas", OperacionActiva: getOperacionActiva()})
}

func handlePOSPage(w http.ResponseWriter, r *http.Request) {
	if !getOperacionActiva() {
		http.Redirect(w, r, "/cajas", http.StatusSeeOther)
		return
	}
	render(w, "ventas/pos.html", PageData{Title: "Punto de Venta", Active: "ventas", OperacionActiva: true})
}

func handleClientesPage(w http.ResponseWriter, r *http.Request) {
	render(w, "clientes/list.html", PageData{Title: "Clientes", Active: "clientes", OperacionActiva: getOperacionActiva()})
}

func handleClienteFormPage(w http.ResponseWriter, r *http.Request) {
	render(w, "clientes/form.html", PageData{Title: "Nuevo Cliente", Active: "clientes", OperacionActiva: getOperacionActiva()})
}

func handleClienteEditPage(w http.ResponseWriter, r *http.Request) {
	numero := r.PathValue("numero")
	render(w, "clientes/form.html", PageData{Title: "Editar Cliente", Active: "clientes", Data: numero, OperacionActiva: getOperacionActiva()})
}

func handleTicketsPage(w http.ResponseWriter, r *http.Request) {
	render(w, "tickets/list.html", PageData{Title: "Tickets", Active: "tickets", OperacionActiva: getOperacionActiva()})
}

func handleTicketDetailPage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	render(w, "tickets/detail.html", PageData{Title: "Ticket #" + id, Active: "tickets", Data: id, OperacionActiva: getOperacionActiva()})
}

func handleCajasPage(w http.ResponseWriter, r *http.Request) {
	render(w, "cajas/operacion.html", PageData{Title: "Caja", Active: "cajas", OperacionActiva: getOperacionActiva()})
}

func handleReportesPage(w http.ResponseWriter, r *http.Request) {
	render(w, "reportes/dashboard.html", PageData{Title: "Reportes", Active: "reportes", OperacionActiva: getOperacionActiva()})
}

func handleProveedoresPage(w http.ResponseWriter, r *http.Request) {
	render(w, "proveedores/list.html", PageData{Title: "Proveedores", Active: "proveedores", OperacionActiva: getOperacionActiva()})
}

func handleUsuariosPage(w http.ResponseWriter, r *http.Request) {
	render(w, "usuarios/list.html", PageData{Title: "Usuarios", Active: "usuarios", OperacionActiva: getOperacionActiva()})
}


