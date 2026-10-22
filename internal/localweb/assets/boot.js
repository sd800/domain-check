(() => {
  const read = key => { try { return localStorage.getItem(key); } catch { return null; } };
  const theme = read('domain-check.theme');
  document.documentElement.dataset.theme = ['light', 'dark'].includes(theme) ? theme : 'auto';
  const saved = read('domain-check.language');
  document.documentElement.lang = saved === 'zh' || saved !== 'en' && (navigator.language || '').toLowerCase().startsWith('zh') ? 'zh-CN' : 'en';
})();
