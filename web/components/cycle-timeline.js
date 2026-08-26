/**
 * CycleTimeline：循环时间线组件。
 * 被追溯页与器械包详情共用；渲染器械包完整环节留痕（时间/操作人/设备/参数快照）。
 */
window.CSSD = window.CSSD || {};

CSSD.CycleTimeline = function (cycles) {
  const wrap = document.createElement('div');
  if (!cycles || cycles.length === 0) {
    wrap.innerHTML = '<div class="empty">暂无环节记录</div>';
    return wrap;
  }
  const ul = document.createElement('ul');
  ul.className = 'timeline';
  cycles.forEach((c) => {
    const li = document.createElement('li');
    const fromLabel = c.fromStage ? (CSSD.constants.PackStage[c.fromStage] || {}).label || c.fromStage : '登记';
    const head = document.createElement('div');
    head.className = 'tl-head';
    head.innerHTML = CSSD.StageBadge(c.stage)
      + '<span class="small muted">' + CSSD.esc(fromLabel) + ' → ' + CSSD.esc((CSSD.constants.PackStage[c.stage] || { label: c.stage }).label) + '</span>';
    const meta = document.createElement('div');
    meta.className = 'tl-meta';
    const parts = [];
    parts.push('时间：' + CSSD.esc(CSSD.fmtTime(c.createdAt)));
    if (c.operator) parts.push('操作人：' + CSSD.esc(c.operator));
    if (c.deviceId) parts.push('设备：' + CSSD.esc(c.deviceId));
    if (c.note) parts.push('备注：' + CSSD.esc(c.note));
    meta.textContent = parts.join(' ｜ ');
    li.appendChild(head);
    li.appendChild(meta);
    if (c.params && Object.keys(c.params).length > 0) {
      const params = document.createElement('div');
      params.className = 'tl-params mono';
      params.textContent = Object.keys(c.params).map((k) => k + '=' + c.params[k]).join(', ');
      li.appendChild(params);
    }
    ul.appendChild(li);
  });
  wrap.appendChild(ul);
  return wrap;
};
