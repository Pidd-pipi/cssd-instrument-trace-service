/**
 * 应用入口：路径路由 + 页面渲染 + 健康指示器。
 * 路由：/ 工作台、/packs 器械包、/sterilization 灭菌、/issue 发放回收、/trace 追溯。
 */
window.CSSD = window.CSSD || {};

(function () {
  const routes = {
    '/': { title: '工作台', render: CSSD.pages.dashboard },
    '/packs': { title: '器械包管理', render: CSSD.pages.packs },
    '/sterilization': { title: '灭菌管理', render: CSSD.pages.sterilization },
    '/issue': { title: '发放回收', render: CSSD.pages.issue },
    '/trace': { title: '追溯查询', render: CSSD.pages.trace }
  };

  function currentPath() {
    let p = window.location.pathname;
    if (p.endsWith('/') && p.length > 1) p = p.slice(0, -1);
    return routes[p] ? p : '/';
  }

  function render() {
    const path = currentPath();
    const route = routes[path];
    const app = document.getElementById('app');
    app.innerHTML = '<div class="loading">页面加载中...</div>';
    document.title = route.title + ' - CSSD 器械追溯服务';
    document.querySelectorAll('.nav a').forEach((a) => {
      a.classList.toggle('active', a.dataset.route === path);
    });
    try {
      route.render(app);
    } catch (e) {
      app.innerHTML = '<div class="alert error">页面渲染失败: ' + CSSD.esc(e.message) + '</div>';
    }
  }

  // 路径路由：拦截导航点击，使用 history.pushState。
  document.addEventListener('click', (e) => {
    const a = e.target.closest('a[href]');
    if (!a) return;
    const href = a.getAttribute('href');
    if (href.startsWith('/') && !href.startsWith('/api/')) {
      e.preventDefault();
      if (currentPath() !== href) {
        window.history.pushState({}, '', href);
        render();
      } else {
        render();
      }
    }
  });
  window.addEventListener('popstate', render);

  // 健康指示器：周期性探测 /healthz。
  async function checkHealth() {
    const el = document.getElementById('health-indicator');
    if (!el) return;
    try {
      const data = await CSSD.api.healthz();
      el.className = 'health ok';
      el.textContent = '● 服务正常';
      el.title = '服务健康：' + (data && data.service ? data.service : '') + '，器械包 ' + (data && data.packs != null ? data.packs : '-') + ' 个';
    } catch (e) {
      el.className = 'health bad';
      el.textContent = '● 服务异常';
    }
  }
  checkHealth();
  setInterval(checkHealth, 15000);

  render();
})();
