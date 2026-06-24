(function(){
  var modalStack = [];

  function topModal(){
    for(var i=modalStack.length-1;i>=0;i--){
      var el = document.getElementById(modalStack[i]);
      if(el && el.classList.contains('open')) return modalStack[i];
    }
    return null;
  }
  function registerModal(id){ if(modalStack.indexOf(id)===-1) modalStack.push(id); }
  function unregisterModal(id){ var i=modalStack.indexOf(id); if(i>-1) modalStack.splice(i,1); }
  window.registerModal=registerModal; window.unregisterModal=unregisterModal;

  var isInputFocused = function(){
    var t = document.activeElement;
    return t && (t.tagName==='INPUT' || t.tagName==='TEXTAREA' || t.tagName==='SELECT');
  };
  var isCobrarInputFocused = function(){
    var t = document.activeElement;
    return t && t.id && t.id.indexOf('pago')!==-1;
  };

  document.addEventListener('keydown', function(e){
    var key = e.key;
    var ctrl = e.ctrlKey || e.metaKey;

    // Ctrl+Z — deshacer último producto
    if(ctrl && key==='z'){
      e.preventDefault();
      if(typeof window.deshacerUltimo === 'function') window.deshacerUltimo();
      return;
    }
    // Ctrl+L — limpiar carrito
    if(ctrl && key==='l'){
      e.preventDefault();
      if(typeof window.cancelarCarrito === 'function') window.cancelarCarrito();
      return;
    }
    // Ctrl+D — duplicar última línea
    if(ctrl && key==='d'){
      e.preventDefault();
      if(typeof window.duplicarUltimo === 'function') window.duplicarUltimo();
      return;
    }

    // ESC — cerrar modal más reciente
    if(key==='Escape'){
      var tm = topModal();
      if(tm && tm==='confirm-modal' && typeof window.cerrarConfirmModal === 'function'){ e.preventDefault(); window.cerrarConfirmModal(); return; }
      if(tm && tm==='qv-modal' && typeof window.ocultarQuickView === 'function'){ e.preventDefault(); window.ocultarQuickView(); return; }
      if(tm && tm==='new-client-modal' && typeof window.hideNewClientModal === 'function'){ e.preventDefault(); window.hideNewClientModal(); return; }
      if(tm && tm==='help-modal' && typeof window.cerrarHelpModal === 'function'){ e.preventDefault(); window.cerrarHelpModal(); return; }
      if(tm && tm==='cobro-modal' && typeof window.cerrarCobroModal === 'function'){ e.preventDefault(); window.cerrarCobroModal(); return; }
      if(document.getElementById('cart-panel') && document.getElementById('cart-panel').classList.contains('open')){
        if(typeof window.toggleCart === 'function'){ e.preventDefault(); window.toggleCart(); }
      }
      return;
    }

    // Si está escribiendo en un input, no interceptar teclas de función (excepto F2 en search)
    if(isInputFocused()){
      // Permitir F2 incluso en input
      if(key!=='F2') return;
    }
    // F1 en modal de cobrar no debe reabrir
    if(key==='F1' && isCobrarInputFocused()) return;
    // F12 en modal de cobrar
    if(key==='F12' && isCobrarInputFocused()) return;

    switch(key){
      case 'F1':
        e.preventDefault();
        if(typeof window.cobrar === 'function') window.cobrar();
        break;
      case 'F2':
        e.preventDefault();
        var inp = document.getElementById('search');
        if(inp){ inp.focus(); inp.select(); }
        break;
      case 'F3':
        e.preventDefault();
        var cs = document.getElementById('client-search');
        if(cs){ cs.focus(); cs.select(); }
        break;
      case 'F4':
        e.preventDefault();
        if(typeof window.handlePrintLastTicket === 'function') window.handlePrintLastTicket();
        break;
      case 'F5':
        e.preventDefault();
        if(typeof window.refreshProductos === 'function') window.refreshProductos();
        break;
      case 'F8':
        e.preventDefault();
        if(typeof window.handleSinVenta === 'function') window.handleSinVenta();
        break;
      case 'F9':
        e.preventDefault();
        if(typeof window.handleDescuentoGlobal === 'function') window.handleDescuentoGlobal();
        break;
      case 'F12':
        e.preventDefault();
        if(typeof window.handleCerrarCaja === 'function') window.handleCerrarCaja();
        break;
    }
  });

  // Inyectar estilo para help button
  var style = document.createElement('style');
  style.textContent = '.help-btn{position:fixed;bottom:1rem;left:1rem;width:36px;height:36px;border-radius:50%;background:var(--bg3);border:1.5px solid var(--border);color:var(--text2);font-size:1rem;font-weight:700;cursor:pointer;z-index:999;display:flex;align-items:center;justify-content:center;transition:all .15s;box-shadow:var(--shadow-md);}.help-btn:hover{background:var(--accent);color:white;border-color:var(--accent);}.help-modal{display:none;position:fixed;inset:0;background:rgba(0,0,0,.6);z-index:1200;align-items:center;justify-content:center;}.help-modal.open{display:flex;}.help-content{background:var(--bg2);border:1.5px solid var(--border);border-radius:var(--radius-lg);padding:1.3rem;width:90%;max-width:480px;max-height:80vh;overflow-y:auto;box-shadow:var(--shadow-lg);}.help-content h3{margin:0 0 1rem;font-size:1.1rem;color:var(--text);display:flex;align-items:center;gap:.5rem;}.help-grid{display:grid;grid-template-columns:auto 1fr;gap:.35rem .75rem;font-size:.82rem;}.help-grid .key{padding:.15rem .45rem;background:var(--bg3);border:1px solid var(--border);border-radius:4px;font-family:monospace;font-size:.78rem;color:var(--accent);text-align:center;min-width:60px;}.help-grid .desc{color:var(--text2);display:flex;align-items:center;}';
  document.head.appendChild(style);
})();
