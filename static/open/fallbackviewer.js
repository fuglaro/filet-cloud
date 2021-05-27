
/**
 * Displays the file contents as best it can using inbuilt
 * browser image and video elements, otherwise fall back to
 * a basic page. Video elements will also support compatible audio.
 */
function load(path, mime) {
	enc = encodeURIComponent
	/* Attempts to load the selected file on a given html element.
	Intended for "img" and "video" */
	function tryElement(element, fallback) {
		doc = document.createElement(element)
		doc.controls = "controls"
		doc.onload = doc.oncanplay = ()=>document.body.replaceChildren(doc)
		doc.onerror = fallback
		doc.src = `/file?path=${enc(path)}`
	}
	// Hit the fallback approach attempting to load the content.
	tryElement("img", ()=>tryElement("video", ()=> {
		document.getElementById("download").href = `/file?path=${enc(path)}`
		document.getElementById("fail").style.visibility = "visible"
		document.getElementById("details").innerText = `${path}\n(${mime})`
	}))
}
