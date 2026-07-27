// Client-side search over a pre-built index. The index is loaded as a plain
// <script> (window.GLX_SEARCH_INDEX) rather than fetched, so search works when
// the site is opened directly from disk via file:// (where fetch is blocked).
(function () {
  var box = document.getElementById('search-box');
  var results = document.getElementById('search-results');
  var status = document.getElementById('search-status');
  var index = window.GLX_SEARCH_INDEX || [];
  if (!box || !results) {
    return;
  }

  function render(query) {
    var q = query.trim().toLowerCase();
    results.textContent = '';
    if (!q) {
      status.textContent = index.length + ' entries — start typing to search.';
      return;
    }
    var matches = index.filter(function (e) {
      return (e.name && e.name.toLowerCase().indexOf(q) !== -1) ||
             (e.sub && e.sub.toLowerCase().indexOf(q) !== -1);
    }).slice(0, 100);

    status.textContent = matches.length
      ? matches.length + ' match' + (matches.length === 1 ? '' : 'es')
      : 'No matches.';

    matches.forEach(function (e) {
      var li = document.createElement('li');
      var a = document.createElement('a');
      a.href = e.url;
      a.textContent = e.name || '(untitled)';
      li.appendChild(a);
      if (e.sub) {
        var sub = document.createElement('span');
        sub.className = 'muted';
        sub.textContent = ' ' + e.sub;
        li.appendChild(sub);
      }
      if (e.kind) {
        var kind = document.createElement('span');
        kind.className = 'kind';
        kind.textContent = e.kind;
        li.appendChild(kind);
      }
      results.appendChild(li);
    });
  }

  box.addEventListener('input', function () {
    render(box.value);
  });

  // Support deep-linking via ?q=term
  try {
    var initial = new URLSearchParams(window.location.search).get('q');
    if (initial) {
      box.value = initial;
    }
  } catch (e) { /* URLSearchParams unavailable — ignore */ }

  render(box.value);
})();
