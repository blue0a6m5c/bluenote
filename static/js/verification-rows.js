/*
 * Copyright © 2026 Joseph Quigley.
 * Copyright © 2026 BlueNote contributors.
 *
 * This file is part of WriteFreely.
 *
 * WriteFreely is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License, included
 * in the LICENSE file in this source code package.
 */

// Adds and removes rel="me" fields on the blog settings page. Existing rows
// remain editable without JavaScript; scripting is only needed to add or
// remove rows interactively.
(function() {
	var maxLinks = 5;
	var container = document.getElementById('verification-links');
	var addButton = document.getElementById('add-verification');
	if (!container || !addButton) {
		return;
	}

	function rows() {
		return container.getElementsByClassName('verification-row');
	}

	function syncAddButton() {
		addButton.disabled = rows().length >= maxLinks;
	}

	function newRow() {
		var row = document.createElement('div');
		row.className = 'verification-row';

		var input = document.createElement('input');
		input.type = 'text';
		input.name = 'verification_link_row';
		input.placeholder = 'https://writing.exchange/@writefreely';

		var remove = document.createElement('button');
		remove.type = 'button';
		remove.className = 'remove-verification';
		remove.title = 'Remove this link';
		remove.setAttribute('aria-label', 'Remove this verification link');
		remove.appendChild(document.createTextNode('\u00d7'));

		row.appendChild(input);
		row.appendChild(remove);
		return row;
	}

	addButton.addEventListener('click', function() {
		if (rows().length >= maxLinks) {
			return;
		}
		var row = newRow();
		container.appendChild(row);
		row.getElementsByTagName('input')[0].focus();
		syncAddButton();
	});

	container.addEventListener('click', function(e) {
		if (!e.target || e.target.className !== 'remove-verification') {
			return;
		}
		var row = e.target.parentNode;
		if (rows().length > 1) {
			container.removeChild(row);
		} else {
			row.getElementsByTagName('input')[0].value = '';
		}
		syncAddButton();
	});

	syncAddButton();
})();
