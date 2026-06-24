(function(){
  var searchCache = {};
  var cacheKeys = [];
  var MAX_CACHE = 50;
  var debounceTimer = null;
  var selectedIndex = -1;
  var results = [];
  var popularPreloaded = false;

  function getFromCache(q){
    return searchCache[q];
  }

  function setCache(q, data){
    searchCache[q] = data;
    var idx = cacheKeys.indexOf(q);
    if(idx>-1) cacheKeys.splice(idx,1);
    cacheKeys.push(q);
    if(cacheKeys.length > MAX_CACHE){
      var old = cacheKeys.shift();
      delete searchCache[old];
    }
  }

  function highlight(text, query){
    if(!query) return text;
    var re = new RegExp('('+query.replace(/[.*+?^${}()|[\]\\]/g,'\\$&').split('').join('|')+')','gi');
    return text.replace(re, '<strong>$1</strong>');
  }

  function closeDropdown(){
    var dd = document.getElementById('search-dropdown');
    if(dd){ dd.remove(); }
    selectedIndex = -1;
  }

  function renderDropdown(items, q){
    closeDropdown();
    if(!items || !items.length){ return; }
    var dd = document.createElement('div');
    dd.id = 'search-dropdown';
    dd.setAttribute('role', 'listbox');
    dd.style.cssText = 'position:absolute;top:100%;left:0;right:0;background:var(--bg2);border:1.5px solid var(--border);border-radius:0 0 10px 10px;max-height:360px;overflow-y:auto;z-index:500;box-shadow:var(--shadow-lg);margin-top:2px;';
    dd.innerHTML = items.map(function(item, i){
      var stockColor = item.stock <= 0 ? 'color:var(--danger)' : item.stock < 10 ? 'color:var(--warning)' : 'color:var(--success)';
      var img = item.imagen ? '<img src="'+item.imagen+'" style="width:32px;height:32px;object-fit:cover;border-radius:6px;">' : '<div style="width:32px;height:32px;border-radius:6px;background:var(--bg3);display:flex;align-items:center;justify-content:center;font-size:.7rem;color:var(--text3);"><i class="ti ti-box"></i></div>';
      return '<div role="option" id="search-opt-'+i+'" class="search-result-item" data-index="'+i+'" onclick="searchSelectResult('+i+')" style="display:flex;align-items:center;gap:.5rem;padding:.4rem .6rem;cursor:pointer;border-bottom:1px solid var(--border);transition:background .1s;" onmouseenter="this.style.background=\'var(--bg3)\'" onmouseleave="this.style.background=\'\'">'+
        img+
        '<div style="flex:1;min-width:0;"><div style="font-size:.85rem;color:var(--text);white-space:nowrap;overflow:hidden;text-overflow:ellipsis;">'+highlight(item.nombre, q)+'</div><div style="font-size:.7rem;color:var(--text3);">'+highlight(item.codigo, q)+'</div></div>'+
        '<div style="text-align:right;flex-shrink:0;"><div style="font-size:.9rem;font-weight:700;color:var(--text);">$'+item.precio.toFixed(2)+'</div><div style="font-size:.7rem;font-weight:600;'+stockColor+'">'+(item.stock > 0 ? item.stock : 'agotado')+'</div></div>'+
      '</div>';
    }).join('');
    var inp = document.getElementById('search');
    if(inp && inp.parentNode) inp.parentNode.style.position = 'relative';
    inp.parentNode.appendChild(dd);
  }

  function doSearch(q){
    q = q.trim();
    if(q.length < 2){
      closeDropdown();
      return;
    }
    var cached = getFromCache(q);
    if(cached){
      results = cached;
      selectedIndex = -1;
      renderDropdown(cached, q);
      return;
    }
    var dd = document.getElementById('search-dropdown');
    if(!dd){
      var loading = document.createElement('div');
      loading.id = 'search-dropdown';
      loading.style.cssText = 'position:absolute;top:100%;left:0;right:0;background:var(--bg2);border:1.5px solid var(--border);border-radius:0 0 10px 10px;z-index:500;box-shadow:var(--shadow-lg);padding:.6rem;text-align:center;font-size:.8rem;color:var(--text3);';
      loading.textContent = 'Buscando...';
      var inp = document.getElementById('search');
      if(inp && inp.parentNode){ inp.parentNode.style.position = 'relative'; inp.parentNode.appendChild(loading); }
    }
    fetch('/api/productos/search?q='+encodeURIComponent(q)+'&limit=20')
      .then(function(r){ if(!r.ok) throw new Error('Error'); return r.json(); })
      .then(function(data){
        var items = data || [];
        setCache(q, items);
        results = items;
        selectedIndex = -1;
        renderDropdown(items, q);
      })
      .catch(function(){
        closeDropdown();
      });
  }

  function searchInputHandler(e){
    var q = this.value;
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(function(){ doSearch(q); }, 200);
  }

  function searchKeydownHandler(e){
    var dd = document.getElementById('search-dropdown');
    if(!dd){ return; }
    var items = dd.querySelectorAll('.search-result-item');
    if(!items.length){ return; }
    if(e.key === 'ArrowDown'){
      e.preventDefault();
      selectedIndex = Math.min(selectedIndex + 1, items.length - 1);
      updateSelection(items);
    } else if(e.key === 'ArrowUp'){
      e.preventDefault();
      selectedIndex = Math.max(selectedIndex - 1, -1);
      updateSelection(items);
    } else if(e.key === 'Enter'){
      e.preventDefault();
      if(selectedIndex >= 0 && selectedIndex < results.length){
        var item = results[selectedIndex];
        if(typeof window.searchSelectResult === 'function'){
          window.searchSelectResult(selectedIndex);
        } else {
          searchSelectResult(selectedIndex);
        }
      }
    } else if(e.key === 'Escape'){
      closeDropdown();
    }
  }

  function updateSelection(items){
    items.forEach(function(el, i){
      el.style.background = (i === selectedIndex) ? 'var(--accent)' : '';
      el.style.color = (i === selectedIndex) ? 'white' : '';
      el.setAttribute('aria-selected', (i === selectedIndex) ? 'true' : 'false');
    });
    if(selectedIndex >= 0 && items[selectedIndex]){
      items[selectedIndex].scrollIntoView({block:'nearest'});
    }
  }

  function searchSelectResult(idx){
    if(idx < 0 || idx >= results.length) return;
    var item = results[idx];
    closeDropdown();
    var inp = document.getElementById('search');
    if(inp){ inp.value = ''; }
    if(typeof window.agregarAlCarrito === 'function' && item.codigo){
      window.agregarAlCarrito(item.codigo);
    }
    if(typeof window.searchProductos === 'function') window.searchProductos('');
  }
  window.searchSelectResult = searchSelectResult;

  function onBlur(e){
    setTimeout(closeDropdown, 200);
  }

  function onFocus(e){
    var q = this.value.trim();
    if(q.length >= 2) doSearch(q);
  }

  function initSearch(){
    var inp = document.getElementById('search');
    if(!inp) return;
    inp.removeEventListener('input', searchInputHandler);
    inp.removeEventListener('keydown', searchKeydownHandler);
    inp.removeEventListener('blur', onBlur);
    inp.removeEventListener('focus', onFocus);
    inp.addEventListener('input', searchInputHandler);
    inp.addEventListener('keydown', searchKeydownHandler);
    inp.addEventListener('blur', onBlur);
    inp.addEventListener('focus', onFocus);
    inp.setAttribute('role', 'combobox');
    inp.setAttribute('aria-expanded', 'false');
    inp.setAttribute('autocomplete', 'off');

    // Preload popular products
    if(!popularPreloaded){
      popularPreloaded = true;
      fetch('/api/productos/search?q=&limit=20')
        .then(function(r){ return r.json(); })
        .then(function(data){
          if(data && data.length){
            setCache('', data);
          }
        })
        .catch(function(){});
    }
  }

  if(document.readyState === 'loading'){
    document.addEventListener('DOMContentLoaded', initSearch);
  } else {
    initSearch();
  }
})();
