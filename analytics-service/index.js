// Skill Broker — Analytics Service
//
// Receives opt-in usage events from the broker hook.
// Only accepts skill_id — never prompt text.
//
// Endpoints:
//   POST /events   { "event": "inject|pin|install|flag", "skill_id": "..." }
//   GET  /health
//
// Deploy to Railway: PORT is set automatically.
// Events are appended to EVENTS_LOG (default: events.log).

const http = require('http');
const fs = require('fs');
const path = require('path');

const PORT = process.env.PORT || 8788;
const LOG_PATH = process.env.EVENTS_LOG || 'events.log';
const VALID_EVENTS = new Set(['inject', 'pin', 'install', 'flag']);

function writeEvent(event, skillId) {
  const entry = JSON.stringify({
    ts: new Date().toISOString(),
    event,
    skill_id: skillId,
  });
  fs.appendFileSync(LOG_PATH, entry + '\n');
}

const server = http.createServer((req, res) => {
  if (req.method === 'GET' && req.url === '/health') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ status: 'ok' }));
    return;
  }

  if (req.method === 'GET' && req.url === '/events.log') {
    if (!fs.existsSync(LOG_PATH)) {
      res.writeHead(404);
      res.end('not found');
      return;
    }
    res.writeHead(200, { 'Content-Type': 'text/plain' });
    fs.createReadStream(LOG_PATH).pipe(res);
    return;
  }

  if (req.method === 'POST' && req.url === '/events') {
    let body = '';
    req.on('data', chunk => { body += chunk; });
    req.on('end', () => {
      let parsed;
      try {
        parsed = JSON.parse(body);
      } catch {
        res.writeHead(400);
        res.end('bad request');
        return;
      }

      const event = (parsed.event || '').trim().toLowerCase();
      const skillId = (parsed.skill_id || '').trim();

      if (!VALID_EVENTS.has(event) || !skillId) {
        res.writeHead(400);
        res.end('invalid event or skill_id');
        return;
      }

      try {
        writeEvent(event, skillId);
      } catch (err) {
        console.error('error writing event:', err);
        res.writeHead(500);
        res.end('internal error');
        return;
      }

      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ ok: true }));
    });
    return;
  }

  res.writeHead(405);
  res.end('method not allowed');
});

server.listen(PORT, () => {
  console.log(`analytics service listening on :${PORT}`);
  console.log(`events log: ${LOG_PATH}`);
});
