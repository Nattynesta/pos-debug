package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// --- Productos ---

func handleProductosList(w http.ResponseWriter, r *http.Request) {
	ps, err := listProductos()
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	q := r.URL.Query().Get("q")
	if q != "" {
		var filtered []Producto
		q = strings.ToLower(q)
		for _, p := range ps {
			if strings.Contains(strings.ToLower(p.Codigo), q) || strings.Contains(strings.ToLower(p.Descripcion), q) {
				filtered = append(filtered, p)
			}
		}
		ps = filtered
	}
	jsonResp(w, ps)
}

func handleProductosCreate(w http.ResponseWriter, r *http.Request) {
	var p Producto
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		jsonErr(w, "JSON invalido", 400)
		return
	}
	_, err := db.Exec(`INSERT INTO PRODUCTOS (codigo, descripcion, tventa, pcosto, pventa, dept, provid, umedida, mayoreo, iprioridad, dinventario, dinvminimo, dinvmaximo, porcentaje_ganancia, componentes, impuestos) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.Codigo, p.Descripcion, p.Tventa, p.Pcosto, p.Pventa, p.Dept, p.Provid, p.Umedida, p.Mayoreo, p.Iprioridad, p.Dinventario, p.Dinvminimo, p.Dinvmaximo, p.PorcentajeGanancia, p.Componentes, p.Impuestos)
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}
	jsonResp(w, map[string]string{"ok": "Producto creado"})
}

func handleProductosGet(w http.ResponseWriter, r *http.Request) {
	codigo := r.PathValue("codigo")
	var p Producto
	err := db.QueryRow(`SELECT codigo, descripcion, tventa, COALESCE(pcosto,0), COALESCE(pventa,0), dept, provid, umedida, COALESCE(mayoreo,0), iprioridad, COALESCE(dinventario,0), COALESCE(dinvminimo,0), COALESCE(dinvmaximo,0), COALESCE(checado_en,''), COALESCE(porcentaje_ganancia,0), COALESCE(componentes,''), COALESCE(impuestos,'') FROM PRODUCTOS WHERE codigo=?`, codigo).Scan(&p.Codigo, &p.Descripcion, &p.Tventa, &p.Pcosto, &p.Pventa, &p.Dept, &p.Provid, &p.Umedida, &p.Mayoreo, &p.Iprioridad, &p.Dinventario, &p.Dinvminimo, &p.Dinvmaximo, &p.ChecadoEn, &p.PorcentajeGanancia, &p.Componentes, &p.Impuestos)
	if err == sql.ErrNoRows {
		jsonErr(w, "Producto no encontrado", 404)
		return
	}
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	jsonResp(w, p)
}

func handleProductosUpdate(w http.ResponseWriter, r *http.Request) {
	codigo := r.PathValue("codigo")
	var p Producto
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		jsonErr(w, "JSON invalido", 400)
		return
	}
	_, err := db.Exec(`UPDATE PRODUCTOS SET descripcion=?, tventa=?, pcosto=?, pventa=?, dept=?, provid=?, umedida=?, mayoreo=?, iprioridad=?, dinventario=?, dinvminimo=?, dinvmaximo=?, checado_en=?, porcentaje_ganancia=?, componentes=?, impuestos=? WHERE codigo=?`,
		p.Descripcion, p.Tventa, p.Pcosto, p.Pventa, p.Dept, p.Provid, p.Umedida, p.Mayoreo, p.Iprioridad, p.Dinventario, p.Dinvminimo, p.Dinvmaximo, p.ChecadoEn, p.PorcentajeGanancia, p.Componentes, p.Impuestos, codigo)
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}
	jsonResp(w, map[string]string{"ok": "Producto actualizado"})
}

func handleProductosDelete(w http.ResponseWriter, r *http.Request) {
	codigo := r.PathValue("codigo")
	db.Exec("DELETE FROM PRODUCTOS WHERE codigo=?", codigo)
	jsonResp(w, map[string]string{"ok": "Producto eliminado"})
}

// --- Clientes ---

func handleClientesList(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT numero, COALESCE(nombre,''), COALESCE(direccion,''), COALESCE(telefono,''), COALESCE(dsaldoactual,0), COALESCE(dtactualizasaldo,''), COALESCE(limite_credito,0), COALESCE(ultimo_pago_en,''), COALESCE(folio,0) FROM CLIENTES ORDER BY nombre`)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var cs []Cliente
	for rows.Next() {
		var c Cliente
		rows.Scan(&c.Numero, &c.Nombre, &c.Direccion, &c.Telefono, &c.Dsaldoactual, &c.Dtactualizasaldo, &c.LimiteCredito, &c.UltimoPagoEn, &c.Folio)
		cs = append(cs, c)
	}
	q := r.URL.Query().Get("q")
	if q != "" && cs == nil {
		cs = []Cliente{}
	}
	jsonResp(w, cs)
}

func handleClientesSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	rows, err := db.Query(`SELECT numero, COALESCE(nombre,''), COALESCE(direccion,''), COALESCE(telefono,''), COALESCE(dsaldoactual,0), COALESCE(dtactualizasaldo,''), COALESCE(limite_credito,0), COALESCE(ultimo_pago_en,''), COALESCE(folio,0) FROM CLIENTES WHERE nombre LIKE ? OR CAST(numero AS TEXT) LIKE ?`, "%"+q+"%", "%"+q+"%")
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var cs []Cliente
	for rows.Next() {
		var c Cliente
		rows.Scan(&c.Numero, &c.Nombre, &c.Direccion, &c.Telefono, &c.Dsaldoactual, &c.Dtactualizasaldo, &c.LimiteCredito, &c.UltimoPagoEn, &c.Folio)
		cs = append(cs, c)
	}
	if cs == nil {
		cs = []Cliente{}
	}
	jsonResp(w, cs)
}

func handleClientesCreate(w http.ResponseWriter, r *http.Request) {
	var c Cliente
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		jsonErr(w, "JSON invalido", 400)
		return
	}
	_, err := db.Exec(`INSERT INTO CLIENTES (nombre, direccion, telefono, dsaldoactual, limite_credito, folio) VALUES (?,?,?,?,?,COALESCE((SELECT MAX(folio)+1 FROM CLIENTES),1))`,
		c.Nombre, c.Direccion, c.Telefono, c.Dsaldoactual, c.LimiteCredito)
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}
	jsonResp(w, map[string]string{"ok": "Cliente creado"})
}

func handleClientesGet(w http.ResponseWriter, r *http.Request) {
	numero := r.PathValue("numero")
	var c Cliente
	err := db.QueryRow(`SELECT numero, COALESCE(nombre,''), COALESCE(direccion,''), COALESCE(telefono,''), COALESCE(dsaldoactual,0), COALESCE(dtactualizasaldo,''), COALESCE(limite_credito,0), COALESCE(ultimo_pago_en,''), COALESCE(folio,0) FROM CLIENTES WHERE numero=?`, numero).Scan(&c.Numero, &c.Nombre, &c.Direccion, &c.Telefono, &c.Dsaldoactual, &c.Dtactualizasaldo, &c.LimiteCredito, &c.UltimoPagoEn, &c.Folio)
	if err == sql.ErrNoRows {
		jsonErr(w, "Cliente no encontrado", 404)
		return
	}
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	jsonResp(w, c)
}

func handleClientesUpdate(w http.ResponseWriter, r *http.Request) {
	numero := r.PathValue("numero")
	var c Cliente
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		jsonErr(w, "JSON invalido", 400)
		return
	}
	_, err := db.Exec(`UPDATE CLIENTES SET nombre=?, direccion=?, telefono=?, dsaldoactual=?, dtactualizasaldo=?, limite_credito=?, ultimo_pago_en=? WHERE numero=?`,
		c.Nombre, c.Direccion, c.Telefono, c.Dsaldoactual, c.Dtactualizasaldo, c.LimiteCredito, c.UltimoPagoEn, numero)
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}
	jsonResp(w, map[string]string{"ok": "Cliente actualizado"})
}

func handleClientesDelete(w http.ResponseWriter, r *http.Request) {
	numero := r.PathValue("numero")
	db.Exec("DELETE FROM CLIENTES WHERE numero=?", numero)
	jsonResp(w, map[string]string{"ok": "Cliente eliminado"})
}

// --- Proveedores ---

func handleProveedoresList(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT num, COALESCE(nombre,''), COALESCE(direccion,''), COALESCE(telefonos,'') FROM PROV ORDER BY nombre")
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var ps []Proveedor
	for rows.Next() {
		var p Proveedor
		rows.Scan(&p.Num, &p.Nombre, &p.Direccion, &p.Telefonos)
		ps = append(ps, p)
	}
	if ps == nil {
		ps = []Proveedor{}
	}
	jsonResp(w, ps)
}

func handleProveedoresCreate(w http.ResponseWriter, r *http.Request) {
	var p Proveedor
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		jsonErr(w, "JSON invalido", 400)
		return
	}
	_, err := db.Exec("INSERT INTO PROV (nombre, direccion, telefonos) VALUES (?,?,?)", p.Nombre, p.Direccion, p.Telefonos)
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}
	jsonResp(w, map[string]string{"ok": "Proveedor creado"})
}

func handleProveedoresGet(w http.ResponseWriter, r *http.Request) {
	num := r.PathValue("num")
	var p Proveedor
	err := db.QueryRow("SELECT num, COALESCE(nombre,''), COALESCE(direccion,''), COALESCE(telefonos,'') FROM PROV WHERE num=?", num).Scan(&p.Num, &p.Nombre, &p.Direccion, &p.Telefonos)
	if err == sql.ErrNoRows {
		jsonErr(w, "Proveedor no encontrado", 404)
		return
	}
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	jsonResp(w, p)
}

func handleProveedoresUpdate(w http.ResponseWriter, r *http.Request) {
	num := r.PathValue("num")
	var p Proveedor
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		jsonErr(w, "JSON invalido", 400)
		return
	}
	_, err := db.Exec("UPDATE PROV SET nombre=?, direccion=?, telefonos=? WHERE num=?", p.Nombre, p.Direccion, p.Telefonos, num)
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}
	jsonResp(w, map[string]string{"ok": "Proveedor actualizado"})
}

// --- Departamentos ---

func handleDepartamentosList(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, COALESCE(nombre,''), COALESCE(porcentaje_impuesto,0), COALESCE(activo,'t') FROM DEPARTAMENTOS ORDER BY nombre")
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var ds []Departamento
	for rows.Next() {
		var d Departamento
		rows.Scan(&d.ID, &d.Nombre, &d.PorcentajeImpuesto, &d.Activo)
		ds = append(ds, d)
	}
	if ds == nil {
		ds = []Departamento{}
	}
	jsonResp(w, ds)
}

func handleDepartamentosCreate(w http.ResponseWriter, r *http.Request) {
	var d Departamento
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		jsonErr(w, "JSON invalido", 400)
		return
	}
	_, err := db.Exec("INSERT INTO DEPARTAMENTOS (nombre, porcentaje_impuesto, activo) VALUES (?,?,?)", d.Nombre, d.PorcentajeImpuesto, d.Activo)
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}
	jsonResp(w, map[string]string{"ok": "Departamento creado"})
}

func handleDepartamentosUpdate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var d Departamento
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		jsonErr(w, "JSON invalido", 400)
		return
	}
	_, err := db.Exec("UPDATE DEPARTAMENTOS SET nombre=?, porcentaje_impuesto=?, activo=? WHERE id=?", d.Nombre, d.PorcentajeImpuesto, d.Activo, id)
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}
	jsonResp(w, map[string]string{"ok": "Departamento actualizado"})
}

// --- Medidas ---

func handleMedidasList(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT codigo, COALESCE(nombre,'') FROM MEDIDAS ORDER BY nombre")
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var ms []Medida
	for rows.Next() {
		var m Medida
		rows.Scan(&m.Codigo, &m.Nombre)
		ms = append(ms, m)
	}
	if ms == nil {
		ms = []Medida{}
	}
	jsonResp(w, ms)
}

func handleMedidasCreate(w http.ResponseWriter, r *http.Request) {
	var m Medida
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		jsonErr(w, "JSON invalido", 400)
		return
	}
	_, err := db.Exec("INSERT INTO MEDIDAS (nombre) VALUES (?)", m.Nombre)
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}
	jsonResp(w, map[string]string{"ok": "Medida creada"})
}

// --- Usuarios ---

func handleUsuariosList(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, COALESCE(nombre_completo,''), COALESCE(direccion,''), COALESCE(telefono,''), usuario, COALESCE(activo,'t'), COALESCE(created_on,''), COALESCE(correo,''), esta_en_caja_id FROM USUARIOS ORDER BY usuario")
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var us []Usuario
	for rows.Next() {
		var u Usuario
		rows.Scan(&u.ID, &u.NombreCompleto, &u.Direccion, &u.Telefono, &u.Usuario, &u.Activo, &u.CreatedOn, &u.Correo, &u.EstaEnCajaID)
		us = append(us, u)
	}
	if us == nil {
		us = []Usuario{}
	}
	jsonResp(w, us)
}

func handleUsuariosCreate(w http.ResponseWriter, r *http.Request) {
	var u Usuario
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		jsonErr(w, "JSON invalido", 400)
		return
	}
	_, err := db.Exec("INSERT INTO USUARIOS (nombre_completo, direccion, telefono, usuario, clave, activo, created_on, correo) VALUES (?,?,?,?,?,?,?,?)",
		u.NombreCompleto, u.Direccion, u.Telefono, u.Usuario, hashPassword(u.Usuario), u.Activo, now(), u.Correo)
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}
	jsonResp(w, map[string]string{"ok": "Usuario creado"})
}

func handleUsuariosUpdate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var u Usuario
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		jsonErr(w, "JSON invalido", 400)
		return
	}
	q := "UPDATE USUARIOS SET nombre_completo=?, direccion=?, telefono=?, activo=?, correo=?"
	args := []interface{}{u.NombreCompleto, u.Direccion, u.Telefono, u.Activo, u.Correo}
	if u.Usuario != "" {
		q += ", usuario=?"
		args = append(args, u.Usuario)
	}
	q += " WHERE id=?"
	args = append(args, id)
	_, err := db.Exec(q, args...)
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}
	jsonResp(w, map[string]string{"ok": "Usuario actualizado"})
}

// --- Cajas ---

func handleCajasList(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, COALESCE(nombre,''), COALESCE(ultima_ip,''), COALESCE(ultimo_ingreso,''), COALESCE(nombre_pc,'') FROM CAJAS ORDER BY nombre")
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var cs []Caja
	for rows.Next() {
		var c Caja
		rows.Scan(&c.ID, &c.Nombre, &c.UltimaIP, &c.UltimoIngreso, &c.NombrePC)
		cs = append(cs, c)
	}
	if cs == nil {
		cs = []Caja{}
	}
	jsonResp(w, cs)
}

func handleCajasCreate(w http.ResponseWriter, r *http.Request) {
	var c Caja
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		jsonErr(w, "JSON invalido", 400)
		return
	}
	_, err := db.Exec("INSERT INTO CAJAS (nombre, ultima_ip, nombre_pc) VALUES (?,?,?)", c.Nombre, c.UltimaIP, c.NombrePC)
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}
	jsonResp(w, map[string]string{"ok": "Caja creada"})
}

// --- Operaciones (Apertura/Cierre de Caja) ---

func handleOperacionesList(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT id, COALESCE(dinero_en_caja,0), COALESCE(tipo_de_cambio,0), inicio_usuario_id, inicio_en, cerro_en, caja_id, COALESCE(abierta,'t'), COALESCE(ventas,0), COALESCE(salidas,0), COALESCE(entradas,0), COALESCE(pagos,0), COALESCE(impuestos,0), COALESCE(ganancias,0), COALESCE(ingresos_tarjeta,0), COALESCE(ingresos_vales,0), COALESCE(ingresos_efectivo,0) FROM OPERACIONES ORDER BY id DESC LIMIT 50`)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var ops []Operacion
	for rows.Next() {
		var o Operacion
		rows.Scan(&o.ID, &o.DineroEnCaja, &o.TipoDeCambio, &o.InicioUsuarioID, &o.InicioEn, &o.CerroEn, &o.CajaID, &o.Abierta, &o.Ventas, &o.Salidas, &o.Entradas, &o.Pagos, &o.Impuestos, &o.Ganancias, &o.IngresosTarjeta, &o.IngresosVales, &o.IngresosEfectivo)
		ops = append(ops, o)
	}
	if ops == nil {
		ops = []Operacion{}
	}
	jsonResp(w, ops)
}

func handleOperacionActiva(w http.ResponseWriter, r *http.Request) {
	var o Operacion
	err := db.QueryRow(`SELECT id, COALESCE(dinero_en_caja,0), COALESCE(tipo_de_cambio,0), inicio_usuario_id, inicio_en, cerro_en, caja_id, COALESCE(abierta,'t'), COALESCE(ventas,0), COALESCE(salidas,0), COALESCE(entradas,0), COALESCE(pagos,0), COALESCE(impuestos,0), COALESCE(ganancias,0), COALESCE(ingresos_tarjeta,0), COALESCE(ingresos_vales,0), COALESCE(ingresos_efectivo,0) FROM OPERACIONES WHERE abierta='t' LIMIT 1`).Scan(&o.ID, &o.DineroEnCaja, &o.TipoDeCambio, &o.InicioUsuarioID, &o.InicioEn, &o.CerroEn, &o.CajaID, &o.Abierta, &o.Ventas, &o.Salidas, &o.Entradas, &o.Pagos, &o.Impuestos, &o.Ganancias, &o.IngresosTarjeta, &o.IngresosVales, &o.IngresosEfectivo)
	if err == sql.ErrNoRows {
		jsonResp(w, map[string]interface{}{"activa": false})
		return
	}
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]interface{}{"activa": true, "operacion": o})
}

func handleOperacionAbrir(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CajaID       int     `json:"caja_id"`
		UsuarioID    int     `json:"usuario_id"`
		DineroEnCaja float64 `json:"dinero_en_caja"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "JSON invalido", 400)
		return
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM OPERACIONES WHERE abierta='t'").Scan(&count)
	if count > 0 {
		jsonErr(w, "Ya hay una operacion abierta", 400)
		return
	}

	_, err := db.Exec(`INSERT INTO OPERACIONES (dinero_en_caja, tipo_de_cambio, inicio_usuario_id, inicio_en, caja_id, abierta) VALUES (?,1,?,?,?,'t')`,
		req.DineroEnCaja, req.UsuarioID, now(), req.CajaID)
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}
	jsonResp(w, map[string]string{"ok": "Caja abierta"})
}

func handleOperacionCerrar(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_, err := db.Exec(`UPDATE OPERACIONES SET cerro_en=?, abierta='f' WHERE id=? AND abierta='t'`, now(), id)
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}
	jsonResp(w, map[string]string{"ok": "Caja cerrada"})
}

// --- Tickets (Ventas POS) ---

func handleTicketsList(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit := 50
	offset := (page - 1) * limit
	rows, err := db.Query(`SELECT t.id, t.folio, t.caja_id, t.cajero_id, COALESCE(t.nombre,''), t.creado_en, COALESCE(t.subtotal,0), COALESCE(t.impuestos,0), COALESCE(t.total,0), COALESCE(t.ganancia,0), t.esta_abierto, t.cliente_id, t.vendido_en, t.es_modificable, COALESCE(t.pago_con,0), COALESCE(t.moneda,''), COALESCE(t.numero_articulos,0), t.pagado_en, t.esta_cancelado, t.operacion_id, COALESCE(t.forma_pago,''), COALESCE(t.referencia,''), COALESCE(t.total_devuelto,0) FROM VENTATICKETS t ORDER BY t.creado_en DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var ts []VentaTicket
	for rows.Next() {
		var t VentaTicket
		rows.Scan(&t.ID, &t.Folio, &t.CajaID, &t.CajeroID, &t.Nombre, &t.CreadoEn, &t.Subtotal, &t.Impuestos, &t.Total, &t.Ganancia, &t.EstaAbierto, &t.ClienteID, &t.VendidoEn, &t.EsModificable, &t.PagoCon, &t.Moneda, &t.NumeroArticulos, &t.PagadoEn, &t.EstaCancelado, &t.OperacionID, &t.FormaPago, &t.Referencia, &t.TotalDevuelto)
		ts = append(ts, t)
	}
	if ts == nil {
		ts = []VentaTicket{}
	}
	jsonResp(w, ts)
}

func handleTicketCrear(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CajaID    int `json:"caja_id"`
		CajeroID  int `json:"cajero_id"`
		ClienteID *int `json:"cliente_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "JSON invalido", 400)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer tx.Rollback()

	folio := nextFolio(tx)

	var operacionID int
	err = tx.QueryRow("SELECT id FROM OPERACIONES WHERE abierta='t' LIMIT 1").Scan(&operacionID)
	if err != nil {
		jsonErr(w, "No hay caja abierta", 400)
		return
	}

	res, err := tx.Exec(`INSERT INTO VENTATICKETS (folio, caja_id, cajero_id, creado_en, esta_abierto, operacion_id, es_modificable, nombre) VALUES (?,?,?,?,'t',?,'t','PV)`, folio, req.CajaID, req.CajeroID, now(), operacionID)
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}
	id, _ := res.LastInsertId()
	tx.Commit()

	jsonResp(w, map[string]int64{"id": id, "folio": int64(folio)})
}

func handleTicketGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var t VentaTicket
	err := db.QueryRow(`SELECT id, folio, caja_id, cajero_id, COALESCE(nombre,''), creado_en, COALESCE(subtotal,0), COALESCE(impuestos,0), COALESCE(total,0), COALESCE(ganancia,0), esta_abierto, cliente_id, vendido_en, es_modificable, COALESCE(pago_con,0), COALESCE(moneda,''), COALESCE(numero_articulos,0), pagado_en, esta_cancelado, operacion_id, COALESCE(forma_pago,''), COALESCE(referencia,''), COALESCE(total_devuelto,0) FROM VENTATICKETS WHERE id=?`, id).Scan(&t.ID, &t.Folio, &t.CajaID, &t.CajeroID, &t.Nombre, &t.CreadoEn, &t.Subtotal, &t.Impuestos, &t.Total, &t.Ganancia, &t.EstaAbierto, &t.ClienteID, &t.VendidoEn, &t.EsModificable, &t.PagoCon, &t.Moneda, &t.NumeroArticulos, &t.PagadoEn, &t.EstaCancelado, &t.OperacionID, &t.FormaPago, &t.Referencia, &t.TotalDevuelto)
	if err == sql.ErrNoRows {
		jsonErr(w, "Ticket no encontrado", 404)
		return
	}
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	rows, err := db.Query(`SELECT id, ticket_id, producto_codigo, producto_nombre, cantidad, COALESCE(ganancia,0), departamento_id, COALESCE(pagado_en,''), COALESCE(usa_mayoreo,'f'), COALESCE(porcentaje_descuento,0), COALESCE(componentes,''), COALESCE(impuestos_usados,''), COALESCE(impuesto_unitario,0), COALESCE(precio_usado,0), COALESCE(cantidad_devuelta,0), COALESCE(fue_devuelto,'f'), COALESCE(porcentaje_pagado,0) FROM VENTATICKETS_ARTICULOS WHERE ticket_id=?`, id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var a TicketArticulo
			rows.Scan(&a.ID, &a.TicketID, &a.ProductoCodigo, &a.ProductoNombre, &a.Cantidad, &a.Ganancia, &a.DepartamentoID, &a.PagadoEn, &a.UsaMayoreo, &a.PorcentajeDescuento, &a.Componentes, &a.ImpuestosUsados, &a.ImpuestoUnitario, &a.PrecioUsado, &a.CantidadDevuelta, &a.FueDevuelto, &a.PorcentajePagado)
			t.Articulos = append(t.Articulos, a)
		}
	}
	if t.Articulos == nil {
		t.Articulos = []TicketArticulo{}
	}

	jsonResp(w, t)
}

func handleTicketAddArticulo(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		ProductoCodigo string  `json:"producto_codigo"`
		Cantidad       float64 `json:"cantidad"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "JSON invalido", 400)
		return
	}

	var p Producto
	err := db.QueryRow(`SELECT codigo, COALESCE(descripcion,''), tventa, COALESCE(pcosto,0), COALESCE(pventa,0), dept, provid, umedida, COALESCE(mayoreo,0), iprioridad, COALESCE(dinventario,0), COALESCE(dinvminimo,0), COALESCE(dinvmaximo,0), COALESCE(checado_en,''), COALESCE(porcentaje_ganancia,0), COALESCE(componentes,''), COALESCE(impuestos,'') FROM PRODUCTOS WHERE codigo=?`, req.ProductoCodigo).Scan(&p.Codigo, &p.Descripcion, &p.Tventa, &p.Pcosto, &p.Pventa, &p.Dept, &p.Provid, &p.Umedida, &p.Mayoreo, &p.Iprioridad, &p.Dinventario, &p.Dinvminimo, &p.Dinvmaximo, &p.ChecadoEn, &p.PorcentajeGanancia, &p.Componentes, &p.Impuestos)
	if err != nil {
		jsonErr(w, "Producto no encontrado", 404)
		return
	}

	precio := p.Pventa
	ganancia := precio - p.Pcosto

	tx, err := db.Begin()
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(`INSERT INTO VENTATICKETS_ARTICULOS (ticket_id, producto_codigo, producto_nombre, cantidad, ganancia, precio_usado, departamento_id, impuesto_unitario) VALUES (?,?,?,?,?,?,?,0)`,
		id, p.Codigo, p.Descripcion, req.Cantidad, ganancia*req.Cantidad, precio, p.Dept)
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}

	tx.Exec(`UPDATE VENTATICKETS SET subtotal = (SELECT COALESCE(SUM(precio_usado * cantidad),0) FROM VENTATICKETS_ARTICULOS WHERE ticket_id=?), total = subtotal, ganancia = (SELECT COALESCE(SUM(ganancia),0) FROM VENTATICKETS_ARTICULOS WHERE ticket_id=?), numero_articulos = (SELECT COUNT(*) FROM VENTATICKETS_ARTICULOS WHERE ticket_id=?) WHERE id=?`, id, id, id, id)

	tx.Exec(`INSERT INTO VENTAS (producto_codigo, cantidad, fecha, ticket_id) VALUES (?,?,?,?)`, p.Codigo, req.Cantidad, now(), id)

	tx.Commit()
	jsonResp(w, map[string]string{"ok": "Articulo agregado"})
}

func handleTicketRemoveArticulo(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	artID := r.PathValue("artId")
	tx, err := db.Begin()
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer tx.Rollback()

	tx.Exec("DELETE FROM VENTATICKETS_ARTICULOS WHERE id=? AND ticket_id=?", artID, id)
	tx.Exec(`UPDATE VENTATICKETS SET subtotal = (SELECT COALESCE(SUM(precio_usado * cantidad),0) FROM VENTATICKETS_ARTICULOS WHERE ticket_id=?), total = subtotal, ganancia = (SELECT COALESCE(SUM(ganancia),0) FROM VENTATICKETS_ARTICULOS WHERE ticket_id=?), numero_articulos = (SELECT COUNT(*) FROM VENTATICKETS_ARTICULOS WHERE ticket_id=?) WHERE id=?`, id, id, id, id)
	tx.Commit()
	jsonResp(w, map[string]string{"ok": "Articulo eliminado"})
}

func handleTicketCobrar(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		PagoCon   float64 `json:"pago_con"`
		FormaPago string  `json:"forma_pago"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "JSON invalido", 400)
		return
	}
	if req.FormaPago == "" {
		req.FormaPago = "e"
	}

	tx, err := db.Begin()
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer tx.Rollback()

	var total float64
	tx.QueryRow("SELECT COALESCE(total,0) FROM VENTATICKETS WHERE id=?", id).Scan(&total)

	var operacionID int
	tx.QueryRow("SELECT operacion_id FROM VENTATICKETS WHERE id=?", id).Scan(&operacionID)

	_, err = tx.Exec(`UPDATE VENTATICKETS SET esta_abierto='f', pagado_en=?, pago_con=?, forma_pago=?, total_devuelto=?, vendido_en=? WHERE id=?`,
		now(), req.PagoCon, req.FormaPago, req.PagoCon-total, now(), id)
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}

	tx.Exec(`UPDATE OPERACIONES SET ventas = ventas + ?, ingresos_efectivo = ingresos_efectivo + ?, ganancias = ganancias + (SELECT COALESCE(ganancia,0) FROM VENTATICKETS WHERE id=?) WHERE id=?`,
		total, req.PagoCon, id, operacionID)

	tx.Commit()
	jsonResp(w, map[string]string{"ok": "Cobro exitoso", "cambio": fmt.Sprintf("%.2f", req.PagoCon-total)})
}

func handleTicketCancelar(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_, err := db.Exec(`UPDATE VENTATICKETS SET esta_cancelado='t', esta_abierto='f' WHERE id=?`, id)
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}
	jsonResp(w, map[string]string{"ok": "Ticket cancelado"})
}

// --- Movimientos ---

func handleMovimientosList(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT id, operacion_id, COALESCE(monto,0), cuando_fue, COALESCE(comentarios,''), tipo, cliente_id, caja_id, cajero_id FROM MOVIMIENTOS ORDER BY cuando_fue DESC LIMIT 100`)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var ms []Movimiento
	for rows.Next() {
		var m Movimiento
		rows.Scan(&m.ID, &m.OperacionID, &m.Monto, &m.CuandoFue, &m.Comentarios, &m.Tipo, &m.ClienteID, &m.CajaID, &m.CajeroID)
		ms = append(ms, m)
	}
	if ms == nil {
		ms = []Movimiento{}
	}
	jsonResp(w, ms)
}

func handleMovimientoCrear(w http.ResponseWriter, r *http.Request) {
	var m Movimiento
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		jsonErr(w, "JSON invalido", 400)
		return
	}
	m.CuandoFue = now()
	_, err := db.Exec(`INSERT INTO MOVIMIENTOS (operacion_id, monto, cuando_fue, comentarios, tipo, cliente_id, caja_id, cajero_id) VALUES (?,?,?,?,?,?,?,?)`,
		m.OperacionID, m.Monto, m.CuandoFue, m.Comentarios, m.Tipo, m.ClienteID, m.CajaID, m.CajeroID)
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}

	if m.Tipo == "E" {
		db.Exec("UPDATE OPERACIONES SET entradas = entradas + ? WHERE id=?", m.Monto, m.OperacionID)
	} else if m.Tipo == "S" {
		db.Exec("UPDATE OPERACIONES SET salidas = salidas + ? WHERE id=?", m.Monto, m.OperacionID)
	}

	jsonResp(w, map[string]string{"ok": "Movimiento registrado"})
}

// --- Inventario ---

func handleHistorialInventario(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT id, usuario_id, cuando_fue, tipo, COALESCE(habia,0), cantidad, codigo_producto, caja_id FROM HISTORIAL_INVENTARIO ORDER BY cuando_fue DESC LIMIT 100`)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	type HI struct {
		ID             int     `json:"id"`
		UsuarioID      int     `json:"usuario_id"`
		CuandoFue      string  `json:"cuando_fue"`
		Tipo           string  `json:"tipo"`
		Habia          float64 `json:"habia"`
		Cantidad       float64 `json:"cantidad"`
		CodigoProducto string  `json:"codigo_producto"`
		CajaID         *int    `json:"caja_id"`
	}
	var hs []HI
	for rows.Next() {
		var h HI
		rows.Scan(&h.ID, &h.UsuarioID, &h.CuandoFue, &h.Tipo, &h.Habia, &h.Cantidad, &h.CodigoProducto, &h.CajaID)
		hs = append(hs, h)
	}
	if hs == nil {
		hs = []HI{}
	}
	jsonResp(w, hs)
}

func handleInventarioAjustar(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CodigoProducto string  `json:"codigo_producto"`
		Cantidad       float64 `json:"cantidad"`
		Tipo           string  `json:"tipo"`
		UsuarioID      int     `json:"usuario_id"`
		CajaID         *int    `json:"caja_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "JSON invalido", 400)
		return
	}
	var habia float64
	db.QueryRow("SELECT COALESCE(dinventario,0) FROM PRODUCTOS WHERE codigo=?", req.CodigoProducto).Scan(&habia)

	tx, err := db.Begin()
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer tx.Rollback()

	cantidad := req.Cantidad
	if req.Tipo == "E" {
		cantidad = req.Cantidad
	} else {
		cantidad = -req.Cantidad
	}

	tx.Exec("UPDATE PRODUCTOS SET dinventario = dinventario + ? WHERE codigo=?", cantidad, req.CodigoProducto)
	tx.Exec(`INSERT INTO HISTORIAL_INVENTARIO (usuario_id, cuando_fue, tipo, habia, cantidad, codigo_producto, caja_id) VALUES (?,?,?,?,?,?,?)`,
		req.UsuarioID, now(), req.Tipo, habia, req.Cantidad, req.CodigoProducto, req.CajaID)
	tx.Commit()

	jsonResp(w, map[string]string{"ok": "Inventario ajustado"})
}

// --- Impuestos ---

func handleImpuestosList(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, COALESCE(nombre,''), COALESCE(porcentaje,0), COALESCE(defecto,'f'), COALESCE(activo,'t') FROM IMPUESTOS ORDER BY nombre")
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	type Impuesto struct {
		ID          int     `json:"id"`
		Nombre      string  `json:"nombre"`
		Porcentaje  float64 `json:"porcentaje"`
		Defecto     string  `json:"defecto"`
		Activo      string  `json:"activo"`
	}
	var is []Impuesto
	for rows.Next() {
		var i Impuesto
		rows.Scan(&i.ID, &i.Nombre, &i.Porcentaje, &i.Defecto, &i.Activo)
		is = append(is, i)
	}
	if is == nil {
		is = []Impuesto{}
	}
	jsonResp(w, is)
}

func handleImpuestosCreate(w http.ResponseWriter, r *http.Request) {
	var i struct {
		Nombre     string  `json:"nombre"`
		Porcentaje float64 `json:"porcentaje"`
		Defecto    string  `json:"defecto"`
		Activo     string  `json:"activo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&i); err != nil {
		jsonErr(w, "JSON invalido", 400)
		return
	}
	_, err := db.Exec("INSERT INTO IMPUESTOS (nombre, porcentaje, defecto, activo) VALUES (?,?,?,?)", i.Nombre, i.Porcentaje, i.Defecto, i.Activo)
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}
	jsonResp(w, map[string]string{"ok": "Impuesto creado"})
}

func handleImpuestosUpdate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var i struct {
		Nombre     string  `json:"nombre"`
		Porcentaje float64 `json:"porcentaje"`
		Defecto    string  `json:"defecto"`
		Activo     string  `json:"activo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&i); err != nil {
		jsonErr(w, "JSON invalido", 400)
		return
	}
	_, err := db.Exec("UPDATE IMPUESTOS SET nombre=?, porcentaje=?, defecto=?, activo=? WHERE id=?", i.Nombre, i.Porcentaje, i.Defecto, i.Activo, id)
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}
	jsonResp(w, map[string]string{"ok": "Impuesto actualizado"})
}

// --- Promociones ---

func handlePromocionesList(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT id, COALESCE(nombre,''), COALESCE(producto_codigo,''), COALESCE(desde,0), COALESCE(hasta,0), COALESCE(precio_promocion,0) FROM PROMOCIONES_POR_CANTIDAD ORDER BY nombre`)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	type Promocion struct {
		ID             int     `json:"id"`
		Nombre         string  `json:"nombre"`
		ProductoCodigo string  `json:"producto_codigo"`
		Desde          float64 `json:"desde"`
		Hasta          float64 `json:"hasta"`
		PrecioPromocion float64 `json:"precio_promocion"`
	}
	var ps []Promocion
	for rows.Next() {
		var p Promocion
		rows.Scan(&p.ID, &p.Nombre, &p.ProductoCodigo, &p.Desde, &p.Hasta, &p.PrecioPromocion)
		ps = append(ps, p)
	}
	if ps == nil {
		ps = []Promocion{}
	}
	jsonResp(w, ps)
}

func handlePromocionesCreate(w http.ResponseWriter, r *http.Request) {
	var p struct {
		Nombre          string  `json:"nombre"`
		ProductoCodigo  string  `json:"producto_codigo"`
		Desde           float64 `json:"desde"`
		Hasta           float64 `json:"hasta"`
		PrecioPromocion float64 `json:"precio_promocion"`
	}
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		jsonErr(w, "JSON invalido", 400)
		return
	}
	_, err := db.Exec("INSERT INTO PROMOCIONES_POR_CANTIDAD (nombre, producto_codigo, desde, hasta, precio_promocion) VALUES (?,?,?,?,?)", p.Nombre, p.ProductoCodigo, p.Desde, p.Hasta, p.PrecioPromocion)
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}
	jsonResp(w, map[string]string{"ok": "Promocion creada"})
}

func handlePromocionesDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	db.Exec("DELETE FROM PROMOCIONES_POR_CANTIDAD WHERE id=?", id)
	jsonResp(w, map[string]string{"ok": "Promocion eliminada"})
}

// --- Reportes ---

func handleReportesDashboard(w http.ResponseWriter, r *http.Request) {
	d := DashboardReport{}

	db.QueryRow(`SELECT COUNT(*) FROM VENTATICKETS WHERE DATE(creado_en)=DATE('now')`).Scan(&d.VentasHoy)
	db.QueryRow(`SELECT COALESCE(SUM(total),0) FROM VENTATICKETS WHERE DATE(creado_en)=DATE('now') AND esta_cancelado='f'`).Scan(&d.IngresosHoy)
	db.QueryRow(`SELECT COALESCE(SUM(ganancia),0) FROM VENTATICKETS WHERE DATE(creado_en)=DATE('now') AND esta_cancelado='f'`).Scan(&d.GananciaHoy)
	db.QueryRow(`SELECT COUNT(*) FROM VENTATICKETS WHERE strftime('%Y-%m', creado_en)=strftime('%Y-%m','now')`).Scan(&d.VentasMes)
	db.QueryRow(`SELECT COALESCE(SUM(total),0) FROM VENTATICKETS WHERE strftime('%Y-%m', creado_en)=strftime('%Y-%m','now') AND esta_cancelado='f'`).Scan(&d.IngresosMes)
	db.QueryRow(`SELECT COUNT(*) FROM PRODUCTOS WHERE COALESCE(dinventario,0) > 0`).Scan(&d.ProductosStock)
	db.QueryRow(`SELECT COALESCE(SUM(dinventario * pcosto),0) FROM PRODUCTOS WHERE COALESCE(dinventario,0) > 0`).Scan(&d.ValorInventario)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM OPERACIONES WHERE abierta='t'").Scan(&count)
	d.OperacionActiva = count > 0

	db.QueryRow(`SELECT COUNT(*) FROM VENTATICKETS WHERE esta_abierto='t' AND esta_cancelado='f'`).Scan(&d.TicketsAbiertos)

	jsonResp(w, d)
}

func handleReportesVentasDiarias(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT DATE(creado_en) as dia, COUNT(*) as tickets, COALESCE(SUM(total),0) as total, COALESCE(SUM(ganancia),0) as ganancia FROM VENTATICKETS WHERE esta_cancelado='f' GROUP BY DATE(creado_en) ORDER BY dia DESC LIMIT 30`)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var rs []map[string]interface{}
	for rows.Next() {
		var dia string
		var tickets int
		var total, ganancia float64
		rows.Scan(&dia, &tickets, &total, &ganancia)
		rs = append(rs, map[string]interface{}{"dia": dia, "tickets": tickets, "total": total, "ganancia": ganancia})
	}
	if rs == nil {
		rs = []map[string]interface{}{}
	}
	jsonResp(w, rs)
}

func handleReportesTopProductos(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT a.producto_nombre, SUM(a.cantidad) as vendidos, SUM(a.cantidad * a.precio_usado) as total FROM VENTATICKETS_ARTICULOS a JOIN VENTATICKETS t ON t.id=a.ticket_id WHERE t.esta_cancelado='f' GROUP BY a.producto_nombre ORDER BY vendidos DESC LIMIT 20`)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var rs []map[string]interface{}
	for rows.Next() {
		var nombre string
		var vendidos, total float64
		rows.Scan(&nombre, &vendidos, &total)
		rs = append(rs, map[string]interface{}{"nombre": nombre, "vendidos": vendidos, "total": total})
	}
	if rs == nil {
		rs = []map[string]interface{}{}
	}
	jsonResp(w, rs)
}
