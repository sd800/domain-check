(() => {
  'use strict';
  const root = document.documentElement;
  const languageButton = document.getElementById('language-toggle');
  const themeButton = document.getElementById('theme-toggle');
  const status = document.getElementById('copy-status');
  let timer;
  function save(key, value) {
    try { localStorage.setItem('domain-check.website.' + key, value); } catch (_) {}
  }
  function updateLabels() {
    const chinese = root.dataset.language === 'zh';
    root.lang = chinese ? 'zh-CN' : 'en';
    languageButton.setAttribute('aria-label', chinese ? 'Switch to English' : '切换为简体中文');
    const names = chinese ? { auto: '跟随系统', light: '浅色', dark: '深色' } : { auto: 'System', light: 'Light', dark: 'Dark' };
    const label = (chinese ? '外观：' : 'Appearance: ') + names[root.dataset.theme];
    themeButton.setAttribute('aria-label', label);
    themeButton.title = label;
    const dark = root.dataset.theme === 'dark' || root.dataset.theme === 'auto' && matchMedia('(prefers-color-scheme: dark)').matches;
    document.querySelector('meta[name="theme-color"]').content = dark ? '#0d1421' : '#f8faff';
    document.querySelector('[data-copy="setup-command"]').setAttribute('aria-label', chinese ? '复制 domain setup' : 'Copy domain setup');
  }
  languageButton.addEventListener('click', () => {
    root.dataset.language = root.dataset.language === 'en' ? 'zh' : 'en';
    save('language', root.dataset.language);
    updateLabels();
  });
  themeButton.addEventListener('click', () => {
    const modes = ['auto', 'light', 'dark'];
    root.dataset.theme = modes[(modes.indexOf(root.dataset.theme) + 1) % modes.length];
    save('theme', root.dataset.theme);
    updateLabels();
  });
  matchMedia('(prefers-color-scheme: dark)').addEventListener('change', updateLabels);
  async function copy(text) {
    try {
      if (navigator.clipboard && window.isSecureContext) {
        await navigator.clipboard.writeText(text);
        return true;
      }
    } catch (_) {}
    const field = document.createElement('textarea');
    field.value = text;
    field.setAttribute('readonly', '');
    field.style.position = 'fixed';
    field.style.top = '0';
    field.style.left = '-9999px';
    const previous = document.activeElement;
    document.body.append(field);
    field.select();
    field.setSelectionRange(0, text.length);
    let copied = false;
    try { copied = document.execCommand('copy'); } catch (_) {}
    field.remove();
    if (previous instanceof HTMLElement) previous.focus({ preventScroll: true });
    return copied;
  }
  document.querySelectorAll('[data-copy]').forEach(button => {
    button.addEventListener('click', async () => {
      const text = button.dataset.copy === 'setup-command' ? 'domain setup' : document.getElementById('install-command').textContent.trim();
      const copied = await copy(text);
      const chinese = root.dataset.language === 'zh';
      status.textContent = copied ? (chinese ? '已复制到剪贴板' : 'Copied to clipboard') : (chinese ? '请选中命令并手动复制' : 'Select the command to copy it manually');
      status.classList.add('is-visible');
      clearTimeout(timer);
      timer = setTimeout(() => status.classList.remove('is-visible'), 3000);
      if (!copied && button.dataset.copy === 'install-command') {
        const selection = window.getSelection();
        const range = document.createRange();
        range.selectNodeContents(document.getElementById('install-command'));
        selection.removeAllRanges(); selection.addRange(range);
      }
    });
  });
  updateLabels();
})();
