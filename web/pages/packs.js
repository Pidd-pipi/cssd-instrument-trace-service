/**
 * 器械包管理页（/packs）：器械包列表 + 条码登记 + 环节流转 + 详情追溯。
 * 共用 StageBadge、CycleTimeline、IssueForm 与 usePacks、useTrace Hook。
 */
window.CSSD = window.CSSD || {};
CSSD.pages = CSSD.pages || {};

CSSD.pages.packs = function (container) {
  const filter = { stage: '', type: '', keyword: '', limit: 200 };
  const hook = CSSD.usePacks(() => filter, renderList);

  function render() {
    container.innerHTML = `
      <div class="section-title"><h2>器械包管理</h2></div>

      <div class="card">
        <h3>📝 条码登记</h3>
        <div class="form-grid">
          <div class="form-row"><label>条码 *</label><input id="reg-barcode" placeholder="唯一条码，如 PK20260825-001"></div>
          <div class="form-row"><label>包名称 *</label><input id="reg-name" placeholder="如：腔镜手术包"></div>
          <div class="form-row"><label>包类型 *</label>
            <select id="reg-type">
              <option value="surgical">手术器械包</option>
              <option value="dressing">敷料包</option>
              <option value="instrument">器械包</option>
              <option value="implant">植入物包</option>
            </select>
          </div>
          <div class="form-row"><label>内含器械（逗号分隔）</label><input id="reg-instruments" placeholder="如：手术剪,止血钳"></div>
          <div class="form-row"><label>登记人</label><input id="reg-operator" placeholder="如：张护士"></div>
          <div class="form-row" style="display:flex;align-items:flex-end"><button class="btn primary" id="reg-submit" style="width:auto">登记器械包</button></div>
        </div>
      </div>

      <div class="card">
        <div class="section-title">
          <h3>器械包列表</h3>
          <span class="muted small">${hook.loading ? '加载中...' : '共 ' + hook.data.length + ' 条'}</span>
        </div>
        <div class="form-grid" style="margin-bottom:12px">
          <div class="form-row"><label>环节过滤</label><select id="f-stage"><option value="">全部</option>${stageOptions()}</select></div>
          <div class="form-row"><label>类型过滤</label><select id="f-type"><option value="">全部</option>${typeOptions()}</select></div>
          <div class="form-row"><label>关键字（条码/名称）</label><input id="f-keyword" placeholder="输入条码或名称"></div>
          <div class="form-row" style="display:flex;align-items:flex-end"><button class="btn" id="f-apply" style="width:auto">查询</button></div>
        </div>
        <div id="pack-table"></div>
      </div>
    `;

    container.querySelector('#reg-submit').addEventListener('click', register);
    container.querySelector('#f-apply').addEventListener('click', applyFilter);
    container.querySelector('#f-keyword').addEventListener('keydown', (e) => { if (e.key === 'Enter') applyFilter(); });
    renderList();
  }

  function stageOptions() {
    return Object.keys(CSSD.constants.PackStage)
      .map((s) => '<option value="' + s + '">' + CSSD.esc(CSSD.constants.PackStage[s].label) + '</option>').join('');
  }
  function typeOptions() {
    return Object.keys(CSSD.constants.PackType)
      .map((t) => '<option value="' + t + '">' + CSSD.esc(CSSD.constants.PackType[t]) + '</option>').join('');
  }

  function applyFilter() {
    filter.stage = container.querySelector('#f-stage').value;
    filter.type = container.querySelector('#f-type').value;
    filter.keyword = container.querySelector('#f-keyword').value.trim();
    hook.refresh();
  }

  async function register() {
    const payload = {
      barcode: container.querySelector('#reg-barcode').value.trim(),
      name: container.querySelector('#reg-name').value.trim(),
      packType: container.querySelector('#reg-type').value,
      instruments: container.querySelector('#reg-instruments').value.split(/[,，]/).map((s) => s.trim()).filter(Boolean),
      operator: container.querySelector('#reg-operator').value.trim() || '系统'
    };
    if (!payload.barcode || !payload.name) {
      CSSD.toast.error('条码与包名称为必填项');
      return;
    }
    try {
      const pack = await CSSD.api.createPack(payload);
      CSSD.toast.success('器械包登记成功：' + pack.barcode);
      container.querySelector('#reg-barcode').value = '';
      container.querySelector('#reg-name').value = '';
      container.querySelector('#reg-instruments').value = '';
      hook.refresh();
    } catch (e) {
      CSSD.toast.error(e.message);
    }
  }

  function renderList() {
    const box = container.querySelector('#pack-table');
    if (!box) return;
    if (hook.loading) { box.innerHTML = '<div class="empty">加载中...</div>'; return; }
    if (hook.error) { box.innerHTML = '<div class="alert error">' + CSSD.esc(hook.error) + '</div>'; return; }
    if (!hook.data.length) { box.innerHTML = '<div class="empty">暂无器械包</div>'; return; }
    box.innerHTML = `
      <table><thead><tr>
        <th>条码</th><th>名称</th><th>类型</th><th>环节</th><th>有效期至</th><th>灭菌结果</th><th>更新时间</th><th>操作</th>
      </tr></thead><tbody>
      ${hook.data.map((p) => `
        <tr>
          <td class="mono">${CSSD.esc(p.barcode)}</td>
          <td>${CSSD.esc(p.name)}</td>
          <td class="small">${CSSD.esc(CSSD.constants.PackType[p.packType] || p.packType)}</td>
          <td>${CSSD.StageBadge(p.stage)}</td>
          <td class="small">${CSSD.esc(CSSD.fmtTime(p.expiryAt))}</td>
          <td>${CSSD.ResultBadge(p.lastBatchResult)}</td>
          <td class="small muted">${CSSD.esc(CSSD.fmtTime(p.updatedAt))}</td>
          <td style="white-space:nowrap">
            <button class="btn small" data-act="detail" data-id="${CSSD.esc(p.id)}">详情</button>
            ${manualNext(p) ? `<button class="btn small primary" data-act="next" data-id="${CSSD.esc(p.id)}" data-stage="${CSSD.esc(manualNext(p))}">${nextLabel(p)}</button>` : ''}
          </td>
        </tr>`).join('')}
      </tbody></table>`;
    box.querySelectorAll('button[data-act="detail"]').forEach((btn) => {
      btn.addEventListener('click', () => showDetail(btn.dataset.id));
    });
    box.querySelectorAll('button[data-act="next"]').forEach((btn) => {
      btn.addEventListener('click', () => doCycle(btn.dataset.id, btn.dataset.stage));
    });
  }

  function manualNext(p) {
    return CSSD.constants.manualCycle[p.stage] || '';
  }
  function nextLabel(p) {
    const next = manualNext(p);
    const info = CSSD.constants.PackStage[next];
    return '推进 → ' + (info ? info.label : next);
  }

  async function doCycle(id, stage) {
    try {
      const pack = await CSSD.api.cyclePack(id, { stage: stage, operator: '护士站' });
      CSSD.toast.success(pack.barcode + ' 已推进到 ' + (CSSD.constants.PackStage[pack.stage] || {}).label);
      hook.refresh();
    } catch (e) {
      CSSD.toast.error(e.message);
    }
  }

  function showDetail(id) {
    const content = document.createElement('div');
    content.innerHTML = '<div class="loading">加载追溯数据...</div>';
    const m = CSSD.modal.open(CSSD.modal.build('器械包详情', content));

    const traceHook = CSSD.useTrace(id);
    traceHook.onChange = () => renderDetail(content, traceHook);
    renderDetail(content, traceHook);
  }

  function renderDetail(content, traceHook) {
    if (traceHook.loading) { content.innerHTML = '<div class="loading">加载中...</div>'; return; }
    if (traceHook.error) { content.innerHTML = '<div class="alert error">' + CSSD.esc(traceHook.error) + '</div>'; return; }
    const view = traceHook.data;
    const p = view.pack;
    const canIssue = p.stage === 'sterilized' && !p.expiryAtExpired;

    content.innerHTML = `
      <div class="kv">
        <div class="k">条码</div><div class="mono">${CSSD.esc(p.barcode)}</div>
        <div class="k">名称</div><div>${CSSD.esc(p.name)}</div>
        <div class="k">类型</div><div>${CSSD.esc(CSSD.constants.PackType[p.packType] || p.packType)}</div>
        <div class="k">当前环节</div><div>${CSSD.StageBadge(p.stage)}</div>
        <div class="k">有效期至</div><div class="small">${CSSD.esc(CSSD.fmtTime(p.expiryAt))}</div>
        <div class="k">灭菌批次</div><div class="small">${CSSD.esc((view.lastBatch && view.lastBatch.batchNo) || '-')} ${CSSD.ResultBadge(p.lastBatchResult)}</div>
        <div class="k">登记时间</div><div class="small">${CSSD.esc(CSSD.fmtTime(p.createdAt))}</div>
      </div>
      <h3>循环时间线</h3>
      <div id="detail-timeline"></div>
      <h3>发放记录</h3>
      <div id="detail-issues"></div>
      <div id="detail-issueform" style="margin-top:12px"></div>
    `;
    content.querySelector('#detail-timeline').appendChild(CSSD.CycleTimeline(view.cycles));
    content.querySelector('#detail-issues').innerHTML = renderIssueTable(view.issues || []);
    if (canIssue) {
      const h = document.createElement('h3');
      h.textContent = '发放登记';
      content.querySelector('#detail-issueform').appendChild(h);
      content.querySelector('#detail-issueform').appendChild(CSSD.IssueForm(p, {
        onSuccess: () => traceHook.refresh()
      }));
    }
  }

  function renderIssueTable(issues) {
    if (!issues.length) return '<div class="empty">暂无发放记录</div>';
    return `<table><thead><tr><th>科室</th><th>手术间</th><th>发放人</th><th>发放时间</th><th>回收人</th><th>状态</th></tr></thead><tbody>
      ${issues.map((r) => `
        <tr>
          <td>${CSSD.esc(r.department)}</td>
          <td>${CSSD.esc(r.operatingRoom)}</td>
          <td>${CSSD.esc(r.issuer)}</td>
          <td class="small">${CSSD.esc(CSSD.fmtTime(r.issuedAt))}</td>
          <td class="small">${CSSD.esc(r.collector || '-')}</td>
          <td>${CSSD.IssueStatusBadge(r.status)}</td>
        </tr>`).join('')}
    </tbody></table>`;
  }

  render();
};
