// Light/dark theme toggle. The initial theme is applied inline in <head> to
// avoid a flash; this wires up the toggle button, keeps its aria-pressed state
// in sync with the active theme (for screen readers), and persists the choice.
(function () {
  var btn = document.getElementById('theme-toggle');
  if (!btn) {
    return;
  }

  function isDark() {
    return document.documentElement.getAttribute('data-theme') === 'dark';
  }

  function syncState() {
    btn.setAttribute('aria-pressed', isDark() ? 'true' : 'false');
  }

  // Reflect the theme applied inline before this script ran.
  syncState();

  btn.addEventListener('click', function () {
    var next = isDark() ? 'light' : 'dark';
    document.documentElement.setAttribute('data-theme', next);
    syncState();
    try {
      localStorage.setItem('glx-theme', next);
    } catch (e) {
      /* localStorage unavailable (e.g. file://) — toggle still works for the session */
    }
  });
})();
