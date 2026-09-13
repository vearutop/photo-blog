(function () {
  var container = document.querySelector('.pixelpeep');
  if (!container || !container.pixelpeepState) return;

  var state = container.pixelpeepState;

  var toggleBtn = document.createElement('button');
  toggleBtn.className = 'pp-editor-toggle';
  toggleBtn.textContent = 'Hide editor';
  document.body.appendChild(toggleBtn);

  var editor = document.createElement('div');
  editor.className = 'pixelpeep-editor';

  toggleBtn.addEventListener('click', function () {
    var hidden = editor.classList.toggle('hidden');
    toggleBtn.textContent = hidden ? 'Show editor' : 'Hide editor';
  });

  function field(row, name, value, size) {
    var input = document.createElement('input');
    input.name = name;
    input.value = value;
    input.type = 'number';
    if (size) input.size = size;
    row.appendChild(input);
    return input;
  }

  // Viewport row: global settings — viewport height (blank = fill the
  // screen), and the initial shared pan/zoom that "Reset zoom" returns to.
  var vpRow = document.createElement('div');
  vpRow.className = 'pp-row';
  var vpLabel = document.createElement('span');
  vpLabel.className = 'pp-index';
  vpLabel.textContent = 'View';
  vpRow.appendChild(vpLabel);

  var vpHeight = field(vpRow, 'height', state.height || '', 5);
  vpHeight.placeholder = 'full';
  var vpZoom = field(vpRow, 'zoom', state.group.scale, 5);
  vpZoom.step = '0.01';
  var vpOffsetX = field(vpRow, 'offsetX', state.group.x, 6);
  var vpOffsetY = field(vpRow, 'offsetY', state.group.y, 6);
  editor.appendChild(vpRow);

  var rows = state.items.map(function (it, i) {
    var row = document.createElement('div');
    row.className = 'pp-row';

    var index = document.createElement('span');
    index.className = 'pp-index';
    index.textContent = (i + 1) + '.';
    row.appendChild(index);

    var label = document.createElement('input');
    label.name = 'label';
    label.value = it.labelEl.textContent;
    label.size = 14;
    row.appendChild(label);

    var zoom = field(row, 'zoom', it.ind.scale, 5);
    zoom.step = '0.01';
    var offsetX = field(row, 'offsetX', it.offset.x, 6);
    var offsetY = field(row, 'offsetY', it.offset.y, 6);

    editor.appendChild(row);

    return { it: it, label: label, zoom: zoom, offsetX: offsetX, offsetY: offsetY };
  });

  var buttons = document.createElement('div');
  buttons.className = 'pp-buttons';

  function button(text) {
    var b = document.createElement('button');
    b.textContent = text;
    buttons.appendChild(b);
    return b;
  }

  var applyBtn = button('Apply to view');
  var fromViewBtn = button('Apply from view');
  var copyBtn = button('Copy as JSON');
  var copyLinkBtn = button('Copy link');
  var pasteBtn = button('Paste from JSON');
  editor.appendChild(buttons);

  function round2(n) {
    return Math.round(n * 100) / 100;
  }

  applyBtn.addEventListener('click', function () {
    state.height = parseFloat(vpHeight.value) || 0;
    state.setHeight(state.height);
    state.group.scale = state.initialGroup.scale = parseFloat(vpZoom.value) || 1;
    state.group.x = state.initialGroup.x = parseFloat(vpOffsetX.value) || 0;
    state.group.y = state.initialGroup.y = parseFloat(vpOffsetY.value) || 0;
    state.applyGroup();

    rows.forEach(function (r) {
      var it = r.it;
      it.labelEl.textContent = r.label.value;
      it.ind.scale = parseFloat(r.zoom.value) || 1;
      it.offset = { x: parseFloat(r.offsetX.value) || 0, y: parseFloat(r.offsetY.value) || 0 };
      state.center(it);
    });
  });

  // Reads back the relative zoom/offset produced by manually dragging and
  // zooming the group and unlocked images, so it can be copied out.
  fromViewBtn.addEventListener('click', function () {
    vpZoom.value = state.group.scale;
    vpOffsetX.value = state.group.x;
    vpOffsetY.value = state.group.y;

    rows.forEach(function (r) {
      var it = r.it;
      var base = state.centerBaseline(it);
      r.zoom.value = it.ind.scale;
      r.offsetX.value = it.ind.x - base.x;
      r.offsetY.value = it.ind.y - base.y;
    });
  });

  // Builds the config object from the current form values — the single
  // source of truth for both "Copy as JSON" and "Copy link".
  function buildConfig() {
    var items = rows.map(function (r) {
      var zoom = parseFloat(r.zoom.value) || 1;
      var offsetX = parseFloat(r.offsetX.value) || 0;
      var offsetY = parseFloat(r.offsetY.value) || 0;

      var entry = { url: r.it.img.getAttribute('src'), label: r.label.value };
      if (zoom !== 1) entry.zoom = round2(zoom);
      if (offsetX) entry.offsetX = round2(offsetX);
      if (offsetY) entry.offsetY = round2(offsetY);

      return entry;
    });

    var height = parseFloat(vpHeight.value) || 0;
    var zoom = parseFloat(vpZoom.value) || 1;
    var offsetX = parseFloat(vpOffsetX.value) || 0;
    var offsetY = parseFloat(vpOffsetY.value) || 0;

    var out = {};
    if (height) out.height = height;
    if (zoom !== 1) out.zoom = round2(zoom);
    if (offsetX) out.offsetX = round2(offsetX);
    if (offsetY) out.offsetY = round2(offsetY);
    out.items = items;

    return out;
  }

  function flash(btn, text) {
    var original = btn.textContent;
    btn.textContent = text;
    setTimeout(function () { btn.textContent = original; }, 1200);
  }

  // navigator.clipboard needs a secure context (HTTPS or localhost) and is
  // simply undefined otherwise (e.g. plain http://); fall back to the old
  // execCommand trick via a throwaway textarea, which works anywhere.
  function copyText(text) {
    if (navigator.clipboard) return navigator.clipboard.writeText(text);

    var ta = document.createElement('textarea');
    ta.value = text;
    ta.style.position = 'fixed';
    ta.style.left = '-9999px';
    document.body.appendChild(ta);
    ta.focus();
    ta.select();
    try {
      document.execCommand('copy');
    } finally {
      document.body.removeChild(ta);
    }

    return Promise.resolve();
  }

  copyBtn.addEventListener('click', function () {
    var json = JSON.stringify(buildConfig(), null, 2);

    copyText(json).then(function () {
      flash(copyBtn, 'Copied!');
    });
  });

  // Points back at this same page's base path (the CLI's own "/", or a
  // dedicated share route the webapp sets via data-share-base) with a
  // "?config=" override, so anyone who can reach this server gets the exact
  // same framing without needing the JSON pasted in manually.
  copyLinkBtn.addEventListener('click', function () {
    var base = container.dataset.shareBase || location.pathname;
    var link = location.origin + base + '?config=' + encodeURIComponent(JSON.stringify(buildConfig()));

    copyText(link).then(function () {
      flash(copyLinkBtn, 'Copied!');
    });
  });

  // Fills the form from a pasted JSON config (e.g. one earlier "Copy as
  // JSON" output), matched to rows by index. Only fills the form; click
  // "Apply to view" afterwards to see it in the viewports.
  pasteBtn.addEventListener('click', function () {
    var text = prompt('Paste pixelpeep JSON config:');
    if (!text) return;

    var cfg;
    try {
      cfg = JSON.parse(text);
    } catch (e) {
      alert('Invalid JSON: ' + e.message);
      return;
    }

    var arr = cfg && cfg.items;
    if (!Array.isArray(arr)) {
      alert('Expected {"items": [...]}.');
      return;
    }

    vpHeight.value = cfg.height || '';
    vpZoom.value = cfg.zoom != null ? cfg.zoom : 1;
    vpOffsetX.value = cfg.offsetX != null ? cfg.offsetX : 0;
    vpOffsetY.value = cfg.offsetY != null ? cfg.offsetY : 0;

    rows.forEach(function (r, i) {
      var entry = arr[i];
      if (!entry) return;

      if (entry.label != null) r.label.value = entry.label;
      r.zoom.value = entry.zoom != null ? entry.zoom : 1;
      r.offsetX.value = entry.offsetX != null ? entry.offsetX : 0;
      r.offsetY.value = entry.offsetY != null ? entry.offsetY : 0;
    });

    if (arr.length !== rows.length) {
      alert('Pasted ' + arr.length + ' entries, editor has ' + rows.length + ' — matched by position, extra entries ignored.');
    }
  });

  container.parentNode.insertBefore(editor, container.nextSibling);
})();
