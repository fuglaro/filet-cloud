package main

import (
	"archive/zip"
	"encoding/json"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

/**
 * Estabilises, and returns a sftp connection.
 */
func sftpConnect(r *http.Request) (*ssh.Client, *sftp.Client, error) {
	user, pass, _ := r.BasicAuth()
	config := &ssh.ClientConfig {
		User: user,
		Auth: []ssh.AuthMethod { ssh.Password(pass), },
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	// estabish ssh connection
	                         //TODO "localhost:22"
	sshConn, err := ssh.Dial("tcp", "192.168.0.18:22", config)
	if err != nil { return nil, nil, err }
	// create new SFTP sftp
	sftp, err := sftp.NewClient(sshConn)
	if err != nil { return nil, nil, err }
	return sshConn, sftp, nil
}

/**
 * Checks if the passesd in error has a value and, if it does,
 * an StatusForbidden error is returned.
 * For the purposes of this webserver, where we are exposing
 * files via SFTP, assuming any error relates to a permission
 * issue, is sufficient.
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
	if check(w, err) { return }
	defer sftp.Close()
	defer sshConn.Close()

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
		contents, err := sftp.ReadDir(r.URL.Query().Get("path"))
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
		path := r.URL.Query().Get("path")
		contents, err := sftp.OpenFile(path, os.O_RDONLY)
		if check(w, err) { return }
		http.ServeContent(w, r, path, time.Time{}, contents)

		/*
		 * Creates a new directory on the server.
		 * The 'path' query parameter identifies the directory to make.
		 */
		case "/newdir":
		err = sftp.Mkdir(r.URL.Query().Get("path"))
		if check(w, err) { return }

		/*
		 * Move or rename a file or directory on the server.
		 * The 'path' query parameter identifies what to change.
		 * The 'to' query parameter identifies the new name or location.
		 * Both parameters should be full paths.
		 */
		case "/rename":
		err = sftp.Rename(r.URL.Query().Get("path"), r.URL.Query().Get("to"))
		if check(w, err) { return }

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
			dest, err := sftp.Create(r.URL.Query().Get("path")+"/"+file.Filename)
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
		// prepare the response as a zip file
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", "attachment; filename=\"down.zip\"")
		// start making the zip file
		zipper := zip.NewWriter(w)
		defer zipper.Close()
		for _, path := range files {
			// get the contents of the file from the server
			contents, err := sftp.OpenFile(path, os.O_RDONLY)
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
