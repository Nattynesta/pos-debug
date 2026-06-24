package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type wsClient struct {
	conn     *websocket.Conn
	userID   int
	username string
	canales  map[int]bool
	send     chan []byte
}

var (
	wsClients   = make(map[*wsClient]bool)
	wsClientsMu sync.Mutex
	wsRegister  = make(chan *wsClient)
	wsUnregister = make(chan *wsClient)
)

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func runWSHub() {
	for {
		select {
		case client := <-wsRegister:
			wsClientsMu.Lock()
			wsClients[client] = true
			wsClientsMu.Unlock()
		case client := <-wsUnregister:
			wsClientsMu.Lock()
			if _, ok := wsClients[client]; ok {
				delete(wsClients, client)
				func() {
					defer func() { recover() }()
					close(client.send)
				}()
			}
			wsClientsMu.Unlock()
		}
	}
}

func broadcastToChannel(canalID int, msg []byte) {
	wsClientsMu.Lock()
	defer wsClientsMu.Unlock()
	for client := range wsClients {
		if !client.canales[canalID] {
			continue
		}
		select {
		case client.send <- msg:
		default:
			delete(wsClients, client)
			func() {
				defer func() { recover() }()
				close(client.send)
			}()
		}
	}
}

func sendJSON(client *wsClient, v interface{}) {
	data, _ := json.Marshal(v)
	select {
	case client.send <- data:
	default:
	}
}

func handleChatWS(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &wsClient{
		conn:    conn,
		canales: make(map[int]bool),
		send:    make(chan []byte, 256),
	}

	defer func() {
		wsUnregister <- client
	}()

	conn.SetReadLimit(512 * 1024)
	conn.SetReadDeadline(time.Now().Add(120 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		return nil
	})

	wsRegister <- client

	go func() {
		for msg := range client.send {
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		}
		conn.Close()
	}()

	authenticated := false
	authTimer := time.AfterFunc(10*time.Second, func() {
		if !authenticated {
			sendJSON(client, map[string]string{"type": "error", "id": "auth", "error": "auth_timeout"})
			wsUnregister <- client
		}
	})
	defer authTimer.Stop()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg struct {
			Type    string          `json:"type"`
			Token   string          `json:"token"`
			Payload string          `json:"payload"`
			ID      string          `json:"id"`
			Extra   json.RawMessage `json:"extra,omitempty"`
		}
		if err := json.Unmarshal(message, &msg); err != nil {
			sendJSON(client, map[string]string{"type": "error", "id": "parse", "error": "invalid_json"})
			continue
		}

		switch msg.Type {
		case "auth":
			if authenticated {
				sendJSON(client, map[string]string{"type": "error", "id": msg.ID, "error": "already_authenticated"})
				continue
			}
			if msg.Token == "" {
				sendJSON(client, map[string]string{"type": "error", "id": msg.ID, "error": "missing_token"})
				continue
			}
			var uid int
			db.QueryRow("SELECT id FROM USUARIOS WHERE usuario=?", msg.Token).Scan(&uid)
			if uid == 0 {
				sendJSON(client, map[string]string{"type": "error", "id": msg.ID, "error": "invalid_token"})
				continue
			}
			var username string
			db.QueryRow("SELECT usuario FROM USUARIOS WHERE id=?", uid).Scan(&username)
			client.userID = uid
			client.username = username
			client.canales[1] = true
			authenticated = true
			authTimer.Stop()
			sendJSON(client, map[string]string{"type": "ack", "id": msg.ID, "status": "authenticated"})

		case "subscribe":
			if !authenticated {
				sendJSON(client, map[string]string{"type": "error", "id": msg.ID, "error": "not_authenticated"})
				continue
			}
			var extra struct {
				CanalID int `json:"canal_id"`
			}
			if msg.Extra != nil {
				json.Unmarshal(msg.Extra, &extra)
			}
			if extra.CanalID > 0 {
				client.canales[extra.CanalID] = true
			}
			sendJSON(client, map[string]string{"type": "ack", "id": msg.ID, "status": "subscribed"})

		case "unsubscribe":
			if !authenticated {
				continue
			}
			var extra struct {
				CanalID int `json:"canal_id"`
			}
			if msg.Extra != nil {
				json.Unmarshal(msg.Extra, &extra)
			}
			delete(client.canales, extra.CanalID)

		case "chat":
			if !authenticated {
				sendJSON(client, map[string]string{"type": "error", "id": msg.ID, "error": "not_authenticated"})
				continue
			}
			if strings.TrimSpace(msg.Payload) == "" {
				sendJSON(client, map[string]string{"type": "error", "id": msg.ID, "error": "empty_message"})
				continue
			}

			var extra struct {
				CanalID int    `json:"canal_id"`
				Tipo    string `json:"tipo"`
				Datos   string `json:"datos"`
			}
			canalID := 1
			tipo := "texto"
			datos := ""
			if msg.Extra != nil {
				json.Unmarshal(msg.Extra, &extra)
				if extra.CanalID > 0 {
					canalID = extra.CanalID
				}
				if extra.Tipo != "" {
					tipo = extra.Tipo
				}
				datos = extra.Datos
			}

			_, err := db.Exec("INSERT INTO CHAT_MESSAGES (usuario_id, mensaje, canal_id, tipo, datos_json) VALUES (?,?,?,?,?)",
				client.userID, msg.Payload, canalID, tipo, datos)
			if err != nil {
				sendJSON(client, map[string]string{"type": "error", "id": msg.ID, "error": "db_error"})
				continue
			}
			var created string
			db.QueryRow("SELECT created_on FROM CHAT_MESSAGES WHERE id=last_insert_rowid()").Scan(&created)
			var mid int
			db.QueryRow("SELECT last_insert_rowid()").Scan(&mid)

			broadcast := map[string]interface{}{
				"type":      "chat",
				"id":        msg.ID,
				"msg_id":    mid,
				"user_id":   client.userID,
				"username":  client.username,
				"message":   msg.Payload,
				"created":   created,
				"canal_id":  canalID,
				"tipo":      tipo,
				"datos":     datos,
			}
			data, _ := json.Marshal(broadcast)
			broadcastToChannel(canalID, data)
			sendJSON(client, map[string]string{"type": "ack", "id": msg.ID, "status": "sent"})

		case "mark_read":
			if !authenticated {
				continue
			}
			var extra struct {
				CanalID int `json:"canal_id"`
				MsgID   int `json:"msg_id"`
			}
			if msg.Extra != nil {
				json.Unmarshal(msg.Extra, &extra)
			}
			if extra.CanalID > 0 && extra.MsgID > 0 {
				db.Exec("INSERT OR REPLACE INTO chat_leidos (usuario_id, canal_id, ultimo_leido_id) VALUES (?,?,?)",
					client.userID, extra.CanalID, extra.MsgID)
			}

		case "ping":
			sendJSON(client, map[string]string{"type": "ack", "id": msg.ID, "status": "pong"})

		default:
			sendJSON(client, map[string]string{"type": "error", "id": msg.ID, "error": "unknown_type"})
		}
	}
}

// REST handlers

type Canal struct {
	ID          int    `json:"id"`
	Nombre      string `json:"nombre"`
	Icono       string `json:"icono"`
	Descripcion string `json:"descripcion"`
	NoLeidos    int    `json:"no_leidos"`
}

func handleChatCanales(w http.ResponseWriter, r *http.Request) {
	uid := getUserID(r)
	rows, err := db.Query(`
		SELECT c.id, c.nombre, c.icono, c.descripcion,
			COALESCE((SELECT COUNT(*) FROM CHAT_MESSAGES m WHERE m.canal_id = c.id AND m.id > COALESCE(l.ultimo_leido_id, 0)), 0) as no_leidos
		FROM chat_canales c
		LEFT JOIN chat_leidos l ON l.canal_id = c.id AND l.usuario_id = ?
		ORDER BY c.id
	`, uid)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	canales := make([]Canal, 0)
	for rows.Next() {
		var ca Canal
		rows.Scan(&ca.ID, &ca.Nombre, &ca.Icono, &ca.Descripcion, &ca.NoLeidos)
		canales = append(canales, ca)
	}
	jsonResp(w, canales)
}

func handleChatMensajes(w http.ResponseWriter, r *http.Request) {
	uid := getUserID(r)
	canalID := queryIntParam(r, "canal_id", 1)
	limit := queryIntParam(r, "limit", 50)
	if limit < 1 {
		limit = 1
	}
	if limit > 200 {
		limit = 200
	}

	if r.Method == "GET" {
		beforeID := 0
		if b := r.URL.Query().Get("before_id"); b != "" {
			if v, err := strconv.Atoi(b); err == nil {
				beforeID = v
			}
		}
		afterID := 0
		if a := r.URL.Query().Get("after_id"); a != "" {
			if v, err := strconv.Atoi(a); err == nil {
				afterID = v
			}
		}

		var rows *sql.Rows
		var err error

		if afterID > 0 {
			rows, err = db.Query(`
				SELECT cm.id, cm.usuario_id, cm.mensaje, cm.created_on, u.usuario, cm.tipo, COALESCE(cm.datos_json,'')
				FROM CHAT_MESSAGES cm JOIN USUARIOS u ON u.id=cm.usuario_id
				WHERE cm.canal_id = ? AND cm.id > ?
				ORDER BY cm.id ASC LIMIT ?
			`, canalID, afterID, limit)
		} else if beforeID > 0 {
			rows, err = db.Query(`
				SELECT cm.id, cm.usuario_id, cm.mensaje, cm.created_on, u.usuario, cm.tipo, COALESCE(cm.datos_json,'')
				FROM CHAT_MESSAGES cm JOIN USUARIOS u ON u.id=cm.usuario_id
				WHERE cm.canal_id = ? AND cm.id < ?
				ORDER BY cm.id DESC LIMIT ?
			`, canalID, beforeID, limit)
		} else {
			rows, err = db.Query(`
				SELECT cm.id, cm.usuario_id, cm.mensaje, cm.created_on, u.usuario, cm.tipo, COALESCE(cm.datos_json,'')
				FROM CHAT_MESSAGES cm JOIN USUARIOS u ON u.id=cm.usuario_id
				WHERE cm.canal_id = ?
				ORDER BY cm.id DESC LIMIT ?
			`, canalID, limit)
		}
		if err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		defer rows.Close()

		msgs := make([]map[string]interface{}, 0)
		for rows.Next() {
			var id, uid int
			var msg, created, usuario, tipo, datos string
			rows.Scan(&id, &uid, &msg, &created, &usuario, &tipo, &datos)
			m := map[string]interface{}{
				"id": id, "user_id": uid, "message": msg, "created": created,
				"username": usuario, "type": "chat", "canal_id": canalID, "tipo": tipo,
			}
			if datos != "" {
				var dj interface{}
				if json.Unmarshal([]byte(datos), &dj) == nil {
					m["datos"] = dj
				}
			}
			msgs = append(msgs, m)
		}

		if len(msgs) > 0 {
			lastID := 0
			for _, m := range msgs {
				if id, ok := m["id"].(int); ok && id > lastID {
					lastID = id
				}
			}
			if lastID > 0 && uid > 0 {
				db.Exec("INSERT OR REPLACE INTO chat_leidos (usuario_id, canal_id, ultimo_leido_id) VALUES (?,?,?)",
					uid, canalID, lastID)
			}
		}

		jsonResp(w, msgs)
		return
	}

	if r.Method == "POST" {
		var body struct {
			Mensaje string `json:"mensaje"`
			CanalID int    `json:"canal_id"`
			Tipo    string `json:"tipo"`
			Datos   string `json:"datos"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if strings.TrimSpace(body.Mensaje) == "" {
			jsonErr(w, "Mensaje vacío", 400)
			return
		}
		if body.CanalID <= 0 {
			body.CanalID = 1
		}
		if body.Tipo == "" {
			body.Tipo = "texto"
		}
		if uid <= 0 {
			jsonErr(w, "No autenticado", 401)
			return
		}
		_, err := db.Exec("INSERT INTO CHAT_MESSAGES (usuario_id, mensaje, canal_id, tipo, datos_json) VALUES (?,?,?,?,?)",
			uid, body.Mensaje, body.CanalID, body.Tipo, body.Datos)
		if err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		var msgCreated string
		db.QueryRow("SELECT created_on FROM CHAT_MESSAGES WHERE id=last_insert_rowid()").Scan(&msgCreated)
		var mid int
		db.QueryRow("SELECT last_insert_rowid()").Scan(&mid)
		var usuario string
		db.QueryRow("SELECT usuario FROM USUARIOS WHERE id=?", uid).Scan(&usuario)

		msgData, _ := json.Marshal(map[string]interface{}{
			"type": "chat", "id": "", "msg_id": mid, "user_id": uid,
			"username": usuario, "message": body.Mensaje, "created": msgCreated,
			"canal_id": body.CanalID, "tipo": body.Tipo, "datos": body.Datos,
		})
		broadcastToChannel(body.CanalID, msgData)
		jsonResp(w, map[string]string{"ok": "enviado"})
		return
	}

	if r.Method == "DELETE" {
		if !isAdmin(r) {
			jsonErr(w, "Solo administradores", 403)
			return
		}
		canalID := queryIntParam(r, "canal_id", 0)
		if canalID > 0 {
			_, err := db.Exec("DELETE FROM CHAT_MESSAGES WHERE canal_id = ?", canalID)
			if err != nil {
				jsonErr(w, err.Error(), 500)
				return
			}
			jsonResp(w, map[string]string{"ok": "Mensajes eliminados del canal"})
		} else {
			_, err := db.Exec("DELETE FROM CHAT_MESSAGES")
			if err != nil {
				jsonErr(w, err.Error(), 500)
				return
			}
			jsonResp(w, map[string]string{"ok": "Todos los mensajes eliminados"})
		}
		return
	}

	jsonErr(w, "Method not allowed", 405)
}

func handleChatLeido(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		jsonErr(w, "Method not allowed", 405)
		return
	}
	uid := getUserID(r)
	if uid <= 0 {
		jsonErr(w, "No autenticado", 401)
		return
	}
	var body struct {
		CanalID int `json:"canal_id"`
		MsgID   int `json:"msg_id"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.CanalID <= 0 || body.MsgID <= 0 {
		jsonErr(w, "canal_id y msg_id requeridos", 400)
		return
	}
	_, err := db.Exec("INSERT OR REPLACE INTO chat_leidos (usuario_id, canal_id, ultimo_leido_id) VALUES (?,?,?)",
		uid, body.CanalID, body.MsgID)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"ok": "marcado"})
}

func handleChatUsuarios(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, usuario, COALESCE(nombre_completo,'') FROM USUARIOS WHERE activo='t' ORDER BY usuario")
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	type ChatUser struct {
		ID            int    `json:"id"`
		Usuario       string `json:"usuario"`
		NombreCompleto string `json:"nombre_completo"`
	}
	us := make([]ChatUser, 0)
	for rows.Next() {
		var u ChatUser
		rows.Scan(&u.ID, &u.Usuario, &u.NombreCompleto)
		us = append(us, u)
	}
	jsonResp(w, us)
}

func handleChatOnline(w http.ResponseWriter, r *http.Request) {
	wsClientsMu.Lock()
	count := len(wsClients)
	wsClientsMu.Unlock()
	jsonResp(w, map[string]int{"count": count})
}

func getUserID(r *http.Request) int {
	cookie, err := r.Cookie("session")
	if err != nil || cookie.Value == "" {
		return 0
	}
	var uid int
	db.QueryRow("SELECT id FROM USUARIOS WHERE usuario=?", cookie.Value).Scan(&uid)
	return uid
}

func queryIntParam(r *http.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
