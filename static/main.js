
var cart = []
let curPath = ()=>document.getElementById('path').innerText
let cwd = ()=>curPath().replace(/[^/]+$/g, '')
let cwdParent = ()=>cwd().replace(/[^/]+\/$/g, '')
let dataEl = document.getElementById('data')
let sel = document.getElementById('cart').style
let enc = encodeURIComponent
// Basic error handler for queries to the server.
let check = okayFn=> async r=> {
	if (r.ok) r.json().then(j=> okayFn(j)).catch(e=> okayFn(null))
	else alert('Error: ' + await r.text()) }

/**
 * Switches between cart and normal selection modes.
 * If the current path is a file, it will just add the,
 * file to the cart or switch to the cart, if the file is
 * already in the cart.
 */
function cartSel() {
	sel.filter = sel.filter ? '' : 'drop-shadow(0.2rem 0.2rem 0.2rem indigo)'
	if (sel.filter && !curPath().endsWith('/') && !cart.includes(curPath())) {
		// If a file is selected, add to the cart without switching to cart mode.
		nav(curPath())
		sel.filter = ''
	}
	nav(curPath(), 1)
}

/**
 * Navigate to and show a file or directory, or,
 * if cart selection mode is active, show the cart and add and
 * remove from it instead of navigating.
 * For folders, path should be terminated in a "/"
 * Sets the current files or directories for future actions.
 * Updates the folder area with interactive tree navigation.
 * Displays file contents as best it can.
 * @param {bool} forceNav Ignore cart selection mode in navigating
 * @param {bool} refresh Force redraw the working dir contents
 */
function nav(path, forceNav, refresh) {
	// Function to generate directory entry selector elements.
	dirSel = (isFile, name, path)=> {
		nib = document.createElement("h2")
		nib.onclick = ()=>nav(path)
		nib.innerText = `${isFile?'\u{1F4C4}':'\u{1F4C2}'} ${name}`
		return nib}

	// Handle addition and removal from cart if in cart selection mode
	if (sel.filter && !forceNav) {
		index = cart.indexOf(path)
		index < 0 ? cart.push(path) : cart.splice(index, 1)
	}
	document.getElementById('cartlen').innerText = cart.length.toString()
	// Show the cart entries, if it's selected
	dataEl.replaceChildren(...
		sel.filter ? cart.map(c=> dirSel(!c.endsWith('/'), c, c)) : [])

	// Handle directory navigation, or file content display
	if ((refresh || path.endsWith("/")) && (!sel.filter || forceNav)) {
		dir = path.replace(/[^/]+$/g, '')
		// Query the contents of the directory
		fetch(`dir?path=${enc(dir)}`)
		.then(check(r=> { r.sort()
			// Update the location to this directory if we aren't just refreshing
			if (path.endsWith("/"))
				document.getElementById('path').innerText = path
			// Display the directory contents
			document.getElementById('dir').replaceChildren(...r.map(([f, n])=>
				dirSel(f, n, dir + n + (f?"":"/"))))}))
	}

	// Open any file contents in the data pane
	if (!path.endsWith("/") && !sel.filter) {
		document.getElementById('path').innerText = path
		doc = document.createElement("iframe")
		doc.frameBorder = "0"
		doc.src = `open?path=${enc(path)}`
		dataEl.replaceChildren(doc)
	}
}

/**
 * Make a directory on the server.
 * The user is prompted for the name of the new directory and
 * it is made inside the current directory.
 */
function makedir() {
	if (!(newDir = prompt("New Folder:", ""))) return
	fetch(`newdir?path=${enc(cwd() + newDir)}`)
	.then(check(r=> nav(cwd())))
}

/**
 * Start uploading of files to the server.
 * Files are uploaded into the current directory.
 * This is called after the file selection popup has been loaded
 * and files have been selected.
 */
 function send() {
	uploadEl = document.getElementById('upload')
	uploadForm = new FormData()
	for (i = 0; i < uploadEl.files.length; i++)
		uploadForm.append("files[]", uploadEl.files[i])
	fetch(`upload?path=${enc(cwd())}`, {method: 'POST', body: uploadForm})
	.then(check(r=> nav(cwd())))
}

/**
 * Rename a file or directory on the server.
 * The user is prompted for the new name.
 */
function rename() {
	folder = curPath().replace(/[^/]+\/?$/g, '')
	oldName = curPath().replace(/\/?$/g, '').split('/').pop()
	if (!(newName = prompt("Rename:", oldName)) || newName == oldName) return
	fetch(`rename?path=${enc(folder + oldName)}&to=${enc(folder + newName)}`)
	.then(check(r=> nav(folder + newName + (folder==cwd()?'':'/'), 0, 1)))
}

/**
 * Open a file in a new window for viewing.
 */
function tabopen() {
	if (curPath() == cwd()) return // ignore folders
	window.open(`/open?path=${enc(curPath())}`, "_blank")
}

/**
 * Download a file or a zip of multiple items.
 * Supports downloading the selected file (if a file),
 * a single file from the cart, or zipping multiple files
 * and folders in the cart.
 */
function download() {
	if (cart.length) paths = cart
	else if (!curPath().endsWith('/')) paths = [curPath()]
	else return alert('Please select a file, or items with the cart.')
	downloadEl = document.getElementById('download')
	endpoint = (cart.length == 1)?'file':'zip'
	downloadEl.href = endpoint+'?'+paths.map(p=> `path=${enc(p)}`).join('&')
	downloadEl.click()
}

/**
 * Moves everything in the cart to the current folder.
 */
function move() {
	if (!cart.length) return alert('Please select items with the cart.')
	confirmMsg = `Move ${cart.length} item${cart.length>1?'s':''} to ${cwd()} ?`
	if (!confirm(confirmMsg)) return
	/* Loop through the paths in the cart sorted backwards so sub-items
		 are moved before parent items */
	cart.sort().reverse().forEach(item => {
		dest = cwd() + item.replace(/\/?$/g, '').split('/').pop()
		fetch(`rename?path=${enc(item)}&to=${enc(dest)}`)
		.then(check(r=> {
			cart.splice(cart.indexOf(item), 1)
			nav(cwd(), 1, 1)
		}))
	})
}
