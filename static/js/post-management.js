// BlueNote post management. Uses the existing authenticated mutation APIs.
(function () {
 'use strict';
 var table = document.getElementById('management-posts');
 if (!table) return;
 var busy = false;
 table.addEventListener('click', function (event) {
  var button = event.target;
  var operation = button.getAttribute('data-operation');
  if (!operation || busy) return;
  var row = button.closest('tr');
  var id = row.getAttribute('data-post-id');
  if (operation === 'delete' && !window.confirm('Delete this post permanently?')) return;
  busy = true;
  button.disabled = true;
  var error = document.getElementById('post-list-error');
  error.textContent = '';
  var request = new XMLHttpRequest();
  var deleting = operation === 'delete';
  var collectionPath = '/api/collections/' + encodeURIComponent(table.getAttribute('data-alias'));
  var path = deleting ? collectionPath + '/posts/' + encodeURIComponent(id) + '/delete' : collectionPath + '/' + operation;
  request.open(deleting ? 'DELETE' : 'POST', path, true);
  request.timeout = 30000;
  function failed() {
   busy = false;
   button.disabled = false;
   error.textContent = 'The operation failed. Please reload the page and try again.';
  }
  request.onerror = request.ontimeout = failed;
  request.onload = function () {
   var success = deleting && request.status === 204;
   if (!deleting && request.status === 200) {
    try {
     var response = JSON.parse(request.responseText);
     success = response.data.length === 1 && response.data[0].id === id && response.data[0].code === 200;
    } catch (e) { success = false; }
   }
   if (success) window.location.reload();
   else failed();
  };
  if (!deleting) request.setRequestHeader('Content-Type', 'application/json');
  request.setRequestHeader('X-CSRF-Token', table.getAttribute('data-csrf-token'));
  request.send(deleting ? null : JSON.stringify([{id: id}]));
 });
})();
