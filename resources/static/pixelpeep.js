(function () {
  // A pixelpeep instance is any ".pixelpeep" element carrying its config as
  // JSON, either in a <script type="application/json"> child (pixelpeep CLI)
  // or a nested code fence (the `:::{.pixelpeep}` markdown widget):
  // {height, zoom, offsetX, offsetY, items: [{url, label, zoom, offsetX, offsetY}, ...]}.
  // Each instance gets its own group/drag state, so multiple widgets can live
  // on one page.
  //
  // Two-layer transform per image: a shared "group" pan/zoom (initial value
  // from the top-level zoom/offsetX/offsetY, and what "Reset zoom" returns
  // to), and a per-image "individual" correction nested inside it for
  // aligning imperfectly-framed shots (or a configured base zoom
  // multiplier). Locked images follow the group; unlocked ones are
  // dragged/zoomed independently, and the correction is kept when re-locked.

  function readConfig(container) {
    var script = container.querySelector('script[type="application/json"]');
    var text = script ? script.textContent : ((container.querySelector('pre code') || {}).textContent);
    if (!text) return null;

    try {
      return JSON.parse(text);
    } catch (e) {
      console.error('pixelpeep: invalid JSON config', e);
      return null;
    }
  }

  function buildDOM(container, images) {
    var toolbar = document.createElement('div');
    toolbar.className = 'toolbar';
    var resetBtn = document.createElement('button');
    resetBtn.textContent = 'Reset zoom';
    toolbar.appendChild(resetBtn);
    var zoomOutBtn = document.createElement('button');
    zoomOutBtn.textContent = '−';
    zoomOutBtn.title = 'Zoom out';
    toolbar.appendChild(zoomOutBtn);
    var zoomInBtn = document.createElement('button');
    zoomInBtn.textContent = '+';
    zoomInBtn.title = 'Zoom in';
    toolbar.appendChild(zoomInBtn);
    var zoomLabel = document.createElement('span');
    zoomLabel.className = 'zoom-pct';
    toolbar.appendChild(zoomLabel);

    var row = document.createElement('div');
    row.className = 'row';

    container.innerHTML = '';
    container.appendChild(toolbar);
    container.appendChild(row);

    var items = images.map(function (image) {
      var vp = document.createElement('div');
      vp.className = 'vp';

      var label = document.createElement('span');
      label.className = 'label';
      label.textContent = image.label || (image.url || '').split('/').pop();
      vp.appendChild(label);

      var lockBtn = document.createElement('button');
      lockBtn.className = 'lock';
      lockBtn.title = 'Lock/unlock this image to the group';
      lockBtn.textContent = '🔒';
      vp.appendChild(lockBtn);

      var group = document.createElement('div');
      group.className = 'group';
      var img = document.createElement('img');
      img.src = image.url;
      group.appendChild(img);
      vp.appendChild(group);

      row.appendChild(vp);

      return {
        vp: vp, group: group, img: img, labelEl: label, lockBtn: lockBtn, locked: true,
        ind: { x: 0, y: 0, scale: image.zoom || 1 },
        offset: { x: image.offsetX || 0, y: image.offsetY || 0 }
      };
    });

    return { resetBtn: resetBtn, zoomOutBtn: zoomOutBtn, zoomInBtn: zoomInBtn, zoomLabel: zoomLabel, items: items };
  }

  function clampScale(s) {
    return Math.min(20, Math.max(0.1, s));
  }

  function init(container) {
    var config = readConfig(container);
    if (!config || !Array.isArray(config.items) || config.items.length < 2) return;

    // Default (no explicit height): the container fills the viewport height
    // (100vh) and each .vp stretches to fill it via flexbox — .vp has no
    // in-flow content (image/label/lock are all absolutely positioned), so it
    // only gets a height by stretching from a definite-height ancestor.
    // Explicit height: the container instead sizes to its content (toolbar +
    // the fixed-height .vp row), like a normal in-page block.
    function setHeight(h) {
      if (h) {
        container.style.setProperty('--vp-height', h + 'px');
        container.style.setProperty('--pp-height', 'auto');
      } else {
        container.style.removeProperty('--vp-height');
        container.style.removeProperty('--pp-height');
      }
    }
    setHeight(config.height);

    var built = buildDOM(container, config.items);
    var items = built.items;
    var initialGroup = { x: config.offsetX || 0, y: config.offsetY || 0, scale: config.zoom || 1 };
    var group = { x: initialGroup.x, y: initialGroup.y, scale: initialGroup.scale };

    function applyGroup() {
      var t = 'translate(' + group.x + 'px,' + group.y + 'px) scale(' + group.scale + ')';
      items.forEach(function (it) { it.group.style.transform = t; });
      built.zoomLabel.textContent = Math.round(group.scale * 100) + '%';
    }

    function applyIndividual(it) {
      it.img.style.transform = 'translate(' + it.ind.x + 'px,' + it.ind.y + 'px) scale(' + it.ind.scale + ')';
    }

    // Rescales the group (or, when unlocked, just this image) so that the
    // point (px, py) — in the viewport's own pixel space — stays under the
    // cursor/fingers.
    function zoomAt(it, px, py, factor) {
      if (it.locked) {
        var ns = clampScale(group.scale * factor);
        group.x = px - ns * (px - group.x) / group.scale;
        group.y = py - ns * (py - group.y) / group.scale;
        group.scale = ns;
        applyGroup();
      } else {
        // Reference point in the group's local space, since the individual
        // layer is nested inside the group's transform.
        var gx = (px - group.x) / group.scale, gy = (py - group.y) / group.scale;
        var nis = clampScale(it.ind.scale * factor);
        it.ind.x = gx - nis * (gx - it.ind.x) / it.ind.scale;
        it.ind.y = gy - nis * (gy - it.ind.y) / it.ind.scale;
        it.ind.scale = nis;
        applyIndividual(it);
      }
    }

    // Used by the toolbar +/- buttons, which have no single viewport or
    // pointer position to anchor to: zooms the group around the center of
    // the first viewport.
    function zoomGroupBy(factor) {
      var rect = items[0].vp.getBoundingClientRect();
      var ns = clampScale(group.scale * factor);
      var px = rect.width / 2, py = rect.height / 2;
      group.x = px - ns * (px - group.x) / group.scale;
      group.y = py - ns * (py - group.y) / group.scale;
      group.scale = ns;
      applyGroup();
    }

    // The position an image would sit at with offset (0,0): centered in its
    // viewport at its current individual scale. offsetX/offsetY, in and out,
    // are always relative to this baseline, so they stay meaningful
    // regardless of viewport size or zoom multiplier.
    function centerBaseline(it) {
      var w = it.vp.clientWidth, h = it.vp.clientHeight;
      return {
        x: (w - it.img.naturalWidth * it.ind.scale) / 2,
        y: (h - it.img.naturalHeight * it.ind.scale) / 2
      };
    }

    function center(it) {
      var base = centerBaseline(it);
      it.ind.x = base.x + it.offset.x;
      it.ind.y = base.y + it.offset.y;
      applyIndividual(it);
    }

    items.forEach(function (it) {
      it.img.addEventListener('load', function () { center(it); });
      if (it.img.complete) center(it);
    });
    applyGroup();

    built.resetBtn.addEventListener('click', function () {
      group.x = initialGroup.x; group.y = initialGroup.y; group.scale = initialGroup.scale;
      applyGroup();
    });
    built.zoomOutBtn.addEventListener('click', function () { zoomGroupBy(1 / 1.2); });
    built.zoomInBtn.addEventListener('click', function () { zoomGroupBy(1.2); });

    items.forEach(function (it) {
      it.lockBtn.addEventListener('pointerdown', function (e) { e.stopPropagation(); });
      it.lockBtn.addEventListener('click', function () {
        it.locked = !it.locked;
        it.vp.classList.toggle('unlocked', !it.locked);
        it.lockBtn.textContent = it.locked ? '🔒' : '🔓';
      });

      // Pointer Events unify mouse/touch/pen. Everything is scoped to this
      // viewport (via setPointerCapture, so a drag keeps tracking even once
      // the pointer leaves the element) — no shared/global drag state needed.
      var pointers = {}; // pointerId -> {x, y}, in this viewport's own pixel space
      var drag = null; // single-pointer pan: {pointerId, target, x0, y0, startX, startY, divisor}
      var pinch = null; // two-pointer pinch: {dist, midX, midY}

      function vpPoint(e) {
        var rect = it.vp.getBoundingClientRect();
        return { x: e.clientX - rect.left, y: e.clientY - rect.top };
      }

      function pinchState() {
        var ids = Object.keys(pointers);
        if (ids.length < 2) return null;
        var a = pointers[ids[0]], b = pointers[ids[1]];
        return {
          dist: Math.hypot(a.x - b.x, a.y - b.y),
          midX: (a.x + b.x) / 2, midY: (a.y + b.y) / 2
        };
      }

      it.vp.addEventListener('pointerdown', function (e) {
        e.preventDefault();
        it.vp.setPointerCapture(e.pointerId);
        pointers[e.pointerId] = vpPoint(e);

        if (Object.keys(pointers).length >= 2) {
          drag = null;
          pinch = pinchState();
          return;
        }

        var target = it.locked ? group : it.ind;
        drag = {
          pointerId: e.pointerId, target: target,
          startX: e.clientX, startY: e.clientY,
          x0: target.x, y0: target.y,
          divisor: it.locked ? 1 : group.scale
        };
        it.vp.style.cursor = 'grabbing';
      });

      it.vp.addEventListener('pointermove', function (e) {
        if (!(e.pointerId in pointers)) return;
        pointers[e.pointerId] = vpPoint(e);

        if (pinch) {
          var next = pinchState();
          if (next) {
            // Rescale anchored at the previous midpoint, then pan by how much
            // the midpoint itself moved, so pinch-zoom and two-finger pan
            // compose naturally in one gesture.
            zoomAt(it, pinch.midX, pinch.midY, next.dist / pinch.dist);
            var target = it.locked ? group : it.ind;
            var dx = next.midX - pinch.midX, dy = next.midY - pinch.midY;
            if (it.locked) {
              target.x += dx; target.y += dy;
              applyGroup();
            } else {
              target.x += dx / group.scale; target.y += dy / group.scale;
              applyIndividual(it);
            }
            pinch = next;
          }
          return;
        }

        if (!drag || e.pointerId !== drag.pointerId) return;
        var ddx = (e.clientX - drag.startX) / drag.divisor;
        var ddy = (e.clientY - drag.startY) / drag.divisor;
        drag.target.x = drag.x0 + ddx;
        drag.target.y = drag.y0 + ddy;
        if (drag.target === group) applyGroup(); else applyIndividual(it);
      });

      function endPointer(e) {
        delete pointers[e.pointerId];
        if (Object.keys(pointers).length < 2) pinch = null;
        if (drag && e.pointerId === drag.pointerId) {
          drag = null;
          it.vp.style.cursor = 'grab';
        }
      }
      it.vp.addEventListener('pointerup', endPointer);
      it.vp.addEventListener('pointercancel', endPointer);

      it.vp.addEventListener('wheel', function (e) {
        e.preventDefault();
        var p = vpPoint(e);
        // Proportional to deltaY (clamped) so a fine-grained trackpad pinch
        // zooms in small steps, not the same jump as a full mouse-wheel notch.
        var dy = Math.max(-100, Math.min(100, e.deltaY));
        zoomAt(it, p.x, p.y, Math.exp(-dy * 0.0015));
      }, { passive: false });
    });

    // Exposed for an optional external editor UI (see pixelpeep-editor.js).
    container.pixelpeepState = {
      items: items, centerBaseline: centerBaseline, center: center,
      group: group, initialGroup: initialGroup, applyGroup: applyGroup,
      height: config.height || 0, setHeight: setHeight
    };
  }

  document.querySelectorAll('.pixelpeep').forEach(init);
})();
