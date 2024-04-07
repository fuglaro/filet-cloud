package main

import (
	"archive/zip"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/crypto/ssh"
)

var sshport = "22"
var upgrader = websocket.Upgrader{}
var privateKey = make([]byte, 512/8)
var connectionID atomic.Uint64 // sequential ID generator making keys for connections.
var connections = map[uint64]*ssh.Client{}

// Attempt to find the Client IP (without the port) for an incomming request.
func clientIP(r *http.Request) string {
	ip := r.RemoteAddr
	for _, header := range []string{"X-Client-IP", "X-Forwarded-For", "X-Real-IP"} {
		v := r.Header.Get(header)
		if v != "" {
			ip = v
			break
		}
	}
	return strings.SplitN(strings.SplitN(ip, ",", 2)[0], ":", 2)[0]
}

/**
 * Checks if the passed in error has a value and, if it does,
 * a StatusForbidden error is provided to the response.
 * For the purposes of this webserver, where we are exposing
 * files via SFTP, assuming any error relates to a permission
 * issue, is sufficient. It breaks some HTTP conventions but
 * its nice and simple.
 * Returns whether the error had a value.
 */
func check(w http.ResponseWriter, e error) bool {
	if e != nil {
		http.Error(w, e.Error(), http.StatusForbidden)
	}
	return e != nil
}

/**
 * Endpoints for content that is served for browser access like
 * images, streaming or downloads, and which requires authentication
 * to be previously established through the WebSocket connection.
 */
func authServeContent(w http.ResponseWriter, r *http.Request) {
	// Ensure Secure Fetch Metadata validity.
	if r.Header.Get("Sec-Fetch-Site") != "same-origin" ||
		(r.Header.Get("Sec-Fetch-Dest") != "audio" &&
			r.Header.Get("Sec-Fetch-Dest") != "image" &&
			r.Header.Get("Sec-Fetch-Dest") != "video" &&
			r.Header.Get("Sec-Fetch-Dest") != "document") {
		http.Error(w, "Invalid Secure Fetch Metadata", http.StatusForbidden)
		return
	}
	// Ensure authentication is successfull and get storage connection.
	ts, err := r.Cookie("__Host-Auth")
	if check(w, err) {
		return
	}
	token, err := jwt.Parse(ts.Value,
		func(token *jwt.Token) (interface{}, error) {
			return privateKey, nil
		},
		jwt.WithValidMethods([]string{"HS512"}),
		jwt.WithAudience(clientIP(r)),
		jwt.WithExpirationRequired())
	if check(w, err) {
		return
	}
	cIDs, err := token.Claims.GetSubject()
	if check(w, err) {
		return
	}
	cID, err := strconv.ParseUint(cIDs, 10, 64)
	sshConn := connections[cID]
	if sshConn == nil {
		http.Error(w, "Invalid authentication token.", http.StatusForbidden)
		return
	}
	// Wrap SSH connection with SFTP interface.
	sftp, err := sftp.NewClient(sshConn)
	if check(w, err) {
		return
	}
	defer sftp.Close()
	user := sshConn.Conn.User()
	prepath := strings.Replace(os.Getenv("FC_DIR"), "USERNAME", string(user), -1) + "/"

	components := strings.SplitN(r.URL.Path, ":", 2)
	switch components[0] {

	/*
	 * Retrieves a file and sends it to the client.
	 * The 'path' query parameter identifies the file to send.
	 */
	case "/file":
		path := prepath + components[1]
		// get the file stat information
		stat, err := sftp.Stat(path)
		if check(w, err) {
			return
		}
		// stream the file contents
		contents, err := sftp.Open(path)
		if check(w, err) {
			return
		}
		defer contents.Close()
		http.ServeContent(w, r, filepath.Base(path), stat.ModTime(), contents)

	/*
	 * Serve a thumbnail image of the file.
	 * This does not support all formats.
	 */
	case "/thumb":
		w.Header().Set("Content-Type", "image/jpeg")
		// Single quoted POSIX command argument input sanitisation,
		// necessary due to needing to travel through the ssh stream.
		ppath := strings.Replace(prepath+components[1], "'", "'\\''", -1)
		cmd := "ffmpeg -i '" + ppath + "' -q:v 16 -vf scale=240:-1 -update 1 -f image2 -vcodec mjpeg -"
		session, err := sshConn.NewSession()
		if check(w, err) {
			return
		}
		defer session.Close()
		session.Stdout = w
		err = session.Run(cmd)
		if check(w, err) {
			return
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
		for _, upath := range r.URL.Query()["path"] {
			path := prepath + upath
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
	}
}

func connect(w http.ResponseWriter, r *http.Request) {
	// Ensure Secure Fetch Metadata validity.
	if r.Header.Get("Sec-Fetch-Site") != "same-origin" ||
		r.Header.Get("Sec-Fetch-Mode") != "websocket" ||
		r.Header.Get("Sec-Fetch-Dest") != "empty" {
		http.Error(w, "Invalid Secure Fetch Metadata", http.StatusForbidden)
		return
	}
	csrfcookie, err := r.Cookie("__Host-CSRFToken")
	if check(w, err) {
		return
	}
	// Establish Websocket connection.
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close()
	_, csrf64, err := c.ReadMessage()
	if err != nil {
		return
	}
	// Validate the CSRF Token
	csrf, err := base64.URLEncoding.DecodeString(string(csrf64))
	if err != nil {
		return
	}
	csrfhash, err := base64.URLEncoding.DecodeString(string(csrfcookie.Value))
	if err != nil {
		return
	}
	hash := hmac.New(sha256.New, privateKey)
	hash.Write(csrf)
	if !hmac.Equal([]byte(csrfhash), hash.Sum(nil)) {
		return
	}
	// Authenticate and establish SSH connection.
	_, user, err := c.ReadMessage()
	if err != nil {
		return
	}
	_, pass, err := c.ReadMessage()
	if err != nil {
		return
	}
	_, code, err := c.ReadMessage()
	if err != nil {
		return
	}
	sshConn, err := ssh.Dial("tcp", "localhost:"+sshport, &ssh.ClientConfig{
		User: string(user),
		Auth: []ssh.AuthMethod{
			ssh.Password(string(pass)),
			ssh.KeyboardInteractive(
				func(name, instruction string, question []string, echos []bool) (answers []string, err error) {
					return []string{string(code)}, nil
				})},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // trust localhost
	})
	if err != nil {
		return
	}
	defer sshConn.Close()

	// Setup for management of the shell sessions.
	var sessRunning = false
	var sess *ssh.Session
	var sessIn io.WriteCloser
	runSession := func() {
		if sessRunning { // Reuse existing sessions if available.
			return
		}
		sess, err = sshConn.NewSession()
		if err != nil {
			c.Close()
			return
		}
		sessIn, err = sess.StdinPipe()
		if err != nil {
			sess.Close()
			c.Close()
			return
		}
		sessOut, err := sess.StdoutPipe()
		if err != nil {
			sess.Close()
			c.Close()
			return
		}
		sess.Stderr = os.Stderr
		_ = sess.Setenv("COLORTERM", "truecolor")
		if sess.RequestPty("xterm-256color", 80, 40, nil) != nil || sess.Shell() != nil {
			sess.Close()
			c.Close()
			return
		}
		sessRunning = true
		go func() {
			buf := make([]byte, 20000)
			for {
				// Ferry data coming out of the ssh shell into the websocket.
				mw, err := c.NextWriter(websocket.BinaryMessage)
				if err != nil {
					sess.Close()
					c.Close()
					return
				}
				mw.Write([]byte("-1                  ")) // term header
				n, err := sessOut.Read(buf)
				mw.Write(buf[:n])
				if mw.Close() != nil {
					sess.Close()
					c.Close()
					return
				}
				if err != nil {
					sessRunning = false
					sess.Close()
					if err != io.EOF {
						c.Close()
						return
					}
					// Send a term closed message
					if c.WriteMessage(websocket.BinaryMessage, []byte("-1                  ")) != nil {
						c.Close()
					}
					return
				}
			}
		}()
	}

	// Wrap SSH connection with SFTP interface.
	sftp, err := sftp.NewClient(sshConn)
	if err != nil {
		return
	}
	// Associate the connection with a unique ID for subsequent authenticated access.
	connID := connectionID.Add(1)
	connections[connID] = sshConn
	// Ensure the connection is cleared when the WebSocket connection closes.
	defer delete(connections, connID)
	// Handle messages on established authenticated connection.
	type Msg struct {
		Action string
		Data   string
		Path   string
		To     string
		Id     int
		Rows   int
		Cols   int
	}
	prepath := strings.Replace(os.Getenv("FC_DIR"), "USERNAME", string(user), -1) + "/"
	for {
		mtype, re, err := c.NextReader()
		if err != nil {
			return
		}
		if mtype == websocket.BinaryMessage {

			/* Handle file upload.
			 * Unpack the message header to get the path name then store the rest. */
			idbuf := make([]byte, 20)
			if _, err = io.ReadFull(re, idbuf); err != nil {
				return
			}
			id, err := strconv.Atoi(strings.TrimSpace(string(idbuf)))
			if err != nil {
				return
			}
			pathlenbuf := make([]byte, 20)
			if _, err = io.ReadFull(re, pathlenbuf); err != nil {
				return
			}
			pathlen, err := strconv.Atoi(strings.TrimSpace(string(pathlenbuf)))
			if err != nil {
				return
			}
			pathbuf := make([]byte, pathlen)
			if _, err = io.ReadFull(re, pathbuf); err != nil {
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
			if _, err = io.Copy(dest, re); err != nil {
				return
			}
			if c.WriteJSON(map[string]interface{}{"id": id}) != nil {
				return
			}
			continue
		}

		m := Msg{}
		mstr, err := io.ReadAll(re)
		err = json.Unmarshal(mstr, &m)
		if err != nil {
			return
		}
	action:
		switch m.Action {
		// Prepares and sends an authentication JWT
		// for allowing authenticated access to authServeContent.
		case "token":
			t := jwt.NewWithClaims(jwt.SigningMethodHS512, &jwt.RegisteredClaims{
				Audience:  jwt.ClaimStrings{clientIP(r)},
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 5)),
				Subject:   strconv.FormatUint(connID, 10)})
			s, err := t.SignedString(privateKey)
			if err != nil {
				return
			}
			if c.WriteJSON(map[string]interface{}{"id": m.Id, "msg": s}) != nil {
				return
			}

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

		case "termData":
			if m.Rows == -1 {
				runSession()
				continue
			}
			if !sessRunning {
				continue // Probaly just ignorable leftover messages in transit after an ended session.
			}
			if m.Rows != 0 {
				sess.WindowChange(m.Rows, m.Cols)
				continue
			}
			_, err = sessIn.Write([]byte(m.Data))
			if err != nil {
				if !sessRunning {
					continue
				}
				return
			}

		default:
			return
		}
	}
}

func main() {
	// Handle options.
	p := os.Getenv("FC_SSH_PORT")
	if p != "" {
		sshport = p
	}
	addr := os.Getenv("FC_LISTEN")
	if addr == "" {
		addr = ":443"
	}
	cert := os.Getenv("FC_CERT_FILE")
	key := os.Getenv("FC_KEY_FILE")
	domain := os.Getenv("FC_DOMAIN")
	if domain == "" && (cert == "" || key == "") {

		fmt.Print(`
filet-cloud: The lean and powerful 💪 personal cloud ⛅.

Usage (environment variables):

  FC_CERT_FILE:
  FC_KEY_FILE:
    The credentials to use for TLS connections.
  FC_DIR:
    The folder path to use when serving storage, rather than the root.
    Supports a USERNAME token to serve a different tree for each user.
  FC_DOMAIN:
    The domain to use with the included Let's Encrypt integration.
    Use of this implies acceptance of the LetsEncrypt Terms of Service.
  FC_LISTEN:
    The address to listen on. Defaults to :443.
  FC_SSH_PORT:
    The port to use to connect locally.

This service can only be served over HTTPS connections, requiring
either FC_CERT_FILE and FC_KEY_FILE to be specified, or,
if you accept the LetsEncrypt Terms of Service, you can use the
automatic LetsEncrypt configuration by specifying FC_DOMAIN.

`)
		return
	}

	// Set up the standard security middleware.
	SMW := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
			w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Vary", "Sec-Fetch-Dest, Sec-Fetch-Mode, Sec-Fetch-Site")
			w.Header().Set("Cache-Control", "max-age=36000")
			next.ServeHTTP(w, r)
		})
	}

	// Redirect HTTP to HTTPS.
	go func() {
		log.Fatal(http.ListenAndServe(":8080",
			SMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				redirect := "https://" + strings.Split(r.Host, ":")[0]
				if strings.Contains(addr, ":") {
					redirect += ":" + strings.Split(addr, ":")[1]
				}
				http.Redirect(w, r, redirect+r.URL.Path, http.StatusTemporaryRedirect)
			}))))
	}()

	// Generate private key for JWT signing.
	_, err := rand.Read(privateKey)
	if err != nil {
		log.Fatal("Failed to generate cryptographically secure pseudorandom private JWT signing key.")
		return
	}

	// Serve all endpoints.
	http.Handle("/", SMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/browse:/", http.StatusSeeOther)
	})))
	http.Handle("/connect", SMW(http.HandlerFunc(connect)))
	http.Handle("/authenticate", SMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Ensure Secure Fetch Metadata validity.
		if r.Header.Get("Sec-Fetch-Site") != "same-origin" ||
			r.Header.Get("Sec-Fetch-Dest") != "empty" {
			http.Error(w, "Invalid Secure Fetch Metadata", http.StatusForbidden)
			return
		}
		// Register the rebounded JWT as a secure authentication cookie
		// for allowing authenticated access to authServeContent.
		b, err := io.ReadAll(r.Body)
		if check(w, err) {
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "__Host-Auth",
			Value:    string(b),
			MaxAge:   300,
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})
	})))
	http.Handle("/logout", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Clear-Site-Data", "\"*\"")
		http.Redirect(w, r, "/browse:/", http.StatusSeeOther)
	}))
	http.Handle("/file:/", SMW(http.HandlerFunc(authServeContent)))
	http.Handle("/thumb:/", SMW(http.HandlerFunc(authServeContent)))
	http.Handle("/zip", SMW(http.HandlerFunc(authServeContent)))
	mainPage := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Ensure Secure Fetch Metadata validity.
		if r.Header.Get("Sec-Fetch-Dest") != "document" {
			http.Error(w, "Invalid Secure Fetch Metadata", http.StatusForbidden)
			return
		}
		// Set up security cookies.
		nonceb := make([]byte, 128/8)
		_, err = rand.Read(nonceb)
		if check(w, err) {
			return
		}
		nonce := base64.URLEncoding.EncodeToString(nonceb)
		w.Header().Set("Content-Security-Policy", "sandbox allow-downloads allow-forms "+
			"allow-same-origin allow-scripts; default-src 'none'; frame-ancestors 'none'; "+
			"form-action 'none'; img-src 'self'; media-src 'self'; font-src 'self'; "+
			"connect-src 'self'; style-src-elem 'self' 'unsafe-inline'; "+
			"style-src-attr 'unsafe-inline'; "+
			"script-src-elem 'self' 'nonce-"+nonce+"';")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Embedder-Policy", "require-corp")
		w.Header().Set("Referrer-Policy", "same-origin")
		t, err := template.ParseFiles("static/main.html")
		if check(w, err) {
			return
		}
		// Prepare a Singed Double Submit Cookie CSRF Token.
		var hashData = make([]byte, 512/8)
		_, err = rand.Read(hashData)
		if err != nil {
			log.Fatal("Failed to generate cryptographically secure CSRF random identifier.")
			return
		}
		hash := hmac.New(sha256.New, privateKey)
		hash.Write(hashData)
		http.SetCookie(w, &http.Cookie{
			Name:     "__Host-CSRFToken",
			Value:    base64.URLEncoding.EncodeToString(hash.Sum(nil)),
			Path:     "/",
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})
		w.Header().Set("Cache-Control", "no-cache")
		// Resolve template and send.
		t.Execute(w, struct {
			Nonce string
			CSRF  string
		}{Nonce: nonce, CSRF: base64.URLEncoding.EncodeToString(hashData)})
	})
	http.Handle("/browse:/", SMW(mainPage))
	http.Handle("/open:/", SMW(mainPage))
	http.Handle("/static/", SMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Ensure Secure Fetch Metadata validity.
		if r.Header.Get("Sec-Fetch-Site") != "same-origin" ||
			(r.Header.Get("Sec-Fetch-Dest") != "script" &&
				r.Header.Get("Sec-Fetch-Dest") != "style" &&
				r.Header.Get("Sec-Fetch-Dest") != "font") {
			http.Error(w, "Invalid Secure Fetch Metadata", http.StatusForbidden)
			return
		}
		http.StripPrefix("/static/", http.FileServer(http.Dir("static"))).ServeHTTP(w, r)
	})))
	http.Handle("/favicon.ico", SMW(http.FileServer(http.Dir("static"))))

	fmt.Fprintf(os.Stderr, "FC_CERT_FILE=%v\n", cert)
	fmt.Fprintf(os.Stderr, "FC_KEY_FILE=%v\n", key)
	fmt.Fprintf(os.Stderr, "FC_DIR=%v\n", os.Getenv("FC_DIR"))
	fmt.Fprintf(os.Stderr, "FC_DOMAIN=%v\n", domain)
	fmt.Fprintf(os.Stderr, "FC_LISTEN=%v\n", addr)
	fmt.Fprintf(os.Stderr, "FC_SSH_PORT=%v\n", sshport)
	fmt.Fprintf(os.Stderr, "\nListening...\n")
	if os.Getenv("FC_DOMAIN") != "" {
		log.Fatal(http.Serve(autocert.NewListener(domain), nil))
	} else {
		log.Fatal(http.ListenAndServeTLS(addr, cert, key, nil))
	}
}
