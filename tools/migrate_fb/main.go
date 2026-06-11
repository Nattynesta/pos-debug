package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/nakagami/firebirdsql"
	_ "modernc.org/sqlite"
)

func main() {
	sqlitePath := os.Args[1]

	fb, err := sql.Open("firebirdsql", "sysdba:masterkey@localhost/pdv_test?charset=NONE")
	if err != nil {
		log.Fatalf("FB connect: %v", err)
	}
	defer fb.Close()

	var n int
	fb.QueryRow("SELECT COUNT(*) FROM PRODUCTOS").Scan(&n)
	fmt.Printf("Firebird: %d productos\n", n)

	tx, err := fb.Begin()
	if err != nil {
		log.Fatalf("FB tx: %v", err)
	}

	rows, err := tx.Query(`SELECT codigo, descripcion, tventa, pcosto, pventa, dept, provid, umedida, mayoreo, iprioridad, dinventario, dinvminimo, dinvmaximo, porcentaje_ganancia, componentes, impuestos FROM PRODUCTOS`)
	if err != nil {
		log.Fatalf("FB query: %v", err)
	}
	defer rows.Close()

	sq, err := sql.Open("sqlite", sqlitePath)
	if err != nil {
		log.Fatalf("SQLite open: %v", err)
	}
	defer sq.Close()

	stx, err := sq.Begin()
	if err != nil {
		log.Fatalf("SQLite tx: %v", err)
	}

	stmt, err := stx.Prepare(`INSERT OR REPLACE INTO PRODUCTOS (codigo, descripcion, tventa, pcosto, pventa, dept, provid, umedida, mayoreo, iprioridad, dinventario, dinvminimo, dinvmaximo, porcentaje_ganancia, componentes, impuestos) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		log.Fatalf("SQLite prep: %v", err)
	}

	count := 0
	for rows.Next() {
		var codigo, descripcion, tventa string
		var pcosto, pventa, mayoreo, dinventario, dinvminimo, dinvmaximo sql.NullFloat64
		var dept, provid, umedida, iprioridad, porcentajeGanancia sql.NullInt64
		var componentes, impuestos sql.NullString

		rows.Scan(&codigo, &descripcion, &tventa, &pcosto, &pventa, &dept, &provid, &umedida, &mayoreo, &iprioridad, &dinventario, &dinvminimo, &dinvmaximo, &porcentajeGanancia, &componentes, &impuestos)
		stmt.Exec(codigo, descripcion, tventa, nullFloat(pcosto), nullFloat(pventa), nullInt(dept), nullInt(provid), nullInt(umedida), nullFloat(mayoreo), nullInt(iprioridad), nullFloat(dinventario), nullFloat(dinvminimo), nullFloat(dinvmaximo), nullInt(porcentajeGanancia), nullStr(componentes), nullStr(impuestos))
		count++
	}
	rows.Close()
	stmt.Close()

	// Users
	urows, err := tx.Query("SELECT id, COALESCE(nombre_completo,''), COALESCE(direccion,''), COALESCE(telefono,''), usuario, clave, COALESCE(activo,'t'), COALESCE(created_on,''), COALESCE(correo,''), COALESCE(rol,'helper') FROM USUARIOS")
	if err == nil {
		defer urows.Close()
		ustmt, _ := stx.Prepare(`INSERT OR REPLACE INTO USUARIOS (id, nombre_completo, direccion, telefono, usuario, clave, activo, created_on, correo, rol) VALUES (?,?,?,?,?,?,?,?,?,?)`)
		ucount := 0
		for urows.Next() {
			var id int
			var nombre, dir, tel, usuario, clave, activo, createdOn, correo, rol string
			urows.Scan(&id, &nombre, &dir, &tel, &usuario, &clave, &activo, &createdOn, &correo, &rol)
			ustmt.Exec(id, nombre, dir, tel, usuario, clave, activo, createdOn, correo, rol)
			ucount++
		}
		ustmt.Close()
		fmt.Printf("Migrados %d usuarios\n", ucount)
	}

	// Departments
	drows, err := tx.Query("SELECT id, nombre, COALESCE(porcentaje_impuesto,0), COALESCE(activo,1) FROM DEPARTAMENTOS")
	if err == nil {
		defer drows.Close()
		dstmt, _ := stx.Prepare(`INSERT OR REPLACE INTO DEPARTAMENTOS (id, nombre, porcentaje_impuesto, activo) VALUES (?,?,?,?)`)
		dcount := 0
		for drows.Next() {
			var id int
			var nombre string
			var pct, activo int
			drows.Scan(&id, &nombre, &pct, &activo)
			dstmt.Exec(id, nombre, pct, activo)
			dcount++
		}
		dstmt.Close()
		fmt.Printf("Migrados %d departamentos\n", dcount)
	}

	// Measures
	mrows, err := tx.Query("SELECT codigo, nombre FROM MEDIDAS")
	if err == nil {
		defer mrows.Close()
		mstmt, _ := stx.Prepare(`INSERT OR REPLACE INTO MEDIDAS (codigo, nombre) VALUES (?,?)`)
		mcount := 0
		for mrows.Next() {
			var codigo int
			var nombre string
			mrows.Scan(&codigo, &nombre)
			mstmt.Exec(codigo, nombre)
			mcount++
		}
		mstmt.Close()
		fmt.Printf("Migrados %d medidas\n", mcount)
	}

	// Providers
	prows, err := tx.Query("SELECT num, COALESCE(nombre,''), COALESCE(direccion,''), COALESCE(telefonos,'') FROM PROV")
	if err == nil {
		defer prows.Close()
		pstmt, _ := stx.Prepare(`INSERT OR REPLACE INTO PROV (num, nombre, direccion, telefonos) VALUES (?,?,?,?)`)
		pcount := 0
		for prows.Next() {
			var num int
			var nombre, dir, tel string
			prows.Scan(&num, &nombre, &dir, &tel)
			pstmt.Exec(num, nombre, dir, tel)
			pcount++
		}
		pstmt.Close()
		fmt.Printf("Migrados %d proveedores\n", pcount)
	}

	fmt.Printf("Migracion completa: %d productos\n", count)
	stx.Commit()
}

func nullFloat(n sql.NullFloat64) interface{} {
	if n.Valid { return n.Float64 }
	return nil
}
func nullInt(n sql.NullInt64) interface{} {
	if n.Valid { return n.Int64 }
	return nil
}
func nullStr(n sql.NullString) interface{} {
	if n.Valid { return n.String }
	return nil
}
