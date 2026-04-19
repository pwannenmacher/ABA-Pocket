/* ABA Pocket – Frontend JavaScript */

// ── Mobile nav toggle ──────────────────────────────────────────
const navToggle = document.getElementById('navToggle');
const navMenu   = document.getElementById('navMenu');

if (navToggle && navMenu) {
  navToggle.addEventListener('click', () => {
    navMenu.classList.toggle('open');
    navToggle.setAttribute('aria-expanded', navMenu.classList.contains('open'));
  });

  // Close on outside click
  document.addEventListener('click', (e) => {
    if (!navToggle.contains(e.target) && !navMenu.contains(e.target)) {
      navMenu.classList.remove('open');
    }
  });
}

// ── Global search dropdown ─────────────────────────────────────
const globalSearch = document.getElementById('globalSearch');
const dropdown     = document.getElementById('search-results-dropdown');

if (globalSearch && dropdown) {
  // Close dropdown when clicking outside
  document.addEventListener('click', (e) => {
    if (!globalSearch.contains(e.target) && !dropdown.contains(e.target)) {
      dropdown.innerHTML = '';
    }
  });

  // Close on Escape
  globalSearch.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') dropdown.innerHTML = '';
    if (e.key === 'Enter') {
      e.preventDefault();
      const q = globalSearch.value.trim();
      if (q) window.location.href = '/search?q=' + encodeURIComponent(q);
    }
  });
}

// ── Flash message auto-dismiss ─────────────────────────────────
const flash = document.getElementById('flashMsg');
if (flash) {
  setTimeout(() => flash.style.transition = 'opacity 0.5s', 3000);
  setTimeout(() => { flash.style.opacity = '0'; setTimeout(() => flash.remove(), 500); }, 3500);
}

// ── Entry row removal (admin) ──────────────────────────────────
function removeRow(btn) {
  const row = btn.closest('tr');
  if (row) row.remove();
}

// ── Markdown preview (optional enhancement) ────────────────────
// When a textarea with class 'entry-input' changes, we could show
// a live preview – keeping it simple for now.

// ── HTMX config ───────────────────────────────────────────────
document.addEventListener('DOMContentLoaded', () => {
  // Ensure htmx is configured
  if (typeof htmx !== 'undefined') {
    htmx.config.defaultSwapStyle = 'innerHTML';
  }
});
