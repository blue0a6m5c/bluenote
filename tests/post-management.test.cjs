// Run with: node --test tests/post-management.test.cjs
const test = require('node:test');
const assert = require('node:assert/strict');
const vm = require('node:vm');
const fs = require('node:fs');
const source = fs.readFileSync(require('node:path').join(__dirname, '../static/js/post-management.js'), 'utf8');

function setup(operation, confirm = true) {
 let click, request, reloads = 0;
 const error = {textContent: ''};
 const button = {disabled: false, getAttribute: () => operation, closest: () => ({getAttribute: () => 'post123456'})};
 const table = {getAttribute: name => name === 'data-alias' ? 'myblog' : 'csrf-token', addEventListener: (_, fn) => { click = fn; }};
 vm.runInNewContext(source, {
  document: {getElementById: id => id === 'management-posts' ? table : error},
  window: {confirm: () => confirm, location: {reload: () => {reloads++;}}},
  XMLHttpRequest: function () {
   request = this;
   this.headers = {};
   this.open = (method, path) => {this.method = method; this.path = path;};
   this.setRequestHeader = (name, value) => {this.headers[name] = value;};
   this.send = body => {this.body = body;};
  }
 });
 click({target: button});
 return {request, button, error, reloads: () => reloads};
}

test('pin success checks the per-post result before reload', () => {
 const s = setup('pin');
 assert.equal(s.request.method, 'POST');
 assert.equal(s.request.path, '/api/collections/myblog/pin');
 assert.deepEqual(JSON.parse(s.request.body), [{id: 'post123456'}]);
 assert.equal(s.request.headers['X-CSRF-Token'], 'csrf-token');
 s.request.status = 200;
 s.request.responseText = JSON.stringify({data: [{id: 'post123456', code: 200}]});
 s.request.onload();
 assert.equal(s.reloads(), 1);
});
test('HTTP 200 with per-post failure does not report success', () => {
 const s = setup('unpin');
 s.request.status = 200;
 s.request.responseText = JSON.stringify({data: [{id: 'post123456', code: 403}]});
 s.request.onload();
 assert.equal(s.reloads(), 0);
 assert.equal(s.button.disabled, false);
 assert.ok(s.error.textContent);
});
test('delete cancellation sends no request', () => {
 assert.equal(setup('delete', false).request, undefined);
});
test('delete only reloads on confirmed success', () => {
 const s = setup('delete');
 assert.equal(s.request.method, 'DELETE');
 assert.equal(s.request.path, '/api/collections/myblog/posts/post123456/delete');
 assert.equal(s.request.headers['X-CSRF-Token'], 'csrf-token');
 s.request.status = 204;
 s.request.onload();
 assert.equal(s.reloads(), 1);
});
test('timeout re-enables the control and displays an error', () => {
 const s = setup('pin');
 s.request.ontimeout();
 assert.equal(s.button.disabled, false);
 assert.ok(s.error.textContent);
 assert.equal(s.reloads(), 0);
});
