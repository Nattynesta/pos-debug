-- Schema exacto de "abarrotes punto de venta" (Firebird → SQLite)
-- Traducido fielmente de la base de datos original PDVDATA.FDB

CREATE TABLE IF NOT EXISTS USUARIOS (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nombre_completo TEXT,
    direccion TEXT,
    telefono TEXT,
    usuario TEXT NOT NULL,
    clave TEXT NOT NULL,
    activo TEXT DEFAULT 't',
    permisos BLOB,
    created_on TEXT,
    correo TEXT,
    esta_en_caja_id INTEGER,
    rol TEXT DEFAULT 'helper'
);

CREATE TABLE IF NOT EXISTS CAJAS (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nombre TEXT,
    ultima_ip TEXT NOT NULL,
    ultimo_ingreso TEXT,
    nombre_pc TEXT
);

CREATE TABLE IF NOT EXISTS OPERACIONES (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    dinero_en_caja REAL DEFAULT 0,
    tipo_de_cambio REAL DEFAULT 0,
    inicio_usuario_id INTEGER NOT NULL,
    inicio_en TEXT NOT NULL,
    cerro_en TEXT,
    caja_id INTEGER NOT NULL,
    abierta TEXT DEFAULT 't',
    ventas REAL DEFAULT 0,
    salidas REAL DEFAULT 0,
    entradas REAL DEFAULT 0,
    pagos REAL DEFAULT 0,
    impuestos REAL DEFAULT 0,
    ganancias REAL DEFAULT 0,
    abono_id INTEGER,
    ingresos_tarjeta REAL DEFAULT 0,
    ingresos_vales REAL DEFAULT 0,
    ingresos_efectivo REAL DEFAULT 0,
    FOREIGN KEY (caja_id) REFERENCES CAJAS(id),
    FOREIGN KEY (inicio_usuario_id) REFERENCES USUARIOS(id)
);

CREATE TABLE IF NOT EXISTS CLIENTES (
    numero INTEGER PRIMARY KEY AUTOINCREMENT,
    nombre TEXT,
    direccion TEXT,
    telefono TEXT,
    dsaldoactual REAL,
    dtactualizasaldo TEXT,
    limite_credito REAL,
    ultimo_pago_en TEXT,
    folio INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS FACTURACION_CLIENTES (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    rfc TEXT NOT NULL,
    nombre TEXT,
    calle TEXT,
    noexterior TEXT,
    nointerior TEXT,
    colonia TEXT,
    localidad TEXT,
    municipio TEXT,
    estado TEXT,
    pais TEXT,
    email TEXT,
    referencia TEXT,
    codigopostal INTEGER
);

CREATE TABLE IF NOT EXISTS FACTURACION_EMISORES (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    rfc TEXT NOT NULL,
    nombre TEXT,
    calle TEXT,
    noexterior TEXT,
    nointerior TEXT,
    colonia TEXT,
    localidad TEXT,
    municipio TEXT,
    estado TEXT,
    pais TEXT,
    email TEXT,
    referencia TEXT,
    codigopostal INTEGER,
    activo TEXT
);

CREATE TABLE IF NOT EXISTS FACTURACION_CERTIFICADOS (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    emisor_id INTEGER NOT NULL,
    numero_serie TEXT NOT NULL,
    vigencia_inicio TEXT NOT NULL,
    vigencia_fin TEXT NOT NULL,
    clave_llave_privada TEXT NOT NULL,
    FOREIGN KEY (emisor_id) REFERENCES FACTURACION_EMISORES(id)
);

CREATE TABLE IF NOT EXISTS FACTURACION_FOLIOS (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    emisor_id INTEGER NOT NULL,
    serie TEXT,
    folio_inicial INTEGER NOT NULL,
    folio_final INTEGER NOT NULL,
    siguiente_folio INTEGER NOT NULL,
    numero_aprobacion TEXT NOT NULL,
    ano_aprobacion INTEGER NOT NULL,
    FOREIGN KEY (emisor_id) REFERENCES FACTURACION_EMISORES(id)
);

CREATE TABLE IF NOT EXISTS FACTURACION_INFORMES (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    mes INTEGER,
    ano INTEGER,
    generado_en TEXT,
    contenido BLOB,
    enviado_en TEXT,
    tipo TEXT DEFAULT 'n'
);

CREATE TABLE IF NOT EXISTS DEPARTAMENTOS (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nombre TEXT NOT NULL,
    porcentaje_impuesto INTEGER DEFAULT 0,
    activo TEXT DEFAULT 1,
    UNIQUE(nombre, activo)
);

CREATE TABLE IF NOT EXISTS DEPTS (
    num INTEGER PRIMARY KEY AUTOINCREMENT,
    nombre TEXT
);

CREATE TABLE IF NOT EXISTS PROV (
    num INTEGER PRIMARY KEY AUTOINCREMENT,
    nombre TEXT,
    direccion TEXT,
    telefonos TEXT
);

CREATE TABLE IF NOT EXISTS MEDIDAS (
    codigo INTEGER PRIMARY KEY AUTOINCREMENT,
    nombre TEXT
);

CREATE TABLE IF NOT EXISTS PRODUCTOS (
    codigo TEXT PRIMARY KEY,
    descripcion TEXT,
    tventa TEXT NOT NULL,
    pcosto REAL,
    pventa REAL,
    dept INTEGER,
    provid INTEGER,
    umedida INTEGER,
    mayoreo REAL,
    iprioridad INTEGER,
    dinventario REAL,
    dinvminimo REAL,
    dinvmaximo REAL,
    checado_en TEXT,
    porcentaje_ganancia INTEGER DEFAULT 0,
    componentes TEXT,
    impuestos TEXT,
    FOREIGN KEY (dept) REFERENCES DEPARTAMENTOS(id),
    FOREIGN KEY (provid) REFERENCES PROV(num),
    FOREIGN KEY (umedida) REFERENCES MEDIDAS(codigo)
);

CREATE TABLE IF NOT EXISTS PRODUCTOS_BASE (
    codigo TEXT PRIMARY KEY,
    descripcion TEXT
);

CREATE TABLE IF NOT EXISTS IMPUESTOS (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nombre TEXT,
    porcentaje REAL,
    defecto TEXT,
    activo TEXT
);

CREATE TABLE IF NOT EXISTS PROMOCIONES_POR_CANTIDAD (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nombre TEXT,
    producto_codigo TEXT,
    desde REAL,
    hasta REAL,
    precio_promocion REAL,
    FOREIGN KEY (producto_codigo) REFERENCES PRODUCTOS(codigo)
);

CREATE TABLE IF NOT EXISTS CONFIGURACION (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    parametro TEXT,
    valor TEXT,
    caja_id INTEGER
);

CREATE TABLE IF NOT EXISTS VENTATICKETS (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    folio INTEGER,
    caja_id INTEGER NOT NULL,
    cajero_id INTEGER NOT NULL,
    nombre TEXT,
    creado_en TEXT,
    subtotal REAL DEFAULT 0,
    impuestos REAL DEFAULT 0,
    total REAL DEFAULT 0,
    ganancia REAL DEFAULT 0,
    esta_abierto TEXT DEFAULT 't',
    cliente_id INTEGER,
    vendido_en TEXT,
    es_modificable TEXT DEFAULT 't',
    pago_con REAL,
    moneda TEXT,
    numero_articulos INTEGER DEFAULT 0,
    pagado_en TEXT,
    esta_cancelado TEXT DEFAULT 'f',
    operacion_id INTEGER NOT NULL,
    old_ticket_id INTEGER,
    notas BLOB,
    imprimir_nota TEXT DEFAULT 't',
    forma_pago TEXT,
    referencia TEXT,
    factura_id INTEGER,
    total_devuelto REAL DEFAULT 0,
    FOREIGN KEY (caja_id) REFERENCES CAJAS(id),
    FOREIGN KEY (cajero_id) REFERENCES USUARIOS(id),
    FOREIGN KEY (cliente_id) REFERENCES CLIENTES(numero),
    FOREIGN KEY (operacion_id) REFERENCES OPERACIONES(id),
    FOREIGN KEY (factura_id) REFERENCES FACTURAS(id)
);

CREATE TABLE IF NOT EXISTS VENTATICKETS_ARTICULOS (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ticket_id INTEGER NOT NULL,
    producto_codigo TEXT NOT NULL,
    producto_nombre TEXT NOT NULL,
    cantidad REAL NOT NULL,
    ganancia REAL,
    departamento_id INTEGER,
    pagado_en TEXT,
    usa_mayoreo TEXT DEFAULT 'f',
    porcentaje_descuento REAL,
    componentes TEXT,
    impuestos_usados TEXT,
    impuesto_unitario REAL,
    precio_usado REAL,
    cantidad_devuelta REAL DEFAULT 0,
    fue_devuelto TEXT DEFAULT 'f',
    porcentaje_pagado INTEGER DEFAULT 0,
    FOREIGN KEY (ticket_id) REFERENCES VENTATICKETS(id),
    FOREIGN KEY (producto_codigo) REFERENCES PRODUCTOS(codigo)
);

CREATE TABLE IF NOT EXISTS VENTAS (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    producto_codigo TEXT,
    cantidad REAL,
    fecha TEXT,
    ticket_id INTEGER,
    FOREIGN KEY (ticket_id) REFERENCES VENTATICKETS(id),
    FOREIGN KEY (producto_codigo) REFERENCES PRODUCTOS(codigo)
);

CREATE TABLE IF NOT EXISTS FACTURAS (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    serie TEXT,
    folio TEXT,
    folio_id INTEGER NOT NULL,
    generada_en TEXT NOT NULL,
    transaccion_de TEXT,
    facturacion_cliente_id INTEGER NOT NULL,
    facturacion_emisor_id INTEGER NOT NULL,
    ventaticket_id INTEGER,
    ventaticket_folio INTEGER,
    subtotal REAL,
    impuestos REAL,
    total REAL,
    xml BLOB NOT NULL,
    cancelada_en TEXT,
    tipo TEXT DEFAULT 'n',
    informe_id INTEGER,
    certificado_id INTEGER,
    FOREIGN KEY (folio_id) REFERENCES FACTURACION_FOLIOS(id),
    FOREIGN KEY (facturacion_cliente_id) REFERENCES FACTURACION_CLIENTES(id),
    FOREIGN KEY (facturacion_emisor_id) REFERENCES FACTURACION_EMISORES(id),
    FOREIGN KEY (ventaticket_id) REFERENCES VENTATICKETS(id),
    FOREIGN KEY (informe_id) REFERENCES FACTURACION_INFORMES(id),
    FOREIGN KEY (certificado_id) REFERENCES FACTURACION_CERTIFICADOS(id)
);

CREATE TABLE IF NOT EXISTS MOVIMIENTOS (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    operacion_id INTEGER NOT NULL,
    monto REAL,
    cuando_fue TEXT NOT NULL,
    comentarios TEXT,
    tipo TEXT NOT NULL,
    cliente_id INTEGER,
    caja_id INTEGER NOT NULL,
    cajero_id INTEGER NOT NULL,
    abono_id INTEGER,
    FOREIGN KEY (operacion_id) REFERENCES OPERACIONES(id),
    FOREIGN KEY (cliente_id) REFERENCES CLIENTES(numero),
    FOREIGN KEY (caja_id) REFERENCES CAJAS(id),
    FOREIGN KEY (cajero_id) REFERENCES USUARIOS(id)
);

CREATE TABLE IF NOT EXISTS ABONOS (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cliente_id INTEGER,
    dtfecha TEXT,
    dmonto REAL,
    bcontar TEXT,
    ganancia REAL DEFAULT 0,
    FOREIGN KEY (cliente_id) REFERENCES CLIENTES(numero)
);

CREATE TABLE IF NOT EXISTS HISTORIAL_INVENTARIO (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    usuario_id INTEGER NOT NULL,
    cuando_fue TEXT NOT NULL,
    tipo TEXT NOT NULL,
    habia REAL,
    cantidad REAL NOT NULL,
    codigo_producto TEXT NOT NULL,
    caja_id INTEGER,
    FOREIGN KEY (usuario_id) REFERENCES USUARIOS(id),
    FOREIGN KEY (codigo_producto) REFERENCES PRODUCTOS(codigo),
    FOREIGN KEY (caja_id) REFERENCES CAJAS(id)
);

CREATE TABLE IF NOT EXISTS HISTORIAL_USUARIOS (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    usuario_id INTEGER,
    cuando TEXT,
    caja_id INTEGER,
    movimiento TEXT,
    FOREIGN KEY (usuario_id) REFERENCES USUARIOS(id),
    FOREIGN KEY (caja_id) REFERENCES CAJAS(id)
);

CREATE TABLE IF NOT EXISTS PRODUCTOS_OFF (
    codigo TEXT PRIMARY KEY,
    image_url TEXT,
    image_small TEXT,
    name TEXT,
    last_sync TEXT,
    FOREIGN KEY (codigo) REFERENCES PRODUCTOS(codigo)
);

CREATE TABLE IF NOT EXISTS SCHEMA_INFO (
    version_db INTEGER
);
