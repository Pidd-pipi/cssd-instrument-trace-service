/**
 * 发放回收页（/issue）：发放登记（扫码）+ 回收扫码闭环 + 丢失待查。
 * 共用 IssueForm 组件与 StageBadge。
 */
window.CSSD = window.CSSD || {};
CSSD.pages = CSSD.pages || {};

CSSD.pages.issue = function (container) {
  container.innerHTML = '<div class="loading">加载发放回收数据...</div>';

  let openIssues = [];
  let lost = [];
  let resolvedPack = null;

  async function load() {
    try {
      const [o, l] = await Promise.all([
        CSSD.api.listIssues({ status: 'issued', limit: 200 }),
        CSSD.api.lostList()
      ]);
      openIssues = o || [];
      lost = l || [];
      // 丢失清单（/api/lost）内部会触发一次丢失扫描，可能把某条记录由
      // issued→lost。两条请求是并发的，listIssues 可能在标记前读到该记录，
      // 造成同一条同时落在「未回收发放记录」与「丢失待查」两列。
      // 以丢失清单为准去重：已进入丢失态的记录不再出现在未回收列表。
      const lostIds = new Set(lost.map((e) => e.issue && e.issue.id));
      openIssues = openIssues.filter((r) => !lostIds.has(r.id));
    } catch (e) {
      CSSD.toast.error('加载数据失败: ' + e.message);
    }
    render();
  }

  function render() {
    container.innerHTML = `
      <div class="section-title"><h2>发放回收</h2></div>

      <div class="grid cols-2">
        <div class="card">
          <h3>📤 发放登记（扫码）</h3>
          <div class="form-row">
            <label>器械包条码</label>
            <div style="display:flex;gap:8px">
              <input id="issue-barcode" placeholder="扫描或输入条码，如 PK20260825-001">
              <button class="btn" id="issue-lookup" style="width:auto">查询</button>
            </div>
          </div>
          <div id="issue-resolve" class="alert warn small">请先输入条码查询器械包（仅「已灭菌 + 未过期 + 批次合格」可发放）</div>
          <div id="issue-form-wrap"></div>
        </div>

        <div class="card">
          <h3>🧺 回收扫码（闭环）</h3>
          <div class="form-row">
            <label>器械包条码</label>
            <div style="display:flex;gap:8px">
              <input id="collect-barcode" placeholder="扫描使用完毕的器械包条码">
              <button class="btn primary" id="collect-submit" style="width:auto">回收</button>
            </div>
          </div>
          <div class="alert small muted">回收要求器械包处于「使用中」且存在未回收发放记录，回收后回到「待回收」。</div>
        </div>
      </div>

      <div class="grid cols-2">
        <div class="card">
          <h3>未回收发放记录（${openIssues.length}）</h3>
          ${openIssues.length === 0 ? '<div class="empty">暂无未回收发放记录</div>' : `
          <table><thead><tr><th>条码</th><th>科室</th><th>手术间</th><th>发放人</th><th>发放时间</th><th>操作</th></tr></thead><tbody>
          ${openIssues.map((r) => `
            <tr>
              <td class="mono">${CSSD.esc(r.barcode)}</td>
              <td>${CSSD.esc(r.department)}</td>
              <td>${CSSD.esc(r.operatingRoom)}</td>
              <td>${CSSD.esc(r.issuer)}</td>
              <td class="small">${CSSD.esc(CSSD.fmtTime(r.issuedAt))}</td>
              <td><button class="btn small" data-act="inuse" data-id="${CSSD.esc(r.packId)}">标记使用中</button></td>
            </tr>`).join('')}
          </tbody></table>`}
        </div>

        <div class="card">
          <h3>⏰ 丢失待查（${lost.length}）</h3>
          ${lost.length === 0 ? '<div class="empty">暂无丢失待查器械包</div>' : `
          <table><thead><tr><th>条码</th><th>科室</th><th>手术间</th><th>发放时间</th><th>超时</th></tr></thead><tbody>
          ${lost.map((e) => `
            <tr>
              <td class="mono">${CSSD.esc(e.pack.barcode)}</td>
              <td>${CSSD.esc(e.issue.department)}</td>
              <td>${CSSD.esc(e.issue.operatingRoom)}</td>
              <td class="small">${CSSD.esc(CSSD.fmtTime(e.issue.issuedAt))}</td>
              <td class="small" style="color:var(--danger)">${Math.round(e.overdueHours)}h</td>
            </tr>`).join('')}
          </tbody></table>`}
        </div>
      </div>
    `;

    container.querySelector('#issue-lookup').addEventListener('click', lookupPack);
    container.querySelector('#issue-barcode').addEventListener('keydown', (e) => { if (e.key === 'Enter') lookupPack(); });
    container.querySelector('#collect-submit').addEventListener('click', doCollect);
    container.querySelector('#collect-barcode').addEventListener('keydown', (e) => { if (e.key === 'Enter') doCollect(); });
    container.querySelectorAll('button[data-act="inuse"]').forEach((btn) => {
      btn.addEventListener('click', () => markInUse(btn.dataset.id));
    });
  }

  async function lookupPack() {
    const barcode = container.querySelector('#issue-barcode').value.trim();
    const box = container.querySelector('#issue-resolve');
    const wrap = container.querySelector('#issue-form-wrap');
    wrap.innerHTML = '';
    if (!barcode) { box.className = 'alert warn small'; box.textContent = '请输入条码'; return; }
    try {
      const view = await CSSD.api.traceByBarcode(barcode);
      const p = view.pack;
      resolvedPack = p;
      if (p.stage === 'sterilized' && p.lastBatchResult === 'pass') {
        const expired = p.expiryAt && new Date(p.expiryAt).getTime() < Date.now();
        if (expired) {
          box.className = 'alert error small';
          box.textContent = '⛔ 器械包已过期，禁止发放（需重新清洗灭菌）';
          return;
        }
        box.className = 'alert success small';
        box.innerHTML = '✅ 可发放：' + CSSD.esc(p.barcode) + '（' + CSSD.esc(p.name) + '），有效期至 ' + CSSD.esc(CSSD.fmtTime(p.expiryAt));
        wrap.appendChild(CSSD.IssueForm(p, { onSuccess: () => load() }));
      } else {
        box.className = 'alert error small';
        box.textContent = '⛔ 器械包当前环节为「' + (CSSD.constants.PackStage[p.stage] || { label: p.stage }).label + '」、灭菌结果「' + (p.lastBatchResult === 'pass' ? '合格' : p.lastBatchResult === 'fail' ? '不合格' : '未判定') + '」，不可发放';
      }
    } catch (e) {
      box.className = 'alert error small';
      box.textContent = '查询失败：' + e.message;
    }
  }

  async function doCollect() {
    const barcode = container.querySelector('#collect-barcode').value.trim();
    if (!barcode) { CSSD.toast.error('请输入回收条码'); return; }
    try {
      const view = await CSSD.api.traceByBarcode(barcode);
      const p = view.pack;
      const record = await CSSD.api.collectPack(p.id, { operator: '回收员', collector: '回收员' });
      CSSD.toast.success('回收闭环成功：' + barcode);
      container.querySelector('#collect-barcode').value = '';
      load();
    } catch (e) {
      CSSD.toast.error(e.message);
    }
  }

  async function markInUse(packId) {
    try {
      const pack = await CSSD.api.cyclePack(packId, { stage: 'in_use', operator: '手术室' });
      CSSD.toast.success('已标记使用中：' + pack.barcode);
      load();
    } catch (e) {
      CSSD.toast.error(e.message);
    }
  }

  load();
};
