/**
* Interface for all operations interacting with the storage and server.
*/
class Storage {

	/**
	* Return the current path determined by the url.
	*/
	path() {
		return decodeURI(window.location.pathname.replace(/^\/.+:\//, '/'))
	}

	/**
	* Returns whether the display should be in preview mode.
	*/
	isPreview() {
		return window.location.pathname.startsWith("/preview:")
	}

	/**
	* Retrieve the contents of a file as a string.
	* On error returns an empty string.
	*/
	async readFile(path) {
		let r = await fetch(`/file?path=${encodeURIComponent(path)}`)
		if (!r.ok) throw new Error("FileReadError: " + await r.text())
		return await r.text()
	}

	/**
	* Retrieve a link to the file for streaming the contents.
	*/
	fileLink(path) {
		return `/file?path=${encodeURIComponent(path)}`
	}

	/**
	* Retrieve a link to a generated thumbnail (if supported) for a file.
	*/
	thumbLink(path) {
		return `/thumb?path=${encodeURIComponent(path)}`
	}

	/**
	* Retrieve a link to a zip file of all the given paths.
	*/
	zipLink(paths) {
		return '/zip?' + paths.map(p => `path=${encodeURIComponent(p)}`).join('&')
	}

	/**
	* Upload files (given as a list of File objects),
	* potentially overwriting existing files on the server,
	* into the specified directory.
	*/
	async writeFiles(directory, files) {
		let uploadForm = new FormData()
		for (var i = 0; i < files.length; i++) uploadForm.append("files[]", files[i])
		let r = await fetch(`/upload?path=${encodeURIComponent(directory)}`,
			{ method: 'POST', body: uploadForm })
		if (!r.ok) throw new Error("FileWriteError: " + await r.text())
		return await r.text()
	}

	/**
	* Retrieve the contents of a directory as a list of strings,
	* where child directories end in (forward) slashes.
	* Results are returned sorted by name but with folder first.
	*/
	async listDir(path) {
		let r = await fetch(`/dir?path=${encodeURIComponent(path)}`)
		if (!r.ok) throw new Error("ListDirError: " + await r.text())
		r = await r.json()
		r.sort()
		return r.map(([isFile, name]) => name + (isFile ? "" : "/"))
	}

	/**
	* Create a new directory with the given path.
	*/
	async newDir(path) {
		let r = await fetch(`/newdir?path=${encodeURIComponent(path)}`)
		if (!r.ok) throw new Error("NewDirError: " + await r.text())
		return await r.text()
	}

	/**
	* Create a new file with the given path.
	*/
	async newFile(path) {
		let r = await fetch(`/newfile?path=${encodeURIComponent(path)}`)
		if (!r.ok) throw new Error("NewFileError: " + await r.text())
		return await r.text()
	}
	
	/**
	* Rename or move a file or folder.
	*/
	async rename(oldpath, newpath) {
		let r = await fetch(`/rename?path=${encodeURIComponent(oldpath)
			}&to=${encodeURIComponent(newpath)}`)
		if (!r.ok) throw new Error("MoveError: " + await r.text())
		return await r.text()
	}


  /**
  * Delete the file or folder at the given path.
	*/
	async remove(path) {
		let r = await fetch(`/remove?path=${encodeURIComponent(path)}`)
		if (!r.ok) throw new Error("RemoveError: " + await r.text())
		return await r.text()
	}
}
const storage = new Storage();
