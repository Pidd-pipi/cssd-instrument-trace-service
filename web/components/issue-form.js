/**
 * IssueForm：发放登记表单组件。
 * 被发放页与器械包详情共用；校验「已灭菌 + 未过期 + 批次参数合格」由后端完成。
 */
window.CSSD = window.CSSD || {};

CSSD.IssueForm = function (pack, opts) {
  opts = opts || {};
  const wrap = document.createElement('div');

  const info = document.createElement('div');
  info.className = 'alert success';
  info.style.display = 'none';
  wrap.appendChild(info);

  const errBox = document.createElement('div');
  errBox.className = 'alert error';
  errBox.style.display = 'none';
  wrap.appendChild(errBox);

  const form = document.createElement('div');
  form.className = 'form-grid';
  form.innerHTML = [
    '<div class="form-row"><label>使用科室</label><input type="text" id="issue-department" placeholder="如：普外科" required></div>',
    '<div class="form-row"><label>手术间</label><input type="text" id="issue-room" placeholder="如：3号手术间" required></div>',
    '<div class="form-row"><label>发放人</label><input type="text" id="issue-issuer" placeholder="如：王护士" required></div>'
  ].join('');
  wrap.appendChild(form);

  const actions = document.createElement('div');
  actions.className = 'form-row';
  actions.innerHTML = '<button class="btn primary" id="issue-submit">确认发放</button>';
  wrap.appendChild(actions);

  const submitBtn = actions.querySelector('#issue-submit');
  submitBtn.addEventListener('click', async () => {
    const department = form.querySelector('#issue-department').value.trim();
    const operatingRoom = form.querySelector('#issue-room').value.trim();
    const issuer = form.querySelector('#issue-issuer').value.trim();
    if (!department || !operatingRoom || !issuer) {
      showErr('请完整填写使用科室、手术间与发放人');
      return;
    }
    submitBtn.disabled = true;
    try {
      const record = await CSSD.api.issuePack(pack.id, {
        department: department,
        operatingRoom: operatingRoom,
        issuer: issuer,
        operator: issuer
      });
      showOk('发放成功：' + pack.barcode + ' → ' + department + ' ' + operatingRoom);
      form.querySelector('#issue-department').value = '';
      form.querySelector('#issue-room').value = '';
      form.querySelector('#issue-issuer').value = '';
      if (opts.onSuccess) opts.onSuccess(record);
    } catch (e) {
      showErr(e.message);
    } finally {
      submitBtn.disabled = false;
    }
  });

  function showOk(msg) {
    info.textContent = msg;
    info.style.display = 'block';
    errBox.style.display = 'none';
  }
  function showErr(msg) {
    errBox.textContent = msg;
    errBox.style.display = 'block';
    info.style.display = 'none';
  }

  return wrap;
};
