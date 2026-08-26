/**
 * 追溯查询页（/trace）：按条码/器械包追溯完整循环，按灭菌批次查询器械包去向。
 * 共用 CycleTimeline 组件与 useTrace Hook。
 */
window.CSSD = window.CSSD || {};
CSSD.pages = CSSD.pages || {};

CSSD.pages.trace = function (container) {
  container.innerHTML = `
    <div class="section-title"><h2>追溯查询</h2></div>
    <div class="card">
      <div class="tabs">
        <span class="tab active" data-tab="pack">按器械包追溯</span>
        <span class="tab" data-tab="batch">按灭菌批次查询</span>
      </div>
      <div id="trace-panel"></div>
    </div>
  `;

  container.querySelectorAll('.tab').forEach((tab) => {
    tab.addEventListener('click', () => {
      container.querySelectorAll('.tab').forEach((t) => t.classList.remove('active'));
      tab.classList.add('active');
      if (tab.dataset.tab === 'pack') renderPackPanel();
      else renderBatchPanel();
    });
  });
  renderPackPanel();

  function renderPackPanel() {
    const panel = container.querySelector('#trace-panel');
    panel.innerHTML = `
      <div class="form-row">
        <label>器械包条码或 ID</label>
        <div style="display:flex;gap:8px">
          <input id="trace-keyword" placeholder="输入条码，如 PK20260825-001">
          <button class="btn primary" id="trace-query" style="width:auto">追溯</button>
        </div>
      </div>
      <div id="trace-result"><div class="empty">输入条码或 ID 后点击追溯，可查看完整循环留痕</div></div>
    `;
    panel.querySelector('#trace-query').addEventListener('click', doPackTrace);
    panel.querySelector('#trace-keyword').addEventListener('keydown', (e) => { if (e.key === 'Enter') doPackTrace(); });
  }

  async function doPackTrace() {
    const kw = container.querySelector('#trace-keyword').value.trim();
    const result = container.querySelector('#trace-result');
    if (!kw) { CSSD.toast.error('请输入条码或 ID'); return; }
    result.innerHTML = '<div class="loading">追溯中...</div>';
    try {
      let view;
      if (kw.startsWith('pack_')) {
        view = await CSSD.api.packTrace(kw);
      } else {
        view = await CSSD.api.traceByBarcode(kw);
      }
      const p = view.pack;
      result.innerHTML = `
        <div class="kv">
          <div class="k">条码</div><div class="mono">${CSSD.esc(p.barcode)}</div>
          <div class="k">名称</div><div>${CSSD.esc(p.name)}</div>
          <div class="k">类型</div><div>${CSSD.esc(CSSD.constants.PackType[p.packType] || p.packType)}</div>
          <div class="k">当前环节</div><div>${CSSD.StageBadge(p.stage)}</div>
          <div class="k">有效期至</div><div class="small">${CSSD.esc(CSSD.fmtTime(p.expiryAt))}</div>
          <div class="k">灭菌批次</div><div class="small">${CSSD.esc((view.lastBatch && view.lastBatch.batchNo) || '-')} ${CSSD.ResultBadge(p.lastBatchResult)}</div>
          <div class="k">登记时间</div><div class="small">${CSSD.esc(CSSD.fmtTime(p.createdAt))}</div>
        </div>
        <h3>完整循环时间线（${view.cycles.length} 条环节记录）</h3>
        <div id="trace-timeline"></div>
        <h3>发放回收记录</h3>
        <div id="trace-issues"></div>
      `;
      result.querySelector('#trace-timeline').appendChild(CSSD.CycleTimeline(view.cycles));
      result.querySelector('#trace-issues').innerHTML = renderIssues(view.issues || []);
    } catch (e) {
      result.innerHTML = '<div class="alert error">' + CSSD.esc(e.message) + '</div>';
    }
  }

  function renderIssues(issues) {
    if (!issues.length) return '<div class="empty">暂无发放记录</div>';
    return `<table><thead><tr><th>科室</th><th>手术间</th><th>发放人</th><th>发放时间</th><th>回收人</th><th>回收时间</th><th>状态</th></tr></thead><tbody>
      ${issues.map((r) => `
        <tr>
          <td>${CSSD.esc(r.department)}</td>
          <td>${CSSD.esc(r.operatingRoom)}</td>
          <td>${CSSD.esc(r.issuer)}</td>
          <td class="small">${CSSD.esc(CSSD.fmtTime(r.issuedAt))}</td>
          <td class="small">${CSSD.esc(r.collector || '-')}</td>
          <td class="small">${CSSD.esc(CSSD.fmtTime(r.collectedAt))}</td>
          <td>${CSSD.IssueStatusBadge(r.status)}</td>
        </tr>`).join('')}
    </tbody></table>`;
  }

  function renderBatchPanel() {
    const panel = container.querySelector('#trace-panel');
    panel.innerHTML = `
      <div class="form-row">
        <label>灭菌批次号或 ID</label>
        <div style="display:flex;gap:8px">
          <input id="batch-keyword" placeholder="输入批次号，如 SB20260825-001">
          <button class="btn primary" id="batch-query" style="width:auto">查询去向</button>
        </div>
      </div>
      <div id="batch-result"><div class="empty">输入批次号后点击查询，可查看该批全部器械包去向</div></div>
    `;
    panel.querySelector('#batch-query').addEventListener('click', doBatchTrace);
    panel.querySelector('#batch-keyword').addEventListener('keydown', (e) => { if (e.key === 'Enter') doBatchTrace(); });
  }

  async function doBatchTrace() {
    const kw = container.querySelector('#batch-keyword').value.trim();
    const result = container.querySelector('#batch-result');
    if (!kw) { CSSD.toast.error('请输入批次号或 ID'); return; }
    result.innerHTML = '<div class="loading">查询中...</div>';
    try {
      // 支持批次号（SB20260825-xxx）或内部批次 ID（batch_xxx）：批次号先解析为 ID。
      let batchId = kw;
      if (!kw.startsWith('batch_')) {
        const list = await CSSD.api.listSterilizations(500);
        const hit = (list || []).find((b) => b.batchNo === kw);
        if (!hit) throw new Error('未找到批次 ' + kw);
        batchId = hit.id;
      }
      const view = await CSSD.api.batchPacks(batchId);
      const b = view.batch;
      result.innerHTML = `
        <div class="kv">
          <div class="k">批次号</div><div class="mono">${CSSD.esc(b.batchNo)}</div>
          <div class="k">灭菌器</div><div>${CSSD.esc(b.sterilizerName)}</div>
          <div class="k">参数</div><div class="mono">${CSSD.esc(b.tempC)}℃ / ${CSSD.esc(b.durationMin)}min / ${CSSD.esc(b.pressureKPa)}kPa</div>
          <div class="k">判定结果</div><div>${CSSD.ResultBadge(b.result)}</div>
        </div>
        <table style="margin-top:12px"><thead><tr><th>条码</th><th>名称</th><th>当前环节</th><th>有效期至</th><th>最新去向</th></tr></thead><tbody>
        ${(view.packs || []).map((item) => `
          <tr>
            <td class="mono">${CSSD.esc(item.pack.barcode)}</td>
            <td>${CSSD.esc(item.pack.name)}</td>
            <td>${CSSD.StageBadge(item.pack.stage)}</td>
            <td class="small">${CSSD.esc(CSSD.fmtTime(item.pack.expiryAt))}</td>
            <td class="small">${item.latestIssue ? CSSD.esc(item.latestIssue.department + ' ' + item.latestIssue.operatingRoom + '（' + CSSD.esc((CSSD.constants.IssueStatus[item.latestIssue.status] || { label: item.latestIssue.status }).label) + '）') : '未发放'}</td>
          </tr>`).join('')}
        </tbody></table>`;
    } catch (e) {
      result.innerHTML = '<div class="alert error">' + CSSD.esc(e.message) + '</div>';
    }
  }
};
