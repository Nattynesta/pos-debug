package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type ChatMessage struct {
	ID               int     `json:"id"`
	UsuarioID        int     `json:"usuario_id"`
	UsuarioNombre    string  `json:"usuario_nombre"`
	UsuarioFoto      string  `json:"usuario_foto,omitempty"`
	Mensaje          string  `json:"mensaje"`
	Tipo             string  `json:"tipo"`
	ArchivoRuta      string  `json:"archivo_ruta,omitempty"`
	ArchivoNombre    string  `json:"archivo_nombre,omitempty"`
	DuracionSegundos float64 `json:"duracion_segundos,omitempty"`
	CreatedOn        string  `json:"created_on"`
}

type ChatDeleteAction struct {
	Action string `json:"action"`
	ID     int    `json:"id,omitempty"`
}

type WSClient struct {
	conn *websocket.Conn
	send chan []byte
}

type WSHub struct {
	mu      sync.RWMutex
	clients map[*WSClient]bool
}

var chatHub = &WSHub{clients: make(map[*WSClient]bool)}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *WSHub) broadcast(msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		select {
		case c.send <- msg:
		default:
			close(c.send)
			delete(h.clients, c)
		}
	}
}

func broadcastChatMessage(msg ChatMessage) {
	data, _ := json.Marshal(msg)
	chatHub.broadcast(data)
}

func handleChatWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS upgrade: %v", err)
		return
	}
	client := &WSClient{conn: conn, send: make(chan []byte, 64)}
	chatHub.mu.Lock()
	chatHub.clients[client] = true
	chatHub.mu.Unlock()

	go func() {
		defer func() {
			chatHub.mu.Lock()
			delete(chatHub.clients, client)
			chatHub.mu.Unlock()
			conn.Close()
		}()
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()

	go func() {
		defer conn.Close()
		for msg := range client.send {
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				break
			}
		}
	}()
}

func handleChatPage(w http.ResponseWriter, r *http.Request) {
	render(w, r, "chat.html", PageData{Title: "Chat", Active: "chat", OperacionActiva: getOperacionActiva()})
}

func handleChatMessages(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT m.id, m.usuario_id, COALESCE(NULLIF(u.nombre_completo,''),u.usuario,'?'), COALESCE(u.foto,''), m.mensaje, m.tipo, COALESCE(m.archivo_ruta,''), COALESCE(m.archivo_nombre,''), COALESCE(m.duracion_segundos,0), m.created_on FROM CHAT_MESSAGES m LEFT JOIN USUARIOS u ON u.id=m.usuario_id ORDER BY m.id ASC LIMIT 100`)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	msgs := make([]ChatMessage, 0)
	for rows.Next() {
		var msg ChatMessage
		rows.Scan(&msg.ID, &msg.UsuarioID, &msg.UsuarioNombre, &msg.UsuarioFoto, &msg.Mensaje, &msg.Tipo, &msg.ArchivoRuta, &msg.ArchivoNombre, &msg.DuracionSegundos, &msg.CreatedOn)
		msgs = append(msgs, msg)
	}
	if msgs == nil {
		msgs = []ChatMessage{}
	}
	jsonResp(w, msgs)
}

func handleChatUpload(w http.ResponseWriter, r *http.Request) {
	uid, _, err := validateSession(r)
	if err != nil {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	ct := r.Header.Get("Content-Type")

	// JSON text-only message
	if strings.HasPrefix(ct, "application/json") {
		var req struct {
			Mensaje string `json:"mensaje"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Mensaje == "" {
			jsonErr(w, "Mensaje vacio", http.StatusBadRequest)
			return
		}
		result, err := db.Exec(`INSERT INTO CHAT_MESSAGES (usuario_id, mensaje, tipo, created_on) VALUES (?, ?, 'texto', ?)`, uid, req.Mensaje, time.Now().Format(time.RFC3339))
		if err != nil {
			jsonErr(w, "Error DB: "+err.Error(), http.StatusInternalServerError)
			return
		}
		id, _ := result.LastInsertId()
		var usuarioNombre, usuarioFoto string
		db.QueryRow("SELECT COALESCE(NULLIF(nombre_completo,''),usuario,'?'), COALESCE(foto,'') FROM USUARIOS WHERE id=?", uid).Scan(&usuarioNombre, &usuarioFoto)
		msg := ChatMessage{
			ID: int(id), UsuarioID: uid, UsuarioNombre: usuarioNombre, UsuarioFoto: usuarioFoto,
			Mensaje: req.Mensaje, Tipo: "texto", CreatedOn: time.Now().Format(time.RFC3339),
		}
		broadcastChatMessage(msg)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(msg)
		return
	}

	// Multipart file upload
	err = r.ParseMultipartForm(10 << 20)
	if err != nil {
		jsonErr(w, "Archivo demasiado grande (max 10MB)", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("archivo")
	if err != nil {
		jsonErr(w, "Error al leer archivo", http.StatusBadRequest)
		return
	}
	defer file.Close()

	contentType := handler.Header.Get("Content-Type")
	var tipo, extension string

	if strings.HasPrefix(contentType, "audio/") {
		tipo = "audio"
		extension = ".webm"
	} else if strings.HasPrefix(contentType, "image/") {
		tipo = "imagen"
		extension = filepath.Ext(handler.Filename)
		if extension == "" {
			extension = ".jpg"
		}
	} else {
		jsonErr(w, "Tipo no soportado. Solo audio/imagen", http.StatusBadRequest)
		return
	}

	uploadDir := "uploads/chat"
	os.MkdirAll(uploadDir, 0755)

	filename := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), tipo, extension)
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

	var duracion float64
	if tipo == "audio" {
		duracionStr := r.FormValue("duracion")
		if duracionStr != "" {
			duracion, _ = strconv.ParseFloat(duracionStr, 64)
		}
	}

	mensaje := r.FormValue("mensaje")
	if mensaje == "" {
		if tipo == "audio" {
			mensaje = "Nota de voz"
		} else {
			mensaje = "Imagen"
		}
	}

	result, err := db.Exec(`INSERT INTO CHAT_MESSAGES (usuario_id, mensaje, tipo, archivo_ruta, archivo_nombre, duracion_segundos, created_on) VALUES (?, ?, ?, ?, ?, ?, ?)`, uid, mensaje, tipo, "/uploads/chat/"+filename, handler.Filename, duracion, time.Now().Format(time.RFC3339))
	if err != nil {
		jsonErr(w, "Error DB: "+err.Error(), http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()

	var usuarioNombre, usuarioFoto string
	db.QueryRow("SELECT COALESCE(NULLIF(nombre_completo,''),usuario,'?'), COALESCE(foto,'') FROM USUARIOS WHERE id=?", uid).Scan(&usuarioNombre, &usuarioFoto)

	msg := ChatMessage{
		ID:               int(id),
		UsuarioID:        uid,
		UsuarioNombre:    usuarioNombre,
		UsuarioFoto:      usuarioFoto,
		Mensaje:          mensaje,
		Tipo:             tipo,
		ArchivoRuta:      "/uploads/chat/" + filename,
		ArchivoNombre:    handler.Filename,
		DuracionSegundos: duracion,
		CreatedOn:        time.Now().Format(time.RFC3339),
	}

	broadcastChatMessage(msg)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msg)
}

func handleChatDeleteMsg(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		jsonErr(w, "Solo admin puede borrar mensajes", http.StatusForbidden)
		return
	}
	idStr := r.PathValue("id")
	msgID, err := strconv.Atoi(idStr)
	if err != nil {
		jsonErr(w, "ID invalido", http.StatusBadRequest)
		return
	}

	var archivoRuta string
	db.QueryRow("SELECT COALESCE(archivo_ruta,'') FROM CHAT_MESSAGES WHERE id=?", msgID).Scan(&archivoRuta)

	if _, err := db.Exec("DELETE FROM CHAT_MESSAGES WHERE id=?", msgID); err != nil {
		jsonErr(w, "Error al borrar: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if archivoRuta != "" {
		localPath := "." + archivoRuta
		os.Remove(localPath)
	}

	action := ChatDeleteAction{Action: "delete", ID: msgID}
	data, _ := json.Marshal(action)
	chatHub.broadcast(data)

	jsonResp(w, map[string]string{"status": "ok"})
}

func handleChatClearAll(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		jsonErr(w, "Solo admin puede borrar mensajes", http.StatusForbidden)
		return
	}

	var paths []string
	rows, _ := db.Query("SELECT COALESCE(archivo_ruta,'') FROM CHAT_MESSAGES WHERE archivo_ruta != ''")
	if rows != nil {
		for rows.Next() {
			var p string
			rows.Scan(&p)
			if p != "" {
				paths = append(paths, p)
			}
		}
		rows.Close()
	}

	if _, err := db.Exec("DELETE FROM CHAT_MESSAGES"); err != nil {
		jsonErr(w, "Error al limpiar: "+err.Error(), http.StatusInternalServerError)
		return
	}

	for _, p := range paths {
		os.Remove("." + p)
	}

	action := ChatDeleteAction{Action: "clear"}
	data, _ := json.Marshal(action)
	chatHub.broadcast(data)

	jsonResp(w, map[string]string{"status": "ok"})
}
