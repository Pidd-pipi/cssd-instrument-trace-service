/**
 * 前端 API 层：统一封装 fetch，处理统一响应格式 {code, message, data}。
 */
window.CSSD = window.CSSD || {};

CSSD.esc = function (value) {
  if (value === null || value === undefined) return '';
  return String(value)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
};

CSSD.fmtTime = function (iso) {
  if (!iso) return '-';
  const d = new Date(iso);
  if (isNaN(d.getTime())) return String(iso);
  const pad = (n) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
};

CSSD.api = (function () {
  async function request(method, url, body) {
    const opts = { method: method, headers: { 'Content-Type': 'application/json' } };
    if (body !== undefined && body !== null) opts.body = JSON.stringify(body);
    let resp;
    try {
      resp = await fetch(url, opts);
    } catch (e) {
      throw new Error('网络请求失败: ' + e.message);
    }
    let data = null;
    try { data = await resp.json(); } catch (e) { /* 非 JSON 响应 */ }
    if (!resp.ok || (data && data.code !== 0)) {
      const msg = (data && data.message) ? data.message : ('HTTP ' + resp.status);
      const err = new Error(msg);
      err.status = resp.status;
      err.data = data;
      throw err;
    }
    return data ? data.data : null;
  }

  function qs(params) {
    const url = new URLSearchParams();
    Object.keys(params || {}).forEach((k) => {
      const v = params[k];
      if (v !== undefined && v !== null && v !== '') url.set(k, v);
    });
    const s = url.toString();
    return s ? '?' + s : '';
  }

  return {
    get: (url) => request('GET', url),
    post: (url, body) => request('POST', url, body || {}),
    // 器械包
    createPack: (payload) => request('POST', '/api/packs', payload),
    listPacks: (filter) => request('GET', '/api/packs' + qs(filter)),
    getPack: (id) => request('GET', '/api/packs/' + encodeURIComponent(id)),
    cyclePack: (id, payload) => request('POST', '/api/packs/' + encodeURIComponent(id) + '/cycle', payload),
    issuePack: (id, payload) => request('POST', '/api/packs/' + encodeURIComponent(id) + '/issue', payload),
    collectPack: (id, payload) => request('POST', '/api/packs/' + encodeURIComponent(id) + '/collect', payload),
    packTrace: (id) => request('GET', '/api/packs/' + encodeURIComponent(id) + '/trace'),
    // 灭菌
    createSterilization: (payload) => request('POST', '/api/sterilizations', payload),
    listSterilizations: (limit) => request('GET', '/api/sterilizations' + qs({ limit: limit || 100 })),
    getSterilization: (id) => request('GET', '/api/sterilizations/' + encodeURIComponent(id)),
    completeSterilization: (id, payload) => request('POST', '/api/sterilizations/' + encodeURIComponent(id) + '/complete', payload),
    batchPacks: (id) => request('GET', '/api/sterilizations/' + encodeURIComponent(id) + '/packs'),
    listSterilizers: () => request('GET', '/api/sterilizers'),
    createSterilizer: (payload) => request('POST', '/api/sterilizers', payload),
    // 发放回收 / 追溯 / 总览 / 审计
    listIssues: (filter) => request('GET', '/api/issues' + qs(filter)),
    lostList: () => request('GET', '/api/lost'),
    traceByBarcode: (barcode) => request('GET', '/api/trace' + qs({ barcode: barcode })),
    dashboard: () => request('GET', '/api/dashboard'),
    auditLogs: (limit) => request('GET', '/api/audit-logs' + qs({ limit: limit || 50 })),
    healthz: () => request('GET', '/healthz')
  };
})();

CSSD.toast = (function () {
  function show(message, type) {
    const root = document.getElementById('toast-root');
    if (!root) return;
    const el = document.createElement('div');
    el.className = 'toast ' + (type || '');
    el.textContent = message;
    root.appendChild(el);
    setTimeout(() => {
      el.style.opacity = '0';
      el.style.transition = 'opacity 0.3s';
      setTimeout(() => el.remove(), 320);
    }, 3200);
  }
  return { success: (m) => show(m, 'success'), error: (m) => show(m, 'error'), info: (m) => show(m, '') };
})();

CSSD.modal = (function () {
  function open(contentEl) {
    const root = document.getElementById('modal-root');
    if (!root) return;
    root.innerHTML = '';
    const mask = document.createElement('div');
    mask.className = 'modal-mask';
    const modal = document.createElement('div');
    modal.className = 'modal';
    modal.appendChild(contentEl);
    mask.appendChild(modal);
    mask.addEventListener('click', (e) => {
      if (e.target === mask) close();
    });
    root.appendChild(mask);
    return { close: close };
  }
  function close() {
    const root = document.getElementById('modal-root');
    if (root) root.innerHTML = '';
  }
  function build(title, bodyEl) {
    const wrap = document.createElement('div');
    const head = document.createElement('div');
    head.style.display = 'flex';
    head.style.justifyContent = 'space-between';
    head.style.alignItems = 'center';
    const h = document.createElement('h3');
    h.textContent = title || '';
    const btn = document.createElement('button');
    btn.className = 'modal-close';
    btn.textContent = '✕';
    btn.addEventListener('click', close);
    head.appendChild(h);
    head.appendChild(btn);
    wrap.appendChild(head);
    wrap.appendChild(bodyEl);
    return wrap;
  }
  return { open: open, close: close, build: build };
})();
