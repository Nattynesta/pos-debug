var chatState = {
  open: false,
  channels: [],
  currentCanalId: null,
  messages: {},
  loadedUpTo: {},
  hasMore: {},
  loadingOlder: {},
  loadingChannels: false,
  ws: null,
  wsReconnectTimer: null,
  sessionToken: '',
  userId: 0,
  username: ''
};

function initChatSidebar(token, uid, uname) {
  chatState.sessionToken = token;
  chatState.userId = uid;
  chatState.username = uname;
  conectarWSChat();
  cargarCanales();
}

function toggleChatSidebar() {
  chatState.open = !chatState.open;
  var sb = document.getElementById('chat-sidebar');
  if (!sb) return;
  sb.classList.toggle('open', chatState.open);
  if (chatState.open && chatState.currentCanalId) {
    setTimeout(function() { scrollChatToBottom(); }, 100);
  }
  if (chatState.open && chatState.channels.length === 0) {
    cargarCanales();
  }
}

function conectarWSChat() {
  if (chatState.ws && chatState.ws.readyState === WebSocket.OPEN) return;
  var proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
  chatState.ws = new WebSocket(proto + '//' + location.host + '/api/chat/ws');
  chatState.ws.onopen = function() {
    if (chatState.sessionToken) {
      chatState.ws.send(JSON.stringify({type: 'auth', id: 'auth_1', token: chatState.sessionToken}));
    }
  };
  chatState.ws.onmessage = function(e) {
    var m = JSON.parse(e.data);
    if (m.type === 'chat') {
      var cid = m.canal_id || 1;
      if (!chatState.messages[cid]) chatState.messages[cid] = [];
      chatState.messages[cid].push(m);
      if (chatState.currentCanalId === cid) {
        appendBurbuja(m);
      }
      actualizarNoLeidos(cid);
      actualizarListaCanales();
    }
  };
  chatState.ws.onclose = function() {
    chatState.ws = null;
    chatState.wsReconnectTimer = setTimeout(conectarWSChat, 3000);
  };
  chatState.ws.onerror = function() {
    if (chatState.ws) chatState.ws.close();
  };
}

function enviarMensajeChat() {
  var input = document.getElementById('chat-input');
  if (!input) return;
  var texto = input.value.trim();
  if (!texto || !chatState.currentCanalId) return;
  input.value = '';
  autoExpandChatInput(input);
  if (chatState.ws && chatState.ws.readyState === WebSocket.OPEN) {
    chatState.ws.send(JSON.stringify({
      type: 'chat',
      id: 'msg_' + Date.now(),
      payload: texto,
      extra: { canal_id: chatState.currentCanalId }
    }));
  } else {
    fetch('/api/chat/mensajes', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({mensaje: texto, canal_id: chatState.currentCanalId})
    }).catch(function(){});
  }
}

function marcarLeido(canalId, msgId) {
  if (chatState.ws && chatState.ws.readyState === WebSocket.OPEN) {
    chatState.ws.send(JSON.stringify({
      type: 'mark_read',
      id: 'read_' + Date.now(),
      extra: { canal_id: canalId, msg_id: msgId }
    }));
  } else {
    fetch('/api/chat/leido', {
      method: 'PUT',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({canal_id: canalId, msg_id: msgId})
    }).catch(function(){});
  }
}

// -- Channel loading --

function cargarCanales() {
  if (chatState.loadingChannels) return;
  chatState.loadingChannels = true;
  fetch('/api/chat/canales').then(function(r) { return r.json(); }).then(function(canales) {
    chatState.channels = canales;
    renderizarListaCanales();
    if (!chatState.currentCanalId && canales.length > 0) {
      seleccionarCanal(canales[0].id);
    }
    chatState.loadingChannels = false;
  }).catch(function() { chatState.loadingChannels = false; });
}

function seleccionarCanal(canalId) {
  if (chatState.currentCanalId === canalId) return;
  chatState.currentCanalId = canalId;
  renderizarListaCanales();
  var header = document.getElementById('chat-header-title');
  var canal = chatState.channels.find(function(c) { return c.id === canalId; });
  if (header && canal) {
    header.innerHTML = '<i class="ti ti-' + (canal.icono || 'hash') + '"></i> ' + escapeHtml(canal.nombre);
  }
  document.getElementById('chat-mensajes').innerHTML = '';
  document.getElementById('chat-empty').style.display = 'block';
  document.getElementById('chat-bottom').style.display = 'block';
  chatState.messages[canalId] = [];
  chatState.hasMore[canalId] = true;
  chatState.loadedUpTo[canalId] = null;
  cargarMensajesCanal(canalId, false);
}

function cargarMensajesCanal(canalId, prepend) {
  if (chatState.loadingOlder[canalId]) return;
  chatState.loadingOlder[canalId] = true;
  var url = '/api/chat/mensajes?canal_id=' + canalId + '&limit=30';
  if (prepend && chatState.loadedUpTo[canalId]) {
    url += '&before_id=' + chatState.loadedUpTo[canalId];
  }
  fetch(url).then(function(r) { return r.json(); }).then(function(msgs) {
    var cont = document.getElementById('chat-mensajes');
    var empty = document.getElementById('chat-empty');
    if (!msgs.length) {
      if (!prepend) { empty.style.display = 'block'; }
      chatState.hasMore[canalId] = false;
      chatState.loadingOlder[canalId] = false;
      return;
    }
    empty.style.display = 'none';
    if (!chatState.messages[canalId]) chatState.messages[canalId] = [];
    if (prepend) {
      chatState.messages[canalId] = msgs.reverse().concat(chatState.messages[canalId]);
      var scrollHeightBefore = cont.scrollHeight;
      cont.insertAdjacentHTML('afterbegin', msgs.reverse().map(renderBurbuja).join(''));
      chatState.loadedUpTo[canalId] = msgs[msgs.length - 1].id;
      cont.scrollTop = cont.scrollHeight - scrollHeightBefore;
    } else {
      chatState.messages[canalId] = msgs.reverse();
      cont.innerHTML = msgs.reverse().map(renderBurbuja).join('');
      chatState.loadedUpTo[canalId] = msgs[msgs.length - 1].id;
      if (msgs.length > 0) {
        var lastId = msgs[msgs.length - 1].id;
        marcarLeido(canalId, lastId);
        actualizarNoLeidos(canalId);
      }
      setTimeout(scrollChatToBottom, 50);
    }
    if (msgs.length < 30) chatState.hasMore[canalId] = false;
    chatState.loadingOlder[canalId] = false;
  }).catch(function() { chatState.loadingOlder[canalId] = false; });
}

function appendBurbuja(m) {
  var cont = document.getElementById('chat-mensajes');
  var empty = document.getElementById('chat-empty');
  if (empty) empty.style.display = 'none';
  cont.insertAdjacentHTML('beforeend', renderBurbuja(m));
  marcarLeido(m.canal_id || chatState.currentCanalId, m.msg_id || m.id);
  scrollChatToBottom();
}

function renderBurbuja(m) {
  var isOwn = (m.user_id || 0) === chatState.userId;
  var content = m.message || m.mensaje || '';
  var user = m.username || m.usuario || '?';
  var t = m.created || m.created_on || m.timestamp || '';
  var avatar = user.charAt(0).toUpperCase();
  var timeStr = formatChatTime(t);
  return '<div class="cb cb-' + (isOwn ? 'own' : 'other') + '">' +
    (!isOwn ? '<div class="cb-avatar">' + escapeHtml(avatar) + '</div>' : '') +
    '<div class="cb-body">' +
    (!isOwn ? '<div class="cb-user">' + escapeHtml(user) + '</div>' : '') +
    '<div class="cb-text">' + escapeHtml(content) + '</div>' +
    '<div class="cb-time">' + timeStr + '</div>' +
    '</div></div>';
}

function formatChatTime(ts) {
  if (!ts) return '';
  var m = ts.match(/^(\d{4})-(\d{2})-(\d{2})[T ](\d{2}):(\d{2})/);
  if (!m) return ts;
  var d = new Date(+m[1], +m[2] - 1, +m[3], +m[4], +m[5]);
  var now = new Date();
  var sameDay = d.getFullYear() === now.getFullYear() && d.getMonth() === now.getMonth() && d.getDate() === now.getDate();
  var opts = sameDay ? {hour: '2-digit', minute: '2-digit'} : {day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit'};
  return d.toLocaleTimeString('es-MX', opts);
}

function renderizarListaCanales() {
  var list = document.getElementById('chat-canal-list');
  if (!list) return;
  var html = '';
  for (var i = 0; i < chatState.channels.length; i++) {
    var c = chatState.channels[i];
    var active = c.id === chatState.currentCanalId ? ' active' : '';
    var badge = c.no_leidos > 0 ? '<span class="cc-badge">' + c.no_leidos + '</span>' : '';
    html += '<div class="cc-item' + active + '" onclick="seleccionarCanal(' + c.id + ')">' +
      '<span class="cc-icon"><i class="ti ti-' + (c.icono || 'hash') + '"></i></span>' +
      '<span class="cc-name">' + escapeHtml(c.nombre) + '</span>' +
      badge +
      '</div>';
  }
  list.innerHTML = html;

  var btn = document.getElementById('chat-toggle-badge');
  if (btn) {
    var total = 0;
    for (var j = 0; j < chatState.channels.length; j++) {
      total += chatState.channels[j].no_leidos || 0;
    }
    btn.textContent = total > 0 ? total : '';
    btn.style.display = total > 0 ? 'flex' : 'none';
  }
}

function actualizarNoLeidos(canalId) {
  for (var i = 0; i < chatState.channels.length; i++) {
    if (chatState.channels[i].id === canalId) {
      chatState.channels[i].no_leidos = 0;
      break;
    }
  }
}

function actualizarListaCanales() {
  renderizarListaCanales();
}

function scrollChatToBottom() {
  var cont = document.getElementById('chat-mensajes');
  if (cont) cont.scrollTop = cont.scrollHeight;
}

function autoExpandChatInput(el) {
  el.style.height = 'auto';
  el.style.height = Math.min(el.scrollHeight, 120) + 'px';
}

function escapeHtml(s) {
  if (typeof s !== 'string') return '';
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

document.addEventListener('DOMContentLoaded', function() {
  document.addEventListener('keydown', function(e) {
    if (e.ctrlKey && e.shiftKey && e.key === 'C') {
      e.preventDefault();
      toggleChatSidebar();
    }
    if (e.key === 'Escape' && chatState.open) {
      toggleChatSidebar();
    }
    if (e.key === 'Enter' && !e.shiftKey && chatState.open && document.activeElement === document.getElementById('chat-input')) {
      e.preventDefault();
      enviarMensajeChat();
    }
  });

  document.addEventListener('click', function(e) {
    var sb = document.getElementById('chat-sidebar');
    var btn = document.querySelector('.chat-toggle-btn');
    if (chatState.open && sb && !sb.contains(e.target) && btn && !btn.contains(e.target)) {
      // click outside closes sidebar on mobile
      if (window.innerWidth < 1024) {
        toggleChatSidebar();
      }
    }
  });

  // Scroll to load older messages
  document.addEventListener('scroll', function(e) {
    var cont = document.getElementById('chat-mensajes');
    if (!cont || chatState.loadingOlder[chatState.currentCanalId] || !chatState.hasMore[chatState.currentCanalId]) return;
    if (cont.scrollTop < 60) {
      cargarMensajesCanal(chatState.currentCanalId, true);
    }
  }, true);
});
