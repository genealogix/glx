// Light/dark theme toggle. The initial theme is applied inline in <head> to
// avoid a flash; this only wires up the toggle button and persists the choice.
(function () {
  var btn = document.getElementById('theme-toggle');
  if (!btn) {
    return;
  }
  btn.addEventListener('click', function () {
    var current = document.documentElement.getAttribute('data-theme') === 'dark' ? 'dark' : 'light';
    var next = current === 'dark' ? 'light' : 'dark';
    document.documentElement.setAttribute('data-theme', next);
    try {
      localStorage.setItem('glx-theme', next);
    } catch (e) {
      /* localStorage unavailable (e.g. file://) — toggle still works for the session */
    }
  });
})();
