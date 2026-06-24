var SoundManager = (function(){
  var muted = localStorage.getItem('pos.soundMuted') === 'true';
  var volume = parseFloat(localStorage.getItem('pos.soundVolume') || '0.5');
  var unlocked = false;
  var ctx = null;

  function getCtx(){
    if(!ctx) ctx = new (window.AudioContext || window.webkitAudioContext)();
    return ctx;
  }

  function playTone(freq, duration, type, vol){
    if(muted || !unlocked) return;
    if(window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;
    if(window.matchMedia('(prefers-reduced-sound: reduce)').matches) return;
    try{
      var c = getCtx();
      var osc = c.createOscillator();
      var gain = c.createGain();
      osc.type = type || 'sine';
      osc.frequency.setValueAtTime(freq, c.currentTime);
      gain.gain.setValueAtTime((vol||1)*volume*0.3, c.currentTime);
      gain.gain.exponentialRampToValueAtTime(0.001, c.currentTime + duration);
      osc.connect(gain);
      gain.connect(c.destination);
      osc.start();
      osc.stop(c.currentTime + duration);
    }catch(e){}
  }

  function play(name){
    switch(name){
      case 'beep':
        playTone(1200, 0.08, 'sine');
        break;
      case 'success':
        playTone(800, 0.1, 'sine', 0.8);
        setTimeout(function(){ playTone(1000, 0.1, 'sine', 0.8); }, 100);
        setTimeout(function(){ playTone(1200, 0.15, 'sine', 0.8); }, 200);
        break;
      case 'error':
        playTone(250, 0.25, 'sawtooth', 0.6);
        break;
      case 'notification':
        playTone(880, 0.08, 'sine', 0.5);
        setTimeout(function(){ playTone(1100, 0.08, 'sine', 0.5); }, 80);
        break;
      case 'alert':
        playTone(440, 0.15, 'square', 0.4);
        setTimeout(function(){ playTone(440, 0.15, 'square', 0.4); }, 200);
        break;
    }
  }

  function unlock(){
    if(unlocked) return;
    unlocked = true;
    try{ getCtx().resume(); }catch(e){}
  }

  function setMuted(v){
    muted = v;
    localStorage.setItem('pos.soundMuted', v);
    updateIcon();
  }

  function isMuted(){ return muted; }

  function setVolume(v){
    volume = Math.max(0, Math.min(1, v));
    localStorage.setItem('pos.soundVolume', volume);
  }

  function getVolume(){ return volume; }

  function updateIcon(){
    var btn = document.getElementById('sound-toggle');
    if(btn) btn.innerHTML = muted ? '<i class="ti ti-volume-off"></i>' : '<i class="ti ti-volume"></i>';
  }

  document.addEventListener('click', unlock, {once:true});
  document.addEventListener('touchstart', unlock, {once:true});
  document.addEventListener('keydown', unlock, {once:true});

  return { play:play, setMuted:setMuted, isMuted:isMuted, setVolume:setVolume, getVolume:getVolume, unlock:unlock, updateIcon:updateIcon };
})();
