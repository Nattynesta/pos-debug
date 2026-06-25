package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"abarrotes-pdv/printer"
)

// --- Productos ---

type PaginationParams struct {
	Page   int
	Limit  int
	Offset int
}

func parsePagination(r *http.Request) PaginationParams {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 50
	}
	return PaginationParams{
		Page:   page,
		Limit:  limit,
		Offset: (page - 1) * limit,
	}
}

func handleProductosList(w http.ResponseWriter, r *http.Request) {
	hasPage := r.URL.Query().Has("page") || r.URL.Query().Has("limit")
	p := parsePagination(r)
	search := r.URL.Query().Get("q")

	var total int
	var rows *sql.Rows
	var err error

	if search != "" {
		searchArg := strings.ReplaceAll(search, " ", "* ") + "*"
		if hasPage {
			db.QueryRow(`SELECT COUNT(*) FROM PRODUCTOS p JOIN productos_fts fts ON p.codigo = fts.codigo WHERE productos_fts MATCH ?`, searchArg).Scan(&total)
		}

		query := `
			SELECT p.codigo, p.descripcion, p.tventa, COALESCE(p.pcosto,0), COALESCE(p.pventa,0),
			p.dept, p.provid, p.umedida, COALESCE(p.mayoreo,0), p.iprioridad,
			COALESCE(p.dinventario,0), COALESCE(p.dinvminimo,0), COALESCE(p.dinvmaximo,0),
			COALESCE(p.checado_en,''), COALESCE(p.porcentaje_ganancia,0), COALESCE(p.componentes,''), COALESCE(p.impuestos,''),
			COALESCE(p.imagen_local,''),
			COALESCE(p.imagen_local,''),
			COALESCE(p.imagen_thumb,''),
			COALESCE(p.marca,''), COALESCE(p.categorias,''),
			COALESCE(p.ingredientes,''), COALESCE(p.nutriscore,''),
			COALESCE(p.cantidad_presentacion,''), COALESCE(p.nutricion,'')
			FROM PRODUCTOS p
			JOIN productos_fts fts ON p.codigo = fts.codigo
			WHERE productos_fts MATCH ?
			ORDER BY rank`
		if hasPage {
			query += ` LIMIT ? OFFSET ?`
			rows, err = db.Query(query, searchArg, p.Limit, p.Offset)
		} else {
			rows, err = db.Query(query, searchArg)
		}
	} else {
		if hasPage {
			db.QueryRow(`SELECT COUNT(*) FROM PRODUCTOS`).Scan(&total)
		}

		query := `
			SELECT p.codigo, p.descripcion, p.tventa, COALESCE(p.pcosto,0), COALESCE(p.pventa,0),
			p.dept, p.provid, p.umedida, COALESCE(p.mayoreo,0), p.iprioridad,
			COALESCE(p.dinventario,0), COALESCE(p.dinvminimo,0), COALESCE(p.dinvmaximo,0),
			COALESCE(p.checado_en,''), COALESCE(p.porcentaje_ganancia,0), COALESCE(p.componentes,''), COALESCE(p.impuestos,''),
			COALESCE(p.imagen_local,''),
			COALESCE(p.imagen_thumb,''),
			COALESCE(p.marca,''), COALESCE(p.categorias,''),
			COALESCE(p.ingredientes,''), COALESCE(p.nutriscore,''),
			COALESCE(p.cantidad_presentacion,''), COALESCE(p.nutricion,'')
			FROM PRODUCTOS p
			ORDER BY p.descripcion`
		if hasPage {
			query += ` LIMIT ? OFFSET ?`
			rows, err = db.Query(query, p.Limit, p.Offset)
		} else {
			rows, err = db.Query(query)
		}
	}

	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	ps := make([]Producto, 0)
	for rows.Next() {
		var pr Producto
		rows.Scan(&pr.Codigo, &pr.Descripcion, &pr.Tventa, &pr.Pcosto, &pr.Pventa, &pr.Dept, &pr.Provid, &pr.Umedida, &pr.Mayoreo, &pr.Iprioridad, &pr.Dinventario, &pr.Dinvminimo, &pr.Dinvmaximo, &pr.ChecadoEn, &pr.PorcentajeGanancia, &pr.Componentes, &pr.Impuestos, &pr.ImagenLocal, &pr.ImagenThumb, &pr.Marca, &pr.Categorias, &pr.Ingredientes, &pr.Nutriscore, &pr.CantidadPresentacion, &pr.Nutricion)
		if pr.ImagenThumb == "" && pr.ImagenLocal != "" {
			pr.ImagenThumb = thumbnailURL(pr.Codigo, pr.ImagenLocal)
		}
		ps = append(ps, pr)
	}
	if hasPage {
		pages := (total + p.Limit - 1) / p.Limit
		jsonResp(w, map[string]interface{}{
			"data":  ps,
			"total": total,
			"page":  p.Page,
			"pages": pages,
		})
	} else {
		jsonResp(w, ps)
	}
}

func handleProductosSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	type SearchResult struct {
		Codigo  string  `json:"codigo"`
		Nombre  string  `json:"nombre"`
		Precio  float64 `json:"precio"`
		Stock   float64 `json:"stock"`
		Imagen  string  `json:"imagen"`
		Categoria string `json:"categoria"`
	}

	if q == "" {
		rows, err := db.Query(`SELECT codigo, COALESCE(descripcion,''), COALESCE(pventa,0), COALESCE(dinventario,0), COALESCE(imagen_local,''), COALESCE(categorias,'') FROM PRODUCTOS ORDER BY COALESCE(dinventario,0) DESC LIMIT ?`, limit)
		if err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		defer rows.Close()
		res := make([]SearchResult, 0, limit)
		for rows.Next() {
			var r SearchResult
			rows.Scan(&r.Codigo, &r.Nombre, &r.Precio, &r.Stock, &r.Imagen, &r.Categoria)
			res = append(res, r)
		}
		jsonResp(w, res)
		return
	}

	// Sanitize FTS5 query: escape special chars, treat spaces as AND
	ftsQuery := strings.Map(func(r rune) rune {
		if strings.ContainsRune(`*"^+-()~:`, r) {
			return -1 // drop
		}
		return r
	}, q)
	if ftsQuery == "" {
		jsonResp(w, []SearchResult{})
		return
	}
	ftsQuery = strings.ReplaceAll(ftsQuery, " ", " AND ") + "*"

	rows, err := db.Query(`SELECT p.codigo, COALESCE(p.descripcion,''), COALESCE(p.pventa,0), COALESCE(p.dinventario,0), COALESCE(p.imagen_local,''), COALESCE(p.categorias,'') FROM productos_fts f JOIN PRODUCTOS p ON p.codigo=f.codigo WHERE productos_fts MATCH ? ORDER BY rank LIMIT ?`, ftsQuery, limit)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	res := make([]SearchResult, 0, limit)
	for rows.Next() {
		var r SearchResult
		rows.Scan(&r.Codigo, &r.Nombre, &r.Precio, &r.Stock, &r.Imagen, &r.Categoria)
		res = append(res, r)
	}
	jsonResp(w, res)
}

func handleProductosCreate(w http.ResponseWriter, r *http.Request) {
	var p Producto
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		jsonErr(w, "JSON invalido", 400)
		return
	}
	tx, err := db.Begin()
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(`INSERT INTO PRODUCTOS (codigo, descripcion, tventa, pcosto, pventa, dept, provid, umedida, mayoreo, iprioridad, dinventario, dinvminimo, dinvmaximo, porcentaje_ganancia, componentes, impuestos, imagen_local) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.Codigo, p.Descripcion, p.Tventa, p.Pcosto, p.Pventa, p.Dept, p.Provid, p.Umedida, p.Mayoreo, p.Iprioridad, p.Dinventario, p.Dinvminimo, p.Dinvmaximo, p.PorcentajeGanancia, p.Componentes, p.Impuestos, p.ImagenLocal)
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}

	tx.Commit()
	jsonResp(w, map[string]string{"ok": "Producto creado"})
}

func handleProductosGet(w http.ResponseWriter, r *http.Request) {
	codigo := r.PathValue("codigo")
	var p Producto
	err := db.QueryRow(`
		SELECT 
			p.codigo, p.descripcion, p.tventa, COALESCE(p.pcosto,0), COALESCE(p.pventa,0), 
			p.dept, p.provid, p.umedida, COALESCE(p.mayoreo,0), p.iprioridad, 
			COALESCE(p.dinventario,0), COALESCE(p.dinvminimo,0), COALESCE(p.dinvmaximo,0), 
			COALESCE(p.checado_en,''), COALESCE(p.porcentaje_ganancia,0), COALESCE(p.componentes,''), COALESCE(p.impuestos,''),
			COALESCE(p.imagen_local,''),
			COALESCE(p.imagen_thumb,''),
			COALESCE(p.marca,''), COALESCE(p.categorias,''),
			COALESCE(p.ingredientes,''), COALESCE(p.nutriscore,''),
			COALESCE(p.cantidad_presentacion,''), COALESCE(p.nutricion,'')
		FROM PRODUCTOS p
		WHERE p.codigo=?`, codigo).Scan(&p.Codigo, &p.Descripcion, &p.Tventa, &p.Pcosto, &p.Pventa, &p.Dept, &p.Provid, &p.Umedida, &p.Mayoreo, &p.Iprioridad, &p.Dinventario, &p.Dinvminimo, &p.Dinvmaximo, &p.ChecadoEn, &p.PorcentajeGanancia, &p.Componentes, &p.Impuestos, &p.ImagenLocal, &p.ImagenThumb, &p.Marca, &p.Categorias, &p.Ingredientes, &p.Nutriscore, &p.CantidadPresentacion, &p.Nutricion)
	if p.ImagenThumb == "" && p.ImagenLocal != "" {
		p.ImagenThumb = thumbnailURL(p.Codigo, p.ImagenLocal)
	}
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
	tx, err := db.Begin()
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(`UPDATE PRODUCTOS SET descripcion=?, tventa=?, pcosto=?, pventa=?, dept=?, provid=?, umedida=?, mayoreo=?, iprioridad=?, dinventario=?, dinvminimo=?, dinvmaximo=?, checado_en=?, porcentaje_ganancia=?, componentes=?, impuestos=?, imagen_local=? WHERE codigo=?`,
		p.Descripcion, p.Tventa, p.Pcosto, p.Pventa, p.Dept, p.Provid, p.Umedida, p.Mayoreo, p.Iprioridad, p.Dinventario, p.Dinvminimo, p.Dinvmaximo, p.ChecadoEn, p.PorcentajeGanancia, p.Componentes, p.Impuestos, p.ImagenLocal, codigo)
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}

	tx.Commit()
	jsonResp(w, map[string]string{"ok": "Producto actualizado"})
}

func handleProductosDelete(w http.ResponseWriter, r *http.Request) {
	codigo := r.PathValue("codigo")
	db.Exec("DELETE FROM PRODUCTOS WHERE codigo=?", codigo)
	jsonResp(w, map[string]string{"ok": "Producto eliminado"})
}

func handleProductoUploadImagen(w http.ResponseWriter, r *http.Request) {
	codigo := r.PathValue("codigo")

	r.ParseMultipartForm(10 << 20)
	file, header, err := r.FormFile("imagen")
	if err != nil {
		jsonErr(w, "Imagen requerida", 400)
		return
	}
	defer file.Close()

	if header.Size > 10<<20 {
		jsonErr(w, "Archivo muy grande (max 10MB)", 400)
		return
	}

	buffer := make([]byte, 512)
	if _, err := file.Read(buffer); err != nil {
		jsonErr(w, "Error leyendo archivo", 400)
		return
	}
	contentType := http.DetectContentType(buffer)
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/webp" {
		jsonErr(w, "Tipo de archivo no valido (solo jpg, png, webp)", 400)
		return
	}
	file.Seek(0, 0)

	ext := strings.ToLower(filepath.Ext(header.Filename))
	// Convert webp to jpg
	if ext == ".webp" {
		ext = ".jpg"
	}
	nombre := codigo + ext
	ruta := filepath.Join("static", "img", "productos", nombre)

	if err := os.MkdirAll(filepath.Dir(ruta), 0755); err != nil {
		log.Printf("Error creando directorio: %v", err)
		jsonErr(w, "Error interno", 500)
		return
	}

	dst, err := os.Create(ruta)
	if err != nil {
		log.Printf("Error guardando imagen: %v", err)
		jsonErr(w, "Error interno", 500)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		log.Printf("Error escribiendo imagen: %v", err)
		jsonErr(w, "Error interno", 500)
		return
	}
	dst.Close()

	compressImage(ruta)

	url := "/static/img/productos/" + nombre
	thumbURL := "/static/img/productos/thumbs/" + nombre
	thumbPath := filepath.Join("static", "img", "productos", "thumbs", nombre)

	if err := createThumbnail(ruta, thumbPath); err != nil {
		log.Printf("Error creando thumbnail: %v", err)
	}

	_, err = db.Exec("UPDATE PRODUCTOS SET imagen_local=?, imagen_thumb=? WHERE codigo=?", url, thumbURL, codigo)
	if err != nil {
		log.Printf("Error actualizando BD: %v", err)
		jsonErr(w, "Error interno", 500)
		return
	}

	appCache.Delete("categorias_list")

	jsonResp(w, map[string]string{"ok": "Imagen subida", "url": url, "thumb": thumbURL})
}

// --- Clientes ---

func handleClientesList(w http.ResponseWriter, r *http.Request) {
	hasPage := r.URL.Query().Has("page") || r.URL.Query().Has("limit")
	p := parsePagination(r)
	search := r.URL.Query().Get("q")

	var total int
	var rows *sql.Rows
	var err error

	if search != "" {
		if hasPage {
			db.QueryRow(`SELECT COUNT(*) FROM CLIENTES WHERE nombre LIKE ? OR telefono LIKE ?`, "%"+search+"%", "%"+search+"%").Scan(&total)
		}
		q := `SELECT numero, COALESCE(nombre,''), COALESCE(direccion,''), COALESCE(telefono,''), COALESCE(dsaldoactual,0), COALESCE(dtactualizasaldo,''), COALESCE(limite_credito,0), COALESCE(ultimo_pago_en,''), COALESCE(folio,0) FROM CLIENTES WHERE nombre LIKE ? OR telefono LIKE ? ORDER BY nombre`
		if hasPage {
			q += ` LIMIT ? OFFSET ?`
			rows, err = db.Query(q, "%"+search+"%", "%"+search+"%", p.Limit, p.Offset)
		} else {
			rows, err = db.Query(q, "%"+search+"%", "%"+search+"%")
		}
	} else {
		if hasPage {
			db.QueryRow(`SELECT COUNT(*) FROM CLIENTES`).Scan(&total)
		}
		q := `SELECT numero, COALESCE(nombre,''), COALESCE(direccion,''), COALESCE(telefono,''), COALESCE(dsaldoactual,0), COALESCE(dtactualizasaldo,''), COALESCE(limite_credito,0), COALESCE(ultimo_pago_en,''), COALESCE(folio,0) FROM CLIENTES ORDER BY nombre`
		if hasPage {
			q += ` LIMIT ? OFFSET ?`
			rows, err = db.Query(q, p.Limit, p.Offset)
		} else {
			rows, err = db.Query(q)
		}
	}
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	cs := make([]Cliente, 0)
	for rows.Next() {
		var c Cliente
		rows.Scan(&c.Numero, &c.Nombre, &c.Direccion, &c.Telefono, &c.Dsaldoactual, &c.Dtactualizasaldo, &c.LimiteCredito, &c.UltimoPagoEn, &c.Folio)
		cs = append(cs, c)
	}
	if hasPage {
		pages := (total + p.Limit - 1) / p.Limit
		jsonResp(w, map[string]interface{}{
			"data":  cs,
			"total": total,
			"page":  p.Page,
			"pages": pages,
		})
	} else {
		jsonResp(w, cs)
	}
}

func handleClientesSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	rows, err := db.Query(`SELECT numero, COALESCE(nombre,''), COALESCE(direccion,''), COALESCE(telefono,''), COALESCE(dsaldoactual,0), COALESCE(dtactualizasaldo,''), COALESCE(limite_credito,0), COALESCE(ultimo_pago_en,''), COALESCE(folio,0) FROM CLIENTES WHERE nombre LIKE ? OR direccion LIKE ? OR CAST(numero AS TEXT) LIKE ?`, "%"+q+"%", "%"+q+"%", "%"+q+"%")
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	cs := make([]Cliente, 0)
	for rows.Next() {
		var c Cliente
		rows.Scan(&c.Numero, &c.Nombre, &c.Direccion, &c.Telefono, &c.Dsaldoactual, &c.Dtactualizasaldo, &c.LimiteCredito, &c.UltimoPagoEn, &c.Folio)
		cs = append(cs, c)
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
	ps := make([]Proveedor, 0)
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

func handleProveedoresProductos(w http.ResponseWriter, r *http.Request) {
	num := r.PathValue("num")
	rows, err := db.Query("SELECT codigo, COALESCE(descripcion,''), COALESCE(pventa,0), COALESCE(dinventario,0) FROM PRODUCTOS WHERE provid=? ORDER BY descripcion", num)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	type ProvProd struct {
		Codigo      string  `json:"codigo"`
		Descripcion string  `json:"descripcion"`
		Pventa      float64 `json:"pventa"`
		Stock       float64 `json:"stock"`
	}
	ps := make([]ProvProd, 0)
	for rows.Next() {
		var p ProvProd
		rows.Scan(&p.Codigo, &p.Descripcion, &p.Pventa, &p.Stock)
		ps = append(ps, p)
	}
	if ps == nil {
		ps = []ProvProd{}
	}
	jsonResp(w, ps)
}

func handleProveedoresRecibir(w http.ResponseWriter, r *http.Request) {
	num := r.PathValue("num")
	var req struct {
		Productos []struct {
			Codigo   string  `json:"codigo"`
			Cantidad float64 `json:"cantidad"`
			Pcosto   float64 `json:"pcosto"`
		} `json:"productos"`
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
	for _, p := range req.Productos {
		if p.Cantidad <= 0 {
			continue
		}
		// Check if product exists
		var exists int
		tx.QueryRow("SELECT COUNT(*) FROM PRODUCTOS WHERE codigo=?", p.Codigo).Scan(&exists)
		if exists == 0 {
			jsonErr(w, "Producto no encontrado: "+p.Codigo, 400)
			return
		}
		// Update stock and costo
		if p.Pcosto > 0 {
			tx.Exec("UPDATE PRODUCTOS SET dinventario=COALESCE(dinventario,0)+?, pcosto=?, provid=? WHERE codigo=?", p.Cantidad, p.Pcosto, num, p.Codigo)
		} else {
			tx.Exec("UPDATE PRODUCTOS SET dinventario=COALESCE(dinventario,0)+?, provid=? WHERE codigo=?", p.Cantidad, num, p.Codigo)
		}
	}
	tx.Commit()
	jsonResp(w, map[string]string{"ok": "Stock actualizado"})
}

// --- Departamentos ---

func handleDepartamentosList(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, COALESCE(nombre,''), COALESCE(porcentaje_impuesto,0), COALESCE(activo,'t'), COALESCE(orden,999) FROM DEPARTAMENTOS ORDER BY orden, nombre")
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	ds := make([]Departamento, 0)
	for rows.Next() {
		var d Departamento
		rows.Scan(&d.ID, &d.Nombre, &d.PorcentajeImpuesto, &d.Activo, &d.Orden)
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
	ms := make([]Medida, 0)
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
	rows, err := db.Query("SELECT id, COALESCE(nombre_completo,''), COALESCE(direccion,''), COALESCE(telefono,''), usuario, COALESCE(rol,'helper'), COALESCE(activo,'t'), COALESCE(created_on,''), COALESCE(correo,''), esta_en_caja_id, COALESCE(foto,'') FROM USUARIOS ORDER BY usuario")
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	us := make([]Usuario, 0)
	for rows.Next() {
		var u Usuario
		rows.Scan(&u.ID, &u.NombreCompleto, &u.Direccion, &u.Telefono, &u.Usuario, &u.Rol, &u.Activo, &u.CreatedOn, &u.Correo, &u.EstaEnCajaID, &u.Foto)
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
	if u.Rol == "" {
		u.Rol = "helper"
	}
	pw := u.Clave
	if pw == "" {
		pw = u.Usuario
	}
	hash, err := HashPassword(pw)
	if err != nil {
		jsonErr(w, "Error al generar hash", 500)
		return
	}
	_, err = db.Exec("INSERT INTO USUARIOS (nombre_completo, direccion, telefono, usuario, clave, activo, created_on, correo, rol, foto) VALUES (?,?,?,?,?,?,?,?,?,?)",
		u.NombreCompleto, u.Direccion, u.Telefono, u.Usuario, hash, u.Activo, now(), u.Correo, u.Rol, u.Foto)
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
	q := "UPDATE USUARIOS SET nombre_completo=?, direccion=?, telefono=?, activo=?, correo=?, rol=?"
	args := []interface{}{u.NombreCompleto, u.Direccion, u.Telefono, u.Activo, u.Correo, u.Rol}
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

func handleUsuarioFoto(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	uid, _, err := validateSession(r)
	if err != nil {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}
	_ = uid

	err = r.ParseMultipartForm(5 << 20)
	if err != nil {
		jsonErr(w, "Archivo demasiado grande (max 5MB)", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("foto")
	if err != nil {
		jsonErr(w, "No se recibio archivo", http.StatusBadRequest)
		return
	}
	defer file.Close()

	contentType := handler.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		jsonErr(w, "Solo se permiten imagenes", http.StatusBadRequest)
		return
	}

	ext := filepath.Ext(handler.Filename)
	if ext == "" {
		ext = ".jpg"
	}

	uploadDir := "uploads/usuarios"
	os.MkdirAll(uploadDir, 0755)

	filename := fmt.Sprintf("user_%s%s", id, ext)
	fullPath := filepath.Join(uploadDir, filename)

	dst, err := os.Create(fullPath)
	if err != nil {
		jsonErr(w, "Error al crear archivo", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		jsonErr(w, "Error al escribir", http.StatusInternalServerError)
		return
	}

	fotoURL := "/uploads/usuarios/" + filename
	_, err = db.Exec("UPDATE USUARIOS SET foto=? WHERE id=?", fotoURL, id)
	if err != nil {
		jsonErr(w, "Error DB: "+err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResp(w, map[string]string{"foto": fotoURL})
}

func handleUsuarioPassword(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Clave string `json:"clave"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonErr(w, "JSON invalido", 400)
		return
	}
	if body.Clave == "" {
		jsonErr(w, "La clave no puede estar vacia", 400)
		return
	}
	hash, err := HashPassword(body.Clave)
	if err != nil {
		jsonErr(w, "Error al generar hash", 500)
		return
	}
	_, err = db.Exec("UPDATE USUARIOS SET clave=? WHERE id=?", hash, id)
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}
	jsonResp(w, map[string]string{"ok": "Contrasena actualizada"})
}

// --- Pedidos ---

func handlePedidosList(w http.ResponseWriter, r *http.Request) {
	isAdmin := roleFromContext(r.Context()) == "admin"
	uid := userIDFromContext(r.Context())

	order := ` ORDER BY
		CASE p.prioridad WHEN 'alta' THEN 1 WHEN 'media' THEN 2 WHEN 'baja' THEN 3 END,
		CASE p.estado WHEN 'pendiente' THEN 1 WHEN 'en_proceso' THEN 2 WHEN 'completado' THEN 3 WHEN 'cancelado' THEN 4 END,
		p.created_on DESC`

	var ps []Pedido
	var err error
	var rows *sql.Rows
	q := `SELECT p.id, p.items, p.total, p.prioridad, COALESCE(p.notas,''), COALESCE(p.cliente_nombre,''), COALESCE(p.cliente_direccion,''), COALESCE(p.cliente_telefono,''), p.es_adeudo, p.creado_por_id, p.asignado_a_id, p.estado, p.created_on, COALESCE(p.completado_on,''), COALESCE(cr.usuario,'?'), COALESCE(asi.usuario,'')
		FROM PEDIDOS p
		LEFT JOIN USUARIOS cr ON cr.id = p.creado_por_id
		LEFT JOIN USUARIOS asi ON asi.id = p.asignado_a_id`

	if isAdmin {
		rows, err = db.Query(q + order)
	} else {
		rows, err = db.Query(q+` WHERE p.estado IN ('pendiente','en_proceso') OR p.asignado_a_id = ? OR p.creado_por_id = ?`+order, uid, uid)
	}
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var p Pedido
		rows.Scan(&p.ID, &p.Items, &p.Total, &p.Prioridad, &p.Notas, &p.ClienteNombre, &p.ClienteDireccion, &p.ClienteTelefono, &p.EsAdeudo, &p.CreadoPorID, &p.AsignadoAID, &p.Estado, &p.CreatedOn, &p.CompletadoOn, &p.CreadoPorNombre, &p.AsignadoANombre)
		ps = append(ps, p)
	}
	jsonResp(w, ps)
}

func handlePedidosCreate(w http.ResponseWriter, r *http.Request) {
	usuarioID := userIDFromContext(r.Context())
	if usuarioID == 0 {
		jsonErr(w, "Usuario no encontrado", 400)
		return
	}

	var p Pedido
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		jsonErr(w, "JSON invalido", 400)
		return
	}
	if p.Total == 0 {
		jsonErr(w, "Total requerido", 400)
		return
	}
	if p.Prioridad == "" {
		p.Prioridad = "media"
	}

	_, err := db.Exec(`INSERT INTO PEDIDOS (items, total, prioridad, notas, cliente_nombre, cliente_direccion, cliente_telefono, es_adeudo, creado_por_id, estado) VALUES (?,?,?,?,?,?,?,?,?,'pendiente')`,
		p.Items, p.Total, p.Prioridad, p.Notas, p.ClienteNombre, p.ClienteDireccion, p.ClienteTelefono, p.EsAdeudo, usuarioID)
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}
	jsonResp(w, map[string]string{"ok": "Pedido creado"})
}

func handlePedidosAsignar(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		AsignadoAID int `json:"asignado_a_id"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	usuarioID := userIDFromContext(r.Context())
	isAdmin := roleFromContext(r.Context()) == "admin"

	// Non-admin can only self-assign
	targetID := body.AsignadoAID
	if !isAdmin && targetID != 0 && targetID != usuarioID {
		jsonErr(w, "Solo administradores pueden asignar a otros usuarios", 403)
		return
	}
	if targetID == 0 {
		targetID = usuarioID
	}

	_, err := db.Exec("UPDATE PEDIDOS SET asignado_a_id=?, estado='en_proceso' WHERE id=? AND (estado='pendiente' OR estado='en_proceso')", targetID, id)
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}
	db.Exec("INSERT INTO PEDIDOS_LOG (pedido_id, usuario_id, accion) VALUES (?,?,'asignar')", id, usuarioID)
	jsonResp(w, map[string]string{"ok": "Pedido asignado"})
}

func handlePedidosEstado(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Estado string `json:"estado"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonErr(w, "JSON invalido", 400)
		return
	}
	valid := map[string]bool{"pendiente": true, "en_proceso": true, "completado": true, "cancelado": true}
	if !valid[body.Estado] {
		jsonErr(w, "Estado invalido", 400)
		return
	}

	usuarioID := userIDFromContext(r.Context())

	// If completado, create a credit VENTATICKETS
	if body.Estado == "completado" {
		var itemsJSON string
		var total float64
		var clienteNombre, clienteDireccion string
		err := db.QueryRow("SELECT items, total, COALESCE(cliente_nombre,''), COALESCE(cliente_direccion,'') FROM PEDIDOS WHERE id=?", id).Scan(&itemsJSON, &total, &clienteNombre, &clienteDireccion)
		if err == nil && total > 0 {
			// Create credit ticket
			var operacionID int
			err = db.QueryRow("SELECT id FROM OPERACIONES WHERE abierta='t' LIMIT 1").Scan(&operacionID)
			if err == nil {
				var cajaID int
				db.QueryRow("SELECT id FROM CAJAS ORDER BY id LIMIT 1").Scan(&cajaID)
				if cajaID == 0 {
					cajaID = 1
				}
				folio := 0
				db.QueryRow("SELECT COALESCE(MAX(folio), 0) + 1 FROM VENTATICKETS").Scan(&folio)

					nombre := "Pedido #" + id
				if clienteNombre != "" {
					nombre = clienteNombre + " (Pedido #" + id + ")"
				}

				// Parse items first to count them
				type PedidoItem struct {
					Codigo   string  `json:"codigo"`
					Nombre   string  `json:"nombre"`
					Cantidad float64 `json:"cantidad"`
					Precio   float64 `json:"precio"`
				}
				var items []PedidoItem
				json.Unmarshal([]byte(itemsJSON), &items)

				// Look up cliente_id from CLIENTES table
				var clienteID *int
				if clienteNombre != "" {
					var cid int
					if db.QueryRow("SELECT numero FROM CLIENTES WHERE nombre=? LIMIT 1", clienteNombre).Scan(&cid) == nil {
						clienteID = &cid
					}
				}

				_, err := db.Exec(`INSERT INTO VENTATICKETS (folio, caja_id, cajero_id, prioridad, cliente_id, creado_en, esta_abierto, operacion_id, es_modificable, nombre, total, subtotal, forma_pago, esta_cancelado, numero_articulos) VALUES (?,?,?,0,?,?,'f',?,'f',?,?,?,'c','f',?)`,
					folio, cajaID, usuarioID, clienteID, now(), operacionID, nombre, total, total, len(items))
				if err == nil {
					var ticketID int64
					db.QueryRow("SELECT last_insert_rowid()").Scan(&ticketID)
					for _, it := range items {
						db.Exec(`INSERT INTO VENTATICKETS_ARTICULOS (ticket_id, producto_codigo, producto_nombre, cantidad, ganancia, precio_usado, impuesto_unitario) VALUES (?,?,?,?,0,?,0)`,
							ticketID, it.Codigo, it.Nombre, it.Cantidad, it.Precio)
					}
				}
			}
		}
	}

	q := "UPDATE PEDIDOS SET estado=?"
	if body.Estado == "completado" {
		q += ", completado_on=datetime('now','localtime')"
	}
	q += " WHERE id=?"
	_, err := db.Exec(q, body.Estado, id)
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}

	db.Exec("INSERT INTO PEDIDOS_LOG (pedido_id, usuario_id, accion) VALUES (?,?,?)", id, usuarioID, body.Estado)
	jsonResp(w, map[string]string{"ok": "Estado actualizado"})
}

func handlePedidosStats(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT u.id, u.usuario, COALESCE(u.nombre_completo,'?'),
			COALESCE((SELECT COUNT(*) FROM PEDIDOS_LOG WHERE usuario_id=u.id AND accion='asignar'),0) as tomados,
			COALESCE((SELECT COUNT(*) FROM PEDIDOS WHERE asignado_a_id=u.id AND estado='completado'),0) as completados,
			COALESCE((SELECT COALESCE(SUM(total),0) FROM PEDIDOS WHERE asignado_a_id=u.id AND estado='completado'),0) as total_vendido
		FROM USUARIOS u
		WHERE u.rol='helper' OR u.rol='admin'
		ORDER BY tomados DESC, completados DESC`)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	type StatEntry struct {
		ID           int     `json:"id"`
		Usuario      string  `json:"usuario"`
		Nombre       string  `json:"nombre"`
		Tomados      int     `json:"tomados"`
		Completados  int     `json:"completados"`
		TotalVendido float64 `json:"total_vendido"`
	}
	stats := make([]StatEntry, 0)
	for rows.Next() {
		var s StatEntry
		rows.Scan(&s.ID, &s.Usuario, &s.Nombre, &s.Tomados, &s.Completados, &s.TotalVendido)
		stats = append(stats, s)
	}
	jsonResp(w, stats)
}

// --- Cajas ---

func handleCajasList(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, COALESCE(nombre,''), COALESCE(ultima_ip,''), COALESCE(ultimo_ingreso,''), COALESCE(nombre_pc,'') FROM CAJAS ORDER BY nombre")
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	cs := make([]Caja, 0)
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

func handleCajaDefault(w http.ResponseWriter, r *http.Request) {
	machine := "localhost"
	var c Caja
	err := db.QueryRow("SELECT id, COALESCE(nombre,''), COALESCE(ultima_ip,''), COALESCE(ultimo_ingreso,''), COALESCE(nombre_pc,'') FROM CAJAS ORDER BY id LIMIT 1").Scan(&c.ID, &c.Nombre, &c.UltimaIP, &c.UltimoIngreso, &c.NombrePC)
	if err == sql.ErrNoRows {
		_, err = db.Exec("INSERT INTO CAJAS (nombre, ultima_ip, nombre_pc) VALUES (?,?,?)", "Caja Principal", "127.0.0.1", machine)
		if err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		db.QueryRow("SELECT id, COALESCE(nombre,''), COALESCE(ultima_ip,''), COALESCE(ultimo_ingreso,''), COALESCE(nombre_pc,'') FROM CAJAS ORDER BY id LIMIT 1").Scan(&c.ID, &c.Nombre, &c.UltimaIP, &c.UltimoIngreso, &c.NombrePC)
	}
	jsonResp(w, c)
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
	ops := make([]Operacion, 0)
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
	oid, _ := strconv.Atoi(id)
	logAudit(db, getUserIDForAudit(r), "caja_closed", "operacion", oid, "", r.RemoteAddr)
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

	q := r.URL.Query().Get("q")
	estado := r.URL.Query().Get("estado")
	prioridad := r.URL.Query().Get("prioridad")
	soloAdeudos := r.URL.Query().Get("solo_adeudos")

	where := []string{"1=1"}
	args := []interface{}{}

	if q != "" {
		where = append(where, `(t.folio LIKE ? OR COALESCE(c.nombre,'') LIKE ? OR COALESCE(c.direccion,'') LIKE ?)`)
		args = append(args, "%"+q+"%", "%"+q+"%", "%"+q+"%")
	}
	if estado != "" {
		switch estado {
		case "abierto":
			where = append(where, "t.esta_abierto='t' AND t.esta_cancelado='f'")
		case "pagado":
			where = append(where, "t.esta_abierto='f' AND t.esta_cancelado='f'")
		case "cancelado":
			where = append(where, "t.esta_cancelado='t'")
		case "credito":
			where = append(where, "t.forma_pago='c' AND t.esta_abierto='f' AND t.esta_cancelado='f'")
		}
	}
	if prioridad != "" {
		where = append(where, "t.prioridad=?")
		args = append(args, prioridad)
	}
	if soloAdeudos == "1" {
		where = append(where, "(t.forma_pago='c' AND t.esta_abierto='f' AND t.esta_cancelado='f')")
	}

	whereClause := strings.Join(where, " AND ")

	var total int
	if r.URL.Query().Has("page") || r.URL.Query().Has("limit") {
		countQuery := `SELECT COUNT(*) FROM VENTATICKETS t LEFT JOIN CLIENTES c ON t.cliente_id=c.numero WHERE ` + whereClause
		db.QueryRow(countQuery, args...).Scan(&total)

		rows, err := db.Query(`SELECT t.id, t.folio, t.caja_id, t.cajero_id, COALESCE(t.nombre,''), t.prioridad, t.creado_en, COALESCE(t.subtotal,0), COALESCE(t.impuestos,0), COALESCE(t.total,0), COALESCE(t.ganancia,0), t.esta_abierto, t.cliente_id, COALESCE(t.vendido_en,''), t.es_modificable, COALESCE(t.pago_con,0), COALESCE(t.moneda,''), COALESCE(t.numero_articulos,0), COALESCE(t.pagado_en,''), t.esta_cancelado, t.operacion_id, COALESCE(t.forma_pago,''), COALESCE(t.referencia,''), COALESCE(t.total_devuelto,0), COALESCE(c.nombre,''), COALESCE(c.direccion,'') FROM VENTATICKETS t LEFT JOIN CLIENTES c ON t.cliente_id=c.numero WHERE `+whereClause+` ORDER BY t.creado_en DESC LIMIT ? OFFSET ?`, append(args, limit, offset)...)
		if err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		defer rows.Close()
		ts := make([]VentaTicket, 0)
		for rows.Next() {
			var t VentaTicket
			if err := rows.Scan(&t.ID, &t.Folio, &t.CajaID, &t.CajeroID, &t.Nombre, &t.Prioridad, &t.CreadoEn, &t.Subtotal, &t.Impuestos, &t.Total, &t.Ganancia, &t.EstaAbierto, &t.ClienteID, &t.VendidoEn, &t.EsModificable, &t.PagoCon, &t.Moneda, &t.NumeroArticulos, &t.PagadoEn, &t.EstaCancelado, &t.OperacionID, &t.FormaPago, &t.Referencia, &t.TotalDevuelto, &t.ClienteNombre, &t.ClienteDireccion); err != nil {
				fmt.Printf("Error scanning ticket row: %v\n", err)
				continue
			}
			ts = append(ts, t)
		}
		if ts == nil {
			ts = []VentaTicket{}
		}
		pages := (total + limit - 1) / limit
		jsonResp(w, map[string]interface{}{
			"data":  ts,
			"total": total,
			"page":  page,
			"pages": pages,
		})
		return
	}

	rows, err := db.Query(`SELECT t.id, t.folio, t.caja_id, t.cajero_id, COALESCE(t.nombre,''), t.prioridad, t.creado_en, COALESCE(t.subtotal,0), COALESCE(t.impuestos,0), COALESCE(t.total,0), COALESCE(t.ganancia,0), t.esta_abierto, t.cliente_id, COALESCE(t.vendido_en,''), t.es_modificable, COALESCE(t.pago_con,0), COALESCE(t.moneda,''), COALESCE(t.numero_articulos,0), COALESCE(t.pagado_en,''), t.esta_cancelado, t.operacion_id, COALESCE(t.forma_pago,''), COALESCE(t.referencia,''), COALESCE(t.total_devuelto,0), COALESCE(c.nombre,''), COALESCE(c.direccion,'') FROM VENTATICKETS t LEFT JOIN CLIENTES c ON t.cliente_id=c.numero WHERE `+whereClause+` ORDER BY t.creado_en DESC`)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	ts := make([]VentaTicket, 0)
	for rows.Next() {
		var t VentaTicket
		if err := rows.Scan(&t.ID, &t.Folio, &t.CajaID, &t.CajeroID, &t.Nombre, &t.Prioridad, &t.CreadoEn, &t.Subtotal, &t.Impuestos, &t.Total, &t.Ganancia, &t.EstaAbierto, &t.ClienteID, &t.VendidoEn, &t.EsModificable, &t.PagoCon, &t.Moneda, &t.NumeroArticulos, &t.PagadoEn, &t.EstaCancelado, &t.OperacionID, &t.FormaPago, &t.Referencia, &t.TotalDevuelto, &t.ClienteNombre, &t.ClienteDireccion); err != nil {
			fmt.Printf("Error scanning ticket row: %v\n", err)
			continue
		}
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
		Prioridad int `json:"prioridad"`
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

	res, err := tx.Exec(`INSERT INTO VENTATICKETS (folio, caja_id, cajero_id, prioridad, cliente_id, creado_en, esta_abierto, operacion_id, es_modificable, nombre) VALUES (?,?,?,?,?,?,'t',?,'t','PV')`, folio, req.CajaID, req.CajeroID, req.Prioridad, req.ClienteID, now(), operacionID)
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
	err := db.QueryRow(`SELECT t.id, t.folio, t.caja_id, t.cajero_id, COALESCE(t.nombre,''), t.prioridad, t.creado_en, COALESCE(t.subtotal,0), COALESCE(t.impuestos,0), COALESCE(t.total,0), COALESCE(t.ganancia,0), t.esta_abierto, t.cliente_id, COALESCE(t.vendido_en,''), t.es_modificable, COALESCE(t.pago_con,0), COALESCE(t.moneda,''), COALESCE(t.numero_articulos,0), COALESCE(t.pagado_en,''), t.esta_cancelado, t.operacion_id, COALESCE(t.forma_pago,''), COALESCE(t.referencia,''), COALESCE(t.total_devuelto,0), COALESCE(c.nombre,''), COALESCE(c.direccion,''), COALESCE(u.usuario,'') FROM VENTATICKETS t LEFT JOIN CLIENTES c ON t.cliente_id=c.numero LEFT JOIN USUARIOS u ON u.id=t.cajero_id WHERE t.id=?`, id).Scan(&t.ID, &t.Folio, &t.CajaID, &t.CajeroID, &t.Nombre, &t.Prioridad, &t.CreadoEn, &t.Subtotal, &t.Impuestos, &t.Total, &t.Ganancia, &t.EstaAbierto, &t.ClienteID, &t.VendidoEn, &t.EsModificable, &t.PagoCon, &t.Moneda, &t.NumeroArticulos, &t.PagadoEn, &t.EstaCancelado, &t.OperacionID, &t.FormaPago, &t.Referencia, &t.TotalDevuelto, &t.ClienteNombre, &t.ClienteDireccion, &t.CajeroNombre)
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
	if req.Cantidad <= 0 {
		jsonErr(w, "Cantidad debe ser mayor a 0", 400)
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

	// Check if product already exists in this ticket → accumulate quantity
	var existingID int
	err = tx.QueryRow(`SELECT id FROM VENTATICKETS_ARTICULOS WHERE ticket_id=? AND producto_codigo=?`, id, req.ProductoCodigo).Scan(&existingID)
	if err == nil {
		// Product exists: update cantidad and ganancia
		_, err = tx.Exec(`UPDATE VENTATICKETS_ARTICULOS SET cantidad = cantidad + ?, ganancia = ganancia + ? WHERE id=?`,
			req.Cantidad, ganancia*req.Cantidad, existingID)
	} else {
		// New product: insert row
		_, err = tx.Exec(`INSERT INTO VENTATICKETS_ARTICULOS (ticket_id, producto_codigo, producto_nombre, cantidad, ganancia, precio_usado, departamento_id, impuesto_unitario) VALUES (?,?,?,?,?,?,?,0)`,
			id, p.Codigo, p.Descripcion, req.Cantidad, ganancia*req.Cantidad, precio, p.Dept)
	}
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}

	tx.Exec(`UPDATE VENTATICKETS SET subtotal = (SELECT COALESCE(SUM(precio_usado * cantidad),0) FROM VENTATICKETS_ARTICULOS WHERE ticket_id=?), total = (SELECT COALESCE(SUM(precio_usado * cantidad),0) FROM VENTATICKETS_ARTICULOS WHERE ticket_id=?), ganancia = (SELECT COALESCE(SUM(ganancia),0) FROM VENTATICKETS_ARTICULOS WHERE ticket_id=?), numero_articulos = (SELECT COUNT(*) FROM VENTATICKETS_ARTICULOS WHERE ticket_id=?) WHERE id=?`, id, id, id, id, id)

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
	tx.Exec(`UPDATE VENTATICKETS SET subtotal = (SELECT COALESCE(SUM(precio_usado * cantidad),0) FROM VENTATICKETS_ARTICULOS WHERE ticket_id=?), total = (SELECT COALESCE(SUM(precio_usado * cantidad),0) FROM VENTATICKETS_ARTICULOS WHERE ticket_id=?), ganancia = (SELECT COALESCE(SUM(ganancia),0) FROM VENTATICKETS_ARTICULOS WHERE ticket_id=?), numero_articulos = (SELECT COUNT(*) FROM VENTATICKETS_ARTICULOS WHERE ticket_id=?) WHERE id=?`, id, id, id, id, id)
	tx.Commit()
	jsonResp(w, map[string]string{"ok": "Articulo eliminado"})
}

func handleTicketCobrar(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Pagos []PagoRequest `json:"pagos"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "JSON invalido", 400)
		return
	}
	if len(req.Pagos) == 0 {
		jsonErr(w, "Se requiere al menos un pago", 400)
		return
	}

	var estaAbierto, formaPagoActual string
	var total, ganancia, pagoCon float64
	var operacionID int
	var clienteID sql.NullInt64
	err := db.QueryRow("SELECT esta_abierto, COALESCE(total,0), COALESCE(ganancia,0), operacion_id, cliente_id, COALESCE(pago_con,0), COALESCE(forma_pago,'') FROM VENTATICKETS WHERE id=?", id).Scan(&estaAbierto, &total, &ganancia, &operacionID, &clienteID, &pagoCon, &formaPagoActual)
	if err != nil {
		jsonErr(w, "Ticket no encontrado", 404)
		return
	}
	if estaAbierto != "t" && formaPagoActual != "c" {
		jsonErr(w, "El ticket ya fue cobrado o cancelado", 400)
		return
	}

	restante := total - pagoCon
	if restante < 0.01 {
		jsonErr(w, "El ticket ya está saldado", 400)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Printf("Error en transaccion: %v", err)
		jsonErr(w, "Error interno", 500)
		return
	}
	defer tx.Rollback()

	tid, _ := strconv.Atoi(id)

	var sumaMontos float64
	var pagoEfectivoRecibido, pagoEfectivoMonto float64
	var formaPago string
	for _, p := range req.Pagos {
		if p.Metodo == "" {
			p.Metodo = "e"
		}
		_, err = createPago(tx, tid, p)
		if err != nil {
			jsonErr(w, err.Error(), 400)
			return
		}
		sumaMontos += p.Monto
		if p.Metodo == "e" {
			pagoEfectivoRecibido += p.Recibido
			pagoEfectivoMonto += p.Monto
		}
		if formaPago == "" {
			formaPago = p.Metodo
		}
	}
	if sumaMontos < restante-0.01 {
		jsonErr(w, "El total de pagos no cubre el monto restante", 400)
		return
	}
	cambioEfectivo := pagoEfectivoRecibido - pagoEfectivoMonto
	if cambioEfectivo < 0 {
		cambioEfectivo = 0
	}
	totalCambio := sumaMontos - restante
	if totalCambio < 0 {
		totalCambio = 0
	}
	nuevoPagoCon := pagoCon + sumaMontos

	if estaAbierto == "t" {
		_, err = tx.Exec(`UPDATE VENTATICKETS SET esta_abierto='f', pagado_en=?, pago_con=?, forma_pago=?, total_devuelto=?, vendido_en=? WHERE id=?`,
			now(), nuevoPagoCon, formaPago, totalCambio, now(), id)
	} else {
		_, err = tx.Exec(`UPDATE VENTATICKETS SET pago_con=?, total_devuelto=? WHERE id=?`,
			nuevoPagoCon, totalCambio, id)
	}
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}

	var ingresosEfectivo float64
	for _, p := range req.Pagos {
		switch p.Metodo {
		case "e":
			ingresosEfectivo += p.Monto
		case "c":
			if clienteID.Valid {
				tx.Exec(`UPDATE CLIENTES SET dsaldoactual = COALESCE(dsaldoactual,0) + ?, dtactualizasaldo = ? WHERE numero = ?`, p.Monto, now(), clienteID.Int64)
			}
		}
	}
	_, err = tx.Exec(`UPDATE OPERACIONES SET ventas = ventas + ?, ingresos_efectivo = ingresos_efectivo + ?, ganancias = ganancias + ? WHERE id=?`,
		total, ingresosEfectivo, ganancia, operacionID)
	if err != nil {
		log.Printf("Error updating operacion: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		log.Printf("Error en commit: %v", err)
		jsonErr(w, "Error al guardar", 500)
		return
	}
	logAudit(db, getUserIDForAudit(r), "ticket_paid", "ticket", tid, fmt.Sprintf("Monto: %.2f, forma: %s", total, formaPago), r.RemoteAddr)
	jsonResp(w, map[string]string{"ok": "Cobro exitoso", "cambio": fmt.Sprintf("%.2f", cambioEfectivo), "total_pagado": fmt.Sprintf("%.2f", sumaMontos), "ticket_id": id})
}

func handleTicketPagosList(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tid, err := strconv.Atoi(id)
	if err != nil {
		jsonErr(w, "ID invalido", 400)
		return
	}
	pagos, err := listPagos(tid)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	jsonResp(w, pagos)
}

func handleTicketPrint(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tid, err := strconv.Atoi(id)
	if err != nil {
		jsonErr(w, "ID invalido", 400)
		return
	}

	var t VentaTicket
	db.QueryRow(`SELECT v.id, v.folio, COALESCE(v.nombre,''), COALESCE(v.subtotal,0), COALESCE(v.total,0), COALESCE(v.pago_con,0), COALESCE(v.total_devuelto,0), COALESCE(v.forma_pago,'e'), COALESCE(v.pagado_en,''), COALESCE(u.nombre_completo,'') FROM VENTATICKETS v LEFT JOIN USUARIOS u ON v.cajero_id=u.id WHERE v.id=?`, tid).Scan(&t.ID, &t.Folio, &t.Nombre, &t.Subtotal, &t.Total, &t.PagoCon, &t.TotalDevuelto, &t.FormaPago, &t.PagadoEn, &t.CajeroNombre)

	pagos, _ := listPagos(tid)
	tpagos := make([]printer.TicketPago, 0, len(pagos))
	for _, p := range pagos {
		tpagos = append(tpagos, printer.TicketPago{Metodo: p.Metodo, Monto: p.Monto})
	}

	titems := make([]printer.TicketItem, 0)
	artRows, err := db.Query(`SELECT COALESCE(producto_nombre,''), cantidad, COALESCE(precio_usado,0) FROM VENTATICKETS_ARTICULOS WHERE ticket_id=?`, tid)
	if err == nil {
		defer artRows.Close()
		for artRows.Next() {
			var it printer.TicketItem
			artRows.Scan(&it.Nombre, &it.Cantidad, &it.Precio)
			it.Total = it.Precio * it.Cantidad
			titems = append(titems, it)
		}
	}

	fecha := t.PagadoEn
	if fecha == "" {
		fecha = now()
	}
	folio := 0
	if t.Folio != nil {
		folio = *t.Folio
	}

	td := printer.TicketData{
		Negocio:  negociosName,
		Folio:    folio,
		Fecha:    fecha,
		Cajero:   t.CajeroNombre,
		Items:    titems,
		Subtotal: t.Subtotal,
		Total:    t.Total,
		Pagos:    tpagos,
		Cambio:   t.TotalDevuelto,
	}

	printerDevice := os.Getenv("PRINTER_DEVICE")
	if printerDevice == "" || printerDevice == "stdout" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		printer.PrintTicket(w, td)
		return
	}
	if strings.HasPrefix(printerDevice, "/dev/") || strings.HasPrefix(printerDevice, "/dev/usb/") {
		f, err := os.OpenFile(printerDevice, os.O_WRONLY, 0)
		if err != nil {
			jsonErr(w, "Error abriendo impresora: "+err.Error(), 500)
			return
		}
		defer f.Close()
		printer.PrintTicket(f, td)
		printer.Beep(f)
		jsonResp(w, map[string]string{"ok": "Impreso correctamente"})
		return
	}
	jsonResp(w, map[string]string{"ok": "Proxy impresion", "device": printerDevice})
}

func handleBarcodeLookup(w http.ResponseWriter, r *http.Request) {
	codigo := r.PathValue("codigo")
	var p Producto
	err := db.QueryRow(`SELECT codigo, descripcion, COALESCE(pventa,0), COALESCE(dinventario,0) FROM PRODUCTOS WHERE codigo=?`, codigo).Scan(&p.Codigo, &p.Descripcion, &p.Pventa, &p.Dinventario)
	if err != nil {
		jsonErr(w, "Producto no encontrado", 404)
		return
	}
	jsonResp(w, p)
}

func handleTicketCancelar(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := userIDFromContext(r.Context())
	role := roleFromContext(r.Context())

	var cajeroID int
	err := db.QueryRow("SELECT cajero_id FROM VENTATICKETS WHERE id=?", id).Scan(&cajeroID)
	if err != nil {
		jsonErr(w, "Ticket no encontrado", 404)
		return
	}
	if role != "admin" && cajeroID != userID {
		jsonErr(w, "No autorizado para cancelar este ticket", 403)
		return
	}

	_, err = db.Exec(`UPDATE VENTATICKETS SET esta_cancelado='t', esta_abierto='f' WHERE id=?`, id)
	if err != nil {
		log.Printf("Error cancelando ticket %s: %v", id, err)
		jsonErr(w, "Error al cancelar ticket", 500)
		return
	}
	tid, _ := strconv.Atoi(id)
	logAudit(db, userID, "ticket_cancelled", "ticket", tid, fmt.Sprintf("Cancelado por usuario %d", userID), r.RemoteAddr)
	jsonResp(w, map[string]string{"ok": "Ticket cancelado"})
}

func handleTicketActualizarPrioridad(w http.ResponseWriter, r *http.Request) {
	if !isHelperOrAdmin(r) {
		jsonErr(w, "No autorizado", 401)
		return
	}
	id := r.PathValue("id")
	var req struct {
		Prioridad int `json:"prioridad"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "JSON invalido", 400)
		return
	}
	_, err := db.Exec(`UPDATE VENTATICKETS SET prioridad=? WHERE id=? AND esta_abierto='t'`, req.Prioridad, id)
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}
	jsonResp(w, map[string]string{"ok": "Prioridad actualizada"})
}

func handleTicketDelete(w http.ResponseWriter, r *http.Request) {
	usuarioID := userIDFromContext(r.Context())
	if !isAdmin(r) {
		var cajeroID int
		db.QueryRow("SELECT cajero_id FROM VENTATICKETS WHERE id=?", r.PathValue("id")).Scan(&cajeroID)
		if cajeroID != usuarioID {
			jsonErr(w, "Solo admin o el creador del ticket puede borrarlo", 403)
			return
		}
	}
	id := r.PathValue("id")
	tx, err := db.Begin()
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer tx.Rollback()
	tx.Exec("DELETE FROM PAGOS WHERE ticket_id=?", id)
	tx.Exec("DELETE FROM VENTATICKETS_ARTICULOS WHERE ticket_id=?", id)
	tx.Exec("DELETE FROM VENTAS WHERE ticket_id=?", id)
	_, err = tx.Exec("DELETE FROM VENTATICKETS WHERE id=?", id)
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}
	tx.Commit()
	tid, _ := strconv.Atoi(id)
	logAudit(db, usuarioID, "ticket_deleted", "ticket", tid, "Ticket eliminado", r.RemoteAddr)
	jsonResp(w, map[string]string{"ok": "Ticket eliminado"})
}

// --- Movimientos ---

func handleMovimientosList(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT id, operacion_id, COALESCE(monto,0), cuando_fue, COALESCE(comentarios,''), tipo, cliente_id, caja_id, cajero_id FROM MOVIMIENTOS ORDER BY cuando_fue DESC LIMIT 100`)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	ms := make([]Movimiento, 0)
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
	hs := make([]HI, 0)
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
	role := roleFromContext(r.Context())
	if role != "admin" {
		jsonErr(w, "Solo administradores pueden ajustar inventario", 403)
		return
	}

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
		log.Printf("Error en transaccion: %v", err)
		jsonErr(w, "Error interno", 500)
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
	logAudit(db, req.UsuarioID, "inventory_adjusted", "product", 0, fmt.Sprintf("Producto: %s, tipo: %s, cantidad: %.2f", req.CodigoProducto, req.Tipo, req.Cantidad), r.RemoteAddr)

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
	is := make([]Impuesto, 0)
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
	ps := make([]Promocion, 0)
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
	if d.IngresosHoy > 0 {
		d.MargenHoy = d.GananciaHoy / d.IngresosHoy * 100
	}
	db.QueryRow(`SELECT COUNT(*) FROM VENTATICKETS WHERE strftime('%Y-%m', creado_en)=strftime('%Y-%m','now')`).Scan(&d.VentasMes)
	db.QueryRow(`SELECT COALESCE(SUM(total),0) FROM VENTATICKETS WHERE strftime('%Y-%m', creado_en)=strftime('%Y-%m','now') AND esta_cancelado='f'`).Scan(&d.IngresosMes)
	db.QueryRow(`SELECT COALESCE(SUM(ganancia),0) FROM VENTATICKETS WHERE strftime('%Y-%m', creado_en)=strftime('%Y-%m','now') AND esta_cancelado='f'`).Scan(&d.GananciaMes)
	if d.IngresosMes > 0 {
		d.MargenMes = d.GananciaMes / d.IngresosMes * 100
	}
	if d.VentasHoy > 0 {
		d.TicketPromedio = d.IngresosHoy / float64(d.VentasHoy)
	}
	db.QueryRow(`SELECT COUNT(*) FROM PRODUCTOS WHERE COALESCE(dinventario,0) > 0`).Scan(&d.ProductosStock)
	db.QueryRow(`SELECT COALESCE(SUM(dinventario * pcosto),0) FROM PRODUCTOS WHERE COALESCE(dinventario,0) > 0`).Scan(&d.ValorInventario)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM OPERACIONES WHERE abierta='t'").Scan(&count)
	d.OperacionActiva = count > 0

	db.QueryRow(`SELECT COUNT(*) FROM VENTATICKETS WHERE esta_abierto='t' AND esta_cancelado='f'`).Scan(&d.TicketsAbiertos)

	jsonResp(w, d)
}

func dateFilter(r *http.Request) (string, string) {
	desde := r.URL.Query().Get("desde")
	hasta := r.URL.Query().Get("hasta")
	if desde == "" {
		desde = "1970-01-01"
	}
	if hasta == "" {
		hasta = "2099-12-31"
	}
	return desde, hasta
}

func handleReportesVentasDiarias(w http.ResponseWriter, r *http.Request) {
	desde, hasta := dateFilter(r)
	rows, err := db.Query(`SELECT DATE(creado_en) as dia, COUNT(*) as tickets, COALESCE(SUM(total),0) as total, COALESCE(SUM(ganancia),0) as ganancia FROM VENTATICKETS WHERE esta_cancelado='f' AND DATE(creado_en) >= ? AND DATE(creado_en) <= ? GROUP BY DATE(creado_en) ORDER BY dia DESC LIMIT 90`, desde, hasta)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	rs := make([]map[string]interface{}, 0)
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
	desde, hasta := dateFilter(r)
	rows, err := db.Query(`SELECT a.producto_nombre, SUM(a.cantidad) as vendidos, SUM(a.cantidad * a.precio_usado) as total FROM VENTATICKETS_ARTICULOS a JOIN VENTATICKETS t ON t.id=a.ticket_id WHERE t.esta_cancelado='f' AND DATE(t.creado_en) >= ? AND DATE(t.creado_en) <= ? GROUP BY a.producto_nombre ORDER BY vendidos DESC LIMIT 20`, desde, hasta)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	rs := make([]map[string]interface{}, 0)
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

func handleReportesMetodosPago(w http.ResponseWriter, r *http.Request) {
	desde, hasta := dateFilter(r)
	rows, err := db.Query(`SELECT COALESCE(p.metodo,'e') as metodo, COUNT(*) as cantidad, COALESCE(SUM(p.monto),0) as total FROM PAGOS p JOIN VENTATICKETS t ON t.id=p.ticket_id WHERE t.esta_cancelado='f' AND DATE(t.creado_en) >= ? AND DATE(t.creado_en) <= ? GROUP BY p.metodo ORDER BY total DESC`, desde, hasta)
	if err != nil {
		rows2, err2 := db.Query(`SELECT COALESCE(forma_pago,'e') as metodo, COUNT(*) as cantidad, COALESCE(SUM(total),0) as total FROM VENTATICKETS WHERE esta_cancelado='f' AND DATE(creado_en) >= ? AND DATE(creado_en) <= ? GROUP BY forma_pago ORDER BY total DESC`, desde, hasta)
		if err2 != nil {
			jsonErr(w, err2.Error(), 500)
			return
		}
		rows = rows2
	}
	defer rows.Close()
	metodos := map[string]string{"e": "Efectivo", "t": "Tarjeta", "v": "Vales", "c": "Credito", "x": "Transferencia"}
	rs := make([]map[string]interface{}, 0)
	for rows.Next() {
		var metodo string
		var cantidad int
		var total float64
		rows.Scan(&metodo, &cantidad, &total)
		nombre := metodo
		if n, ok := metodos[metodo]; ok {
			nombre = n
		}
		rs = append(rs, map[string]interface{}{"metodo": metodo, "nombre": nombre, "cantidad": cantidad, "total": total})
	}
	if rs == nil {
		rs = []map[string]interface{}{}
	}
	jsonResp(w, rs)
}

func handleReportesVentasPorHora(w http.ResponseWriter, r *http.Request) {
	desde, hasta := dateFilter(r)
	rows, err := db.Query(`SELECT CAST(strftime('%H', creado_en) AS INTEGER) as hora, COUNT(*) as tickets, COALESCE(SUM(total),0) as total FROM VENTATICKETS WHERE esta_cancelado='f' AND DATE(creado_en) >= ? AND DATE(creado_en) <= ? GROUP BY hora ORDER BY hora`, desde, hasta)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	rs := make([]map[string]interface{}, 0)
	for rows.Next() {
		var hora, tickets int
		var total float64
		rows.Scan(&hora, &tickets, &total)
		rs = append(rs, map[string]interface{}{"hora": hora, "tickets": tickets, "total": total})
	}
	if rs == nil {
		rs = []map[string]interface{}{}
	}
	jsonResp(w, rs)
}

func handleReportesVentasPorCajero(w http.ResponseWriter, r *http.Request) {
	desde, hasta := dateFilter(r)
	rows, err := db.Query(`SELECT COALESCE(u.nombre_completo, u.usuario, '?') as nombre, COUNT(*) as tickets, COALESCE(SUM(t.total),0) as total, COALESCE(SUM(t.ganancia),0) as ganancia FROM VENTATICKETS t LEFT JOIN USUARIOS u ON u.id=t.cajero_id WHERE t.esta_cancelado='f' AND DATE(t.creado_en) >= ? AND DATE(t.creado_en) <= ? GROUP BY t.cajero_id ORDER BY total DESC`, desde, hasta)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	rs := make([]map[string]interface{}, 0)
	for rows.Next() {
		var nombre string
		var tickets int
		var total, ganancia float64
		rows.Scan(&nombre, &tickets, &total, &ganancia)
		rs = append(rs, map[string]interface{}{"nombre": nombre, "tickets": tickets, "total": total, "ganancia": ganancia})
	}
	if rs == nil {
		rs = []map[string]interface{}{}
	}
	jsonResp(w, rs)
}

func handleReportesExportCSV(w http.ResponseWriter, r *http.Request) {
	tipo := r.URL.Query().Get("tipo")
	desde, hasta := dateFilter(r)

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=reporte_"+tipo+"_"+desde+"_"+hasta+".csv")

	switch tipo {
	case "ventas-diarias":
		w.Write([]byte("Dia,Tickets,Total,Ganancia\n"))
		rows, err := db.Query(`SELECT DATE(creado_en) as dia, COUNT(*), COALESCE(SUM(total),0), COALESCE(SUM(ganancia),0) FROM VENTATICKETS WHERE esta_cancelado='f' AND DATE(creado_en) >= ? AND DATE(creado_en) <= ? GROUP BY DATE(creado_en) ORDER BY dia`, desde, hasta)
		if err != nil {
			return
		}
		defer rows.Close()
		for rows.Next() {
			var dia string
			var tickets int
			var total, ganancia float64
			rows.Scan(&dia, &tickets, &total, &ganancia)
			w.Write([]byte(fmt.Sprintf("%s,%d,%.2f,%.2f\n", dia, tickets, total, ganancia)))
		}
	case "top-productos":
		w.Write([]byte("Producto,Vendidos,Total\n"))
		rows, err := db.Query(`SELECT a.producto_nombre, SUM(a.cantidad), SUM(a.cantidad * a.precio_usado) FROM VENTATICKETS_ARTICULOS a JOIN VENTATICKETS t ON t.id=a.ticket_id WHERE t.esta_cancelado='f' AND DATE(t.creado_en) >= ? AND DATE(t.creado_en) <= ? GROUP BY a.producto_nombre ORDER BY SUM(a.cantidad) DESC`, desde, hasta)
		if err != nil {
			return
		}
		defer rows.Close()
		for rows.Next() {
			var nombre string
			var vendidos, total float64
			rows.Scan(&nombre, &vendidos, &total)
			w.Write([]byte(fmt.Sprintf("%s,%.0f,%.2f\n", nombre, vendidos, total)))
		}
	case "metodos-pago":
		w.Write([]byte("Metodo,Cantidad,Total\n"))
		metodos := map[string]string{"e": "Efectivo", "t": "Tarjeta", "v": "Vales", "c": "Credito", "x": "Transferencia"}
		rows, err := db.Query(`SELECT COALESCE(forma_pago,'e'), COUNT(*), COALESCE(SUM(total),0) FROM VENTATICKETS WHERE esta_cancelado='f' AND DATE(creado_en) >= ? AND DATE(creado_en) <= ? GROUP BY forma_pago`, desde, hasta)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var metodo string
				var cantidad int
				var total float64
				rows.Scan(&metodo, &cantidad, &total)
				nombre := metodo
				if n, ok := metodos[metodo]; ok {
					nombre = n
				}
				w.Write([]byte(fmt.Sprintf("%s,%d,%.2f\n", nombre, cantidad, total)))
			}
		}
	case "cajeros":
		w.Write([]byte("Cajero,Tickets,Total,Ganancia\n"))
		rows, err := db.Query(`SELECT COALESCE(u.nombre_completo,u.usuario,'?'), COUNT(*), COALESCE(SUM(t.total),0), COALESCE(SUM(t.ganancia),0) FROM VENTATICKETS t LEFT JOIN USUARIOS u ON u.id=t.cajero_id WHERE t.esta_cancelado='f' AND DATE(t.creado_en) >= ? AND DATE(t.creado_en) <= ? GROUP BY t.cajero_id ORDER BY SUM(t.total) DESC`, desde, hasta)
		if err != nil {
			return
		}
		defer rows.Close()
		for rows.Next() {
			var nombre string
			var tickets int
			var total, ganancia float64
			rows.Scan(&nombre, &tickets, &total, &ganancia)
			w.Write([]byte(fmt.Sprintf("%s,%d,%.2f,%.2f\n", nombre, tickets, total, ganancia)))
		}
	default:
		w.Write([]byte("tipo no valido"))
	}
}

func handleAdminResetVentas(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Confirm bool `json:"confirm"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !req.Confirm {
		jsonErr(w, "Se requiere confirmacion explicita", 400)
		return
	}

	userID := getUserIDForAudit(r)
	backupDir := filepath.Join(os.TempDir(), "pos-backup-"+time.Now().Format("20060102-150405"))
	os.MkdirAll(backupDir, 0755)

	tx, _ := db.Begin()
	tx.Exec("DELETE FROM VENTAS")
	tx.Exec("DELETE FROM VENTATICKETS_ARTICULOS")
	tx.Exec("DELETE FROM PEDIDOS_LOG")
	tx.Exec("DELETE FROM PEDIDOS")
	tx.Exec("DELETE FROM VENTATICKETS")
	tx.Commit()

	logAudit(db, userID, "admin_reset_ventas", "system", 0, fmt.Sprintf("Backup en: %s", backupDir), r.RemoteAddr)
	jsonResp(w, map[string]string{"ok": "Datos reiniciados", "backup": backupDir})
}

// --- Dashboard Metrics ---

func handleDashboardMetrics(w http.ResponseWriter, r *http.Request) {
	cacheKey := "dashboard_metrics"
	if cached, ok := appCache.Get(cacheKey); ok {
		jsonResp(w, cached)
		return
	}

	var ventasHoy, ticketsHoy, gananciaHoy float64
	var topProductos []map[string]interface{}

	db.QueryRow(`SELECT COALESCE(SUM(total),0), COUNT(*), COALESCE(SUM(ganancia),0) FROM VENTATICKETS WHERE date(creado_en) = date('now') AND esta_cancelado='f'`).Scan(&ventasHoy, &ticketsHoy, &gananciaHoy)

	rows, err := db.Query(`
		SELECT p.descripcion, SUM(va.cantidad) as total_vendido
		FROM VENTATICKETS_ARTICULOS va
		JOIN PRODUCTOS p ON va.producto_codigo = p.codigo
		JOIN VENTATICKETS vt ON va.ticket_id = vt.id
		WHERE date(vt.creado_en) = date('now') AND vt.esta_cancelado='f'
		GROUP BY p.codigo
		ORDER BY total_vendido DESC
		LIMIT 5
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var desc string
			var total float64
			rows.Scan(&desc, &total)
			topProductos = append(topProductos, map[string]interface{}{
				"producto": desc,
				"cantidad": total,
			})
		}
	}
	if topProductos == nil {
		topProductos = []map[string]interface{}{}
	}

	result := map[string]interface{}{
		"ventas_hoy":    ventasHoy,
		"tickets_hoy":   ticketsHoy,
		"ganancia_hoy":  gananciaHoy,
		"top_productos": topProductos,
	}
	appCache.Set(cacheKey, result, 30*time.Second)
	jsonResp(w, result)
}

// --- Categorias ---

func handleCategoriasList(w http.ResponseWriter, r *http.Request) {
	cacheKey := "categorias_list"
	if cached, ok := appCache.Get(cacheKey); ok {
		jsonResp(w, cached)
		return
	}

	rows, err := db.Query(`SELECT DISTINCT categoria FROM PRODUCTOS WHERE categoria != '' AND categoria IS NOT NULL AND activo=1 ORDER BY categoria`)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var categorias []string
	for rows.Next() {
		var cat string
		rows.Scan(&cat)
		categorias = append(categorias, cat)
	}
	if categorias == nil {
		categorias = []string{}
	}
	appCache.Set(cacheKey, categorias, 1*time.Hour)
	jsonResp(w, categorias)
}

// --- Jobs ---

func handleJobCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type   JobType                `json:"type"`
		Params map[string]interface{} `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "JSON invalido", 400)
		return
	}
	if req.Type == "" {
		jsonErr(w, "Tipo de job requerido", 400)
		return
	}

	jobID := generateJobID()
	userID := userIDFromContext(r.Context())

	paramsJSON, _ := json.Marshal(req.Params)
	db.Exec(
		"INSERT INTO jobs (id, type, params, user_id) VALUES (?, ?, ?, ?)",
		jobID, req.Type, string(paramsJSON), userID,
	)

	jobQueue <- Job{
		ID:     jobID,
		Type:   req.Type,
		Params: req.Params,
		UserID: userID,
	}

	jsonResp(w, map[string]interface{}{
		"job_id": jobID,
		"status": "queued",
	})
}

func handleJobStatus(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")

	var status, result string
	err := db.QueryRow("SELECT status, COALESCE(result,'') FROM jobs WHERE id = ?", jobID).Scan(&status, &result)
	if err != nil {
		jsonErr(w, "Job no encontrado", 404)
		return
	}

	jsonResp(w, map[string]string{
		"status": status,
		"result": result,
	})
}


