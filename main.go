package main

import (
	"encoding/json"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

/* Estabilises, and returns a sftp connection.
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

/* Checks if the passesd in error has a value and, if it does,
 * an StatusForbidden error is returned.
 * For the purposes of this webserver, where we are exposing
 * files via SFTP, assuming any error relates to a permission
 * issue, is sufficient.
 * Returns whether the error had a value.
 */
func resError(w http.ResponseWriter, e error) (bool) {
	if e != nil { http.Error(w, e.Error(), http.StatusForbidden) }
	return e != nil
}

/* Resonds to the Http query by performing actions related to the
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
	if resError(w, err) { return }
	defer sftp.Close()
	defer sshConn.Close()

	switch r.URL.Path {
		/*
		 * Serve the main page
		 */
		case "/":
		user, _, _ := r.BasicAuth()
		page, err := template.ParseFiles("default.html")
		if resError(w, err) { return }
		page.Execute(w, struct{P string}{P:"/mnt/usb/filetclouddata/"+user})

		/*
		 * Return the contents of the directory identified by the 'path'
		 * query parameter.
		 * Entries include whether it is a file, and the name. E.g:
		 * ?path=/foo -> [[true, "file1"], [false, "dir1"]]
		 */
		case "/dir":
			// find contents of the directory
			contents, err := sftp.ReadDir(r.URL.Query().Get("path"))
			if resError(w, err) { return }
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
			if resError(w, err) { return }
			http.ServeContent(w, r, path, time.Time{}, contents)

		/*
		 * Creates a new directory on the server.
		 * The 'path' query parameter identifies the directory to make.
		 */
		case "/newdir":
			// stream the file contents
			err = sftp.Mkdir(r.URL.Query().Get("path"))
			if resError(w, err) { return }

		/*
		 * Upload files to the server.
		 * The path specified is the directory to upload to.
		 * Files are recieved from a 'files[]' form parameter
		 * sent in the body of the request.
		 */
		case "/upload":
			err := r.ParseMultipartForm(100 << 20) // 100MB in memory
			if resError(w, err) { return }
			for _, file := range r.MultipartForm.File["files[]"] {
				source, err := file.Open()
				if resError(w, err) { return }
				defer source.Close()
				dest, err := sftp.Create(r.URL.Query().Get("path")+"/"+file.Filename)
				if resError(w, err) { return }
				defer dest.Close()
				_, err = io.Copy(dest, source)
				if resError(w, err) { return }
			}
	}
}

func main() {
	http.HandleFunc("/", urlHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
