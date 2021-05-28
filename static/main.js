var cart = []
let curPath = ()=>document.getElementById('path').innerText
let cwd = ()=>curPath().replace(/[^/]+$/g, '')
let cartMode = ()=>document.getElementById('cart').style.filter
let enc = encodeURIComponent
// Basic error handler for queries to the server.
let check = okayFn=> async r=> {
	if (r.ok) r.json().then(j=> okayFn(j)).catch(e=> okayFn(null))
	else alert('Error: ' + await r.text()) }

/**
 * Switches between cart and normal selection modes.
 * If the current path is a file and not yet in the cart,
 * it will just add the file to the cart.
 */
function cartButton() {
	if (!cartMode() && !curPath().endsWith('/') && !cart.includes(curPath())) {
		// If a file is selected, add to the cart without switching to cart mode.
		cartSel(curPath())
	} else {
		document.getElementById('cart').style.filter =
			cartMode()?'':'drop-shadow(0.2rem 0.2rem 0.2rem indigo)'
		// redraw the cart or contents depending on the resulting cart mode.
		cartMode() ? cartSel() : (!curPath().endsWith("/")?load(curPath()):0)
	}
}

/**
 * Toggle the presence of the given path in the cart, if provided.
 * Refresh the display of the cart, updating the size,
 * and listing the contents if the cart mode is active.
 */
 function cartSel(path) {
	if (path) {
		if (cart.indexOf(path) < 0) cart.push(path)
		else cart.splice(cart.indexOf(path), 1)
	}
	// Update the cart size display
	document.getElementById('cartlen').innerText = cart.length.toString()
	// Show the cart entries, if in cart mode
	if (cartMode())
		document.getElementById('data').replaceChildren(...cart.map(c=> {
			nib = document.createElement("h2")
			nib.onclick = ()=>cartSel(c) // remove the cart item if clicked
			nib.innerText = `${!c.endsWith('/')?'\u{1F4C4}':'\u{1F4C2}'} ${c}`
			return nib
		}))
}

/**
 * Navigate to, and show, a file or directory.
 * For folders, path should be terminated in a "/".
 * Also sets the current files or directories for future actions.
 * Updates the folder area with interactive tree navigation.
 * Displays file contents as best it can.
 */
function nav(path) {
	document.getElementById('path').innerText = path
	document.getElementById('dir').replaceChildren()
	document.getElementById('data').replaceChildren()
	// turn off cart selection mode
	document.getElementById('cart').style.filter = ''
	// Refresh the contents of the directory by querying the server
	fetch(`dir?path=${enc(cwd())}`).then(check(r=> {r.sort()
		// Display the directory contents
		document.getElementById('dir').replaceChildren(...r.map(([f, n])=> {
			nib = document.createElement("h2")
			nib.onclick = ()=>{
				p = cwd() + n + (f?"":"/")
				cartMode()?cartSel(p):(f?load(p):nav(p))
			}
			nib.innerText = `${f?'\u{1F4C4}':'\u{1F4C2}'} ${n}`
			return nib
		}))
	}))
	// Open any file contents in the data pane
	if (!path.endsWith("/")) load(path)
}

/**
 * Open file contents in the data pane.
 * Also sets the current file for future actions.
 */
 function load(path) {
	document.getElementById('path').innerText = path
	doc = document.createElement("iframe")
	doc.frameBorder = "0"
	doc.src = `open?path=${enc(path)}`
	document.getElementById('data').replaceChildren(doc)
}

/**
 * Make a directory on the server.
 * The user is prompted for the name of the new directory and
 * it is made inside the current directory.
 */
function makedir() {
	if (!(newDir = prompt("New Folder:", ""))) return
	fetch(`newdir?path=${enc(cwd() + newDir)}`)
	.then(check(r=> nav(cwd()))) // refresh directory
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
	.then(check(r=> {}))
}

/**
 * Rename a file or directory on the server.
 * The user is prompted for the new name.
 */
function rename() {
	suffix = curPath().endsWith('/')?'/':''
	folder = curPath().replace(/[^/]+\/?$/g, '')
	oldName = curPath().replace(/\/?$/g, '').split('/').pop()
	if (!(newName = prompt("Rename:", oldName)) || newName == oldName) return
	fetch(`rename?path=${enc(folder + oldName)}&to=${enc(folder + newName)}`)
	.then(check(r=> nav(folder + newName + suffix)))
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
		// remove moved items from the cart and refresh directory
		.then(check(r=> {cartSel(item); nav(cwd())}))
	})
}
