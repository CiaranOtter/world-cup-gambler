// ============================================================
// State
// ============================================================
const REFRESH_MS = 30_000;
const STORAGE_KEY = 'wcg_player_id';

let allMatches = [];
let allGroups  = [];
let activeView = 'matches';
let activeMatchday = 'all';
let currentUser = null;

// Surface any unhandled promise rejections visibly during dev
window.addEventListener('unhandledrejection', e => {
  console.error('Unhandled rejection:', e.reason);
});

// ============================================================
// Boot
// ============================================================
document.addEventListener('DOMContentLoaded', () => {
  setupNav();
  setupGame();
  loadAll();
  setInterval(loadAll, REFRESH_MS);
});

async function loadAll() {
  const [matchesRes, groupsRes] = await Promise.all([
    safeFetch('/api/matches'),
    safeFetch('/api/groups'),
  ]);

  if (matchesRes) {
    allMatches = matchesRes;
    sortMatches();
    if (activeView === 'matches')    renderMatches();
    if (activeView === 'leaderboard') renderLeaderboard();
    if (activeView === 'game' && currentUser) renderProfileStats();
  }

  if (groupsRes) {
    allGroups = groupsRes;
    if (activeView === 'standings') renderStandings();
  }

  if (activeView === 'leaderboard') renderLeaderboard();
}

// ============================================================
// Navigation
// ============================================================
function setupNav() {
  document.querySelectorAll('.nav-tab').forEach(btn => {
    btn.addEventListener('click', () => switchView(btn.dataset.view));
  });
}

function switchView(view) {
  activeView = view;
  document.querySelectorAll('.nav-tab').forEach(b =>
    b.classList.toggle('active', b.dataset.view === view));
  document.querySelectorAll('.view').forEach(v =>
    v.classList.toggle('active', v.id === `view-${view}`));

  if (view === 'matches')     renderMatches();
  if (view === 'standings')   renderStandings();
  if (view === 'leaderboard') renderLeaderboard().catch(e => console.error('leaderboard render:', e));
}

// ============================================================
// Matches
// ============================================================
function sortMatches() {
  allMatches.sort((a, b) => {
    const md = Number(a.matchday) - Number(b.matchday);
    if (md !== 0) return md;
    return (a.local_date || '').localeCompare(b.local_date || '');
  });
}

function renderMatches() {
  const container = document.getElementById('matches-list');
  const filterBar = document.getElementById('matchday-filters');

  if (!allMatches.length) {
    container.innerHTML = '<p class="loading">Loading fixtures…</p>';
    return;
  }

  // Filter bar
  const days = [...new Set(allMatches.map(m => String(m.matchday)))].sort((a,b)=>Number(a)-Number(b));
  filterBar.innerHTML = '';
  filterBar.appendChild(makeFilterBtn('All', 'all'));
  days.forEach(d => filterBar.appendChild(makeFilterBtn(`Matchday ${d}`, d)));

  // Match list — group by matchday for day labels
  const filtered = activeMatchday === 'all' ? allMatches
    : allMatches.filter(m => String(m.matchday) === activeMatchday);

  if (!filtered.length) {
    container.innerHTML = '<p class="empty">No fixtures found.</p>';
    return;
  }

  let html = '';
  let lastDay = null;
  for (const m of filtered) {
    if (activeMatchday === 'all' && String(m.matchday) !== lastDay) {
      lastDay = String(m.matchday);
      html += `<div class="match-day-label">Matchday ${esc(lastDay)}</div>`;
    }
    html += renderMatchCard(m);
  }
  container.innerHTML = html;
}

function makeFilterBtn(label, value) {
  const btn = document.createElement('button');
  btn.className = 'filter-btn' + (value === activeMatchday ? ' active' : '');
  btn.textContent = label;
  btn.addEventListener('click', () => {
    activeMatchday = value;
    renderMatches();
  });
  return btn;
}

function renderMatchCard(m) {
  const status = matchStatus(m);
  const hs = m.home_score ?? '–';
  const as = m.away_score ?? '–';
  const stadium = m.stadium?.name_en
    ? `${m.stadium.name_en}${m.stadium.city_en ? ' · ' + m.stadium.city_en : ''}`
    : (m.stadium_id ? `Stadium #${m.stadium_id}` : '');

  const scorers = (m.home_scorers?.length || m.away_scorers?.length) ? `
    <div class="scorers">
      <ul>${(m.home_scorers||[]).map(s=>`<li>${esc(s)}</li>`).join('')}</ul>
      <ul class="away-scorers">${(m.away_scorers||[]).map(s=>`<li>${esc(s)}</li>`).join('')}</ul>
    </div>` : '';

  return `
    <article class="match-card ${status.cls}">
      <div class="match-card-top">
        <span class="group-badge">Group ${esc(m.group||'–')}</span>
        <span class="match-status ${status.cls}">${esc(status.label)}</span>
      </div>
      <div class="scoreboard">
        <div class="team home">
          <div class="team-crest">${crest(m.home_team?.name_en)}</div>
          <div class="team-info">
            <div class="team-name">${esc(m.home_team?.name_en||'TBD')}</div>
            <div class="team-name-fa">${esc(m.home_team?.name_fa||'')}</div>
          </div>
        </div>
        <div class="score-board">
          <span>${esc(String(hs))}</span>
          <span class="sep">–</span>
          <span>${esc(String(as))}</span>
        </div>
        <div class="team away">
          <div class="team-crest">${crest(m.away_team?.name_en)}</div>
          <div class="team-info">
            <div class="team-name">${esc(m.away_team?.name_en||'TBD')}</div>
            <div class="team-name-fa">${esc(m.away_team?.name_fa||'')}</div>
          </div>
        </div>
      </div>
      ${scorers}
      <div class="match-footer">
        <span>${esc(m.local_date||'')}</span>
        ${m.persian_date ? `<span>${esc(m.persian_date)}</span>` : ''}
        ${stadium ? `<span>${esc(stadium)}</span>` : ''}
      </div>
    </article>`;
}

function matchStatus(m) {
  if (m.finished) return { cls: 'finished', label: 'Full time' };
  if (m.time_elapsed && m.time_elapsed !== 'finished' && m.time_elapsed !== '')
    return { cls: 'live', label: m.time_elapsed };
  return { cls: 'scheduled', label: 'Scheduled' };
}

// ============================================================
// Standings
// ============================================================
function renderStandings() {
  const el = document.getElementById('standings-list');
  if (!allGroups.length) {
    el.innerHTML = '<p class="loading">Loading standings…</p>';
    return;
  }

  const sorted = [...allGroups].sort((a,b) => a.group.localeCompare(b.group));
  el.innerHTML = `<div class="groups-grid">${sorted.map(renderGroupCard).join('')}</div>`;
}

function renderGroupCard(g) {
  const teams = [...(g.teams||[])].sort((a,b) => {
    const pd = Number(b.pts) - Number(a.pts);
    if (pd) return pd;
    const gda = (Number(a.gf)||0) - (Number(a.ga)||0);
    const gdb = (Number(b.gf)||0) - (Number(b.ga)||0);
    return gdb - gda;
  });

  const rows = teams.map((t, i) => {
    const team = findTeam(t.team_id);
    const gd = (Number(t.gf)||0) - (Number(t.ga)||0);
    return `<tr>
      <td class="td-pos">${i+1}</td>
      <td>
        <div class="td-en">${esc(team?.name_en || `Team ${t.team_id}`)}</div>
        <div class="td-fa">${esc(team?.name_fa||'')}</div>
      </td>
      <td class="td-center">${esc(t.gf)}</td>
      <td class="td-center">${esc(t.ga)}</td>
      <td class="td-center">${gd>=0?'+':''}${gd}</td>
      <td class="td-pts td-center">${esc(t.pts)}</td>
    </tr>`;
  }).join('');

  return `
    <div class="group-card">
      <div class="group-card-header">
        <div class="group-letter">${esc(g.group)}</div>
        <span>Group ${esc(g.group)}</span>
      </div>
      <table class="group-table">
        <thead><tr>
          <th>#</th><th>Team</th>
          <th class="td-center">GF</th>
          <th class="td-center">GA</th>
          <th class="td-center">GD</th>
          <th class="td-center">Pts</th>
        </tr></thead>
        <tbody>${rows}</tbody>
      </table>
    </div>`;
}

// ============================================================
// Leaderboard
// ============================================================
async function renderLeaderboard() {
  const el = document.getElementById('leaderboard-list');
  const data = await safeFetch('/api/leaderboard');
  if (!data) { el.innerHTML = '<p class="error">Failed to load leaderboard.</p>'; return; }
  if (!data.length) { el.innerHTML = '<p class="empty">No players yet — be the first to join!</p>'; return; }

  const medals = ['🥇','🥈','🥉'];

  const rows = data.map(e => {
    const medal = medals[e.rank-1] || '';
    const rankCls = e.rank <= 3 ? `rank-${e.rank}` : '';
    const isMe = currentUser && e.id === currentUser.id ? 'style="outline:2px solid var(--gold);outline-offset:-2px"' : '';
    const gd = e.goal_diff >= 0 ? `+${e.goal_diff}` : String(e.goal_diff);
    return `<tr class="${rankCls}" ${isMe}>
      <td class="td-center"><span class="rank-medal">${medal||e.rank}</span></td>
      <td>
        <div class="player-cell">
          <div class="player-avatar">${nameInitials(e.name)}</div>
          <div>
            <div class="player-name">${esc(e.name)}</div>
            <div class="player-team">${esc(e.team_name)}</div>
          </div>
        </div>
      </td>
      <td class="td-center hide-mobile">${e.played}</td>
      <td class="td-center hide-mobile">${e.won}/${e.drawn}/${e.lost}</td>
      <td class="td-center hide-mobile">${e.goals_for}</td>
      <td class="td-center hide-mobile">${gd}</td>
      <td class="td-center hide-mobile">${e.points}</td>
      <td class="td-center hide-mobile">${e.knockout_wins}</td>
      <td class="td-center"><span class="stat-score">${e.score}</span></td>
    </tr>`;
  }).join('');

  el.innerHTML = `
    <table class="leaderboard-table">
      <thead><tr>
        <th class="td-center">#</th>
        <th>Player / Team</th>
        <th class="td-center hide-mobile">P</th>
        <th class="td-center hide-mobile">W/D/L</th>
        <th class="td-center hide-mobile">GF</th>
        <th class="td-center hide-mobile">GD</th>
        <th class="td-center hide-mobile">Pts</th>
        <th class="td-center hide-mobile">KO</th>
        <th class="td-center">Score</th>
      </tr></thead>
      <tbody>${rows}</tbody>
    </table>`;
}

// ============================================================
// Game — profile management
// ============================================================
function setupGame() {
  const get = id => {
    const el = document.getElementById(id);
    if (!el) console.error(`setupGame: element #${id} not found`);
    return el;
  };

  get('reg-submit')?.addEventListener('click', registerUser);
  get('reg-name')?.addEventListener('keydown', e => {
    if (e.key === 'Enter') registerUser();
  });
  get('lookup-link')?.addEventListener('click', e => {
    e.preventDefault();
    const f = document.getElementById('lookup-form');
    if (f) f.style.display = f.style.display === 'none' ? 'block' : 'none';
  });
  get('lookup-submit')?.addEventListener('click', lookupUser);
  get('lookup-id')?.addEventListener('keydown', e => {
    if (e.key === 'Enter') lookupUser();
  });
  get('reroll-btn')?.addEventListener('click', rerollTeam);
  get('signout-btn')?.addEventListener('click', signOut);

  // Auto-load saved profile
  const saved = localStorage.getItem(STORAGE_KEY);
  if (saved) loadProfile(saved);
}

async function registerUser() {
  const name = document.getElementById('reg-name').value.trim();
  if (!name) { alert('Please enter your name.'); return; }

  const btn = document.getElementById('reg-submit');
  btn.disabled = true; btn.textContent = 'Drawing…';

  const data = await safeFetch('/api/users', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name }),
  });

  btn.disabled = false; btn.textContent = 'Draw my team ⚽';

  if (!data) return;
  localStorage.setItem(STORAGE_KEY, data.id);
  setCurrentUser(data);
}

async function lookupUser() {
  const id = document.getElementById('lookup-id').value.trim();
  if (!id) return;
  await loadProfile(id);
}

async function loadProfile(id) {
  const data = await safeFetch(`/api/users/${id}`);
  if (!data) {
    localStorage.removeItem(STORAGE_KEY);
    return;
  }
  localStorage.setItem(STORAGE_KEY, data.id);
  setCurrentUser(data);
}

async function rerollTeam() {
  if (!currentUser) return;
  if (!confirm('Are you sure? You only get one re-roll for the entire tournament!')) return;

  const btn = document.getElementById('reroll-btn');
  btn.disabled = true; btn.textContent = 'Drawing…';

  const data = await safeFetch(`/api/users/${currentUser.id}/reroll`, { method: 'POST' });
  btn.disabled = false; btn.textContent = '🎲 Re-roll my team';

  if (!data) return;
  setCurrentUser(data);
}

function signOut() {
  localStorage.removeItem(STORAGE_KEY);
  currentUser = null;
  document.getElementById('game-profile').style.display = 'none';
  document.getElementById('game-register').style.display = '';
  document.getElementById('reg-name').value = '';
}

function setCurrentUser(user) {
  currentUser = user;
  document.getElementById('game-register').style.display = 'none';
  document.getElementById('game-profile').style.display = '';

  document.getElementById('profile-crest').textContent = crest(user.team_name);
  document.getElementById('profile-team-name').textContent = user.team_name || '—';
  document.getElementById('profile-team-fa').textContent = user.team_fa || '';
  document.getElementById('profile-display-name').textContent = user.name;
  document.getElementById('profile-id-display').textContent = user.id;

  const rerollBtn = document.getElementById('reroll-btn');
  const rerollNote = document.getElementById('reroll-note');
  if (user.has_rerolled) {
    rerollBtn.disabled = true;
    rerollNote.textContent = 'Re-roll used — good luck with your team!';
  } else {
    rerollBtn.disabled = false;
    rerollNote.textContent = 'You have 1 re-roll remaining for the tournament.';
  }

  renderProfileStats();
}

function renderProfileStats() {
  if (!currentUser || !allMatches.length) return;
  const el = document.getElementById('profile-stats');
  const tid = currentUser.team_id;

  let played=0, won=0, drawn=0, lost=0, gf=0, ga=0, pts=0, koWins=0;
  for (const m of allMatches) {
    if (!m.finished) continue;
    const hg = parseInt2(m.home_score), ag = parseInt2(m.away_score);
    const isHome = m.home_team?.id === tid;
    const isAway = m.away_team?.id === tid;
    if (!isHome && !isAway) continue;
    played++;
    const myG = isHome ? hg : ag, oppG = isHome ? ag : hg;
    gf += myG; ga += oppG;
    if (myG > oppG) { won++; pts+=3; if (m.type!=='group') koWins++; }
    else if (myG === oppG) { drawn++; pts++; }
    else lost++;
  }
  const gd = gf - ga;
  const score = pts*3 + gf + gd + koWins*5;

  el.innerHTML = [
    ['Played', played], ['Won', won], ['Drawn', drawn], ['Lost', lost],
    ['GF', gf], ['GA', ga], ['GD', gd>=0?`+${gd}`:gd],
    ['Pts', pts], ['KO Wins', koWins], ['Score', score],
  ].map(([l,v]) => `<span class="stat-pill">${l}: <strong>${v}</strong></span>`).join('');
}

// ============================================================
// Helpers
// ============================================================
function findTeam(id) {
  for (const m of allMatches) {
    if (String(m.home_team?.id) === String(id)) return m.home_team;
    if (String(m.away_team?.id) === String(id)) return m.away_team;
  }
  return null;
}

function crest(name) {
  if (!name) return '?';
  return name.split(' ').map(w=>w[0]).slice(0,2).join('').toUpperCase();
}

function nameInitials(name) {
  if (!name) return '?';
  return name.split(' ').map(w=>w[0]).slice(0,2).join('').toUpperCase();
}

function parseInt2(s) {
  const n = parseInt(s, 10);
  return isNaN(n) ? 0 : n;
}

function esc(str) {
  const d = document.createElement('div');
  d.textContent = String(str ?? '');
  return d.innerHTML;
}

async function safeFetch(url, opts) {
  try {
    const res = await fetch(url, opts);
    if (!res.ok) {
      const text = await res.text();
      console.error(`${url} → ${res.status}: ${text}`);
      if (opts?.method === 'POST') alert(`Error: ${text}`);
      return null;
    }
    return await res.json();
  } catch (err) {
    console.error(`${url}:`, err);
    return null;
  }
}