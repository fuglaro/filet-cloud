package main

import (
	"archive/zip"
	"encoding/json"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

/**
 * Estabilishes, and returns a sftp connection.
 * Caller is expected to close both,
 * unless an error is returned.
 */
func sftpConnect(r *http.Request) (*ssh.Client, *sftp.Client, error) {
	user, pass, _ := r.BasicAuth()
	config := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(pass)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // trust localhost
	}
	sshConn, err := ssh.Dial("tcp", "localhost:22", config)
	if err != nil {
		return nil, nil, err
	}
	// create new SFTP sftp
	sftp, err := sftp.NewClient(sshConn)
	if err != nil {
		sshConn.Close()
		return nil, nil, err
	}
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
func check(w http.ResponseWriter, e error) bool {
	if e != nil {
		http.Error(w, e.Error(), http.StatusForbidden)
	}
	return e != nil
}

/**
 * Responds to the Http query by performing actions related to the
 * path provided by the 'path' query option.
 */
func urlHandler(w http.ResponseWriter, r *http.Request) {
	/* Ensure authentiction is successfull and get storage connection */
	sshConn, sftp, err := sftpConnect(r)
	if err != nil {
		w.Header().Add("WWW-Authenticate", `Basic realm="Filet Cloud Login"`)
		http.Error(w, "", http.StatusUnauthorized)
		return
	}
	defer sftp.Close()
	defer sshConn.Close()

	path := r.URL.Query().Get("path")
	switch url := r.URL.Path; {

	/*
	 * Redirect to the home page.
	 */
	case url == "/":
		if os.Getenv("FILETCLOUDDIR") == "" {
			http.Redirect(w, r, "/browse:/", http.StatusSeeOther)
			break
		}
		user, _, _ := r.BasicAuth()
		http.Redirect(w, r, "/browse:/"+os.Getenv("FILETCLOUDDIR")+"/"+user+"/", http.StatusSeeOther)

	/*
	 * Serve a web viewer or editor for the subsequent path.
	 */
	case strings.HasPrefix(url, "/open") || strings.HasPrefix(url, "/preview"):
		url, path, _ = strings.Cut(url, ":")
		if strings.Count(url, "/") > 1 {
			// fallback to generic viewer, if misc media
			if strings.HasSuffix(url, "/misc/media") {
				http.ServeFile(w, r, "static/open/fallback.html")
				break

			}
			// Extact the mime type from the url, ensuring a secure path component.
			mime := strings.SplitN(strings.Replace(url, "/../", "/./", -1), "/", 4)
			loader := "static/open/" + mime[2] + "/" + strings.TrimSuffix(mime[3], ":") + ".html"
			http.ServeFile(w, r, loader)
			break
		}

		// attempt to load file extension based viewer
		loader := "static/open/ext" + filepath.Ext(path) + ".html"
		_, err := os.Stat(loader)
		if err == nil {
			http.ServeFile(w, r, loader)
			break
		}
		// detect the mime type of the file to find a viewer
		contents, err := sftp.Open(path)
		if check(w, err) {
			return
		}
		defer contents.Close()
		buffer := make([]byte, 512) /* 512 bytes is enough to catch headers */
		n, err := contents.Read(buffer)
		mime := strings.Split(http.DetectContentType(buffer[:n]), ";")[0]
		// check if we should fallback to generic viewer
		_, err = os.Stat("static/open/" + mime + ".html")
		if err != nil {
			mime = "misc/media"
		}
		http.Redirect(w, r, url+"/"+mime+":/"+path, http.StatusSeeOther)

	/*
	 * Return the contents of the directory identified by the 'path'
	 * query parameter.
	 * Entries include whether it is a file, and the name. E.g:
	 * ?path=/foo -> [[true, "file1"], [false, "dir1"]]
	 */
	case url == "/dir":
		// find contents of the directory
		contents, err := sftp.ReadDir(path)
		if check(w, err) {
			return
		}
		// build json export
		entries := make([][2]interface{}, len(contents)+1)
		entries[0] = [2]interface{}{false, ".."}
		for i, c := range contents {
			entries[i+1] = [2]interface{}{!c.IsDir(), c.Name()}
		}
		json.NewEncoder(w).Encode(entries)

	/*
	 * Retrieves a file and sends it to the client.
	 * The 'path' query parameter identifies the file to send.
	 */
	case url == "/file":
		// stream the file contents
		contents, err := sftp.Open(path)
		if check(w, err) {
			return
		}
		defer contents.Close()
		http.ServeContent(w, r, filepath.Base(path), time.Time{}, contents)

	/*
	 * Creates a new directory on the server.
	 * The 'path' query parameter identifies the directory to make.
	 */
	case url == "/newdir":
		err = sftp.Mkdir(r.URL.Query().Get("path"))
		check(w, err)

	/*
	 * Creates a new file on the server.
	 * The 'path' query parameter identifies the file name to make.
	 */
	case url == "/newfile":
		dest, err := sftp.Create(r.URL.Query().Get("path"))
		defer dest.Close()
		check(w, err)

	/*
	 * Deletes a file or a folder including all contents.
	 * The 'path' query parameter identifies what to delete.
	 * Folders should be terminated with a '/'
	 * On any error, it will stop the deletion process and bail.
	 */
	case url == "/remove":
		if path[len(path)-1:] != "/" {
			// Delete the file
			err = sftp.Remove(path)
			if check(w, err) {
				return
			}
			break
		}
		// Handle folder deletion
		walk := sftp.Walk(path)
		// First delete all files and collect directories
		var dirs []string
		for walk.Step() {
			if check(w, walk.Err()) {
				return
			}
			if walk.Stat().IsDir() {
				dirs = append(dirs, walk.Path())
				continue
			}
			err = sftp.Remove(walk.Path())
			if check(w, err) {
				return
			}
		}
		// Then delete all the dirs (in reverse order)
		for i := range dirs {
			err = sftp.Remove(dirs[len(dirs)-1-i])
			if check(w, err) {
				return
			}
		}

	/*
	 * Move or rename a file or directory on the server.
	 * The 'path' query parameter identifies what to change.
	 * The 'to' query parameter identifies the new name or location.
	 * Both parameters should be full paths.
	 */
	case url == "/rename":
		err = sftp.Rename(path, r.URL.Query().Get("to"))
		check(w, err)

	/*
	 * Serve a thumbnail image of the file.
	 * This does not support all formats.
	 */
	case url == "/thumb":
		ppath := strings.Replace(path, "'", "\\'", -1)
		cmd := "ffmpeg -i '" + ppath + "' -q:v 16 -vf scale=240:-1 -update 1 -f image2 -"
		session, err := sshConn.NewSession()
		if check(w, err) {
			return
		}
		defer session.Close()
		session.Stdout = w
		_ = session.Run(cmd)

	/*
	 * Upload files to the server.
	 * The path specified is the directory to upload to.
	 * Files are recieved from a 'files[]' form parameter
	 * sent in the body of the request.
	 */
	case url == "/upload":
		err := r.ParseMultipartForm(100 << 20) // 100MB in memory
		if check(w, err) {
			return
		}
		for _, file := range r.MultipartForm.File["files[]"] {
			// upack uploaded file
			source, err := file.Open()
			if check(w, err) {
				return
			}
			defer source.Close()
			// create new file on server
			dest, err := sftp.Create(path + "/" + file.Filename)
			if check(w, err) {
				return
			}
			defer dest.Close()
			// copy contents of uploaded file to the server
			_, err = io.Copy(dest, source)
			if check(w, err) {
				return
			}
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
	case url == "/zip":
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
				if check(w, walk.Err()) {
					return
				}
				if walk.Stat().IsDir() {
					continue
				}
				files = append(files, walk.Path())
			}
		}
		// find the common prefix of all paths
		prefix := ""
		if len(files) > 0 {
			prefix = filepath.Dir(files[0])
		}
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
			if check(w, err) {
				return
			}
			defer contents.Close()
			// add the file to the zip with the relative subpath
			filein, err := zipper.Create(path[len(prefix)+1:])
			if check(w, err) {
				return
			}
			// copy the contents of the file into the zip
			_, err = io.Copy(filein, contents)
			if check(w, err) {
				return
			}
		}

	// Bad endpoint error handling.
	default:
		http.Error(w, "bad endpoint", http.StatusForbidden)
	}
}

func main() {
	http.HandleFunc("/", urlHandler)
	http.HandleFunc("/browse:/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/browse.html")
	})
	http.Handle("/favicon.ico", http.FileServer(http.Dir("static")))
	http.Handle("/static/", http.StripPrefix("/static/",
		http.FileServer(http.Dir("static"))))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
