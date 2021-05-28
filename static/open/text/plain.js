/**
 * View text file content
 * and allow editing and saving back to the server.
 */
async function load(path) {
	// Load the contents from the server
	res = await fetch(`/file?path=${encodeURIComponent(path)}`)
	text = document.getElementById("text")
	text.value = res.ok ? await res.text() : ""
	document.getElementById("wait").style.display = "none"
	document.body.style.margin = 0
	document.body.style.height = "100vh"

	// Prepare the saving mechanism
	upload = document.getElementById('upload')
	upload.onclick = ()=> {
		uploadForm = new FormData()
		filename = path.split('/').pop()
		uploadForm.append("files[]", new File([new Blob([text.value])], filename))
		dir = path.replace(/[^/]+$/g, '')
		fetch(`upload?path=${encodeURIComponent(dir)}`,
			{method: 'POST', body: uploadForm}).then(async r=> {
			if (r.ok) upload.style.display = "none"
			else alert('Error: ' + await r.text()) })
	}

	// Attach the save callback.
	text.oninput = ()=>upload.style.display = "inline"
}
