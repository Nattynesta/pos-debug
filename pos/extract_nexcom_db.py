#!/usr/bin/env python3
"""Extract POS database from Nexcom and create a local SQLite copy."""
import hashlib
import json
import sqlite3
import urllib.request
import os
import sys

NEXCOM_BASE = "http://100.103.116.74:8080"
SCHEMA_FILE = "schema.sql"
DB_PATH = os.path.join(os.path.dirname(__file__), "pos.db")

def fetch_json(path):
    url = f"{NEXCOM_BASE}{path}"
    with urllib.request.urlopen(url, timeout=10) as r:
        return json.loads(r.read())

def apply_schema(conn):
    with open(os.path.join(os.path.dirname(__file__), SCHEMA_FILE)) as f:
        conn.executescript(f.read())
    # Apply same migrations as main.go
    migraciones = [
        "ALTER TABLE USUARIOS ADD COLUMN rol TEXT DEFAULT 'helper'",
    ]
    extra_cols = ['imagen_local', 'marca', 'categorias', 'ingredientes', 'nutriscore',
                  'cantidad_presentacion', 'nutricion', 'off_image_url', 'off_image_small']
    for col in extra_cols:
        migraciones.append(f"ALTER TABLE PRODUCTOS ADD COLUMN {col} TEXT DEFAULT ''")
    migraciones.append("""CREATE TABLE IF NOT EXISTS productos_openfoods (
        codigo TEXT PRIMARY KEY, nombre TEXT, marca TEXT, categorias TEXT,
        ingredientes TEXT, nutricion TEXT, nutriscore TEXT, cantidad_presentacion TEXT,
        imagen_url TEXT, imagen_small TEXT, imagen_grande TEXT, updated_at TEXT
    )""")
    migraciones.append("ALTER TABLE VENTATICKETS ADD COLUMN prioridad INTEGER DEFAULT 0")
    for m in migraciones:
        try:
            conn.execute(m)
        except sqlite3.OperationalError:
            pass  # column already exists
    conn.commit()

def clean_val(v):
    if v is None:
        return None
    if isinstance(v, str) and v.strip() == '':
        return None
    return v

def main():
    print("Conectando a Nexcom POS...")

    # Fetch all data
    productos = fetch_json("/api/productos")
    departamentos = fetch_json("/api/departamentos")
    cajas = fetch_json("/api/cajas")
    usuarios = fetch_json("/api/usuarios")
    clientes = fetch_json("/api/clientes")

    print(f"  Productos: {len(productos)}")
    print(f"  Departamentos: {len(departamentos)}")
    print(f"  Cajas: {len(cajas)}")
    print(f"  Usuarios: {len(usuarios)}")
    print(f"  Clientes: {len(clientes)}")

    # Create local DB
    if os.path.exists(DB_PATH):
        bak = DB_PATH + ".bak"
        os.rename(DB_PATH, bak)
        print(f"  Backup existente: {bak}")

    conn = sqlite3.connect(DB_PATH)
    conn.execute("PRAGMA journal_mode=WAL")
    conn.execute("PRAGMA foreign_keys=OFF")

    print("Aplicando schema...")
    apply_schema(conn)
    cur = conn.cursor()

    # Insert DEPARTAMENTOS
    for d in departamentos:
        cur.execute(
            "INSERT OR REPLACE INTO DEPARTAMENTOS (id, nombre, porcentaje_impuesto, activo) VALUES (?,?,?,?)",
            (d['id'], d['nombre'], d.get('porcentaje_impuesto', 0), d.get('activo', 't'))
        )

    # Insert CAJAS
    for c in cajas:
        cur.execute(
            "INSERT OR REPLACE INTO CAJAS (id, nombre, ultima_ip, ultimo_ingreso, nombre_pc) VALUES (?,?,?,?,?)",
            (c['id'], c.get('nombre', ''), c.get('ultima_ip', ''), c.get('ultimo_ingreso', ''), c.get('nombre_pc', ''))
        )

    # Insert USUARIOS
    for u in usuarios:
        cur.execute(
            "INSERT OR REPLACE INTO USUARIOS (id, nombre_completo, direccion, telefono, usuario, clave, activo, created_on, correo, esta_en_caja_id, rol) VALUES (?,?,?,?,?,?,?,?,?,?,?)",
            (u['id'], u.get('nombre_completo', ''), u.get('direccion', ''),
             u.get('telefono', ''), u['usuario'], hashlib.sha256(b'admin').hexdigest(),  # hashed password
             u.get('activo', 't'), u.get('created_on', ''),
             u.get('correo', ''), u.get('esta_en_caja_id'), u.get('rol', 'helper'))
        )

    # Insert CLIENTES
    for cl in clientes:
        cur.execute(
            "INSERT OR REPLACE INTO CLIENTES (numero, nombre, direccion, telefono, dsaldoactual, dtactualizasaldo, limite_credito, ultimo_pago_en, folio) VALUES (?,?,?,?,?,?,?,?,?)",
            (cl['numero'], cl.get('nombre', ''), cl.get('direccion', ''),
             cl.get('telefono', ''), cl.get('dsaldoactual', 0),
             cl.get('dtactualizasaldo', ''), cl.get('limite_credito', 0),
             cl.get('ultimo_pago_en', ''), cl.get('folio', 1))
        )

    # Insert PRODUCTOS + productos_openfoods
    inserted = 0
    for p in productos:
        cur.execute("""INSERT OR REPLACE INTO PRODUCTOS
            (codigo, descripcion, tventa, pcosto, pventa, dept, provid, umedida,
             mayoreo, iprioridad, dinventario, dinvminimo, dinvmaximo, checado_en,
             porcentaje_ganancia, componentes, impuestos,
             imagen_local, marca, categorias, ingredientes, nutriscore,
             cantidad_presentacion, nutricion, off_image_url, off_image_small)
            VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)""",
            (p['codigo'], p.get('descripcion', ''), p.get('tventa', 'U'),
             clean_val(p.get('pcosto')), clean_val(p.get('pventa')),
             p.get('dept'), p.get('provid'), p.get('umedida'),
             clean_val(p.get('mayoreo')), p.get('iprioridad'),
             clean_val(p.get('dinventario')), clean_val(p.get('dinvminimo')),
             clean_val(p.get('dinvmaximo')), p.get('checado_en', ''),
             p.get('porcentaje_ganancia', 0), p.get('componentes', ''),
             p.get('impuestos', ''),
             p.get('imagen_local', ''), p.get('marca', ''), p.get('categorias', ''),
             p.get('ingredientes', ''), p.get('nutriscore', ''),
             p.get('cantidad_presentacion', ''), p.get('nutricion', ''),
             p.get('off_image_url', ''), p.get('off_image_small', '')))

        # Insert into productos_openfoods if there's OFF data
        if p.get('off_image_url') or p.get('off_name'):
            cur.execute("""INSERT OR REPLACE INTO productos_openfoods
                (codigo, nombre, marca, categorias, ingredientes, nutricion,
                 nutriscore, cantidad_presentacion, imagen_url, imagen_small, imagen_grande, updated_at)
                VALUES (?,?,?,?,?,?,?,?,?,?,?,datetime('now'))""",
                (p['codigo'], p.get('off_name', p.get('descripcion', '')),
                 p.get('marca', ''), p.get('categorias', ''),
                 p.get('ingredientes', ''), p.get('nutricion', ''),
                 p.get('nutriscore', ''), p.get('cantidad_presentacion', ''),
                 p.get('off_image_url', ''), p.get('off_image_small', ''),
                 p.get('off_image_grande', '')))
        inserted += 1

    # Insert DEPTS (same as DEPARTAMENTOS for compatibility)
    for d in departamentos:
        cur.execute(
            "INSERT OR REPLACE INTO DEPTS (num, nombre) VALUES (?,?)",
            (d['id'], d['nombre'])
        )

    # Seed PROV with default entry (referenced by products with provid=0)
    cur.execute("INSERT OR REPLACE INTO PROV (num, nombre) VALUES (0, 'Sin Proveedor')")
    # Seed MEDIDAS with values referenced by products
    umedidas = set()
    for p in productos:
        if p.get('umedida'):
            umedidas.add(p['umedida'])
    for u in umedidas:
        cur.execute("INSERT OR REPLACE INTO MEDIDAS (codigo, nombre) VALUES (?, 'Unidad ' || ?)", (u, str(u)[-4:]))

    # Insert default schema version
    cur.execute("INSERT OR REPLACE INTO SCHEMA_INFO (version_db) VALUES (1)")

    conn.commit()
    conn.close()
    print(f"\n✅ DB creada en {DB_PATH}")
    print(f"   {inserted} productos insertados")

if __name__ == "__main__":
    main()
