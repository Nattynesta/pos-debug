package main

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

type Producto struct {
	Codigo              string  `json:"codigo"`
	Descripcion         string  `json:"descripcion"`
	Tventa              string  `json:"tventa"`
	Pcosto              float64 `json:"pcosto"`
	Pventa              float64 `json:"pventa"`
	Dept                *int    `json:"dept"`
	Provid              *int    `json:"provid"`
	Umedida             *int    `json:"umedida"`
	Mayoreo             float64 `json:"mayoreo"`
	Iprioridad          *int    `json:"iprioridad"`
	Dinventario         float64 `json:"dinventario"`
	Dinvminimo          float64 `json:"dinvminimo"`
	Dinvmaximo          float64 `json:"dinvmaximo"`
	ChecadoEn           string  `json:"checado_en"`
	PorcentajeGanancia  int     `json:"porcentaje_ganancia"`
	Componentes         string  `json:"componentes"`
	Impuestos           string  `json:"impuestos"`
	ImagenLocal         string  `json:"imagen_local,omitempty"`
	ImagenThumb         string  `json:"imagen_thumb,omitempty"`
	Marca               string  `json:"marca,omitempty"`
	Categorias          string  `json:"categorias,omitempty"`
	Ingredientes        string  `json:"ingredientes,omitempty"`
	Nutriscore          string  `json:"nutriscore,omitempty"`
	CantidadPresentacion string `json:"cantidad_presentacion,omitempty"`
	Nutricion           string  `json:"nutricion,omitempty"`
}

type Cliente struct {
	Numero           int     `json:"numero"`
	Nombre           string  `json:"nombre"`
	Direccion        string  `json:"direccion"`
	Telefono         string  `json:"telefono"`
	Dsaldoactual     float64 `json:"dsaldoactual"`
	Dtactualizasaldo string  `json:"dtactualizasaldo"`
	LimiteCredito    float64 `json:"limite_credito"`
	UltimoPagoEn     string  `json:"ultimo_pago_en"`
	Folio            int     `json:"folio"`
}

type Proveedor struct {
	Num       int    `json:"num"`
	Nombre    string `json:"nombre"`
	Direccion string `json:"direccion"`
	Telefonos string `json:"telefonos"`
}

type Departamento struct {
	ID                int    `json:"id"`
	Nombre            string `json:"nombre"`
	PorcentajeImpuesto int   `json:"porcentaje_impuesto"`
	Activo            string `json:"activo"`
	Orden             int    `json:"orden"`
}

type Medida struct {
	Codigo int    `json:"codigo"`
	Nombre string `json:"nombre"`
}

type Usuario struct {
	ID             int    `json:"id"`
	NombreCompleto string `json:"nombre_completo"`
	Direccion      string `json:"direccion"`
	Telefono       string `json:"telefono"`
	Usuario        string `json:"usuario"`
	Clave          string `json:"clave,omitempty"`
	Rol            string `json:"rol"`
	Activo         string `json:"activo"`
	CreatedOn      string `json:"created_on"`
	Correo         string `json:"correo"`
	EstaEnCajaID   *int   `json:"esta_en_caja_id"`
}

type Caja struct {
	ID           int    `json:"id"`
	Nombre       string `json:"nombre"`
	UltimaIP     string `json:"ultima_ip"`
	UltimoIngreso string `json:"ultimo_ingreso"`
	NombrePC     string `json:"nombre_pc"`
}

type Operacion struct {
	ID              int     `json:"id"`
	DineroEnCaja    float64 `json:"dinero_en_caja"`
	TipoDeCambio    float64 `json:"tipo_de_cambio"`
	InicioUsuarioID int     `json:"inicio_usuario_id"`
	InicioEn        string  `json:"inicio_en"`
	CerroEn         *string `json:"cerro_en"`
	CajaID          int     `json:"caja_id"`
	Abierta         string  `json:"abierta"`
	Ventas          float64 `json:"ventas"`
	Salidas         float64 `json:"salidas"`
	Entradas        float64 `json:"entradas"`
	Pagos           float64 `json:"pagos"`
	Impuestos       float64 `json:"impuestos"`
	Ganancias       float64 `json:"ganancias"`
	IngresosTarjeta float64 `json:"ingresos_tarjeta"`
	IngresosVales   float64 `json:"ingresos_vales"`
	IngresosEfectivo float64 `json:"ingresos_efectivo"`
}

type VentaTicket struct {
	ID             int      `json:"id"`
	Folio          *int     `json:"folio"`
	CajaID         int      `json:"caja_id"`
	CajeroID       int      `json:"cajero_id"`
	Nombre         string   `json:"nombre"`
	Prioridad      int      `json:"prioridad"`
	CreadoEn       string   `json:"creado_en"`
	Subtotal       float64  `json:"subtotal"`
	Impuestos      float64  `json:"impuestos"`
	Total          float64  `json:"total"`
	Ganancia       float64  `json:"ganancia"`
	EstaAbierto    string   `json:"esta_abierto"`
	ClienteID      *int     `json:"cliente_id"`
	VendidoEn      string   `json:"vendido_en"`
	EsModificable  string   `json:"es_modificable"`
	PagoCon        float64  `json:"pago_con"`
	Moneda         string   `json:"moneda"`
	NumeroArticulos int     `json:"numero_articulos"`
	PagadoEn       string   `json:"pagado_en"`
	EstaCancelado  string   `json:"esta_cancelado"`
	OperacionID    int      `json:"operacion_id"`
	FormaPago      string   `json:"forma_pago"`
	Referencia     string   `json:"referencia"`
	TotalDevuelto  float64  `json:"total_devuelto"`
	ClienteNombre  string   `json:"cliente_nombre"`
	ClienteDireccion string `json:"cliente_direccion"`
	CajeroNombre   string   `json:"cajero_nombre"`
	Articulos      []TicketArticulo `json:"articulos,omitempty"`
}

type TicketArticulo struct {
	ID                int     `json:"id"`
	TicketID          int     `json:"ticket_id"`
	ProductoCodigo    string  `json:"producto_codigo"`
	ProductoNombre    string  `json:"producto_nombre"`
	Cantidad          float64 `json:"cantidad"`
	Ganancia          float64 `json:"ganancia"`
	DepartamentoID    *int    `json:"departamento_id"`
	PagadoEn          string  `json:"pagado_en"`
	UsaMayoreo        string  `json:"usa_mayoreo"`
	PorcentajeDescuento float64 `json:"porcentaje_descuento"`
	Componentes       string  `json:"componentes"`
	ImpuestosUsados   string  `json:"impuestos_usados"`
	ImpuestoUnitario  float64 `json:"impuesto_unitario"`
	PrecioUsado       float64 `json:"precio_usado"`
	CantidadDevuelta  float64 `json:"cantidad_devuelta"`
	FueDevuelto       string  `json:"fue_devuelto"`
	PorcentajePagado  int     `json:"porcentaje_pagado"`
}

type Movimiento struct {
	ID          int     `json:"id"`
	OperacionID int     `json:"operacion_id"`
	Monto       float64 `json:"monto"`
	CuandoFue   string  `json:"cuando_fue"`
	Comentarios string  `json:"comentarios"`
	Tipo        string  `json:"tipo"`
	ClienteID   *int    `json:"cliente_id"`
	CajaID      int     `json:"caja_id"`
	CajeroID    int     `json:"cajero_id"`
}

type DashboardReport struct {
	VentasHoy       int     `json:"ventas_hoy"`
	IngresosHoy     float64 `json:"ingresos_hoy"`
	GananciaHoy     float64 `json:"ganancia_hoy"`
	VentasMes       int     `json:"ventas_mes"`
	IngresosMes     float64 `json:"ingresos_mes"`
	ProductosStock  int     `json:"productos_stock"`
	ValorInventario float64 `json:"valor_inventario"`
	OperacionActiva bool    `json:"operacion_activa"`
	TicketsAbiertos int     `json:"tickets_abiertos"`
}

type Pago struct {
	ID        int     `json:"id"`
	TicketID  int     `json:"ticket_id"`
	Metodo    string  `json:"metodo"`
	Monto     float64 `json:"monto"`
	Recibido  float64 `json:"recibido"`
	Cambio    float64 `json:"cambio"`
	Referencia string `json:"referencia"`
	Fecha     string  `json:"fecha"`
}

type PagoRequest struct {
	Metodo    string  `json:"metodo"`
	Monto     float64 `json:"monto"`
	Recibido  float64 `json:"recibido,omitempty"`
	Referencia string `json:"referencia,omitempty"`
}

type Pedido struct {
	ID               int     `json:"id"`
	Items            string  `json:"items"`
	Total            float64 `json:"total"`
	Prioridad        string  `json:"prioridad"`
	Notas            string  `json:"notas"`
	ClienteNombre    string  `json:"cliente_nombre"`
	ClienteDireccion string  `json:"cliente_direccion"`
	ClienteTelefono  string  `json:"cliente_telefono"`
	EsAdeudo         int     `json:"es_adeudo"`
	CreadoPorID      int     `json:"creado_por_id"`
	AsignadoAID      *int    `json:"asignado_a_id"`
	Estado           string  `json:"estado"`
	CreatedOn        string  `json:"created_on"`
	CompletadoOn     string  `json:"completado_on"`
	CreadoPorNombre  string  `json:"creado_por_nombre"`
	AsignadoANombre  string  `json:"asignado_a_nombre"`
}

// --- Pagos ---

func createPago(tx *sql.Tx, ticketID int, p PagoRequest) (int, error) {
	cambio := 0.0
	if p.Metodo == "e" && p.Recibido > p.Monto {
		cambio = p.Recibido - p.Monto
	}
	res, err := tx.Exec(
		`INSERT INTO PAGOS (ticket_id, metodo, monto, recibido, cambio, referencia, fecha) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		ticketID, p.Metodo, p.Monto, p.Recibido, cambio, p.Referencia, now(),
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

func listPagos(ticketID int) ([]Pago, error) {
	rows, err := db.Query(`SELECT id, ticket_id, metodo, monto, COALESCE(recibido,0), COALESCE(cambio,0), COALESCE(referencia,''), fecha FROM PAGOS WHERE ticket_id=? ORDER BY id`, ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ps := make([]Pago, 0)
	for rows.Next() {
		var p Pago
		rows.Scan(&p.ID, &p.TicketID, &p.Metodo, &p.Monto, &p.Recibido, &p.Cambio, &p.Referencia, &p.Fecha)
		ps = append(ps, p)
	}
	return ps, nil
}

func migrateLegacyPagos() {
	var count int
	db.QueryRow("SELECT COUNT(*) FROM PAGOS").Scan(&count)
	if count > 0 {
		return
	}
	rows, err := db.Query(`SELECT id, COALESCE(NULLIF(forma_pago,''),'e'), COALESCE(pago_con,0), COALESCE(total_devuelto,0) FROM VENTATICKETS WHERE esta_abierto='f' AND esta_cancelado='f'`)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var metodo string
		var pagoCon, cambio float64
		rows.Scan(&id, &metodo, &pagoCon, &cambio)
		recibido := pagoCon
		if pagoCon <= 0 {
			recibido = pagoCon + cambio
		}
		monto := pagoCon - cambio
		if monto < 0 {
			monto = 0
		}
		db.Exec(`INSERT INTO PAGOS (ticket_id, metodo, monto, recibido, cambio, fecha) VALUES (?, ?, ?, ?, ?, (SELECT COALESCE(pagado_en, datetime('now','localtime')) FROM VENTATICKETS WHERE id=?))`, id, metodo, monto, recibido, cambio, id)
	}
}

func now() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func today() string {
	return time.Now().Format("2006-01-02")
}

func nextFolio(tx *sql.Tx) int {
	var folio int
	tx.QueryRow("SELECT COALESCE(MAX(folio), 0) + 1 FROM VENTATICKETS").Scan(&folio)
	return folio
}

func listProductos() ([]Producto, error) {
	rows, err := db.Query(`
		SELECT 
			p.codigo, p.descripcion, p.tventa, COALESCE(p.pcosto,0), COALESCE(p.pventa,0), 
			p.dept, p.provid, p.umedida, COALESCE(p.mayoreo,0), p.iprioridad, 
			COALESCE(p.dinventario,0), COALESCE(p.dinvminimo,0), COALESCE(p.dinvmaximo,0), 
			COALESCE(p.checado_en,''), COALESCE(p.porcentaje_ganancia,0), COALESCE(p.componentes,''), COALESCE(p.impuestos,''),
			COALESCE(p.imagen_local,''),
			COALESCE(p.marca,''), COALESCE(p.categorias,''), COALESCE(p.ingredientes,''),
			COALESCE(p.nutriscore,''), COALESCE(p.cantidad_presentacion,''), COALESCE(p.nutricion,'')
		FROM PRODUCTOS p
		ORDER BY p.descripcion`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ps := make([]Producto, 0)
	for rows.Next() {
		var p Producto
		rows.Scan(&p.Codigo, &p.Descripcion, &p.Tventa, &p.Pcosto, &p.Pventa, &p.Dept, &p.Provid, &p.Umedida, &p.Mayoreo, &p.Iprioridad, &p.Dinventario, &p.Dinvminimo, &p.Dinvmaximo, &p.ChecadoEn, &p.PorcentajeGanancia, &p.Componentes, &p.Impuestos, &p.ImagenLocal, &p.Marca, &p.Categorias, &p.Ingredientes, &p.Nutriscore, &p.CantidadPresentacion, &p.Nutricion)
		ps = append(ps, p)
	}
	return ps, nil
}
