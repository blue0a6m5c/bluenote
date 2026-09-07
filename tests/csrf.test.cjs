// Run with: node --test tests/csrf.test.cjs
const test = require('node:test');
const assert = require('node:assert/strict');
const vm = require('node:vm');
const fs = require('node:fs');
const source = fs.readFileSync(require('node:path').join(__dirname, '../static/js/csrf.js'), 'utf8');

test('adds a server-issued CSRF token before sending a mutation', async () => {
 let fetchOptions, sent;
 const window = {};
 vm.runInNewContext(source, {
  window,
  fetch: async (path, options) => {
   assert.equal(path, '/api/csrf');
   fetchOptions = options;
   return {ok: true, json: async () => ({data: {token: 'server-token'}})};
  }
 });
 const request = {
  headers: {},
  setRequestHeader(name, value) { this.headers[name] = value; },
  send(body) { sent = body; }
 };
 window.BlueNoteCSRF.send(request, 'payload');
 await new Promise(resolve => setImmediate(resolve));
 assert.equal(fetchOptions.credentials, 'same-origin');
 assert.equal(request.headers['X-CSRF-Token'], 'server-token');
 assert.equal(sent, 'payload');
});

test('token fetch failure still reaches protected endpoint error handling', async () => {
 let sent = false;
 const window = {};
 vm.runInNewContext(source, {
  window,
  fetch: async () => { throw new Error('network failure'); }
 });
 const request = {
  setRequestHeader() { throw new Error('must not set an unavailable token'); },
  send() { sent = true; }
 };
 window.BlueNoteCSRF.send(request, null);
 await new Promise(resolve => setImmediate(resolve));
 assert.equal(sent, true);
});
