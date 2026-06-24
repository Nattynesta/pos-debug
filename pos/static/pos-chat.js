class ChatSidebar {
  constructor() {
    this.ws = null
    this.userId = 0
    this.username = ''
    this.token = ''
    this.channels = []
    this.currentCanalId = 1
    this.messages = {}
    this.loadedUpTo = {}
    this.hasMore = {}
    this.loadingOlder = {}
    this.mediaRecorder = null
    this.audioChunks = []
    this.timerInterval = null
    this.recordingSeconds = 0
    this.init()
  }

  init() {
    this.readSession()
    this.connectWS()
    this.loadChannels()
    this.attachEvents()
  }

  readSession() {
    var c = document.cookie.split('; ')
    var s = c.find(function(x) { return x.startsWith('session=') })
    this.token = s ? s.split('=')[1] : ''
  }

  // ---- Channels ----

  async loadChannels() {
    try {
      var r = await fetch('/api/chat/canales')
      this.channels = await r.json()
      this.renderChannels()
      if (!this.currentCanalId && this.channels.length) {
        this.switchChannel(this.channels[0].id)
      }
    } catch (e) {}
  }

  renderChannels() {
    var list = document.getElementById('chat-canal-list')
    if (!list) return
    list.innerHTML = this.channels.map(function(c) {
      var active = c.id === this.currentCanalId ? ' active' : ''
      var badge = c.no_leidos > 0 ? '<span class="cc-badge">' + c.no_leidos + '</span>' : ''
      return '<div class="cc-item' + active + '" data-cid="' + c.id + '">' +
        '<span class="cc-icon"><i class="ti ti-' + (c.icono || 'hash') + '"></i></span>' +
        '<span class="cc-name">' + this.esc(c.nombre) + '</span>' + badge + '</div>'
    }.bind(this)).join('')
    list.querySelectorAll('.cc-item').forEach(function(el) {
      el.onclick = function() { this.switchChannel(parseInt(el.dataset.cid)) }.bind(this)
    }.bind(this))
    this.updateBadge()
  }

  updateBadge() {
    var total = this.channels.reduce(function(s, c) { return s + (c.no_leidos || 0) }, 0)
    var badge = document.getElementById('chat-toggle-badge')
    var fabBadge = document.getElementById('chat-fab-badge')
    if (badge) { badge.textContent = total || ''; badge.style.display = total ? 'flex' : 'none' }
    if (fabBadge) { fabBadge.textContent = total || ''; fabBadge.style.display = total ? 'flex' : 'none' }
  }

  switchChannel(canalId) {
    if (this.currentCanalId === canalId) return
    this.currentCanalId = canalId
    this.renderChannels()
    var canal = this.channels.find(function(c) { return c.id === canalId })
    var header = document.getElementById('chat-header-title')
    if (header && canal) {
      header.innerHTML = '<i class="ti ti-' + (canal.icono || 'hash') + '"></i> ' + this.esc(canal.nombre)
    }
    document.getElementById('chat-mensajes').innerHTML = ''
    document.getElementById('chat-empty').style.display = 'block'
    this.messages[canalId] = []
    this.hasMore[canalId] = true
    this.loadedUpTo[canalId] = null
    this.loadMessages(canalId, false)

    // Subscribe via WS
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type: 'subscribe', id: 'sub_' + canalId, extra: { canal_id: canalId } }))
    }
  }

  // ---- Messages ----

  async loadMessages(canalId, prepend) {
    if (this.loadingOlder[canalId]) return
    this.loadingOlder[canalId] = true
    try {
      var url = '/api/chat/mensajes?canal_id=' + canalId + '&limit=30'
      if (prepend && this.loadedUpTo[canalId]) url += '&before_id=' + this.loadedUpTo[canalId]
      var r = await fetch(url)
      var msgs = await r.json()
      var cont = document.getElementById('chat-mensajes')
      var empty = document.getElementById('chat-empty')
      if (!msgs.length) {
        if (!prepend) empty.style.display = 'block'
        this.hasMore[canalId] = false
        this.loadingOlder[canalId] = false
        return
      }
      empty.style.display = 'none'
      if (!this.messages[canalId]) this.messages[canalId] = []
      if (prepend) {
        this.messages[canalId] = msgs.reverse().concat(this.messages[canalId])
        var shBefore = cont.scrollHeight
        cont.insertAdjacentHTML('afterbegin', msgs.reverse().map(this.renderMsg.bind(this)).join(''))
        this.loadedUpTo[canalId] = msgs[msgs.length - 1].id
        cont.scrollTop = cont.scrollHeight - shBefore
      } else {
        this.messages[canalId] = msgs.reverse()
        cont.innerHTML = msgs.reverse().map(this.renderMsg.bind(this)).join('')
        this.loadedUpTo[canalId] = msgs[msgs.length - 1].id
        if (msgs.length) this.markRead(canalId, msgs[msgs.length - 1].id)
        setTimeout(this.scrollBottom.bind(this), 50)
      }
      if (msgs.length < 30) this.hasMore[canalId] = false
    } catch (e) {}
    this.loadingOlder[canalId] = false
  }

  renderMsg(m) {
    var isOwn = (m.user_id || 0) === this.userId
    var content = m.message || m.mensaje || ''
    var user = m.username || m.usuario || '?'
    var t = m.created || m.created_on || ''
    var tipo = m.tipo || 'texto'
    var avatar = user.charAt(0).toUpperCase()
    var timeStr = this.formatTime(t)

    if (tipo === 'voz') {
      var data = m.datos || {}
      var audioUrl = data.url || ''
      return '<div class="cb cb-' + (isOwn ? 'own' : 'other') + '">' +
        (!isOwn ? '<div class="cb-avatar">' + this.esc(avatar) + '</div>' : '') +
        '<div class="cb-body">' +
        (!isOwn ? '<div class="cb-user">' + this.esc(user) + '</div>' : '') +
        '<div class="cb-audio-wrap"><audio controls class="cb-audio" ' +
        (audioUrl ? 'src="' + audioUrl + '"' : '') + '></audio></div>' +
        '<div class="cb-time">' + timeStr + '</div></div></div>'
    }

    return '<div class="cb cb-' + (isOwn ? 'own' : 'other') + '">' +
      (!isOwn ? '<div class="cb-avatar">' + this.esc(avatar) + '</div>' : '') +
      '<div class="cb-body">' +
      (!isOwn ? '<div class="cb-user">' + this.esc(user) + '</div>' : '') +
      '<div class="cb-text">' + this.esc(content) + '</div>' +
      '<div class="cb-time">' + timeStr + '</div></div></div>'
  }

  appendMsg(m) {
    var cid = m.canal_id || this.currentCanalId
    if (!this.messages[cid]) this.messages[cid] = []
    this.messages[cid].push(m)
    if (cid === this.currentCanalId) {
      var cont = document.getElementById('chat-mensajes')
      var empty = document.getElementById('chat-empty')
      if (empty) empty.style.display = 'none'
      cont.insertAdjacentHTML('beforeend', this.renderMsg(m))
      this.markRead(cid, m.msg_id || m.id)
      this.scrollBottom()
    }
    this.markChannelRead(cid)
    this.updateBadge()
  }

  markRead(canalId, msgId) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type: 'mark_read', id: 'r_' + Date.now(), extra: { canal_id: canalId, msg_id: msgId } }))
    } else {
      fetch('/api/chat/leido', { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ canal_id: canalId, msg_id: msgId }) })
    }
  }

  markChannelRead(canalId) {
    for (var i = 0; i < this.channels.length; i++) {
      if (this.channels[i].id === canalId) { this.channels[i].no_leidos = 0; break }
    }
  }

  // ---- WebSocket ----

  connectWS() {
    var proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
    this.ws = new WebSocket(proto + '//' + location.host + '/api/chat/ws')
    this.ws.onopen = function() {
      if (this.token) this.ws.send(JSON.stringify({ type: 'auth', id: 'auth_1', token: this.token }))
    }.bind(this)
    this.ws.onmessage = function(e) {
      var m = JSON.parse(e.data)
      if (m.type === 'ack' && m.status === 'authenticated') {
        this.userId = m.user_id || this.userId
        if (this.currentCanalId) this.ws.send(JSON.stringify({ type: 'subscribe', id: 'sub_init', extra: { canal_id: this.currentCanalId } }))
        return
      }
      if (m.type === 'chat') this.appendMsg(m)
    }.bind(this)
    this.ws.onclose = function() {
      this.ws = null
      setTimeout(this.connectWS.bind(this), 3000)
    }.bind(this)
    this.ws.onerror = function() { if (this.ws) this.ws.close() }.bind(this)
  }

  // ---- Send ----

  async sendMessage(texto) {
    if (!texto || !this.currentCanalId) return
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type: 'chat', id: 'msg_' + Date.now(), payload: texto, extra: { canal_id: this.currentCanalId, tipo: 'texto' } }))
    } else {
      try {
        await fetch('/api/chat/mensajes', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ mensaje: texto, canal_id: this.currentCanalId, tipo: 'texto' })
        })
      } catch (e) {}
    }
  }

  handleInput() {
    var input = document.getElementById('chat-input')
    if (!input) return
    this.sendMessage(input.value.trim())
    input.value = ''
    this.autoExpand(input)
  }

  autoExpand(el) {
    el.style.height = 'auto'
    el.style.height = Math.min(el.scrollHeight, 120) + 'px'
  }

  // ---- Voice ----

  async startRecording() {
    try {
      var stream = await navigator.mediaDevices.getUserMedia({ audio: true })
      var mime = MediaRecorder.isTypeSupported('audio/webm;codecs=opus') ? 'audio/webm;codecs=opus' : 'audio/webm'
      this.mediaRecorder = new MediaRecorder(stream, { mimeType: mime })
      this.audioChunks = []
      this.recordingSeconds = 0

      this.mediaRecorder.ondataavailable = function(e) {
        if (e.data.size > 0) this.audioChunks.push(e.data)
      }.bind(this)

      this.mediaRecorder.onstop = function() {
        if (this.audioChunks.length) {
          var blob = new Blob(this.audioChunks, { type: mime })
          this.sendVoice(blob)
        }
        stream.getTracks().forEach(function(t) { t.stop() })
        clearInterval(this.timerInterval)
        document.getElementById('voice-recorder').classList.add('hidden')
      }.bind(this)

      this.mediaRecorder.start()
      document.getElementById('voice-recorder').classList.remove('hidden')
      this.timerInterval = setInterval(function() {
        this.recordingSeconds++
        var m = Math.floor(this.recordingSeconds / 60)
        var s = this.recordingSeconds % 60
        document.getElementById('voice-timer').textContent = m + ':' + (s < 10 ? '0' : '') + s
        if (this.recordingSeconds >= 120) this.stopRecording(false)
      }.bind(this), 1000)
    } catch (err) {
      alert('Micrófono no disponible: ' + err.message)
    }
  }

  stopRecording(cancel) {
    if (this.mediaRecorder && this.mediaRecorder.state !== 'inactive') {
      this.mediaRecorder.stop()
    }
    if (cancel) {
      this.audioChunks = []
      clearInterval(this.timerInterval)
      document.getElementById('voice-recorder').classList.add('hidden')
    }
  }

  async sendVoice(blob) {
    var url = URL.createObjectURL(blob)
    var datos = JSON.stringify({ url: url, duration: this.recordingSeconds })
    var texto = '🎤 Nota de voz (' + this.recordingSeconds + 's)'

    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type: 'chat', id: 'v_' + Date.now(), payload: texto, extra: { canal_id: this.currentCanalId, tipo: 'voz', datos: datos } }))
    } else {
      try {
        await fetch('/api/chat/mensajes', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ mensaje: texto, canal_id: this.currentCanalId, tipo: 'voz', datos: datos })
        })
      } catch (e) {}
    }
  }

  // ---- UI ----

  toggleSidebar() {
    var sb = document.getElementById('chat-sidebar')
    if (!sb) return
    var open = sb.classList.toggle('open')
    if (open && !this.channels.length) this.loadChannels()
    if (open && this.currentCanalId) setTimeout(this.scrollBottom.bind(this), 100)
  }

  scrollBottom() {
    var cont = document.getElementById('chat-mensajes')
    if (cont) cont.scrollTop = cont.scrollHeight
  }

  formatTime(ts) {
    if (!ts) return ''
    var m = ts.match(/^(\d{4})-(\d{2})-(\d{2})[T ](\d{2}):(\d{2})/)
    if (!m) return ts
    var d = new Date(+m[1], +m[2] - 1, +m[3], +m[4], +m[5])
    var now = new Date()
    var sameDay = d.getFullYear() === now.getFullYear() && d.getMonth() === now.getMonth() && d.getDate() === now.getDate()
    var opts = sameDay ? { hour: '2-digit', minute: '2-digit' } : { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' }
    return d.toLocaleTimeString('es-MX', opts)
  }

  esc(s) {
    if (typeof s !== 'string') return ''
    return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
  }

  // ---- Events ----

  attachEvents() {
    // Toggle buttons
    var toggleBtn = document.getElementById('chat-toggle-btn')
    if (toggleBtn) toggleBtn.onclick = this.toggleSidebar.bind(this)
    // FAB
    var fab = document.getElementById('chat-fab')
    if (fab) fab.onclick = this.toggleSidebar.bind(this)
    // Send
    var sendBtn = document.getElementById('btn-send')
    if (sendBtn) sendBtn.onclick = this.handleInput.bind(this)
    // Input enter
    document.addEventListener('keydown', function(e) {
      var input = document.getElementById('chat-input')
      if (e.key === 'Enter' && !e.shiftKey && input && document.activeElement === input) {
        e.preventDefault()
        this.handleInput()
      }
    }.bind(this))
    // Ctrl+Shift+C
    document.addEventListener('keydown', function(e) {
      if (e.ctrlKey && e.shiftKey && e.key === 'C') {
        e.preventDefault()
        this.toggleSidebar()
      }
    }.bind(this))
    // Escape
    document.addEventListener('keydown', function(e) {
      if (e.key === 'Escape') {
        var sb = document.getElementById('chat-sidebar')
        if (sb && sb.classList.contains('open')) this.toggleSidebar()
      }
    }.bind(this))
    // Scroll to load older messages
    document.addEventListener('scroll', function(e) {
      var cont = document.getElementById('chat-mensajes')
      if (!cont || this.loadingOlder[this.currentCanalId] || !this.hasMore[this.currentCanalId]) return
      if (cont.scrollTop < 60) this.loadMessages(this.currentCanalId, true)
    }.bind(this), true)
    // Voice buttons
    var btnVoice = document.getElementById('btn-voice')
    if (btnVoice) btnVoice.onclick = this.startRecording.bind(this)
    var btnCancelVoice = document.getElementById('btn-cancel-voice')
    if (btnCancelVoice) btnCancelVoice.onclick = function() { this.stopRecording(true) }.bind(this)
    var btnStopVoice = document.getElementById('btn-stop-voice')
    if (btnStopVoice) btnStopVoice.onclick = function() { this.stopRecording(false) }.bind(this)
    // Auto-expand input
    var chatInput = document.getElementById('chat-input')
    if (chatInput) {
      chatInput.oninput = function() { this.autoExpand(chatInput) }.bind(this)
    }
  }
}

// Init
var chatSidebar
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', function() { chatSidebar = new ChatSidebar() })
} else {
  chatSidebar = new ChatSidebar()
}

// Legacy globals for onclick handlers
function toggleChatSidebar() { if (chatSidebar) chatSidebar.toggleSidebar() }
function enviarMensajeChat() { if (chatSidebar) chatSidebar.handleInput() }
function autoExpandChatInput(el) { if (chatSidebar) chatSidebar.autoExpand(el) }
