// Adds a Gorilla CSRF token to cookie-authenticated API mutations.
(function (global) {
 'use strict';

 global.BlueNoteCSRF = {
  send: function (request, body) {
   fetch('/api/csrf', {
    credentials: 'same-origin',
    headers: {'Accept': 'application/json'}
   }).then(function (response) {
    if (!response.ok) throw new Error('Unable to obtain CSRF token');
    return response.json();
   }).then(function (response) {
    request.setRequestHeader('X-CSRF-Token', response.data.token);
   }).catch(function () {
    // Sending without a token deliberately lets the protected endpoint reject
    // the request and keeps the existing UI error handling in control.
   }).then(function () {
    request.send(body);
   });
  }
 };
})(window);
