/* ABA Pocket – Frontend JavaScript */

// ── Mobile nav toggle ──────────────────────────────────────────
const navToggle = document.getElementById('navToggle');
const navMenu   = document.getElementById('navMenu');

if (navToggle && navMenu) {
  navToggle.addEventListener('click', () => {
    navMenu.classList.toggle('open');
    navToggle.setAttribute('aria-expanded', navMenu.classList.contains('open'));
  });
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
  document.addEventListener('click', (e) => {
    if (!globalSearch.contains(e.target) && !dropdown.contains(e.target)) {
      dropdown.innerHTML = '';
    }
  });
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
  setTimeout(() => { flash.style.transition = 'opacity 0.5s'; }, 3000);
  setTimeout(() => { flash.style.opacity = '0'; setTimeout(() => flash.remove(), 500); }, 3500);
}

// ── Medication card: entry row removal ────────────────────────
function removeRow(btn) {
  btn.closest('tr')?.remove();
}

// ═══════════════════════════════════════════════════════════════
// Symptom-Formular: dynamische Tabellen-Verwaltung
// ═══════════════════════════════════════════════════════════════

/**
 * Fügt eine neue, leere Tabellengruppe zum Container hinzu.
 */
function addSymptomTable() {
  const container = document.getElementById('tablesContainer');
  if (!container) return;

  const div = document.createElement('div');
  div.className = 'symptom-table form-section-inner';
  div.innerHTML = `
    <div class="table-header-bar">
      <input type="text" class="table-title-input form-input"
        placeholder="Überschrift dieser Tabelle (optional)">
      <button type="button" class="btn btn-danger btn-sm" onclick="removeTable(this)">Tabelle löschen</button>
    </div>
    <div class="entries-table-wrap">
      <table class="entries-table">
        <colgroup>
          <col style="width:26px">
          <col style="width:38%">
          <col>
          <col style="width:44px">
        </colgroup>
        <tbody class="rows-body"></tbody>
      </table>
    </div>
    <input type="hidden" class="row-count-input" value="0">
    <button type="button" class="btn btn-secondary btn-sm"
      onclick="addTableRow(this.closest('.symptom-table'))">
      + Zeile hinzufügen
    </button>`;
  container.appendChild(div);
  initRowDragDrop(div.querySelector('.rows-body'));
}

/**
 * Entfernt eine Tabellengruppe nach Bestätigung.
 */
function removeTable(btn) {
  if (confirm('Tabelle wirklich löschen?')) {
    btn.closest('.symptom-table')?.remove();
  }
}

/**
 * Fügt einer Tabellengruppe eine neue Zeile hinzu.
 * @param {HTMLElement} tableEl – das .symptom-table-Element
 */
function addTableRow(tableEl) {
  const tbody = tableEl.querySelector('.rows-body');
  if (!tbody) return;

  const tr = document.createElement('tr');
  tr.className = 'table-row';
  tr.innerHTML = `
    <td class="drag-handle" title="Zeile verschieben">⠿</td>
    <td><textarea class="med-input entry-input" rows="2"></textarea></td>
    <td><textarea class="right-input entry-input" rows="2"></textarea></td>
    <td class="entry-action">
      <button type="button" class="btn btn-danger btn-sm"
        onclick="removeTableRow(this)">✕</button>
    </td>`;
  tbody.appendChild(tr);
  tr.querySelector('.med-input')?.focus();
}

/**
 * Entfernt eine Zeile aus einer Tabellengruppe.
 */
function removeTableRow(btn) {
  btn.closest('tr')?.remove();
}

// ── Drag & Drop: Zeilen in Symptom-Tabellen sortieren ──────────

let _dragRow = null;

/**
 * Aktiviert Drag & Drop-Sortierung für einen <tbody>.
 * Drag startet nur vom .drag-handle, damit Textareas normal nutzbar bleiben.
 */
function initRowDragDrop(tbody) {
  if (!tbody) return;

  // Draggable nur über den Handle aktivieren
  tbody.addEventListener('mousedown', e => {
    if (e.target.closest('.drag-handle')) {
      e.target.closest('tr.table-row')?.setAttribute('draggable', 'true');
    } else {
      tbody.querySelectorAll('tr.table-row').forEach(r => r.removeAttribute('draggable'));
    }
  });

  tbody.addEventListener('dragstart', e => {
    const row = e.target.closest('tr.table-row');
    if (!row) return;
    _dragRow = row;
    e.dataTransfer.effectAllowed = 'move';
    setTimeout(() => row.classList.add('dragging'), 0);
  });

  tbody.addEventListener('dragover', e => {
    e.preventDefault();
    const row = e.target.closest('tr.table-row');
    if (!row || row === _dragRow) return;
    const after = e.clientY > row.getBoundingClientRect().top + row.offsetHeight / 2;
    tbody.querySelectorAll('tr').forEach(r =>
      r.classList.remove('drag-over-top', 'drag-over-bottom'));
    row.classList.add(after ? 'drag-over-bottom' : 'drag-over-top');
  });

  tbody.addEventListener('dragleave', e => {
    e.target.closest('tr.table-row')
      ?.classList.remove('drag-over-top', 'drag-over-bottom');
  });

  tbody.addEventListener('drop', e => {
    e.preventDefault();
    const row = e.target.closest('tr.table-row');
    if (!row || !_dragRow || row === _dragRow) return;
    const after = e.clientY > row.getBoundingClientRect().top + row.offsetHeight / 2;
    after ? row.after(_dragRow) : row.before(_dragRow);
    tbody.querySelectorAll('tr').forEach(r =>
      r.classList.remove('drag-over-top', 'drag-over-bottom'));
  });

  tbody.addEventListener('dragend', () => {
    _dragRow?.classList.remove('dragging');
    _dragRow?.removeAttribute('draggable');
    tbody.querySelectorAll('tr').forEach(r =>
      r.classList.remove('drag-over-top', 'drag-over-bottom', 'dragging'));
    _dragRow = null;
  });
}

// Drag & Drop für alle beim Laden vorhandenen Tabellen initialisieren
document.querySelectorAll('#tablesContainer .rows-body').forEach(initRowDragDrop);

// ── Renaming vor dem Submit ────────────────────────────────────
// Benennt alle Felder des Symptom-Formulars sequenziell um, damit
// der Server klar strukturierte Daten erhält (table_N_title, row_N_M_med, …).

document.getElementById('symptomForm')?.addEventListener('submit', function () {
  renumberSymptomTables();
});

function renumberSymptomTables() {
  const tables = document.querySelectorAll('#tablesContainer .symptom-table');

  // Gesamtanzahl Tabellen
  const countInput = document.getElementById('tableCount');
  if (countInput) countInput.value = tables.length;

  tables.forEach((table, t) => {
    // Tabellenüberschrift
    const titleInput = table.querySelector('.table-title-input');
    if (titleInput) titleInput.name = `table_${t}_title`;

    // Zeilen
    const rowInputs = table.querySelectorAll('.row-count-input');
    const rows = table.querySelectorAll('.table-row');

    rowInputs.forEach(inp => {
      inp.name  = `row_count_${t}`;
      inp.value = rows.length;
    });

    rows.forEach((row, m) => {
      const medEl   = row.querySelector('.med-input');
      const rightEl = row.querySelector('.right-input');
      if (medEl)   medEl.name   = `row_${t}_${m}_med`;
      if (rightEl) rightEl.name = `row_${t}_${m}_right`;
    });
  });
}
