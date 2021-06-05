"use strict"
/**
 * View the md file content with SimpleMDE,
 * and allow editing and saving back to the server.
 */
async function load(path) {
	// Load the contents from the server
	let res = await fetch(`/file?path=${encodeURIComponent(path)}`)
	let md = document.getElementById("md")
	md.value = res.ok ? await res.text() : ""
	document.getElementById("wait").style.display = "none"
	document.body.style.margin = 0
	document.body.style.height = "100vh"

	// Prepare the saving mechanism
	let upload = document.getElementById('upload')
	upload.onclick = ()=> {
		let uploadForm = new FormData()
		let filename = path.split('/').pop()
		uploadForm.append("files[]", new File([new Blob([md.value])], filename))
		let dir = path.replace(/[^/]+$/g, '')
		fetch(`upload?path=${encodeURIComponent(dir)}`,
			{method: 'POST', body: uploadForm}).then(async r=> {
			if (r.ok) upload.style.display = "none"
			else alert('Error: ' + await r.text()) })
	}

	// Show SimpleMDE in preview mode and attach the save callback.
	let simplemde = new SimpleMDE({forceSync: true, status: false})
	simplemde.togglePreview()
	// mobiles don't love fullscreening here.
	try {simplemde.toggleFullScreen()} catch(e){}
	simplemde.codemirror.on("change", ()=>upload.style.display = "inline")

	// Move upload into SimpleMDE's toolbar.
	let firstButton = document.querySelector('.fa')
	firstButton.parentNode.insertBefore(upload, firstButton)
}
