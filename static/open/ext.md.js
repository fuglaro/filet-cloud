/**
 * View the md file content with SimpleMDE,
 * and allow editing and saving back to the server.
 */
async function load(path) {
	// Load the contents from the server
	res = await fetch(`/file?path=${encodeURIComponent(path)}`)
	md = document.getElementById("md")
	md.value = res.ok ? await res.text() : ""
	document.getElementById("wait").style.display = "none"
	document.body.style.margin = 0
	document.body.style.height = "100vh"

	// Prepare the saving mechanism
	upload = document.getElementById('upload')
	upload.onclick = ()=> {
		uploadForm = new FormData()
		filename = path.split('/').pop()
		uploadForm.append("files[]", new File([new Blob([md.value])], filename))
		dir = path.replace(/[^/]+$/g, '')
		fetch(`upload?path=${encodeURIComponent(dir)}`,
			{method: 'POST', body: uploadForm}).then(async r=> {
			if (r.ok) upload.style.display = "none"
			else alert('Error: ' + await r.text()) })
	}

	// Show SimpleMDE in preview mode and attach the save callback.
	simplemde = new SimpleMDE({forceSync: true, status: false})
	simplemde.togglePreview()
	simplemde.codemirror.on("change", ()=>upload.style.display = "inline")
}
