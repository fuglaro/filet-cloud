package main

import (
	"archive/zip"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

var port = "22"
var upgrader = websocket.Upgrader{}

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
	sshConn, err := ssh.Dial("tcp", "localhost:"+port, config)
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

/** rp
 * (realpath)
 * Returns the full path on the filesystem for the given url path.
 */
func rp(path string, r *http.Request) string {
	if os.Getenv("FILETCLOUDDIR") == "" {
		return path
	}
	user, _, _ := r.BasicAuth()
	return filepath.Join(os.Getenv("FILETCLOUDDIR"), user, path)
}

/**
 * Request queries that rely on connection to the storage, and therefore
 * the appropriate authentication.
 */
func secureUrlHandler(w http.ResponseWriter, r *http.Request) {
	/* Ensure authentiction is successfull and get storage connection */
	sshConn, sftp, err := sftpConnect(r)
	if err != nil {
		w.Header().Add("WWW-Authenticate", `Basic realm="Filet Cloud Login"`)
		http.Error(w, "", http.StatusUnauthorized)
		return
	}
	defer sftp.Close()
	defer sshConn.Close()

	components := strings.SplitN(r.URL.Path, ":", 2)
	switch components[0] {

	/*
	 * Redirect to the home page.
	 */
	case "/":
		http.Redirect(w, r, "/browse:/", http.StatusSeeOther)

	/*
	 * Retrieves a file and sends it to the client.
	 * The 'path' query parameter identifies the file to send.
	 */
	case "/file":
		path := rp(components[1], r)
		// stream the file contents
		contents, err := sftp.Open(path)
		if check(w, err) {
			return
		}
		defer contents.Close()
		http.ServeContent(w, r, filepath.Base(path), time.Time{}, contents)

	/*
	 * Serve a thumbnail image of the file.
	 * This does not support all formats.
	 */
	case "/thumb":
		path := rp(components[1], r)
		// Single quoted POSIX command argument input sanitisation,
		// necessary due to needing to travel through the ssh stream.
		ppath := strings.Replace(path, "'", "'\\''", -1)
		cmd := "ffmpeg -i '" + ppath + "' -q:v 16 -vf scale=240:-1 -update 1 -f image2 -"
		session, err := sshConn.NewSession()
		if check(w, err) {
			return
		}
		defer session.Close()
		session.Stdout = w
		_ = session.Run(cmd)

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
		for _, upath := range r.URL.Query()["path"] {
			path := rp(upath, r)
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
		http.Error(w, "bad url", http.StatusForbidden)
	}
}

func connect(w http.ResponseWriter, r *http.Request) {
	// Establish Websocket connection.
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close()
	// Authenticate and establish SSH connection.
	_, user, err := c.ReadMessage()
	if err != nil {
		return
	}
	_, pass, err := c.ReadMessage()
	if err != nil {
		return
	}
	sshConn, err := ssh.Dial("tcp", "localhost:"+port, &ssh.ClientConfig{
		User:            string(user),
		Auth:            []ssh.AuthMethod{ssh.Password(string(pass))},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // trust localhost
	})
	if err != nil {
		return
	}
	defer sshConn.Close()
	// Wrap SSH connection with SFTP interface.
	sftp, err := sftp.NewClient(sshConn)
	if err != nil {
		return
	}
	// Handle messages on established authenticated connection.
	type Msg struct {
		Action string
		Path   string
		To     string
		Id     int
	}
	prepath := os.Getenv("FILETCLOUDDIR")
	if prepath != "" {
		prepath += "/" + string(user) + "/"
	}
	for {
		mtype, r, err := c.NextReader()
		if err != nil {
			return
		}
		if mtype == websocket.BinaryMessage {

			/* Handle file upload.
			 * Unpack the message header to get the path name then store the rest. */
			idbuf := make([]byte, 20)
			if _, err = io.ReadFull(r, idbuf); err != nil {
				return
			}
			id, err := strconv.Atoi(strings.TrimSpace(string(idbuf)))
			if err != nil {
				return
			}
			pathlenbuf := make([]byte, 20)
			if _, err = io.ReadFull(r, pathlenbuf); err != nil {
				return
			}
			pathlen, err := strconv.Atoi(strings.TrimSpace(string(pathlenbuf)))
			if err != nil {
				return
			}
			pathbuf := make([]byte, pathlen)
			if _, err = io.ReadFull(r, pathbuf); err != nil {
				return
			}
			path := strings.TrimSpace(string(pathbuf))
			// create new file on server
			dest, err := sftp.Create(prepath + path)
			if err != nil {
				return
			}
			defer dest.Close()
			// copy contents of uploaded file to the server
			if _, err = io.Copy(dest, r); err != nil {
				return
			}
			if c.WriteJSON(map[string]interface{}{"id": id}) != nil {
				return
			}
			continue
		}

		m := Msg{}
		mstr, err := io.ReadAll(r)
		err = json.Unmarshal(mstr, &m)
		if err != nil {
			return
		}
	action:
		switch m.Action {

		/* Returns the contents of the given directory,
		 * including whether each entry is a file. E.g:
		 * [[true, "file1"], [false, "dir1"]] */
		case "dir":
			// find contents of the directory
			contents, err := sftp.ReadDir(prepath + m.Path)
			if err != nil {
				return
			}
			// build json export
			fs := make([][2]interface{}, len(contents)+1)
			fs[0] = [2]interface{}{false, ".."}
			for i, c := range contents {
				fs[i+1] = [2]interface{}{!c.IsDir(), c.Name()}
			}
			if c.WriteJSON(map[string]interface{}{"id": m.Id, "msg": fs}) != nil {
				return
			}

		// Returns the contents of a file as a binary message.
		case "file":
			// stream the file contents
			contents, err := sftp.Open(prepath + m.Path)
			if err != nil {
				return
			}
			defer contents.Close()
			mw, err := c.NextWriter(websocket.BinaryMessage)
			if err != nil {
				return
			}
			idstr := strconv.Itoa(m.Id)
			if len(idstr) > 20 {
				return
			}
			mw.Write([]byte(idstr))
			mw.Write([]byte(strings.Repeat(" ", 20-len(idstr))))
			if _, err = io.Copy(mw, contents); err != nil {
				return
			}
			if mw.Close() != nil {
				return
			}

		// Returns the mime type of a file.
		case "mime":
			// find the mime type of the file
			contents, err := sftp.Open(prepath + m.Path)
			if err != nil {
				return
			}
			defer contents.Close()
			buffer := make([]byte, 512) /* 512 bytes is enough to catch headers */
			n, err := contents.Read(buffer)
			mime := strings.Split(http.DetectContentType(buffer[:n]), ";")[0]
			if c.WriteJSON(map[string]interface{}{"id": m.Id, "msg": mime}) != nil {
				return
			}

		// Creates a new directory.
		case "newdir":
			err = sftp.Mkdir(prepath + m.Path)
			if c.WriteJSON(map[string]interface{}{"id": m.Id, "err": err}) != nil {
				return
			}

		// Creates a new file.
		case "newfile":
			dest, err := sftp.Create(prepath + m.Path)
			dest.Close()
			if c.WriteJSON(map[string]interface{}{"id": m.Id, "err": err}) != nil {
				return
			}

		/* Move or rename a file or directory.
		 * From the given path "path" to the given path "to". */
		case "rename":
			err = sftp.Rename(prepath+m.Path, prepath+m.To)
			if c.WriteJSON(map[string]interface{}{"id": m.Id, "err": err}) != nil {
				return
			}

		/* Deletes a file or a folder including all contents.
		 * The given "path" identifies what to delete.
		 * Folders should be terminated with a '/'
		 * On errors, it may stop the deletion process and bail. */
		case "remove":
			path := prepath + m.Path
			if path[len(path)-1:] != "/" {
				// Delete the file
				err = sftp.Remove(path)
				if c.WriteJSON(map[string]interface{}{"id": m.Id, "err": err}) != nil {
					return
				}
				break
			}
			// Handle folder deletion
			walk := sftp.Walk(path)
			// First delete all files and collect directories
			var dirs []string
			for walk.Step() {
				if walk.Err() != nil {
					if c.WriteJSON(map[string]interface{}{"id": m.Id, "err": err}) != nil {
						return
					}
					break action
				}
				if walk.Stat().IsDir() {
					dirs = append(dirs, walk.Path())
					continue
				}
				if sftp.Remove(walk.Path()) != nil {
					if c.WriteJSON(map[string]interface{}{"id": m.Id, "err": err}) != nil {
						return
					}
					break action
				}
			}
			// Then delete all the dirs (in reverse order)
			for i := range dirs {
				if sftp.Remove(dirs[len(dirs)-1-i]) != nil {
					if c.WriteJSON(map[string]interface{}{"id": m.Id, "err": err}) != nil {
						return
					}
					break action
				}
			}
			if c.WriteJSON(map[string]interface{}{"id": m.Id}) != nil {
				return
			}

		default:
			return
		}
	}
}

func main() {
	p := os.Getenv("FILETCLOUDSSHPORT")
	if p != "" {
		port = p
	}
	http.HandleFunc("/connect", connect)
	http.HandleFunc("/", secureUrlHandler)
	browse := func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/browse.html")
	}
	http.HandleFunc("/browse:/", browse)
	http.HandleFunc("/open:/", browse)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.Handle("/favicon.ico", http.FileServer(http.Dir("static")))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
