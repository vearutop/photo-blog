(function () {
  // Editor for the room-preview widget: drag the sample photo over the wall, calibrate the perspective
  // lines, measure a known length for scale, then copy the resulting config.
  var container = document.querySelector('.room-preview');
  if (!container || !container.roomPreviewState) return;

  var st = container.roomPreviewState, cfg = st.cfg;
  var dot = function (a, b) { return a[0] * b[0] + a[1] * b[1] + a[2] * b[2]; };
  var M = [[.57, .6], [.9, .63]]; // measured segment, fractions of image width.
  var drag;

  var toggleBtn = document.createElement('button');
  toggleBtn.className = 'rp-editor-toggle';
  toggleBtn.textContent = 'Hide editor';
  document.body.appendChild(toggleBtn);

  var editor = document.createElement('div');
  editor.className = 'room-preview-editor';
  editor.innerHTML =
    '<div class="rp-row">' +
    'long side <input type="range" name="widthr" min="10" max="300"> <input type="number" name="width" step="1"> cm · ' +
    'wall <select name="wall"><option value="0">red lines<option value="1">blue lines</select> · ' +
    'camera→wall <input type="number" name="dist"> cm · ' +
    '<label><input type="checkbox" name="meas"> measure</label> <input type="number" name="mlen" value="200"> cm · ' +
    '<label><input type="checkbox" name="amb"> ambient light</label> · ' +
    'crop to <input type="number" name="aspect" step="0.01" placeholder="none"> (long/short, 1.5 = 3:2) · ' +
    'widget max width <input type="number" name="maxw" placeholder="full"> px · ' +
    '<label><input type="checkbox" name="cal"> calibrate</label> <span class="rp-info"></span>' +
    '</div>' +
    '<div class="rp-row"><textarea name="json" spellcheck="false"></textarea></div>' +
    '<div class="rp-row"><button name="copyjson">Copy JSON</button> <button name="copymd">Copy markdown widget</button></div>';
  document.body.appendChild(editor);

  var f = {};
  editor.querySelectorAll('[name]').forEach(function (e) { f[e.name] = e; });
  f.width.value = f.widthr.value = cfg.width;
  f.wall.value = cfg.wall;
  f.dist.value = cfg.dist;
  f.amb.checked = cfg.ambient !== false;
  f.aspect.value = cfg.aspect || '';
  f.maxw.value = cfg.maxWidth || '';

  toggleBtn.addEventListener('click', function () {
    var hidden = editor.classList.toggle('hidden');
    toggleBtn.textContent = hidden ? 'Show editor' : 'Hide editor';
    st.fit();
  });
  // the room should stay fully visible above the panel.
  st.maxHeight = function () { return window.innerHeight - (editor.classList.contains('hidden') ? 0 : editor.offsetHeight); };

  function round(v) {
    return typeof v === 'number' ? Math.round(v * 1e4) / 1e4 : v;
  }

  function out() {
    var o = { room: cfg.room, r: cfg.r, b: cfg.b, wall: cfg.wall, pos: cfg.pos, width: cfg.width, dist: cfg.dist };
    if (cfg.ambient === false) o.ambient = false;
    if (cfg.aspect) o.aspect = cfg.aspect;
    if (cfg.maxWidth) o.maxWidth = cfg.maxWidth;
    return JSON.stringify(o, function (_, v) { return round(v); });
  }

  // Before every render: with "measure" on, the segment M lies in the wall plane and is mlen cm long,
  // which gives the wall distance.
  st.before = function () {
    cfg.wall = +f.wall.value;
    cfg.width = +f.width.value || cfg.width;
    cfg.ambient = f.amb.checked;
    cfg.aspect = +f.aspect.value || 0; // blank: keep the photo as is.
    cfg.maxWidth = +f.maxw.value || 0;
    container.style.maxWidth = cfg.maxWidth ? cfg.maxWidth + 'px' : '';
    var K = st.camera();
    if (K && f.meas.checked) {
      var nn = cfg.wall ? K.u : K.n;
      var Q = M.map(function (p) {
        var r = [p[0] * st.W - K.c[0], p[1] * st.W - K.c[1], K.f];
        return r.map(function (x) { return x / Math.abs(dot(nn, r)); });
      });
      var d = Math.hypot(Q[0][0] - Q[1][0], Q[0][1] - Q[1][1], Q[0][2] - Q[1][2]);
      if (d > 0) f.dist.value = Math.round(f.mlen.value / d);
    }
    cfg.dist = +f.dist.value || cfg.dist;
  };

  st.after = function () {
    var K = st.K, W = st.W, k = st.k;
    f.json.value = out();
    editor.querySelector('.rp-info').textContent = K ?
      'f=' + (K.f | 0) + 'px ≈ ' + (K.f / W * 36).toFixed(0) + 'mm equiv' : 'inconsistent lines';

    var cal = f.cal.checked, meas = f.meas.checked, h = '';
    var circle = function (attrs, p, col) {
      return '<circle ' + attrs + ' cx=' + p[0] * W + ' cy=' + p[1] * W + ' r=' + 8 / k + ' fill=' + col + ' fill-opacity=.6 />';
    };
    var line = function (a, b, col, w, ext) {
      var d = [b[0] - a[0], b[1] - a[1]], s = ext / Math.hypot(d[0], d[1]);
      return '<line x1=' + (a[0] * W - d[0] * W * s) + ' y1=' + (a[1] * W - d[1] * W * s) + ' x2=' + (b[0] * W + d[0] * W * s) +
        ' y2=' + (b[1] * W + d[1] * W * s) + ' stroke=' + col + ' stroke-width=' + w / k + ' />';
    };
    if (cal) ['r', 'b'].forEach(function (g, gi) {
      cfg[g].forEach(function (l, li) {
        var col = gi ? '#4af' : '#f44';
        h += line(l[0], l[1], col, 1, 5000 / W);
        l.forEach(function (p, i) { h += circle('data-g=' + g + ' data-l=' + li + ' data-i=' + i, p, col); });
      });
    });
    if (meas) {
      h += line(M[0], M[1], '#4f4', 2, 0);
      M.forEach(function (p, i) { h += circle('data-g=m data-i=' + i, p, '#4f4'); });
    }
    st.svg.innerHTML = h;
  };

  function setWidth(v) {
    f.width.value = f.widthr.value = Math.round(Math.min(300, Math.max(10, v)));
    st.render();
  }

  ['width', 'dist', 'wall', 'amb', 'cal', 'meas', 'mlen', 'aspect', 'maxw'].forEach(function (n) {
    f[n].addEventListener('input', function () { st.render(); });
  });
  f.widthr.addEventListener('input', function () { setWidth(f.widthr.value); });
  f.width.addEventListener('input', function () { setWidth(f.width.value); });
  f.json.addEventListener('input', function () {
    try {
      var j = JSON.parse(f.json.value);
      ['r', 'b', 'pos', 'wall', 'width', 'dist'].forEach(function (n) { if (j[n] !== undefined) cfg[n] = j[n]; });
      f.width.value = f.widthr.value = cfg.width; f.wall.value = cfg.wall; f.dist.value = cfg.dist;
      st.render();
    } catch (e) {}
  });

  function copy(text) {
    navigator.clipboard.writeText(text);
  }
  f.copyjson.addEventListener('click', function () { copy(out()); });
  f.copymd.addEventListener('click', function () {
    copy(':::{.room-preview}\n```json\n' + out() + '\n```\n:::\n');
  });

  // Dragging: the sample photo (default), or handles of calibration lines and the measured segment.
  var box = st.box;
  box.style.touchAction = 'none';
  function move(e) {
    var b = st.stage.getBoundingClientRect(), p = [(e.clientX - b.left) / st.k / st.W, (e.clientY - b.top) / st.k / st.W];
    if (drag === 'pos') cfg.pos = p; else if (drag.g === 'm') M[drag.i] = p; else cfg[drag.g][drag.l][drag.i] = p;
    st.render();
  }
  box.addEventListener('pointerdown', function (e) {
    var d = e.target.dataset || {};
    if ((f.cal.checked || f.meas.checked) && !d.g) return;
    drag = d.g ? { g: d.g, l: +d.l, i: +d.i } : 'pos';
    box.setPointerCapture(e.pointerId);
    if (drag === 'pos') move(e);
  });
  box.addEventListener('pointermove', function (e) { if (drag) move(e); });
  box.addEventListener('pointerup', function () { drag = null; });
  box.addEventListener('wheel', function (e) {
    e.preventDefault();
    setWidth(f.width.value * (e.deltaY < 0 ? 1.05 : 1 / 1.05));
  }, { passive: false });

  st.fit();
})();
