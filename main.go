package main

import (
	"archive/zip"
	"encoding/json"
	"os/exec"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"html/template"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

/**
 * Estabilises, and returns a sftp connection.
 * Caller is expected to close both,
 * unless an error is returned.
 */
func sftpConnect(r *http.Request) (*ssh.Client, *sftp.Client, error) {
	user, pass, _ := r.BasicAuth()
	config := &ssh.ClientConfig {
		User: user,
		Auth: []ssh.AuthMethod { ssh.Password(pass), },
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	sshConn, err := ssh.Dial("tcp", "localhost:22", config)
	if err != nil { return nil, nil, err }
	// create new SFTP sftp
	sftp, err := sftp.NewClient(sshConn)
	if err != nil { sshConn.Close(); return nil, nil, err }
	return sshConn, sftp, nil
}

/**
 * Checks if the passed in error has a value and, if it does,
 * a StatusForbidden error is provided to the response.
 * For the purposes of this webserver, where we are exposing
 * files via SFTP, assuming any error relates to a permission
 * issue, is sufficient. Breaks some HTTP rules but its nice and simple.
 * Returns whether the error had a value.
 */
func check(w http.ResponseWriter, e error) (bool) {
	if e != nil { http.Error(w, e.Error(), http.StatusForbidden) }
	return e != nil
}

/**
 * Resonds to the Http query by performing actions related to the
 * path provided by the 'path' query option.
*/
func urlHandler(w http.ResponseWriter, r *http.Request) {
	/* Ensure authentiction is successfull and get connection */
	sshConn, sftp, err := sftpConnect(r)
	if err != nil {
		w.Header().Add("WWW-Authenticate", `Basic realm="Filet Cloud Login"`)
		http.Error(w, "", http.StatusUnauthorized)
		return;
	}
	defer sftp.Close()
	defer sshConn.Close()

	path := r.URL.Query().Get("path")
	switch r.URL.Path {
		/*
		 * Serve the main page
		 */
		case "/":
		user, _, _ := r.BasicAuth()
		page, err := template.ParseFiles("template/main.html")
		if check(w, err) { return }
		page.Execute(w, struct{P string}{P:"/mnt/usb/filetclouddata/"+user+"/"})

		/*
		 * Return the contents of the directory identified by the 'path'
		 * query parameter.
		 * Entries include whether it is a file, and the name. E.g:
		 * ?path=/foo -> [[true, "file1"], [false, "dir1"]]
		 */
		case "/dir":
		// find contents of the directory
		contents, err := sftp.ReadDir(path)
		if check(w, err) { return }
		// build json export
		entries := make([][2]interface{}, len(contents))
		for i, c := range contents {
			entries[i] = [2]interface{}{!c.IsDir(), c.Name()}
		}
		json.NewEncoder(w).Encode(entries)

		/*
		 * Retrieves a file and sends it to the client.
		 * The 'path' query parameter identifies the file to send.
		 */
		case "/file":
		// stream the file contents
		contents, err := sftp.Open(path)
		if check(w, err) { return }
		defer contents.Close()
		http.ServeContent(w, r, filepath.Base(path), time.Time{}, contents)

		/*
		 * Creates a new directory on the server.
		 * The 'path' query parameter identifies the directory to make.
		 */
		case "/newdir":
		err = sftp.Mkdir(r.URL.Query().Get("path"))
		if check(w, err) { return }

		/*
		 * Serve a web viewer (and editor if supported)
		 * for the path provided.
		 * @param path The path of the content to display.
		 */
		case "/open":
		// attempt to load file extension based viewer
		page, err := template.ParseFiles(
			"template/open/ext"+filepath.Ext(path)+".html")
		if err == nil {
			page.Execute(w, struct{P string; M string}{P:path, M:filepath.Ext(path)})
			break
		}
		// detect the mime type of the file to find a viewer
		contents, err := sftp.Open(path)
		if check(w, err) { return }
		defer contents.Close()
		buffer := make([]byte, 512) /* 512 bytes is enough to catch headers */
		n, err := contents.Read(buffer)
		mime := strings.Split(http.DetectContentType(buffer[:n]), ";")[0]
		// attempt to load a mime viewer
		page, err = template.ParseFiles("template/open/"+mime+".html")
		if err == nil {
			page.Execute(w, struct{P string; M string}{P:path, M:mime})
			break
		}
		// fallback to generic viewer
		page, err = template.ParseFiles("template/open/fallback.html")
		if check(w, err) { return }
		page.Execute(w, struct{P string; M string}{P:path, M:mime})

		/*
		 * Deletes a file or a folder including all contents.
		 * The 'path' query parameter identifies what to delete.
		 * Folders should be terminated with a '/'
		 * On any error, it will stop the deletion process and bail.
		 */
		case "/remove":
		if path[len(path)-1:] != "/" {
			// Delete the file
			err = sftp.Remove(path)
			if check(w, err) { return }
			break
		}
		// Handle folder deletion
		walk := sftp.Walk(path)
		// First delete all files and collect directories
		var dirs []string
		for walk.Step() {
			if check(w, walk.Err()) { return }
			if walk.Stat().IsDir() {
				dirs = append(dirs, walk.Path())
				continue
			}
			err = sftp.Remove(walk.Path())
			if check(w, err) { return }
		}
		// Then delete all the dirs (in reverse order)
		for i := range dirs {
			err = sftp.Remove(dirs[len(dirs)-1-i])
			if check(w, err) { return }
		}

		/*
		 * Move or rename a file or directory on the server.
		 * The 'path' query parameter identifies what to change.
		 * The 'to' query parameter identifies the new name or location.
		 * Both parameters should be full paths.
		 */
		case "/rename":
		err = sftp.Rename(path, r.URL.Query().Get("to"))
		if check(w, err) { return }

		/*
		 * Serve a thumbnail image of the file.
		 * This does not support all formats.
		 */
		case "/thumb":
		contents, err := sftp.Open(path)
		if check(w, err) { return }
		defer contents.Close()
		cmd := exec.Command("ffmpeg", "-i", "-", "-vframes", "1", "-f", "image2",
			"-vf", "scale=-1:240", "pipe:1")
		cmd.Stdin = contents
		cmd.Stdout = w // browsers rock at detecting its a jpeg
		_ = cmd.Run()

		/*
		 * Upload files to the server.
		 * The path specified is the directory to upload to.
		 * Files are recieved from a 'files[]' form parameter
		 * sent in the body of the request.
		 */
		case "/upload":
		err := r.ParseMultipartForm(100 << 20) // 100MB in memory
		if check(w, err) { return }
		for _, file := range r.MultipartForm.File["files[]"] {
			// upack uploaded file
			source, err := file.Open()
			if check(w, err) { return }
			defer source.Close()
			// create new file on server
			dest, err := sftp.Create(path+"/"+file.Filename)
			if check(w, err) { return }
			defer dest.Close()
			// copy contents of uploaded file to the server
			_, err = io.Copy(dest, source)
			if check(w, err) { return }
		}

		/*
		 * Generate a zip file from a list of files and directories
		 * for downloading.
		 * Files and directories are specified by multiple "path"
		 * query parameters.
		 * Streams the file contents into a zip stream and to the client.
		 *   - SFTPfiles }=> zipper -> http
		 * Note: paths are assumed to be absolute.
		 */
		case "/zip":
		var files []string
		// walk the paths expanding to the list of files inside
		for _, path := range r.URL.Query()["path"] {
			if path[len(path)-1:] != "/" {
				// include files
				files = append(files, path)
				continue
			}
			// expand directories to the files inside
			walk := sftp.Walk(path)
			for walk.Step() {
				if check(w, walk.Err()) { return }
				if walk.Stat().IsDir() { continue }
				files = append(files, walk.Path())
			}
		}
		// find the common prefix of all paths
		prefix := ""
		if len(files)>0 { prefix = filepath.Dir(files[0]) }
		for _, file := range files {
			for prefix != file[:len(prefix)] {
				prefix = filepath.Dir(prefix)
			}
		}
		// start making the zip file
		zipper := zip.NewWriter(w)
		defer zipper.Close()
		for _, path := range files {
			// get the contents of the file from the server
			contents, err := sftp.Open(path)
			if check(w, err) { return }
			defer contents.Close()
			// add the file to the zip with the relative subpath
			filein, err := zipper.Create(path[len(prefix)+1:])
			if check(w, err) { return }
			// copy the contents of the file into the zip
			_, err = io.Copy(filein, contents)
			if check(w, err) { return }
		}

		// Bad endpoint error handling.
		default:
		http.Error(w, "bad endpoint", http.StatusForbidden)
	}
}

func main() {
	http.HandleFunc("/", urlHandler)
	http.Handle("/static/",
		http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
