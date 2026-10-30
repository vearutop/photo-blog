(function () {
  // A room-preview instance is any ".room-preview" element carrying its config as
  // JSON, either in a <script type="application/json"> child or a nested code
  // fence (the `:::{.room-preview}` markdown widget):
  // {room, r, b, wall, pos, width, dist, ambient, items, thumbBase}.
  //
  // It shows an image of an empty room with a photo hung on one of its walls, in
  // the right perspective, and lets visitors flip through photos (items; by
  // default all the images of the album page). Use /room-preview-{room},{sample}
  // to produce the config.
  //
  // Model: r and b are pairs of lines (each a pair of points, as fractions of the
  // room image width) along parallel horizontal edges of two perpendicular walls,
  // each pair meets in a vanishing point. With the principal point at the image
  // centre two perpendicular vanishing points give the focal length, hence true
  // proportions on the wall plane. wall (0|1) selects the wall the photo hangs
  // on (the one along r|b), pos is the photo centre, width is its long side in
  // cm and dist is the camera-to-wall distance in cm.

  var WHITE = 235; // wall brightness that leaves the photo untouched when lighting it.
  var DEF = {
    r: [[[.51, .148], [.75, .058]], [[.533, .4375], [.75, .519]]],
    b: [[[.125, .067], [.508, .148]], [[.308, .457], [.433, .4375]]],
    pos: [.79, .283],
  };

  var dot = function (a, b) { return a[0] * b[0] + a[1] * b[1] + a[2] * b[2]; };
  var cross = function (a, b) { return [a[1] * b[2] - a[2] * b[1], a[2] * b[0] - a[0] * b[2], a[0] * b[1] - a[1] * b[0]]; };
  var norm = function (a) { var l = Math.hypot(a[0], a[1], a[2]); return a.map(function (x) { return x / l; }); };

  function isect(l1, l2) {
    var p1 = l1[0], p2 = l1[1], p3 = l2[0], p4 = l2[1];
    var d = (p1[0] - p2[0]) * (p3[1] - p4[1]) - (p1[1] - p2[1]) * (p3[0] - p4[0]);
    var a = p1[0] * p2[1] - p1[1] * p2[0], b = p3[0] * p4[1] - p3[1] * p4[0];
    return [(a * (p3[0] - p4[0]) - (p1[0] - p2[0]) * b) / d, (a * (p3[1] - p4[1]) - (p1[1] - p2[1]) * b) / d];
  }

  function readConfig(container) {
    var script = container.querySelector('script[type="application/json"]');
    var text = script ? script.textContent : ((container.querySelector('pre code') || {}).textContent);
    if (!text) return null;

    try {
      return JSON.parse(text);
    } catch (e) {
      console.error('room-preview: invalid JSON config', e);
      return null;
    }
  }

  function el(tag, cls, parent) {
    var e = document.createElement(tag);
    if (cls) e.className = cls;
    if (parent) parent.appendChild(e);
    return e;
  }


  // Draws img (sx0,sy0..) mapped onto a destination triangle: affine map of a source triangle onto a destination one.
  function tri(x, img, d0, d1, d2, s0, s1, s2) {
    var cx = (d0[0] + d1[0] + d2[0]) / 3, cy = (d0[1] + d1[1] + d2[1]) / 3;
    var grow = function (p) { // ~1.5px outwards, so neighbours overlap and antialiased clip edges leave no hairline seams.
      var dx = p[0] - cx, dy = p[1] - cy, k = 1 + 1.5 / (Math.hypot(dx, dy) || 1);
      return [cx + dx * k, cy + dy * k];
    };
    var g0 = grow(d0), g1 = grow(d1), g2 = grow(d2);
    var det = s0[0] * (s1[1] - s2[1]) + s1[0] * (s2[1] - s0[1]) + s2[0] * (s0[1] - s1[1]);
    if (!det) return;
    var solve = function (v0, v1, v2) {
      return [
        (v0 * (s1[1] - s2[1]) + v1 * (s2[1] - s0[1]) + v2 * (s0[1] - s1[1])) / det,
        (v0 * (s2[0] - s1[0]) + v1 * (s0[0] - s2[0]) + v2 * (s1[0] - s0[0])) / det,
        (v0 * (s1[0] * s2[1] - s2[0] * s1[1]) + v1 * (s2[0] * s0[1] - s0[0] * s2[1]) + v2 * (s0[0] * s1[1] - s1[0] * s0[1])) / det,
      ];
    };
    var X = solve(d0[0], d1[0], d2[0]), Y = solve(d0[1], d1[1], d2[1]);
    x.save();
    x.beginPath(); x.moveTo(g0[0], g0[1]); x.lineTo(g1[0], g1[1]); x.lineTo(g2[0], g2[1]); x.closePath(); x.clip();
    x.setTransform(X[0], Y[0], X[1], Y[1], X[2], Y[2]);
    x.drawImage(img, 0, 0);
    x.restore();
  }

  function init(container) {
    var cfg = readConfig(container);
    if (!cfg || !cfg.room) return;

    cfg.r = cfg.r || DEF.r;
    cfg.b = cfg.b || DEF.b;
    cfg.pos = cfg.pos || DEF.pos;
    cfg.wall = cfg.wall || 0;
    cfg.width = cfg.width || 90;
    cfg.dist = cfg.dist || 300;

    var base = cfg.thumbBase || '/thumb';
    var items = (cfg.items && cfg.items.length) ? cfg.items : [].map.call(
      document.querySelectorAll('.img-select-checkbox[data-hash]'), function (e) { return e.dataset.hash; });

    container.innerHTML = '';
    container.tabIndex = 0;
    if (cfg.maxWidth) container.style.maxWidth = cfg.maxWidth + 'px';
    var box = el('div', 'rp-box', container);
    var stage = el('div', 'rp-stage', box);
    // Everything (room, light, shadow, perspective-warped photo) is painted into one canvas on change only, so
    // scrolling the page moves a single static bitmap, which is what keeps it smooth on phones.
    var cv = el('canvas', 'rp-canvas', stage);
    var room = new Image(), print = new Image();
    var lit = document.createElement('canvas'); // the photo, cropped to the print aspect, with ambient light baked in.
    var shade = document.createElement('canvas'); // coarse ambient light map of the wall under the photo.
    var svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
    svg.setAttribute('class', 'rp-svg');
    stage.appendChild(svg);
    var prev = el('button', 'rp-nav rp-prev', box), next = el('button', 'rp-nav rp-next', box);
    prev.textContent = '‹'; next.textContent = '›';
    prev.title = 'Previous photo'; next.title = 'Next photo';
    var caption = el('div', 'rp-caption', box);
    if (container.requestFullscreen) { // not available e.g. on iPhone.
      var full = el('button', 'rp-full', box);
      full.textContent = '⛶';
      full.title = 'Full screen';
      full.addEventListener('click', function () {
        container.focus();
        if (document.fullscreenElement === container) document.exitFullscreen(); else container.requestFullscreen();
      });
      document.addEventListener('fullscreenchange', function () { setTimeout(fit, 0); });
    }
    if (items.length < 2) prev.style.display = next.style.display = 'none';

    // st is shared with the editor, before/after are its hooks around render.
    var st = { cfg: cfg, container: container, box: box, stage: stage, svg: svg, W: 0, H: 0, k: 1, px: null, idx: 0, items: items, before: null, after: null };
    container.roomPreviewState = st;

    st.camera = function () {
      var W = st.W, H = st.H;
      var px = function (ls) { return ls.map(function (l) { return l.map(function (p) { return [p[0] * W, p[1] * W]; }); }); };
      var v1 = isect.apply(null, px(cfg.r)), v2 = isect.apply(null, px(cfg.b)), c = [W / 2, H / 2];
      var f2 = -((v1[0] - c[0]) * (v2[0] - c[0]) + (v1[1] - c[1]) * (v2[1] - c[1]));
      if (!(f2 > 0)) return null; // vanishing points not on opposite sides of the centre: lines are inconsistent.
      var f = Math.sqrt(f2), u = norm([v1[0] - c[0], v1[1] - c[1], f]), n = norm([v2[0] - c[0], v2[1] - c[1], f]);
      var up = norm(cross(u, n)); if (up[1] > 0) up = up.map(function (x) { return -x; }); // camera y points down.
      return { f: f, c: c, u: u, n: n, up: up, v: [v1, v2] };
    };

    // Ambient light: the room photo is empty, so the wall under the photo shows how the light falls
    // there. Sample it on a coarse grid (through the same homography) into shade, to multiply the photo by.
    // Assumes a white wall.
    function lightMap(h, iw, ih) {
      var px = st.px, PW = st.pw, PH = st.ph, sc = PW / st.W;
      if (cfg.ambient === false || !px) return false;
      var gw = 24, gh = Math.max(1, Math.round(gw * ih / iw));
      shade.width = gw; shade.height = gh;
      var x = shade.getContext('2d'), im = x.createImageData(gw, gh);
      for (var j = 0; j < gh; j++) for (var i = 0; i < gw; i++) {
        var a = (i + .5) * iw / gw, b = (j + .5) * ih / gh, Z = h[0][2] * a + h[1][2] * b + h[2][2];
        var cx = Math.round((h[0][0] * a + h[1][0] * b + h[2][0]) / Z * sc), cy = Math.round((h[0][1] * a + h[1][1] * b + h[2][1]) / Z * sc);
        var rgb = [0, 0, 0], n = 0;
        for (var dy = -1; dy <= 1; dy++) for (var dx = -1; dx <= 1; dx++, n++) {
          var o = (Math.min(PH - 1, Math.max(0, cy + dy)) * PW + Math.min(PW - 1, Math.max(0, cx + dx))) * 4;
          for (var c = 0; c < 3; c++) rgb[c] += px[o + c];
        }
        for (c = 0; c < 3; c++) im.data[(j * gw + i) * 4 + c] = Math.min(255, 255 * rgb[c] / n / WHITE);
        im.data[(j * gw + i) * 4 + 3] = 255;
      }
      x.putImageData(im, 0, 0);
      return true;
    }

    st.render = function () {
      if (!st.W) return;
      if (st.before) st.before();
      var K = st.K = st.camera();
      var x = cv.getContext('2d'), s = cv.width / st.W; // canvas px per room px.
      x.setTransform(1, 0, 0, 1, 0, 0);
      x.drawImage(room, 0, 0, cv.width, cv.height);
      var nw = print.naturalWidth, nh = print.naturalHeight;
      if (K && nw && items.length) {
        // Centre-crop to the print aspect (long/short side, default 0 keeps the photo as is), keeping the photo orientation.
        var A = cfg.aspect || 0, tgt = A ? (nw >= nh ? A : 1 / A) : nw / nh;
        var iw = Math.round(nw / nh > tgt ? nh * tgt : nw), ih = Math.round(nw / nh > tgt ? nh : nw / tgt);
        var Wc = iw >= ih ? cfg.width : cfg.width * iw / ih, Hc = Wc * ih / iw;
        // The wall runs along one horizontal direction (h, towards its vanishing point vp), the other is its normal.
        var h = cfg.wall ? K.n : K.u, nn = cfg.wall ? K.u : K.n, vp = K.v[cfg.wall ? 1 : 0];
        var r = [cfg.pos[0] * st.W - K.c[0], cfg.pos[1] * st.W - K.c[1], K.f], sg = Math.sign(dot(nn, r)) || 1;
        var P = r.map(function (v) { return v * cfg.dist / (sg * dot(nn, r)); });
        var M = function (X) { return [K.f * X[0] + K.c[0] * X[2], K.f * X[1] + K.c[1] * X[2], X[2]]; };
        var hr = vp[0] > cfg.pos[0] * st.W ? 1 : -1; // photo-right: the way that moves rightwards in the image.
        var U = M(h.map(function (v) { return v * hr; })), V = M(K.up), O = M(P);
        var h1 = U.map(function (v) { return v * Wc / iw; });
        var h2 = V.map(function (v) { return -v * Hc / ih; });
        var h3 = O.map(function (v, i) { return v - U[i] * Wc / 2 + V[i] * Hc / 2; });
        var H3 = [h1, h2, h3];
        // photo pixel (a, b) -> canvas px.
        var proj = function (a, b) {
          var Z = h1[2] * a + h2[2] * b + h3[2];
          return [(h1[0] * a + h2[0] * b + h3[0]) / Z * s, (h1[1] * a + h2[1] * b + h3[1]) / Z * s];
        };
        var c00 = proj(0, 0), c10 = proj(iw, 0), c11 = proj(iw, ih), c01 = proj(0, ih);
        var dw = Math.max(Math.hypot(c10[0] - c00[0], c10[1] - c00[1]), Math.hypot(c11[0] - c01[0], c11[1] - c01[1]));

        // Bake: crop, scale to about the size it is shown at (no mipmaps in the triangle drawing), multiply by light.
        var f = Math.min(1, dw / iw * 1.2);
        lit.width = Math.max(1, Math.round(iw * f)); lit.height = Math.max(1, Math.round(ih * f));
        var lx = lit.getContext('2d');
        lx.imageSmoothingQuality = 'high';
        lx.drawImage(print, (nw - iw) / 2, (nh - ih) / 2, iw, ih, 0, 0, lit.width, lit.height);
        if (lightMap(H3, iw, ih)) {
          lx.globalCompositeOperation = 'multiply';
          lx.drawImage(shade, 0, 0, lit.width, lit.height);
          lx.globalCompositeOperation = 'source-over';
        }

        // Shadow, sized relative to the photo so it follows perspective.
        x.save();
        x.shadowColor = 'rgba(0,0,0,.4)';
        x.shadowBlur = dw * .02; x.shadowOffsetX = dw * .004; x.shadowOffsetY = dw * .01;
        x.fillStyle = '#000';
        x.beginPath(); x.moveTo(c00[0], c00[1]); x.lineTo(c10[0], c10[1]); x.lineTo(c11[0], c11[1]); x.lineTo(c01[0], c01[1]); x.closePath(); x.fill();
        x.restore();

        // Perspective warp: a mesh of small triangles, each an affine map of the baked photo.
        var N = 16, g = [];
        for (var j = 0; j <= N; j++) for (var i = 0; i <= N; i++) {
          var a = i * iw / N, b = j * ih / N, d = proj(a, b);
          g.push({ d: d, s: [a * f, b * f] });
        }
        var T = function (p, q, w) { tri(x, lit, p.d, q.d, w.d, p.s, q.s, w.s); };
        for (j = 0; j < N; j++) for (i = 0; i < N; i++) {
          var p00 = g[j * (N + 1) + i], p10 = g[j * (N + 1) + i + 1], p01 = g[(j + 1) * (N + 1) + i], p11 = g[(j + 1) * (N + 1) + i + 1];
          T(p00, p10, p01); T(p10, p11, p01);
        }
        x.setTransform(1, 0, 0, 1, 0, 0);
      }
      if (st.after) st.after();
    };

    function fit() {
      if (!st.W) return;
      st.fitW = box.clientWidth;
      // Contain: the whole room stays visible, scaled down (and centred) when it is taller than maxHeight().
      st.k = Math.min(box.clientWidth / st.W, st.maxHeight() / st.H);
      var cw = Math.round(st.W * st.k), ch = Math.round(st.H * st.k), dpr = Math.min(window.devicePixelRatio || 1, 2);
      stage.style.left = (box.clientWidth - cw) / 2 + 'px';
      stage.style.width = cw + 'px';
      stage.style.height = ch + 'px';
      cv.width = Math.round(cw * dpr); cv.height = Math.round(ch * dpr);
      cv.style.width = cw + 'px'; cv.style.height = ch + 'px';
      svg.setAttribute('viewBox', '0 0 ' + st.W + ' ' + st.H);
      box.style.height = ch + 'px';
      st.render();
    }

    // Album voting (see album_extra.js): a star for the current photo, only if the album has voting enabled.
    var star;
    function syncVote() {
      if (!window.voteState || !items.length) return;
      if (!star) {
        star = el('span', 'vote-pick', box);
        star.title = 'Pick as one of the best';
        star.addEventListener('click', function () { window.toggleVote(items[st.idx]); });
      }
      star.classList.toggle('picked', !!window.isVotePicked(items[st.idx]));
    }
    if (window.jQuery) window.jQuery(document).on('vote-changed', syncVote);

    st.fit = fit;
    st.maxHeight = function () { return window.innerHeight; };

    st.show = function (i) {
      if (!items.length) return;
      st.idx = (i + items.length) % items.length;
      print.src = base + '/1200w/' + items[st.idx] + '.jpg';
      caption.textContent = items.length > 1 ? (st.idx + 1) + ' / ' + items.length : '';
      syncVote();
    };

    print.addEventListener('load', function () {
      st.render();
      if (items.length > 1) new Image().src = base + '/1200w/' + items[(st.idx + 1) % items.length] + '.jpg';
    });

    room.addEventListener('load', function () {
      st.W = room.naturalWidth; st.H = room.naturalHeight; st.px = null;
      try { // a small copy is enough to see the light; fails for tainted images (another origin): ambient light is then skipped.
        st.pw = Math.min(st.W, 400); st.ph = Math.round(st.H * st.pw / st.W);
        var c = document.createElement('canvas'); c.width = st.pw; c.height = st.ph;
        var x = c.getContext('2d'); x.drawImage(room, 0, 0, st.pw, st.ph); st.px = x.getImageData(0, 0, st.pw, st.ph).data;
      } catch (e) {}
      fit();
    });
    // No need for a 2400w room (15MB decoded) on a phone-sized widget.
    room.src = base + (box.clientWidth * (window.devicePixelRatio || 1) > 1300 ? '/2400w/' : '/1200w/') + cfg.room + '.jpg';

    prev.addEventListener('click', function () { st.show(st.idx - 1); });
    next.addEventListener('click', function () { st.show(st.idx + 1); });
    container.addEventListener('keydown', function (e) {
      if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') return;
      if (e.key === 'ArrowLeft') st.show(st.idx - 1);
      if (e.key === 'ArrowRight') st.show(st.idx + 1);
    });
    // Only a width change matters: scrolling on mobile resizes the window height (address bar) all the time.
    var refit = function () { if (box.clientWidth !== st.fitW) fit(); };
    if (window.ResizeObserver) new ResizeObserver(refit).observe(box);
    window.addEventListener('resize', refit);

    st.show(0);
  }

  document.querySelectorAll('.room-preview').forEach(init);
})();
